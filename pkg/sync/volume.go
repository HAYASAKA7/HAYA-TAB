package sync

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"time"

	"haya-tab/pkg/store"

	"github.com/google/uuid"
)

const (
	// FingerprintFileName is the name of the fingerprint file placed at volume roots
	FingerprintFileName = ".haya-volume-fingerprint"
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
	RelativePath string `json:"relative_path"` // Path relative to volume root
	Title        string `json:"title"`         // Tab title
	Artist       string `json:"artist"`        // Artist name
	Album        string `json:"album"`         // Album name
	Type         string `json:"type"`          // File type (pdf, gp, etc.)
	UploadedAt   string `json:"uploaded_at"`   // ISO 8601 timestamp
	UploadedBy   string `json:"uploaded_by"`   // Device name that uploaded this file
}

// CreateVolumeFingerprint creates a new fingerprint file at the specified path
func (c *WebDAVClient) CreateVolumeFingerprint(remotePath, volumeName, appVersion, deviceName string) (*VolumeFingerprint, error) {
	fingerprint := &VolumeFingerprint{
		VolumeID:    uuid.New().String(),
		VolumeName:  volumeName,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		AppVersion:  appVersion,
		DeviceName:  deviceName,
		LastUpdated: time.Now().UTC().Format(time.RFC3339),
	}

	// Serialize to JSON
	data, err := json.MarshalIndent(fingerprint, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal fingerprint: %w", err)
	}

	// Create temp file
	tempFile, err := os.CreateTemp("", "haya-fingerprint-*.json")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())

	// Write to temp file
	if _, err := tempFile.Write(data); err != nil {
		tempFile.Close()
		return nil, fmt.Errorf("failed to write fingerprint: %w", err)
	}
	tempFile.Close()

	// Upload to WebDAV
	fingerprintPath := path.Join(remotePath, FingerprintFileName)
	tempFileRead, err := os.Open(tempFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to open temp file: %w", err)
	}
	defer tempFileRead.Close()

	if err := c.streamClient.WriteStream(fingerprintPath, tempFileRead, 0644); err != nil {
		return nil, fmt.Errorf("failed to upload fingerprint: %w", err)
	}

	return fingerprint, nil
}

// ReadVolumeFingerprint reads and parses a fingerprint file from WebDAV
func (c *WebDAVClient) ReadVolumeFingerprint(remotePath string) (*VolumeFingerprint, error) {
	fingerprintPath := path.Join(remotePath, FingerprintFileName)

	// Read from WebDAV
	stream, err := c.streamClient.ReadStream(fingerprintPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read fingerprint: %w", err)
	}
	defer stream.Close()

	// Parse JSON
	data, err := io.ReadAll(stream)
	if err != nil {
		return nil, fmt.Errorf("failed to read fingerprint data: %w", err)
	}

	var fingerprint VolumeFingerprint
	if err := json.Unmarshal(data, &fingerprint); err != nil {
		return nil, fmt.Errorf("failed to parse fingerprint: %w", err)
	}

	return &fingerprint, nil
}

// UpdateVolumeFingerprint updates the last_updated timestamp of a fingerprint file
func (c *WebDAVClient) UpdateVolumeFingerprint(remotePath string, fingerprint *VolumeFingerprint) error {
	fingerprint.LastUpdated = time.Now().UTC().Format(time.RFC3339)

	// Serialize to JSON
	data, err := json.MarshalIndent(fingerprint, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal fingerprint: %w", err)
	}

	// Create temp file
	tempFile, err := os.CreateTemp("", "haya-fingerprint-*.json")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())

	// Write to temp file
	if _, err := tempFile.Write(data); err != nil {
		tempFile.Close()
		return fmt.Errorf("failed to write fingerprint: %w", err)
	}
	tempFile.Close()

	// Upload to WebDAV
	fingerprintPath := path.Join(remotePath, FingerprintFileName)
	tempFileRead, err := os.Open(tempFile.Name())
	if err != nil {
		return fmt.Errorf("failed to open temp file: %w", err)
	}
	defer tempFileRead.Close()

	if err := c.streamClient.WriteStream(fingerprintPath, tempFileRead, 0644); err != nil {
		return fmt.Errorf("failed to upload fingerprint: %w", err)
	}

	return nil
}

// FingerprintExists checks if a fingerprint file exists at the specified path
func (c *WebDAVClient) FingerprintExists(remotePath string) bool {
	fingerprintPath := path.Join(remotePath, FingerprintFileName)
	_, err := c.metadataClient.Stat(fingerprintPath)
	return err == nil
}

// ScanVolumes scans the WebDAV root directory for all volumes (directories with fingerprint files)
// Returns a map of mount_path -> VolumeFingerprint
func (c *WebDAVClient) ScanVolumes(rootPath string) (map[string]*VolumeFingerprint, error) {
	volumes := make(map[string]*VolumeFingerprint)

	// List directories in root
	dirs, err := c.ListRemoteDirectories(rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to list directories: %w", err)
	}

	// Check root itself for fingerprint
	if c.FingerprintExists(rootPath) {
		fingerprint, err := c.ReadVolumeFingerprint(rootPath)
		if err == nil {
			volumes[rootPath] = fingerprint
		}
	}

	// Check each subdirectory for fingerprint
	for _, dir := range dirs {
		if c.FingerprintExists(dir) {
			fingerprint, err := c.ReadVolumeFingerprint(dir)
			if err != nil {
				fmt.Printf("[Warning] Failed to read fingerprint at %s: %v\n", dir, err)
				continue
			}
			volumes[dir] = fingerprint
		}
	}

	return volumes, nil
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
		existingVolume.FingerprintPath = path.Join(mountPath, FingerprintFileName)
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
		FingerprintPath: path.Join(mountPath, FingerprintFileName),
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
	// Read existing fingerprint
	fingerprint, err := c.ReadVolumeFingerprint(remotePath)
	if err != nil {
		return fmt.Errorf("failed to read fingerprint: %w", err)
	}

	// Check if file already exists in the list
	fileExists := false
	for i, f := range fingerprint.Files {
		if f.RelativePath == file.RelativePath {
			// Update existing file record
			fingerprint.Files[i] = file
			fileExists = true
			break
		}
	}

	// Add new file if it doesn't exist
	if !fileExists {
		fingerprint.Files = append(fingerprint.Files, file)
	}

	// Update the fingerprint file
	return c.UpdateVolumeFingerprint(remotePath, fingerprint)
}

// RemoveFileFromFingerprint removes a file record from the volume fingerprint
func (c *WebDAVClient) RemoveFileFromFingerprint(remotePath, relativePath string) error {
	// Read existing fingerprint
	fingerprint, err := c.ReadVolumeFingerprint(remotePath)
	if err != nil {
		return fmt.Errorf("failed to read fingerprint: %w", err)
	}

	// Remove the file from the list
	newFiles := make([]FingerprintFile, 0, len(fingerprint.Files))
	for _, f := range fingerprint.Files {
		if f.RelativePath != relativePath {
			newFiles = append(newFiles, f)
		}
	}
	fingerprint.Files = newFiles

	// Update the fingerprint file
	return c.UpdateVolumeFingerprint(remotePath, fingerprint)
}
