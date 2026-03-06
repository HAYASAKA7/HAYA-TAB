package app

import (
	"strings"
	"time"

	"haya-tab/pkg/store"
	syncpkg "haya-tab/pkg/sync"
)

// getFingerprintCache returns the fingerprint cache, creating it if needed
func (a *App) getFingerprintCache() *syncpkg.FingerprintCache {
	a.fingerprintCacheMu.Lock()
	defer a.fingerprintCacheMu.Unlock()

	if a.fingerprintCache == nil {
		settings := a.store.GetSettings()
		if !settings.WebDAVEnabled || settings.WebDAVURL == "" {
			return nil
		}

		client := syncpkg.NewWebDAVClient(
			strings.TrimRight(settings.WebDAVURL, "/"),
			settings.WebDAVUser,
			settings.WebDAVPassword,
		)

		// Create cache with 3 second flush interval for faster updates
		a.fingerprintCache = syncpkg.NewFingerprintCache(client, 3*time.Second)
		a.logger.Info("Fingerprint cache initialized with 3s flush interval")
	}

	return a.fingerprintCache
}

// removeFromFingerprint removes a single file from its volume's fingerprint
func (a *App) removeFromFingerprint(volumeID, relativePath string) {
	settings := a.store.GetSettings()
	if !settings.WebDAVEnabled || settings.WebDAVURL == "" {
		return
	}

	// Get volume info
	volume, err := a.store.GetVolume(volumeID)
	if err != nil || volume == nil {
		a.logger.Error("Failed to get volume %s: %v", volumeID, err)
		return
	}

	client := syncpkg.NewWebDAVClient(
		strings.TrimRight(settings.WebDAVURL, "/"),
		settings.WebDAVUser,
		settings.WebDAVPassword,
	)

	// Read current fingerprint
	fingerprint, err := client.ReadVolumeFingerprint(volume.MountPath)
	if err != nil {
		a.logger.Error("Failed to read fingerprint for volume %s: %v", volumeID, err)
		return
	}

	// Remove file from fingerprint
	newFiles := []syncpkg.FingerprintFile{}
	removed := false
	for _, file := range fingerprint.Files {
		if file.RelativePath != relativePath {
			newFiles = append(newFiles, file)
		} else {
			removed = true
		}
	}

	if !removed {
		a.logger.Info("File %s not found in fingerprint for volume %s", relativePath, volumeID)
		return
	}

	fingerprint.Files = newFiles

	// Update fingerprint on WebDAV
	if err := client.UpdateVolumeFingerprint(volume.MountPath, fingerprint); err != nil {
		a.logger.Error("Failed to update fingerprint for volume %s: %v", volumeID, err)
		return
	}

	a.logger.Info("Removed %s from fingerprint for volume %s", relativePath, volume.Name)
}

// batchRemoveFromFingerprint removes multiple files from a volume's fingerprint
func (a *App) batchRemoveFromFingerprint(volumeID string, relativePaths []string) {
	settings := a.store.GetSettings()
	if !settings.WebDAVEnabled || settings.WebDAVURL == "" {
		return
	}

	if len(relativePaths) == 0 {
		return
	}

	// Get volume info
	volume, err := a.store.GetVolume(volumeID)
	if err != nil || volume == nil {
		a.logger.Error("Failed to get volume %s: %v", volumeID, err)
		return
	}

	client := syncpkg.NewWebDAVClient(
		strings.TrimRight(settings.WebDAVURL, "/"),
		settings.WebDAVUser,
		settings.WebDAVPassword,
	)

	// Group paths by bucket number
	bucketPaths := make(map[int]map[string]bool)
	for _, p := range relativePaths {
		bucketNum := syncpkg.CalculateBucketNumber(p)
		if bucketPaths[bucketNum] == nil {
			bucketPaths[bucketNum] = make(map[string]bool)
		}
		bucketPaths[bucketNum][p] = true
	}

	// Process each bucket
	removedCount := 0
	for bucketNum, pathsToRemove := range bucketPaths {
		// Read current bucket
		bucket, err := client.ReadBucket(volume.MountPath, bucketNum)
		if err != nil {
			a.logger.Error("Failed to read bucket %d for volume %s: %v", bucketNum, volumeID, err)
			continue
		}

		// Remove files from bucket
		newFiles := []syncpkg.FingerprintFile{}
		bucketRemovedCount := 0
		for _, file := range bucket.Files {
			if !pathsToRemove[file.RelativePath] {
				newFiles = append(newFiles, file)
			} else {
				bucketRemovedCount++
			}
		}

		if bucketRemovedCount == 0 {
			continue
		}

		bucket.Files = newFiles

		// Write bucket back
		if err := client.WriteBucket(volume.MountPath, bucketNum, bucket); err != nil {
			a.logger.Error("Failed to write bucket %d for volume %s: %v", bucketNum, volumeID, err)
			continue
		}

		removedCount += bucketRemovedCount
	}

	if removedCount == 0 {
		a.logger.Info("No files removed from fingerprint for volume %s", volumeID)
		return
	}

	a.logger.Info("Removed %d files from fingerprint for volume %s", removedCount, volume.Name)
}

// batchAddToFingerprint adds multiple files to a volume's fingerprint
// OPTIMIZED: Uses in-memory cache with delayed batch writes to avoid blocking user operations
func (a *App) batchAddToFingerprint(volumeID string, tabs []store.Tab) {
	settings := a.store.GetSettings()
	if !settings.WebDAVEnabled || settings.WebDAVURL == "" {
		return
	}

	if len(tabs) == 0 {
		return
	}

	// Get volume info
	volume, err := a.store.GetVolume(volumeID)
	if err != nil || volume == nil {
		a.logger.Error("Failed to get volume %s: %v", volumeID, err)
		return
	}

	// Get or create fingerprint cache
	cache := a.getFingerprintCache()
	if cache == nil {
		a.logger.Error("Failed to get fingerprint cache")
		return
	}

	// Convert tabs to fingerprint files
	fpFiles := make([]syncpkg.FingerprintFile, 0, len(tabs))
	for _, tab := range tabs {
		fpFile := syncpkg.FingerprintFile{
			RelativePath: tab.FilePath,
			Title:        tab.Title,
			Artist:       tab.Artist,
			Album:        tab.Album,
			Type:         tab.Type,
			UploadedAt:   time.Unix(tab.AddedAt, 0).UTC().Format(time.RFC3339),
			UploadedBy:   getDeviceName(),
		}
		fpFiles = append(fpFiles, fpFile)
	}

	// Add files to cache (non-blocking, will be flushed automatically)
	if err := cache.BatchAddFiles(volume.MountPath, fpFiles); err != nil {
		a.logger.Error("Failed to add files to fingerprint cache for volume %s: %v", volumeID, err)
		return
	}

	a.logger.Info("Added %d files to fingerprint cache for volume %s (will be flushed automatically)", len(tabs), volume.Name)
}

