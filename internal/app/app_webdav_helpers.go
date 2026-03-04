package app

import (
	"strings"
	"time"

	"haya-tab/pkg/store"
	syncpkg "haya-tab/pkg/sync"
)

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

	// Create a map for fast lookup
	pathsToRemove := make(map[string]bool)
	for _, path := range relativePaths {
		pathsToRemove[path] = true
	}

	// Remove files from fingerprint
	newFiles := []syncpkg.FingerprintFile{}
	removedCount := 0
	for _, file := range fingerprint.Files {
		if !pathsToRemove[file.RelativePath] {
			newFiles = append(newFiles, file)
		} else {
			removedCount++
		}
	}

	if removedCount == 0 {
		a.logger.Info("No files removed from fingerprint for volume %s", volumeID)
		return
	}

	fingerprint.Files = newFiles

	// Update fingerprint on WebDAV
	if err := client.UpdateVolumeFingerprint(volume.MountPath, fingerprint); err != nil {
		a.logger.Error("Failed to update fingerprint for volume %s: %v", volumeID, err)
		return
	}

	a.logger.Info("Removed %d files from fingerprint for volume %s", removedCount, volume.Name)
}

// batchAddToFingerprint adds multiple files to a volume's fingerprint
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

	// Create a map of existing files for fast lookup
	existingFiles := make(map[string]bool)
	for _, file := range fingerprint.Files {
		existingFiles[file.RelativePath] = true
	}

	// Add new files to fingerprint
	addedCount := 0
	for _, tab := range tabs {
		// Skip if file already exists in fingerprint
		if existingFiles[tab.FilePath] {
			continue
		}

		// Add file to fingerprint
		fpFile := syncpkg.FingerprintFile{
			RelativePath: tab.FilePath,
			Title:        tab.Title,
			Artist:       tab.Artist,
			Album:        tab.Album,
			Type:         tab.Type,
			UploadedAt:   time.Unix(tab.AddedAt, 0).UTC().Format(time.RFC3339),
			UploadedBy:   getDeviceName(),
		}

		fingerprint.Files = append(fingerprint.Files, fpFile)
		addedCount++
	}

	if addedCount == 0 {
		a.logger.Info("No new files added to fingerprint for volume %s", volumeID)
		return
	}

	// Update fingerprint on WebDAV
	if err := client.UpdateVolumeFingerprint(volume.MountPath, fingerprint); err != nil {
		a.logger.Error("Failed to update fingerprint for volume %s: %v", volumeID, err)
		return
	}

	a.logger.Info("Added %d files to fingerprint for volume %s", addedCount, volume.Name)
}

