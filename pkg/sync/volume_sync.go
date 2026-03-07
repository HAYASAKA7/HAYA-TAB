package sync

import (
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"haya-tab/pkg/metadata"
	"haya-tab/pkg/store"
)

// SyncVolumeFiles syncs files from the fingerprint to the local database
// This is used when a device first discovers an existing volume (multi-device sync scenario)
// IMPORTANT: Only syncs files recorded in the fingerprint (user-selected files), NOT all files in the volume
func SyncVolumeFiles(client *WebDAVClient, db *store.DBStore, volume *store.CloudVolume) (int, int, error) {
	addedCount := 0
	skippedCount := 0

	// Read the fingerprint to get files uploaded via the app
	fingerprint, err := client.ReadVolumeFingerprint(volume.MountPath)
	if err != nil {
		fmt.Printf("[Warning] Failed to read fingerprint: %v\n", err)
		return 0, 0, fmt.Errorf("failed to read fingerprint: %w", err)
	}

	// Sync ONLY files from fingerprint (user-selected files)
	for _, fpFile := range fingerprint.Files {
		// Check if file already exists in database
		existingTab, _ := db.GetTabByVolumeAndPath(volume.ID, fpFile.RelativePath)
		if existingTab != nil {
			skippedCount++
			continue
		}

		// Calculate initials for Quick Jump Bar
		az, kana := metadata.CalculateInitials(fpFile.Title, "")

		// Create tab record using metadata from fingerprint
		tab := store.Tab{
			ID:          generateID(),
			Title:       fpFile.Title,
			Artist:      fpFile.Artist,
			Album:       fpFile.Album,
			FilePath:    fpFile.RelativePath,
			VolumeID:    volume.ID,
			Type:        fpFile.Type,
			IsManaged:   false,
			IsCloud:     true,
			CategoryIDs: []string{store.SystemCloudCategoryID},
			AddedAt:     time.Now().Unix(),
			InitialAZ:   az,
			InitialKana: kana,
		}

		if err := db.AddTab(tab); err != nil {
			fmt.Printf("[Warning] Failed to add tab from fingerprint %s: %v\n", fpFile.RelativePath, err)
			continue
		}

		addedCount++
		fmt.Printf("[Info] Synced file from fingerprint: %s\n", fpFile.RelativePath)
	}

	return addedCount, skippedCount, nil
}

// DiscoverAndRegisterVolumes scans WebDAV root for all volumes and registers them
// This is the main entry point for multi-device sync
// It automatically creates volumes for the root directory and first-level subdirectories that don't have fingerprints
func DiscoverAndRegisterVolumes(client *WebDAVClient, db *store.DBStore, rootPath string, cache *VolumeCache) ([]store.CloudVolume, error) {
	// First, scan for existing volumes (directories with fingerprint files)
	volumeMap, err := client.ScanVolumes(rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to scan volumes: %w", err)
	}

	// Mark all existing volumes as unavailable AFTER successful scan
	// Only volumes discovered in this session will be marked as available
	if err := db.MarkAllVolumesUnavailable(); err != nil {
		fmt.Printf("[Warning] Failed to mark volumes as unavailable: %v\n", err)
	}

	// Create volume for root directory if it doesn't exist
	if _, exists := volumeMap[rootPath]; !exists {
		fmt.Printf("[Info] Auto-creating volume for root directory: %s\n", rootPath)
		fingerprint, err := client.CreateVolumeFingerprint(rootPath, "Root", "1.0.0", getDeviceName())
		if err != nil {
			fmt.Printf("[Warning] Failed to create fingerprint for root: %v\n", err)
		} else {
			volumeMap[rootPath] = fingerprint
			fmt.Printf("[Info] Created root volume: Root (%s) at %s\n", fingerprint.VolumeID, rootPath)
		}
	}

	// Get all first-level subdirectories
	subdirs, err := client.ListRemoteDirectories(rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to list directories: %w", err)
	}

	// OPTIMIZED: Parallel fingerprint creation for subdirectories
	// Use a semaphore to limit concurrent WebDAV requests
	const maxConcurrent = 5
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	var mu sync.Mutex // Protect volumeMap

	for _, subdir := range subdirs {
		// Skip if this directory already has a fingerprint
		if _, exists := volumeMap[subdir]; exists {
			continue
		}

		wg.Add(1)
		go func(dir string) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			// Create fingerprint for this subdirectory
			volumeName := path.Base(dir)
			fmt.Printf("[Info] Auto-creating volume for directory: %s\n", dir)

			fingerprint, err := client.CreateVolumeFingerprint(dir, volumeName, "1.0.0", getDeviceName())
			if err != nil {
				fmt.Printf("[Warning] Failed to create fingerprint for %s: %v\n", dir, err)
				return
			}

			// Add to volume map (thread-safe)
			mu.Lock()
			volumeMap[dir] = fingerprint
			mu.Unlock()

			fmt.Printf("[Info] Created volume: %s (%s) at %s\n", volumeName, fingerprint.VolumeID, dir)
		}(subdir)
	}

	// Wait for all parallel operations to complete
	wg.Wait()

	var registeredVolumes []store.CloudVolume

	// Register all volumes (existing + newly created)
	for mountPath, fingerprint := range volumeMap {
		// Register or update volume in database
		volume, err := RegisterOrUpdateVolume(db, mountPath, fingerprint)
		if err != nil {
			fmt.Printf("[Warning] Failed to register volume at %s: %v\n", mountPath, err)
			continue
		}

		registeredVolumes = append(registeredVolumes, *volume)

		// Check if this is a newly discovered volume (first time seeing it on this device)
		tabs, err := db.GetTabsByVolume(volume.ID)
		if err != nil {
			fmt.Printf("[Warning] Failed to check tabs for volume %s: %v\n", volume.ID, err)
			continue
		}

		// If no tabs exist for this volume, it's a new discovery - sync all files
		if len(tabs) == 0 {
			fmt.Printf("[Info] New volume discovered: %s (%s). Syncing files...\n", volume.Name, volume.ID)
			added, skipped, err := SyncVolumeFiles(client, db, volume)
			if err != nil {
				fmt.Printf("[Warning] Failed to sync files for volume %s: %v\n", volume.ID, err)
			} else {
				fmt.Printf("[Info] Volume sync complete: %d added, %d skipped\n", added, skipped)
			}
		}
	}

	// Populate cache with discovered volumes
	if cache != nil {
		cache.SetAll(registeredVolumes, volumeMap)
		fmt.Printf("[Info] Volume cache updated with %d volumes\n", len(registeredVolumes))
	}

	return registeredVolumes, nil
}

// getDeviceName returns a device identifier for fingerprint tracking
func getDeviceName() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown-device"
	}
	return hostname
}

// MigrateExistingCloudFilesToFingerprints migrates existing cloud files in the database to their volume fingerprints
// This is used to backfill fingerprint files with metadata from files that were added before the fingerprint system
func MigrateExistingCloudFilesToFingerprints(client *WebDAVClient, db *store.DBStore) (int, error) {
	// Get all volumes
	volumes, err := db.GetAllVolumes()
	if err != nil {
		return 0, fmt.Errorf("failed to get volumes: %w", err)
	}

	totalMigrated := 0

	for _, volume := range volumes {
		// Get all cloud tabs for this volume
		tabs, err := db.GetTabsByVolume(volume.ID)
		if err != nil {
			fmt.Printf("[Warning] Failed to get tabs for volume %s: %v\n", volume.ID, err)
			continue
		}

		if len(tabs) == 0 {
			continue
		}

		// Read the fingerprint file
		fingerprint, err := client.ReadVolumeFingerprint(volume.MountPath)
		if err != nil {
			fmt.Printf("[Warning] Failed to read fingerprint for volume %s: %v\n", volume.ID, err)
			continue
		}

		// Create a map of existing files in fingerprint for quick lookup
		existingFiles := make(map[string]bool)
		for _, fpFile := range fingerprint.Files {
			existingFiles[fpFile.RelativePath] = true
		}

		// Add missing files to fingerprint
		migratedCount := 0
		for _, tab := range tabs {
			// Skip if file already exists in fingerprint
			if existingFiles[tab.FilePath] {
				continue
			}

			// Add file to fingerprint
			fpFile := FingerprintFile{
				RelativePath: tab.FilePath,
				Title:        tab.Title,
				Artist:       tab.Artist,
				Album:        tab.Album,
				Type:         tab.Type,
				UploadedAt:   time.Unix(tab.AddedAt, 0).UTC().Format(time.RFC3339),
				UploadedBy:   getDeviceName(),
			}

			fingerprint.Files = append(fingerprint.Files, fpFile)
			migratedCount++
		}

		// Update fingerprint file if any files were added
		if migratedCount > 0 {
			if err := client.UpdateVolumeFingerprint(volume.MountPath, fingerprint); err != nil {
				fmt.Printf("[Warning] Failed to update fingerprint for volume %s: %v\n", volume.ID, err)
				continue
			}

			fmt.Printf("[Info] Migrated %d files to fingerprint for volume %s (%s)\n", migratedCount, volume.Name, volume.ID)
			totalMigrated += migratedCount
		}
	}

	return totalMigrated, nil
}

// CheckVolumeHealth checks if all registered volumes are still accessible
// Returns a map of volume_id -> is_available
func CheckVolumeHealth(client *WebDAVClient, db *store.DBStore) (map[string]bool, error) {
	volumes, err := db.GetAllVolumes()
	if err != nil {
		return nil, fmt.Errorf("failed to get volumes: %w", err)
	}

	healthMap := make(map[string]bool)

	for _, volume := range volumes {
		// Check if fingerprint file still exists
		isAvailable := client.FingerprintExists(volume.MountPath)
		healthMap[volume.ID] = isAvailable

		// Update database
		if err := db.MarkVolumeAvailable(volume.ID, isAvailable); err != nil {
			fmt.Printf("[Warning] Failed to update availability for volume %s: %v\n", volume.ID, err)
		}
	}

	return healthMap, nil
}

// ResolveFilePath resolves a cloud file path to its full WebDAV path
// Takes a volume_id and relative path, returns the full WebDAV path
func ResolveFilePath(db *store.DBStore, volumeID, relativePath string) (string, error) {
	if volumeID == "" {
		// Local file, return as-is
		return relativePath, nil
	}

	volume, err := db.GetVolume(volumeID)
	if err != nil {
		return "", fmt.Errorf("failed to get volume: %w", err)
	}
	if volume == nil {
		return "", fmt.Errorf("volume not found: %s", volumeID)
	}
	if !volume.IsAvailable {
		return "", fmt.Errorf("volume is not available: %s", volume.Name)
	}

	// Construct full path
	fullPath := path.Join(volume.MountPath, relativePath)
	return fullPath, nil
}

// parseMetadataFromFilename extracts title and artist from filename
func parseMetadataFromFilename(filename string) (title, artist string) {
	// Remove extension
	name := strings.TrimSuffix(filename, path.Ext(filename))

	// Try to split by " - " (common format: "Artist - Title")
	parts := strings.SplitN(name, " - ", 2)
	if len(parts) == 2 {
		artist = strings.TrimSpace(parts[0])
		title = strings.TrimSpace(parts[1])
	} else {
		title = strings.TrimSpace(name)
		artist = ""
	}

	return title, artist
}

// generateID generates a unique ID for a tab
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
