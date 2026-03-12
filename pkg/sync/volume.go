package sync

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"sync"
	"time"

	"haya-tab/pkg/store"

	"github.com/google/uuid"
)

const (
	// MetadataDirectoryName is the hidden directory that stores fingerprint buckets
	MetadataDirectoryName = "haya-metadata"
	// BucketCount is the number of hash buckets for distributing fingerprint files
	BucketCount = 16
	// BucketFilePrefix is the prefix for bucket files (bucket-00.json to bucket-15.json)
	BucketFilePrefix = "bucket-"
	// LegacyFingerprintFileName is the old fingerprint file name (kept for migration)
	LegacyFingerprintFileName = ".haya-volume-fingerprint"
)

// VolumeFingerprint represents the content of a volume fingerprint file
type VolumeFingerprint struct {
	VolumeID    string              `json:"volume_id"`     // Unique identifier for this volume
	VolumeName  string              `json:"volume_name"`   // User-friendly name
	CreatedAt   string              `json:"created_at"`    // ISO 8601 timestamp
	AppVersion  string              `json:"app_version"`   // Version of the app that created this
	DeviceName  string              `json:"device_name"`   // Name of the device that created this (optional)
	LastUpdated string              `json:"last_updated"`  // ISO 8601 timestamp of last update
	Files       []FingerprintFile   `json:"files"`         // List of files uploaded via the app
}

// FingerprintFile represents metadata for a file uploaded to this volume
type FingerprintFile struct {
	RelativePath string   `json:"relative_path"` // Path relative to volume root
	Title        string   `json:"title"`         // Tab title
	Artist       string   `json:"artist"`        // Artist name
	Album        string   `json:"album"`         // Album name
	Type         string   `json:"type"`          // File type (pdf, gp, etc.)
	Categories   []string `json:"categories"`    // List of category names
	UploadedAt   string   `json:"uploaded_at"`   // ISO 8601 timestamp
	UploadedBy   string   `json:"uploaded_by"`   // Device name that uploaded this file
}

// FingerprintMetadata stores volume-level metadata (stored in bucket-00.json)
type FingerprintMetadata struct {
	VolumeID    string `json:"volume_id"`    // Unique identifier for this volume
	VolumeName  string `json:"volume_name"`  // User-friendly name
	CreatedAt   string `json:"created_at"`   // ISO 8601 timestamp
	AppVersion  string `json:"app_version"`  // Version of the app that created this
	DeviceName  string `json:"device_name"`  // Name of the device that created this (optional)
	LastUpdated string `json:"last_updated"` // ISO 8601 timestamp of last update
	BucketCount int    `json:"bucket_count"` // Always 16
}

// BucketData stores files for a specific bucket
type BucketData struct {
	BucketNumber int                 `json:"bucket_number"` // Bucket number (0-15)
	Files        []FingerprintFile   `json:"files"`         // Files in this bucket
}

// CalculateBucketNumber calculates which bucket (0-15) a file should be stored in
// based on the MD5 hash of its relative path
func CalculateBucketNumber(relativePath string) int {
	hash := md5.Sum([]byte(relativePath))
	// Use last byte of hash, mod 16
	return int(hash[15] % BucketCount)
}

// getMetadataPath returns the path to the metadata directory
func getMetadataPath(volumePath string) string {
	return path.Join(volumePath, MetadataDirectoryName)
}

// getBucketPath returns the path to a specific bucket file
func getBucketPath(volumePath string, bucketNum int) string {
	filename := fmt.Sprintf("%s%02d.json", BucketFilePrefix, bucketNum)
	return path.Join(volumePath, MetadataDirectoryName, filename)
}

// getLegacyFingerprintPath returns the path to the legacy fingerprint file
func getLegacyFingerprintPath(volumePath string) string {
	return path.Join(volumePath, LegacyFingerprintFileName)
}

// CreateVolumeFingerprint creates a new fingerprint file at the specified path
func (c *WebDAVClient) CreateVolumeFingerprint(remotePath, volumeName, appVersion, deviceName string) (*VolumeFingerprint, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	volumeID := uuid.New().String()

	// Create metadata directory (recursively)
	metadataPath := getMetadataPath(remotePath)
	if err := c.MkdirAll(metadataPath); err != nil {
		return nil, fmt.Errorf("failed to create metadata directory: %w", err)
	}

	// Create metadata
	metadata := &FingerprintMetadata{
		VolumeID:    volumeID,
		VolumeName:  volumeName,
		CreatedAt:   now,
		AppVersion:  appVersion,
		DeviceName:  deviceName,
		LastUpdated: now,
		BucketCount: BucketCount,
	}

	// Write metadata to bucket-00.json (with empty files)
	if err := c.WriteMetadata(remotePath, metadata); err != nil {
		return nil, fmt.Errorf("failed to write metadata: %w", err)
	}

	// Initialize empty buckets (bucket-01.json to bucket-15.json)
	for i := 1; i < BucketCount; i++ {
		bucket := &BucketData{
			BucketNumber: i,
			Files:        []FingerprintFile{},
		}
		if err := c.WriteBucket(remotePath, i, bucket); err != nil {
			return nil, fmt.Errorf("failed to initialize bucket %d: %w", i, err)
		}
	}

	// Return VolumeFingerprint for compatibility
	return &VolumeFingerprint{
		VolumeID:    volumeID,
		VolumeName:  volumeName,
		CreatedAt:   now,
		AppVersion:  appVersion,
		DeviceName:  deviceName,
		LastUpdated: now,
		Files:       []FingerprintFile{},
	}, nil
}

// ReadVolumeFingerprint reads and parses a fingerprint file from WebDAV
func (c *WebDAVClient) ReadVolumeFingerprint(remotePath string) (*VolumeFingerprint, error) {
	// Check for legacy format first (migration path)
	if c.legacyFingerprintExists(remotePath) {
		return c.migrateLegacyFingerprint(remotePath)
	}

	// Read metadata from bucket-00
	metadata, err := c.ReadMetadata(remotePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	// Read all buckets and merge files concurrently
	var allFiles []FingerprintFile
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Fetch buckets concurrently with a limited concurrency to avoid overwhelming the server
	sem := make(chan struct{}, 8)

	for i := 0; i < BucketCount; i++ {
		wg.Add(1)
		go func(bucketNum int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			bucket, err := c.ReadBucket(remotePath, bucketNum)
			if err != nil {
				// Skip missing buckets (they might be empty)
				return
			}
			
			mu.Lock()
			allFiles = append(allFiles, bucket.Files...)
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	// Return merged VolumeFingerprint
	return &VolumeFingerprint{
		VolumeID:    metadata.VolumeID,
		VolumeName:  metadata.VolumeName,
		CreatedAt:   metadata.CreatedAt,
		AppVersion:  metadata.AppVersion,
		DeviceName:  metadata.DeviceName,
		LastUpdated: metadata.LastUpdated,
		Files:       allFiles,
	}, nil
}

// UpdateVolumeFingerprint updates the fingerprint by distributing files to buckets
// This function is kept for backward compatibility but now uses the bucket mechanism
func (c *WebDAVClient) UpdateVolumeFingerprint(remotePath string, fingerprint *VolumeFingerprint) error {
	now := time.Now().UTC().Format(time.RFC3339)

	// Update metadata
	metadata := &FingerprintMetadata{
		VolumeID:    fingerprint.VolumeID,
		VolumeName:  fingerprint.VolumeName,
		CreatedAt:   fingerprint.CreatedAt,
		AppVersion:  fingerprint.AppVersion,
		DeviceName:  fingerprint.DeviceName,
		LastUpdated: now,
		BucketCount: BucketCount,
	}

	// Distribute files to buckets
	bucketFiles := make(map[int][]FingerprintFile)
	for _, file := range fingerprint.Files {
		bucketNum := CalculateBucketNumber(file.RelativePath)
		bucketFiles[bucketNum] = append(bucketFiles[bucketNum], file)
	}

	// Ensure metadata directory exists
	metadataPath := getMetadataPath(remotePath)
	if err := c.MkdirAll(metadataPath); err != nil {
		return fmt.Errorf("failed to create metadata directory: %w", err)
	}

	// Write all buckets
	for i := 0; i < BucketCount; i++ {
		bucket := &BucketData{
			BucketNumber: i,
			Files:        bucketFiles[i],
		}
		if bucket.Files == nil {
			bucket.Files = []FingerprintFile{}
		}

		if i == 0 {
			// Write metadata with bucket 0
			if err := c.WriteMetadata(remotePath, metadata); err != nil {
				return fmt.Errorf("failed to write metadata: %w", err)
			}
			// Update bucket 0 files
			if err := c.WriteBucket(remotePath, i, bucket); err != nil {
				return fmt.Errorf("failed to write bucket %d: %w", i, err)
			}
		} else {
			if err := c.WriteBucket(remotePath, i, bucket); err != nil {
				return fmt.Errorf("failed to write bucket %d: %w", i, err)
			}
		}
	}

	return nil
}

// FingerprintExists checks if a fingerprint file exists at the specified path
func (c *WebDAVClient) FingerprintExists(remotePath string) bool {
	// Check for new format (metadata directory)
	metadataPath := getMetadataPath(remotePath)
	_, err := c.metadataClient.Stat(metadataPath)
	if err == nil {
		return true
	}

	// Check for legacy format
	return c.legacyFingerprintExists(remotePath)
}

// legacyFingerprintExists checks if a legacy fingerprint file exists
func (c *WebDAVClient) legacyFingerprintExists(remotePath string) bool {
	legacyPath := getLegacyFingerprintPath(remotePath)
	_, err := c.metadataClient.Stat(legacyPath)
	return err == nil
}

// ScanVolumes scans the WebDAV root directory for all volumes (directories with fingerprint files)
// Returns a map of mount_path -> VolumeFingerprint
// OPTIMIZED: Only reads metadata, not all bucket files, to speed up discovery
func (c *WebDAVClient) ScanVolumes(rootPath string) (map[string]*VolumeFingerprint, error) {
	volumes := make(map[string]*VolumeFingerprint)

	// List directories in root
	dirs, err := c.ListRemoteDirectories(rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to list directories: %w", err)
	}

	// Check root itself for fingerprint
	if c.FingerprintExists(rootPath) {
		fingerprint, err := c.ReadVolumeFingerprintMetadataOnly(rootPath)
		if err == nil {
			volumes[rootPath] = fingerprint
		}
	}

	// Check each subdirectory for fingerprint
	for _, dir := range dirs {
		if c.FingerprintExists(dir) {
			fingerprint, err := c.ReadVolumeFingerprintMetadataOnly(dir)
			if err != nil {
				fmt.Printf("[Warning] Failed to read fingerprint metadata at %s: %v\n", dir, err)
				continue
			}
			volumes[dir] = fingerprint
		}
	}

	return volumes, nil
}

// ReadVolumeFingerprintMetadataOnly reads only the metadata without loading all files
// This is much faster for volume discovery where we don't need the file list
func (c *WebDAVClient) ReadVolumeFingerprintMetadataOnly(remotePath string) (*VolumeFingerprint, error) {
	// Check for legacy format first (migration path)
	if c.legacyFingerprintExists(remotePath) {
		return c.migrateLegacyFingerprint(remotePath)
	}

	// Read metadata from bucket-00
	metadata, err := c.ReadMetadata(remotePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	// Return VolumeFingerprint with empty files list (we only need metadata for discovery)
	return &VolumeFingerprint{
		VolumeID:    metadata.VolumeID,
		VolumeName:  metadata.VolumeName,
		CreatedAt:   metadata.CreatedAt,
		AppVersion:  metadata.AppVersion,
		DeviceName:  metadata.DeviceName,
		LastUpdated: metadata.LastUpdated,
		Files:       []FingerprintFile{}, // Empty - not needed for discovery
	}, nil
}

// RegisterOrUpdateVolume registers a discovered volume in the database or updates if it exists
// This handles the multi-device sync scenario where a device discovers an existing volume
func RegisterOrUpdateVolume(db *store.DBStore, mountPath string, fingerprint *VolumeFingerprint) (*store.CloudVolume, error) {
	// Check if volume already exists in database
	existingVolume, err := db.GetVolume(fingerprint.VolumeID)
	if err != nil {
		return nil, fmt.Errorf("failed to query volume: %w", err)
	}

	now := time.Now().Unix()

	if existingVolume != nil {
		// Volume exists - update mount path and last seen time
		existingVolume.MountPath = mountPath
		existingVolume.FingerprintPath = getMetadataPath(mountPath)
		existingVolume.LastSeenAt = now
		existingVolume.IsAvailable = true

		if err := db.UpdateVolume(*existingVolume); err != nil {
			return nil, fmt.Errorf("failed to update volume: %w", err)
		}

		return existingVolume, nil
	}

	// Volume doesn't exist - create new record
	newVolume := store.CloudVolume{
		ID:              fingerprint.VolumeID,
		Name:            fingerprint.VolumeName,
		MountPath:       mountPath,
		FingerprintPath: getMetadataPath(mountPath),
		CreatedAt:       now,
		LastSeenAt:      now,
		IsAvailable:     true,
	}

	if err := db.AddVolume(newVolume); err != nil {
		return nil, fmt.Errorf("failed to add volume: %w", err)
	}

	return &newVolume, nil
}

// AddFileToFingerprint adds a file record to the volume fingerprint
func (c *WebDAVClient) AddFileToFingerprint(remotePath string, file FingerprintFile) error {
	// Calculate bucket number
	bucketNum := CalculateBucketNumber(file.RelativePath)

	// Read bucket
	bucket, err := c.ReadBucket(remotePath, bucketNum)
	if err != nil {
		return fmt.Errorf("failed to read bucket %d: %w", bucketNum, err)
	}

	// Check if file already exists in the bucket
	fileExists := false
	for i, f := range bucket.Files {
		if f.RelativePath == file.RelativePath {
			// Update existing file record
			bucket.Files[i] = file
			fileExists = true
			break
		}
	}

	// Add new file if it doesn't exist
	if !fileExists {
		bucket.Files = append(bucket.Files, file)
	}

	// Write bucket back
	return c.WriteBucket(remotePath, bucketNum, bucket)
}

// RemoveFileFromFingerprint removes a file record from the volume fingerprint
func (c *WebDAVClient) RemoveFileFromFingerprint(remotePath, relativePath string) error {
	// Calculate bucket number
	bucketNum := CalculateBucketNumber(relativePath)

	// Read bucket
	bucket, err := c.ReadBucket(remotePath, bucketNum)
	if err != nil {
		return fmt.Errorf("failed to read bucket %d: %w", bucketNum, err)
	}

	// Remove the file from the bucket
	newFiles := make([]FingerprintFile, 0, len(bucket.Files))
	for _, f := range bucket.Files {
		if f.RelativePath != relativePath {
			newFiles = append(newFiles, f)
		}
	}
	bucket.Files = newFiles

	// Write bucket back
	return c.WriteBucket(remotePath, bucketNum, bucket)
}

// ReadMetadata reads the volume metadata from bucket-00.json
func (c *WebDAVClient) ReadMetadata(volumePath string) (*FingerprintMetadata, error) {
	bucketPath := getBucketPath(volumePath, 0)

	// Read from WebDAV
	stream, err := c.streamClient.ReadStream(bucketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata bucket: %w", err)
	}
	defer stream.Close()

	// Parse JSON
	data, err := io.ReadAll(stream)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata data: %w", err)
	}

	// Bucket 0 contains both metadata and files
	type Bucket0 struct {
		Metadata FingerprintMetadata `json:"metadata"`
		Files    []FingerprintFile   `json:"files"`
	}

	var bucket0 Bucket0
	if err := json.Unmarshal(data, &bucket0); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	return &bucket0.Metadata, nil
}

// WriteMetadata writes the volume metadata to bucket-00.json
func (c *WebDAVClient) WriteMetadata(volumePath string, metadata *FingerprintMetadata) error {
	bucketPath := getBucketPath(volumePath, 0)

	// Read existing bucket 0 to preserve files
	var existingFiles []FingerprintFile
	bucket, err := c.ReadBucket(volumePath, 0)
	if err == nil {
		existingFiles = bucket.Files
	}

	// Bucket 0 contains both metadata and files
	type Bucket0 struct {
		Metadata FingerprintMetadata `json:"metadata"`
		Files    []FingerprintFile   `json:"files"`
	}

	bucket0 := Bucket0{
		Metadata: *metadata,
		Files:    existingFiles,
	}

	// Serialize to JSON (OPTIMIZED: removed Indent to save space)
	data, err := json.Marshal(bucket0)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Create temp file
	tempFile, err := os.CreateTemp("", "haya-metadata-*.json")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())

	// Write to temp file
	if _, err := tempFile.Write(data); err != nil {
		tempFile.Close()
		return fmt.Errorf("failed to write metadata: %w", err)
	}
	tempFile.Close()

	// Upload to WebDAV
	tempFileRead, err := os.Open(tempFile.Name())
	if err != nil {
		return fmt.Errorf("failed to open temp file: %w", err)
	}
	defer tempFileRead.Close()

	if err := c.streamClient.WriteStream(bucketPath, tempFileRead, 0644); err != nil {
		return fmt.Errorf("failed to upload metadata: %w", err)
	}

	return nil
}

// ReadBucket reads a specific bucket file
func (c *WebDAVClient) ReadBucket(volumePath string, bucketNum int) (*BucketData, error) {
	bucketPath := getBucketPath(volumePath, bucketNum)

	// Read from WebDAV
	stream, err := c.streamClient.ReadStream(bucketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read bucket %d: %w", bucketNum, err)
	}
	defer stream.Close()

	// Parse JSON
	data, err := io.ReadAll(stream)
	if err != nil {
		return nil, fmt.Errorf("failed to read bucket data: %w", err)
	}

	// Bucket 0 has special structure with metadata
	if bucketNum == 0 {
		type Bucket0 struct {
			Metadata FingerprintMetadata `json:"metadata"`
			Files    []FingerprintFile   `json:"files"`
		}
		var bucket0 Bucket0
		if err := json.Unmarshal(data, &bucket0); err != nil {
			return nil, fmt.Errorf("failed to parse bucket 0: %w", err)
		}
		return &BucketData{
			BucketNumber: 0,
			Files:        bucket0.Files,
		}, nil
	}

	// Other buckets have simple structure
	var bucket BucketData
	if err := json.Unmarshal(data, &bucket); err != nil {
		return nil, fmt.Errorf("failed to parse bucket %d: %w", bucketNum, err)
	}

	return &bucket, nil
}

// WriteBucket writes a specific bucket file
func (c *WebDAVClient) WriteBucket(volumePath string, bucketNum int, bucket *BucketData) error {
	bucketPath := getBucketPath(volumePath, bucketNum)

	var data []byte
	var err error

	// Bucket 0 has special structure with metadata
	if bucketNum == 0 {
		// Read existing metadata
		metadata, err := c.ReadMetadata(volumePath)
		if err != nil {
			// If metadata doesn't exist, create default
			metadata = &FingerprintMetadata{
				BucketCount: BucketCount,
			}
		}

		type Bucket0 struct {
			Metadata FingerprintMetadata `json:"metadata"`
			Files    []FingerprintFile   `json:"files"`
		}
		bucket0 := Bucket0{
			Metadata: *metadata,
			Files:    bucket.Files,
		}
		data, err = json.Marshal(bucket0)
	} else {
		// Other buckets have simple structure
		data, err = json.Marshal(bucket)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal bucket %d: %w", bucketNum, err)
	}

	// Create temp file
	tempFile, err := os.CreateTemp("", "haya-bucket-*.json")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())

	// Write to temp file
	if _, err := tempFile.Write(data); err != nil {
		tempFile.Close()
		return fmt.Errorf("failed to write bucket: %w", err)
	}
	tempFile.Close()

	// Upload to WebDAV
	tempFileRead, err := os.Open(tempFile.Name())
	if err != nil {
		return fmt.Errorf("failed to open temp file: %w", err)
	}
	defer tempFileRead.Close()

	if err := c.streamClient.WriteStream(bucketPath, tempFileRead, 0644); err != nil {
		return fmt.Errorf("failed to upload bucket %d: %w", bucketNum, err)
	}

	return nil
}

// migrateLegacyFingerprint migrates a legacy fingerprint file to the new bucket format
func (c *WebDAVClient) migrateLegacyFingerprint(remotePath string) (*VolumeFingerprint, error) {
	// Read old fingerprint file
	legacyPath := getLegacyFingerprintPath(remotePath)
	stream, err := c.streamClient.ReadStream(legacyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read legacy fingerprint: %w", err)
	}
	defer stream.Close()

	data, err := io.ReadAll(stream)
	if err != nil {
		return nil, fmt.Errorf("failed to read legacy fingerprint data: %w", err)
	}

	var oldFingerprint VolumeFingerprint
	if err := json.Unmarshal(data, &oldFingerprint); err != nil {
		return nil, fmt.Errorf("failed to parse legacy fingerprint: %w", err)
	}

	// Create new metadata directory (recursively)
	metadataPath := getMetadataPath(remotePath)
	if err := c.MkdirAll(metadataPath); err != nil {
		return nil, fmt.Errorf("failed to create metadata directory: %w", err)
	}

	// Write metadata to bucket-00
	metadata := &FingerprintMetadata{
		VolumeID:    oldFingerprint.VolumeID,
		VolumeName:  oldFingerprint.VolumeName,
		CreatedAt:   oldFingerprint.CreatedAt,
		AppVersion:  oldFingerprint.AppVersion,
		DeviceName:  oldFingerprint.DeviceName,
		LastUpdated: oldFingerprint.LastUpdated,
		BucketCount: BucketCount,
	}

	// Distribute files to buckets
	bucketFiles := make(map[int][]FingerprintFile)
	for _, file := range oldFingerprint.Files {
		bucketNum := CalculateBucketNumber(file.RelativePath)
		bucketFiles[bucketNum] = append(bucketFiles[bucketNum], file)
	}

	// Write all buckets
	for i := 0; i < BucketCount; i++ {
		bucket := &BucketData{
			BucketNumber: i,
			Files:        bucketFiles[i], // Empty slice if no files
		}

		// For bucket 0, we need to include metadata
		if i == 0 {
			// Write metadata first
			if err := c.WriteMetadata(remotePath, metadata); err != nil {
				return nil, fmt.Errorf("failed to write metadata: %w", err)
			}
			// Then update bucket 0 with files
			if len(bucket.Files) > 0 {
				if err := c.WriteBucket(remotePath, i, bucket); err != nil {
					return nil, fmt.Errorf("failed to write bucket %d: %w", i, err)
				}
			}
		} else {
			if err := c.WriteBucket(remotePath, i, bucket); err != nil {
				return nil, fmt.Errorf("failed to write bucket %d: %w", i, err)
			}
		}
	}

	// Delete old fingerprint file
	if err := c.streamClient.Remove(legacyPath); err != nil {
		// Log warning but don't fail migration
		fmt.Printf("[Warning] Failed to delete legacy fingerprint at %s: %v\n", legacyPath, err)
	}

	// Return migrated fingerprint
	return &oldFingerprint, nil
}
