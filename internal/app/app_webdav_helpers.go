package app

import (
	"strings"
	"time"

	"haya-tab/pkg/store"
	syncpkg "haya-tab/pkg/sync"
)

const (
	// FingerprintCacheFlushInterval is the automatic flush interval for the fingerprint cache.
	FingerprintCacheFlushInterval = 3 * time.Second
	// FingerprintCacheMaxBuckets is the maximum number of buckets kept in the fingerprint cache.
	FingerprintCacheMaxBuckets = 100
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

		// Create cache with configured flush interval and max bucket size
		a.fingerprintCache = syncpkg.NewFingerprintCache(client, FingerprintCacheFlushInterval, FingerprintCacheMaxBuckets)
		a.logger.Info("Fingerprint cache initialized with %v flush interval and %d bucket max size", FingerprintCacheFlushInterval, FingerprintCacheMaxBuckets)
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

	cache := a.getFingerprintCache()
	if cache == nil {
		a.logger.Error("Failed to get fingerprint cache")
		return
	}

	removedCount, err := cache.BatchRemoveFiles(volume.MountPath, []string{relativePath})
	if err != nil {
		a.logger.Error("Failed to remove file from fingerprint cache for volume %s: %v", volumeID, err)
		return
	}

	if removedCount == 0 {
		a.logger.Info("File %s not found in fingerprint for volume %s", relativePath, volumeID)
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

	cache := a.getFingerprintCache()
	if cache == nil {
		a.logger.Error("Failed to get fingerprint cache")
		return
	}

	removedCount, err := cache.BatchRemoveFiles(volume.MountPath, relativePaths)
	if err != nil {
		a.logger.Error("Failed to remove files from fingerprint cache for volume %s: %v", volumeID, err)
		return
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
		// Get categories for this tab
		categories, err := a.store.GetCategoryNamesForTab(tab.ID)
		if err != nil {
			a.logger.Error("Failed to get categories for tab %s: %v", tab.ID, err)
			categories = []string{}
		}

		// Use CloudPath if available, otherwise fallback to FilePath (only for cloud tabs)
		relativePath := tab.CloudPath
		if relativePath == "" {
			if tab.IsCloud {
				relativePath = tab.FilePath
			} else {
				// For local tabs without CloudPath, we can't reliably update fingerprint
				// unless we calculate it, but for now we skip to avoid bad data
				a.logger.Warning("Skipping fingerprint update for local tab %s: no CloudPath", tab.ID)
				continue
			}
		}

		fpFile := syncpkg.FingerprintFile{
			RelativePath: relativePath,
			Title:        tab.Title,
			Artist:       tab.Artist,
			Album:        tab.Album,
			Type:         tab.Type,
			Categories:   categories,
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
