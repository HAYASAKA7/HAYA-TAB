package app

import (
	"errors"
	"fmt"
	apperrors "haya-tab/pkg/errors"
	"haya-tab/pkg/metadata"
	"haya-tab/pkg/store"
	syncpkg "haya-tab/pkg/sync"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// getDeviceName returns a device identifier for fingerprint tracking
func getDeviceName() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown-device"
	}
	return hostname
}

// WebDAVTestConnection tests the WebDAV connection
func (a *App) WebDAVTestConnection(url, user, password string) error {
	// Validate URL format and settings
	if err := a.validateWebDAV(url, user, password); err != nil {
		return err
	}

	url = strings.TrimRight(url, "/")

	client := syncpkg.NewWebDAVClient(url, user, password)
	err := client.TestConnection()

	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "401") || strings.Contains(errMsg, "Unauthorized") {
			return apperrors.NewAppError("errors.webdavAuthFailed", "authentication failed", nil, err)
		} else if strings.Contains(errMsg, "404") || strings.Contains(errMsg, "Not Found") {
			return apperrors.NewAppError("errors.webdavEndpointNotFound", "WebDAV endpoint not found", nil, err)
		} else if strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "Timeout") {
			return apperrors.NewAppError("errors.webdavTimeout", "connection timeout", nil, err)
		} else if strings.Contains(errMsg, "no such host") || strings.Contains(errMsg, "cannot resolve") {
			return apperrors.NewAppError("errors.webdavCannotResolveHost", "cannot resolve hostname", nil, err)
		} else if strings.Contains(errMsg, "connection refused") {
			return apperrors.NewAppError("errors.webdavConnectionRefused", "connection refused", nil, err)
		}
		return apperrors.NewAppError("errors.webdavConnectionFailed", "connection failed", nil, err)
	}

	return nil
}

// WebDAVScanRemoteFiles scans a remote directory
func (a *App) WebDAVScanRemoteFiles(url, user, password, dir string) ([]store.RemoteFile, error) {
	// Sanitize URL: remove trailing slash
	url = strings.TrimRight(url, "/")
	client := syncpkg.NewWebDAVClient(url, user, password)
	return client.ScanRemoteFiles(dir)
}

// WebDAVListRemoteDirectories lists directories in a remote path
func (a *App) WebDAVListRemoteDirectories(url, user, password, dir string) ([]string, error) {
	url = strings.TrimRight(url, "/")
	client := syncpkg.NewWebDAVClient(url, user, password)
	return client.ListRemoteDirectories(dir)
}

// WebDAVListDir lists files and directories in a remote path (non-recursive)
func (a *App) WebDAVListDir(url, user, password, dir string) ([]store.RemoteFile, error) {
	url = strings.TrimRight(url, "/")
	client := syncpkg.NewWebDAVClient(url, user, password)
	return client.ListDir(dir)
}

// WebDAVDownloadFiles downloads selected files and processes them
func (a *App) WebDAVDownloadFiles(url, user, password string, remotePaths []string) error {
	url = strings.TrimRight(url, "/")
	client := syncpkg.NewWebDAVClient(url, user, password)

	a.emitEvent("cloud-progress", map[string]interface{}{
		"status": "start",
		"total":  len(remotePaths),
	})

	// Run in background to avoid blocking UI
	go func() {
		successCount := 0
		skippedCount := 0
		errorCount := 0

		// Get only active volumes for linking
		volumes, _ := a.store.GetActiveVolumes()

		for i, remotePath := range remotePaths {
			fileName := filepath.Base(remotePath)

			// Determine if this file belongs to a volume
			var volumeID string
			var relativePath string
			longestMatch := ""
			for _, vol := range volumes {
				if strings.HasPrefix(remotePath, vol.MountPath+"/") || remotePath == vol.MountPath {
					if len(vol.MountPath) > len(longestMatch) {
						longestMatch = vol.MountPath
						volumeID = vol.ID
					}
				}
			}

			if volumeID != "" {
				relativePath = strings.TrimPrefix(remotePath, longestMatch)
				relativePath = strings.TrimPrefix(relativePath, "/")
			}

			// Create temp file
			tempFile, err := os.CreateTemp("", "haya-tab-download-*.tmp")
			if err != nil {
				a.logger.Error("Failed to create temp file for %s: %v", fileName, err)
				errorCount++
				continue
			}
			tempPath := tempFile.Name()
			tempFile.Close() // Close immediately, DownloadFile will open/create it

			// Download to temp path
			if err := client.DownloadFile(remotePath, tempPath); err != nil {
				a.logger.Error("Failed to download %s: %v", remotePath, err)
				os.Remove(tempPath)
				errorCount++
				continue
			}

			// Process File to get metadata
			parsedTab := a.syncService.ProcessFile(tempPath)

			// If ProcessFile failed to get meaningful title, fallback to filename
			if parsedTab.Title == "" {
				parsedTab.Title = strings.TrimSuffix(fileName, filepath.Ext(fileName))
			}

			// Link to volume if detected
			if volumeID != "" {
				parsedTab.VolumeID = volumeID
				parsedTab.CloudPath = relativePath
			}

			// SaveTab handles ID generation, file moving/renaming, and duplicate checks
			savedTab, err := a.SaveTab(parsedTab, true)
			if err != nil {
				var apperr *apperrors.AppError
				if errors.As(err, &apperr) && apperr.Code == "errors.tabDuplicate" {
					a.logger.Info("Skipping duplicate file %s: %v", fileName, err)
					skippedCount++
				} else {
					a.logger.Error("Failed to save downloaded tab %s: %v", fileName, err)
					errorCount++
				}
				// Clean up temp file
				os.Remove(tempPath)
			} else {
				successCount++
				// Clean up temp file (SaveTab copied it)
				os.Remove(tempPath)

				// If it's linked to a volume, update the fingerprint
				if volumeID != "" && savedTab != nil {
					a.batchAddToFingerprint(volumeID, []store.Tab{*savedTab})
				}
			}

			a.emitEvent("cloud-progress", map[string]interface{}{
				"status":   "progress",
				"current":  i + 1,
				"total":    len(remotePaths),
				"filename": fileName,
			})
		}

		a.emitEvent("cloud-progress", map[string]interface{}{
			"status":  "complete",
			"success": successCount,
			"skipped": skippedCount,
			"errors":  errorCount,
		})

		// Force library refresh when background process finishes, in case modal is closed early
		if successCount > 0 {
			a.emitEvent("tab-updated", nil)
		}
	}()

	return nil
}

// WebDAVUploadFiles uploads local files to a remote directory
func (a *App) WebDAVUploadFiles(url, user, password string, localPaths []string, remoteDir string) error {
	url = strings.TrimRight(url, "/")
	client := syncpkg.NewWebDAVClient(url, user, password)

	a.emitEvent("cloud-upload-progress", map[string]interface{}{
		"status": "start",
		"total":  len(localPaths),
	})

	go func() {
		successCount := 0

		// Get only active volumes (available and deduplicated by mount path)
		volumes, err := a.store.GetActiveVolumes()
		if err != nil {
			a.logger.Error("Failed to get active volumes: %v", err)
			volumes = []store.CloudVolume{}
		}

		// Find the volume that contains remoteDir
		var targetVolume *store.CloudVolume
		longestMatch := ""
		for i := range volumes {
			vol := &volumes[i]
			if strings.HasPrefix(remoteDir, vol.MountPath+"/") || remoteDir == vol.MountPath {
				if len(vol.MountPath) > len(longestMatch) {
					longestMatch = vol.MountPath
					targetVolume = vol
				}
			}
		}

		for i, localPath := range localPaths {
			fileName := filepath.Base(localPath)

			// Upload the file
			if err := client.UploadFile(localPath, remoteDir); err != nil {
				a.logger.Error("Failed to upload %s: %v", localPath, err)
			} else {
				successCount++

				// If we found a volume, update its fingerprint file
				if targetVolume != nil {
					// Get tab metadata and categories if this file is in our library
					tab, _ := a.store.GetTabByPath(localPath)

					var title, artist, album, fileType string
					var categories []string
					if tab != nil {
						// Use existing metadata from library
						title = tab.Title
						artist = tab.Artist
						album = tab.Album
						fileType = tab.Type

						// Get category names
						cats, err := a.store.GetCategoryNamesForTab(tab.ID)
						if err == nil {
							categories = cats
						}
					} else {
						// Parse metadata from filename
						title, artist = parseMetadataFromFilename(fileName)

						// Determine file type
						ext := strings.ToLower(filepath.Ext(fileName))
						fileType = "gp"
						switch ext {
						case ".pdf":
							fileType = "pdf"
						case ".gp", ".gp3", ".gp4", ".gp5", ".gpx", ".xml", ".musicxml", ".mxl":
							fileType = "gp"
						}
					}

					// Calculate relative path within volume
					relativePath := strings.TrimPrefix(remoteDir, targetVolume.MountPath)
					relativePath = strings.TrimPrefix(relativePath, "/")
					if relativePath != "" {
						relativePath = relativePath + "/"
					}
					relativePath = relativePath + fileName

					// Add file record to fingerprint cache (non-blocking)
					fpFile := syncpkg.FingerprintFile{
						RelativePath: relativePath,
						Title:        title,
						Artist:       artist,
						Album:        album,
						Type:         fileType,
						Categories:   categories,
						UploadedAt:   time.Now().UTC().Format(time.RFC3339),
						UploadedBy:   getDeviceName(),
					}

					cache := a.getFingerprintCache()
					if cache != nil {
						if err := cache.AddFile(targetVolume.MountPath, fpFile); err != nil {
							a.logger.Error("Failed to add file to fingerprint cache for %s: %v", fileName, err)
						} else {
							a.logger.Info("Added file to fingerprint cache: %s (will be flushed automatically)", fileName)
						}
					}
				}
			}

			a.emitEvent("cloud-upload-progress", map[string]interface{}{
				"status":   "progress",
				"current":  i + 1,
				"total":    len(localPaths),
				"filename": fileName,
			})
		}

		a.emitEvent("cloud-upload-progress", map[string]interface{}{
			"status":  "complete",
			"success": successCount,
		})
	}()

	return nil
}

// WebDAVAddOnlineFiles adds cloud files to library without downloading (lazy loading)
func (a *App) WebDAVAddOnlineFiles(url, user, password string, remotePaths []string) error {
	url = strings.TrimRight(url, "/")

	// Ensure cloud category exists
	if err := a.store.EnsureCloudCategory(); err != nil {
		a.logger.Error("Failed to ensure cloud category: %v", err)
	}

	// OPTIMIZED: Use cached volumes if available, only discover if cache is stale
	var volumes []store.CloudVolume
	var err error

	// Try to get volumes from cache first
	cachedVolumes := a.volumeCache.GetAll()
	if cachedVolumes != nil {
		a.logger.Info("Using cached volumes (%d volumes)", len(cachedVolumes))
		volumes = cachedVolumes
	} else {
		// Cache miss or stale - discover volumes asynchronously
		a.logger.Info("Cache miss - discovering volumes...")
		volumes, err = a.WebDAVDiscoverVolumes()
		if err != nil {
			a.logger.Error("Failed to discover volumes: %v", err)
			return apperrors.WebDAVDiscoverVolumesError(err)
		}
	}

	a.emitEvent("cloud-progress", map[string]interface{}{
		"status": "start",
		"total":  len(remotePaths),
	})

	go func() {
		successCount := 0
		skippedCount := 0

		a.logger.Info("Processing %d files with %d active volumes", len(remotePaths), len(volumes))

		// Collect all tabs to add in batch
		var tabsToAdd []store.Tab
		// Track added files by volume for batch fingerprint updates
		volumeAddedFiles := make(map[string][]store.Tab)

		for i, remotePath := range remotePaths {
			fileName := filepath.Base(remotePath)

			// Determine which volume this file belongs to
			var volumeID string
			var relativePath string

			// Find the volume with the longest matching mount path
			longestMatch := ""
			for _, vol := range volumes {
				if strings.HasPrefix(remotePath, vol.MountPath+"/") || remotePath == vol.MountPath {
					if len(vol.MountPath) > len(longestMatch) {
						longestMatch = vol.MountPath
						volumeID = vol.ID
					}
				}
			}

			if volumeID != "" {
				// Calculate relative path within volume
				relativePath = strings.TrimPrefix(remotePath, longestMatch)
				relativePath = strings.TrimPrefix(relativePath, "/")
			} else {
				// No volume found
				a.logger.Error("No active volume matched for file: %s", remotePath)
				relativePath = remotePath
			}

			// Check for duplicates by volume and path
			var existingTab *store.Tab
			if volumeID != "" {
				existingTab, _ = a.store.GetTabByVolumeAndPath(volumeID, relativePath)
			} else {
				existingTab, _ = a.store.GetTabByPath(remotePath)
			}

			if existingTab != nil {
				a.logger.Info("Skipping duplicate cloud file %s", fileName)
				skippedCount++
				a.emitEvent("cloud-progress", map[string]interface{}{
					"status":   "progress",
					"current":  i + 1,
					"total":    len(remotePaths),
					"filename": fileName,
				})
				continue
			}

			// Parse metadata from filename (lazy - no download)
			title, artist := parseMetadataFromFilename(fileName)

			// Determine file type
			ext := strings.ToLower(filepath.Ext(fileName))
			fileType := "gp"
			switch ext {
			case ".pdf":
				fileType = "pdf"
			case ".gp", ".gp3", ".gp4", ".gp5", ".gpx", ".xml", ".musicxml", ".mxl":
				fileType = "gp"
			}

			// Create tab record
			tab := store.Tab{
				ID:          generateID(),
				Title:       title,
				Artist:      artist,
				FilePath:    relativePath, // Store relative path within volume
				VolumeID:    volumeID,     // Store volume ID
				Type:        fileType,
				IsManaged:   false,
				IsCloud:     true,
				CategoryIDs: []string{store.SystemCloudCategoryID},
				AddedAt:     time.Now().Unix(),
			}

			if a.pluginManager != nil {
				a.pluginManager.EnhanceMetadata(&tab)
			}
			
			// Always calculate initials (even if no plugin ran, they need to be populated)
			tab.InitialAZ, tab.InitialKana = metadata.CalculateInitials(tab.Title, tab.OriginCountry)

			// Add to batch
			tabsToAdd = append(tabsToAdd, tab)

			// Track for fingerprint update
			if volumeID != "" {
				volumeAddedFiles[volumeID] = append(volumeAddedFiles[volumeID], tab)
			}

			a.emitEvent("cloud-progress", map[string]interface{}{
				"status":   "progress",
				"current":  i + 1,
				"total":    len(remotePaths),
				"filename": fileName,
			})
		}

		// Batch add all tabs at once
		if len(tabsToAdd) > 0 {
			added, err := a.store.BatchAddTabs(tabsToAdd)
			if err != nil {
				a.logger.Error("Failed to batch add tabs: %v", err)
			} else {
				successCount = added
				a.logger.Info("Batch added %d cloud files", added)
			}
		}

		// Update fingerprints for all affected volumes
		for volumeID, tabs := range volumeAddedFiles {
			a.batchAddToFingerprint(volumeID, tabs)
		}

		a.emitEvent("cloud-progress", map[string]interface{}{
			"status":  "complete",
			"success": successCount,
			"skipped": skippedCount,
			"errors":  0,
		})

		// Force library refresh when background process finishes, in case modal is closed early
		if successCount > 0 {
			a.emitEvent("tab-updated", nil)
		}
	}()

	return nil
}

// WebDAVCheckStatus checks if WebDAV connection is available
func (a *App) WebDAVCheckStatus() bool {
	settings := a.store.GetSettings()
	if !settings.WebDAVEnabled || settings.WebDAVURL == "" {
		return false
	}

	client := syncpkg.NewWebDAVClient(
		strings.TrimRight(settings.WebDAVURL, "/"),
		settings.WebDAVUser,
		settings.WebDAVPassword,
	)

	err := client.TestConnection()
	return err == nil
}

// WebDAVReconnect attempts to reconnect and reinitialize WebDAV
// This should be called when connection is restored after being lost
func (a *App) WebDAVReconnect() error {
	settings := a.store.GetSettings()
	if !settings.WebDAVEnabled || settings.WebDAVURL == "" {
		return apperrors.WebDAVNotEnabledError()
	}

	a.logger.Info("Attempting to reconnect to WebDAV...")

	// Test connection first
	if !a.WebDAVCheckStatus() {
		return apperrors.WebDAVConnectionTestFailedError()
	}

	// Reinitialize volumes
	return a.WebDAVInitialize()
}

// monitorWebDAVConnection monitors WebDAV connection status and auto-reconnects when connection is restored
func (a *App) monitorWebDAVConnection() {
	// Initialize lastStatus to current status to avoid false "connection restored" on first check
	settings := a.store.GetSettings()
	lastStatus := false
	if settings.WebDAVEnabled && settings.WebDAVURL != "" {
		lastStatus = a.WebDAVCheckStatus()
	}

	checkInterval := 30 * time.Second // Check every 30 seconds

	for {
		settings := a.store.GetSettings()

		// Only monitor if WebDAV is enabled
		if !settings.WebDAVEnabled || settings.WebDAVURL == "" {
			lastStatus = false
			time.Sleep(checkInterval)
			continue
		}

		currentStatus := a.WebDAVCheckStatus()

		// If connection was lost and now restored, reinitialize
		if !lastStatus && currentStatus {
			a.logger.Info("WebDAV connection restored, reinitializing...")
			go func() {
				if err := a.WebDAVReconnect(); err != nil {
					a.logger.Error("WebDAV reconnection failed: %v", err)
				} else {
					a.logger.Info("WebDAV reconnection successful")
				}
			}()
		} else if lastStatus && !currentStatus {
			a.logger.Info("WebDAV connection lost")
		}

		lastStatus = currentStatus
		time.Sleep(checkInterval)
	}
}

// WebDAVDiscoverVolumes scans WebDAV for all volumes and registers them
// This is the entry point for multi-device sync
func (a *App) WebDAVDiscoverVolumes() ([]store.CloudVolume, error) {
	settings := a.store.GetSettings()
	if !settings.WebDAVEnabled || settings.WebDAVURL == "" {
		return nil, apperrors.WebDAVNotEnabledError()
	}

	client := syncpkg.NewWebDAVClient(
		strings.TrimRight(settings.WebDAVURL, "/"),
		settings.WebDAVUser,
		settings.WebDAVPassword,
	)

	// Discover and register all volumes
	volumes, addedCount, err := syncpkg.DiscoverAndRegisterVolumes(client, a.store, "/", a.volumeCache, AppVersion)
	if err != nil {
		return nil, apperrors.WebDAVDiscoverVolumesError(err)
	}

	if addedCount > 0 {
		// Emit event to refresh frontend if new tabs were added during sync
		a.emitEvent("tab-updated", nil)
	}

	a.logger.Info("Discovered %d volumes", len(volumes))
	return volumes, nil
}

// WebDAVCheckVolumeHealth checks the health of all registered volumes
func (a *App) WebDAVCheckVolumeHealth() (map[string]bool, error) {
	settings := a.store.GetSettings()
	if !settings.WebDAVEnabled || settings.WebDAVURL == "" {
		return nil, apperrors.WebDAVNotEnabledError()
	}

	client := syncpkg.NewWebDAVClient(
		strings.TrimRight(settings.WebDAVURL, "/"),
		settings.WebDAVUser,
		settings.WebDAVPassword,
	)

	healthMap, err := syncpkg.CheckVolumeHealth(client, a.store)
	if err != nil {
		return nil, apperrors.WebDAVVolumeHealthCheckError(err)
	}

	return healthMap, nil
}

// WebDAVMigrateCloudTabs migrates existing cloud tabs to use the volume system
func (a *App) WebDAVMigrateCloudTabs() error {
	return a.store.MigrateCloudTabsToVolumes()
}

// WebDAVGetOrphanedTabsCount returns the count of tabs referencing non-existent volumes
func (a *App) WebDAVGetOrphanedTabsCount() (int, error) {
	return a.store.GetOrphanedTabsCount()
}

// WebDAVCleanupOrphanedTabs removes tabs that reference non-existent volumes
func (a *App) WebDAVCleanupOrphanedTabs() (int, error) {
	return a.store.CleanupOrphanedTabs()
}

// WebDAVInitialize initializes the WebDAV volume system
// This should be called on app startup if WebDAV is enabled
func (a *App) WebDAVInitialize() error {
	settings := a.store.GetSettings()
	if !settings.WebDAVEnabled || settings.WebDAVURL == "" {
		a.logger.Info("WebDAV not enabled, skipping initialization")
		return nil
	}

	a.emitEvent("webdav-sync-progress", map[string]interface{}{
		"status":     "start",
		"messageKey": "toast.syncTask.connecting",
	})

	a.logger.Info("Initializing WebDAV volume system...")

	// Ensure cloud category exists before syncing files
	if err := a.store.EnsureCloudCategory(); err != nil {
		a.logger.Error("Failed to ensure cloud category: %v", err)
		// Don't fail initialization if category creation fails
	}

	client := syncpkg.NewWebDAVClient(
		strings.TrimRight(settings.WebDAVURL, "/"),
		settings.WebDAVUser,
		settings.WebDAVPassword,
	)

	a.emitEvent("webdav-sync-progress", map[string]interface{}{
		"status":     "progress",
		"messageKey": "toast.syncTask.discovering",
	})

	// Discover and register volumes
	volumes, err := a.WebDAVDiscoverVolumes()
	if err != nil {
		a.logger.Error("Failed to discover volumes: %v", err)
		a.emitEvent("webdav-sync-progress", map[string]interface{}{
			"status":     "error",
			"messageKey": "errors.webdavDiscoverVolumesFailed",
		})
		return err
	}

	a.logger.Info("Discovered %d volumes", len(volumes))

	a.emitEvent("webdav-sync-progress", map[string]interface{}{
		"status":     "progress",
		"messageKey": "toast.syncTask.syncing",
	})

	// Migrate existing cloud files to fingerprints
	a.logger.Info("Migrating existing cloud files to fingerprints...")
	migratedCount, err := syncpkg.MigrateExistingCloudFilesToFingerprints(client, a.store)
	if err != nil {
		a.logger.Error("Failed to migrate cloud files to fingerprints: %v", err)
		// Don't fail initialization if migration fails
	} else if migratedCount > 0 {
		a.logger.Info("Migrated %d existing cloud files to fingerprints", migratedCount)
	}

	// Migrate legacy cloud tabs
	if err := a.WebDAVMigrateCloudTabs(); err != nil {
		a.logger.Error("Failed to migrate cloud tabs: %v", err)
		// Don't fail initialization if migration fails
	}

	// Ensure all cloud tabs have the cloud category
	a.logger.Info("Ensuring all cloud tabs have cloud category...")
	addedCount, err := a.store.EnsureCloudTabsHaveCloudCategory()
	if err != nil {
		a.logger.Error("Failed to ensure cloud tabs have cloud category: %v", err)
		// Don't fail initialization if migration fails
	} else if addedCount > 0 {
		a.logger.Info("Added cloud category to %d existing cloud tabs", addedCount)
	}

	if migratedCount > 0 || addedCount > 0 {
		// Emit event to refresh frontend if any migrations occurred
		a.emitEvent("tab-updated", nil)
	}

	// Check volume health
	healthMap, err := a.WebDAVCheckVolumeHealth()
	if err != nil {
		a.logger.Error("Failed to check volume health: %v", err)
		a.emitEvent("webdav-sync-progress", map[string]interface{}{
			"status":     "error",
			"messageKey": "errors.webdavVolumeHealthCheckFailed",
		})
		// Don't fail initialization if health check fails
	} else {
		unavailableCount := 0
		for _, available := range healthMap {
			if !available {
				unavailableCount++
			}
		}
		if unavailableCount > 0 {
			a.logger.Info("%d volumes are currently unavailable", unavailableCount)
		}
	}

	a.logger.Info("WebDAV initialization complete")
	a.emitEvent("webdav-sync-progress", map[string]interface{}{
		"status":     "success",
		"messageKey": "toast.syncTask.success",
	})

	return nil
}

// DownloadCloudTabToLocal downloads a cloud tab to local storage
// IMPORTANT: This preserves existing metadata - does NOT re-parse the file
func (a *App) DownloadCloudTabToLocal(tabID string) error {
	tab, err := a.store.GetTab(tabID)
	if err != nil {
		return fmt.Errorf("failed to get tab: %w", err)
	}
	if tab == nil {
		return apperrors.TabNotFoundError(tabID)
	}
	if !tab.IsCloud {
		return apperrors.TabNotCloudError()
	}

	settings := a.store.GetSettings()
	if !settings.WebDAVEnabled {
		return apperrors.WebDAVNotEnabledError()
	}

	client := syncpkg.NewWebDAVClient(
		strings.TrimRight(settings.WebDAVURL, "/"),
		settings.WebDAVUser,
		settings.WebDAVPassword,
	)

	a.emitEvent("cloud-download-single", map[string]interface{}{
		"status": "start",
		"tabId":  tabID,
	})

	go func() {
		remotePath := tab.FilePath
		fileName := filepath.Base(remotePath)

		// Create temp file
		tempFile, err := os.CreateTemp("", "haya-tab-download-*.tmp")
		if err != nil {
			a.logger.Error("Failed to create temp file: %v", err)
			a.emitEvent("cloud-download-single", map[string]interface{}{
				"status":   "error",
				"tabId":    tabID,
				"messageKey": "errors.fileAccess",
				"errorArgs": map[string]interface{}{"path": "temporary download file"},
			})
			return
		}
		tempPath := tempFile.Name()
		tempFile.Close()

		// Download file
		if err := client.DownloadFile(remotePath, tempPath); err != nil {
			a.logger.Error("Failed to download %s: %v", remotePath, err)
			os.Remove(tempPath)
			a.emitEvent("cloud-download-single", map[string]interface{}{
				"status":   "error",
				"tabId":    tabID,
				"messageKey": "errors.downloadFailed",
			})
			return
		}

		// Move to storage directory
		ext := filepath.Ext(fileName)
		appDir := getAppDir()
		localPath := filepath.Join(appDir, "storage", tab.ID+ext)

		if err := copyFile(tempPath, localPath); err != nil {
			a.logger.Error("Failed to copy to storage: %v", err)
			os.Remove(tempPath)
			a.emitEvent("cloud-download-single", map[string]interface{}{
				"status":   "error",
				"tabId":    tabID,
				"messageKey": "errors.fileAccess",
				"errorArgs": map[string]interface{}{"path": localPath},
			})
			return
		}
		os.Remove(tempPath)

		// CRITICAL: Do NOT call ProcessFile - preserve existing metadata
		// Only update the necessary state fields
		tab.CloudPath = tab.FilePath // Keep original relative path for fingerprint updates
		tab.FilePath = localPath
		tab.IsCloud = false
		tab.IsManaged = true

		// Remove from cloud category, keep other categories
		newCategoryIDs := []string{}
		for _, catID := range tab.CategoryIDs {
			if catID != store.SystemCloudCategoryID {
				newCategoryIDs = append(newCategoryIDs, catID)
			}
		}
		tab.CategoryIDs = newCategoryIDs

		if err := a.store.UpdateTab(*tab); err != nil {
			a.logger.Error("Failed to update tab: %v", err)
			a.emitEvent("cloud-download-single", map[string]interface{}{
				"status":   "error",
				"tabId":    tabID,
				"messageKey": "errors.database",
				"errorArgs": map[string]interface{}{"operation": "update_tab"},
			})
			return
		}

		// Optionally fetch cover if not already present
		if tab.CoverPath == "" && tab.Artist != "" {
			a.fetchCoverAsync(*tab)
		}

		a.emitEvent("cloud-download-single", map[string]interface{}{
			"status": "complete",
			"tabId":  tabID,
			"tab":    tab,
		})
	}()

	return nil
}

// WebDAVCreateVolume creates a new volume with a fingerprint file
func (a *App) WebDAVCreateVolume(volumeName, remotePath string) (*store.CloudVolume, error) {
	settings := a.store.GetSettings()
	if !settings.WebDAVEnabled || settings.WebDAVURL == "" {
		return nil, apperrors.WebDAVNotEnabledError()
	}

	// Validate inputs
	if volumeName == "" {
		return nil, apperrors.InvalidVolumeNameError()
	}
	if remotePath == "" {
		remotePath = "/" // Default to root
	}

	client := syncpkg.NewWebDAVClient(
		strings.TrimRight(settings.WebDAVURL, "/"),
		settings.WebDAVUser,
		settings.WebDAVPassword,
	)

	// Check if fingerprint already exists at this path
	if client.FingerprintExists(remotePath) {
		return nil, apperrors.NewAppError("errors.volumeAlreadyExists", "a volume already exists at this path", map[string]interface{}{"path": remotePath}, nil)
	}

	// Create fingerprint file with app version and device name
	deviceName := getDeviceName()
	fingerprint, err := client.CreateVolumeFingerprint(remotePath, volumeName, AppVersion, deviceName)
	if err != nil {
		return nil, apperrors.VolumeCreationError(err)
	}

	a.logger.Info("Created volume fingerprint: %s at %s", fingerprint.VolumeID, remotePath)

	// Register volume in database
	volume, err := syncpkg.RegisterOrUpdateVolume(a.store, remotePath, fingerprint)
	if err != nil {
		return nil, apperrors.VolumeRegistrationError(err)
	}

	a.logger.Info("Volume created successfully: %s (%s)", volume.Name, volume.ID)
	return volume, nil
}

// parseMetadataFromFilename extracts title and artist from filename
// Supports formats: "Artist - Title.ext" or just "Title.ext"
func parseMetadataFromFilename(filename string) (title, artist string) {
	// Remove extension
	name := strings.TrimSuffix(filename, filepath.Ext(filename))

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
