package app

import (
	"fmt"
	"haya-tab/pkg/store"
	syncpkg "haya-tab/pkg/sync"
	"os"
	"path/filepath"
	"strings"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// WebDAVTestConnection tests the WebDAV connection
func (a *App) WebDAVTestConnection(url, user, password string) error {
	url = strings.TrimRight(url, "/")
	client := syncpkg.NewWebDAVClient(url, user, password)
	return client.TestConnection()
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

	wailsRuntime.EventsEmit(a.ctx, "cloud-progress", map[string]interface{}{
		"status": "start",
		"total":  len(remotePaths),
	})

	// Run in background to avoid blocking UI
	go func() {
		successCount := 0
		skippedCount := 0
		errorCount := 0

		for i, remotePath := range remotePaths {
			fileName := filepath.Base(remotePath)

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

			// SaveTab handles ID generation, file moving/renaming, and duplicate checks
			_, err = a.SaveTab(parsedTab, true)
			if err != nil {
				if strings.Contains(err.Error(), "already exists") {
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
			}

			wailsRuntime.EventsEmit(a.ctx, "cloud-progress", map[string]interface{}{
				"status":   "progress",
				"current":  i + 1,
				"total":    len(remotePaths),
				"filename": fileName,
			})
		}

		wailsRuntime.EventsEmit(a.ctx, "cloud-progress", map[string]interface{}{
			"status":  "complete",
			"success": successCount,
			"skipped": skippedCount,
			"errors":  errorCount,
		})
	}()

	return nil
}

// WebDAVUploadFiles uploads local files to a remote directory
func (a *App) WebDAVUploadFiles(url, user, password string, localPaths []string, remoteDir string) error {
	url = strings.TrimRight(url, "/")
	client := syncpkg.NewWebDAVClient(url, user, password)

	wailsRuntime.EventsEmit(a.ctx, "cloud-upload-progress", map[string]interface{}{
		"status": "start",
		"total":  len(localPaths),
	})

	go func() {
		successCount := 0
		for i, localPath := range localPaths {
			if err := client.UploadFile(localPath, remoteDir); err != nil {
				a.logger.Error("Failed to upload %s: %v", localPath, err)
			} else {
				successCount++
			}

			wailsRuntime.EventsEmit(a.ctx, "cloud-upload-progress", map[string]interface{}{
				"status":   "progress",
				"current":  i + 1,
				"total":    len(localPaths),
				"filename": filepath.Base(localPath),
			})
		}

		wailsRuntime.EventsEmit(a.ctx, "cloud-upload-progress", map[string]interface{}{
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

	wailsRuntime.EventsEmit(a.ctx, "cloud-progress", map[string]interface{}{
		"status": "start",
		"total":  len(remotePaths),
	})

	go func() {
		successCount := 0
		skippedCount := 0

		for i, remotePath := range remotePaths {
			fileName := filepath.Base(remotePath)

			// Check for duplicates by remote path
			existingTab, _ := a.store.GetTabByPath(remotePath)
			if existingTab != nil {
				a.logger.Info("Skipping duplicate cloud file %s", fileName)
				skippedCount++
				wailsRuntime.EventsEmit(a.ctx, "cloud-progress", map[string]interface{}{
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
			if ext == ".pdf" {
				fileType = "pdf"
			} else if ext == ".gp" || ext == ".gp3" || ext == ".gp4" || ext == ".gp5" || ext == ".gpx" || ext == ".xml" || ext == ".musicxml" || ext == ".mxl" {
				fileType = "gp"
			}

			// Create tab record
			tab := store.Tab{
				ID:          generateID(),
				Title:       title,
				Artist:      artist,
				FilePath:    remotePath, // Store remote path
				Type:        fileType,
				IsManaged:   false,
				IsCloud:     true,
				CategoryIDs: []string{store.SystemCloudCategoryID},
				AddedAt:     time.Now().Unix(),
			}

			if err := a.store.AddTab(tab); err != nil {
				a.logger.Error("Failed to add cloud tab %s: %v", fileName, err)
			} else {
				successCount++
			}

			wailsRuntime.EventsEmit(a.ctx, "cloud-progress", map[string]interface{}{
				"status":   "progress",
				"current":  i + 1,
				"total":    len(remotePaths),
				"filename": fileName,
			})
		}

		wailsRuntime.EventsEmit(a.ctx, "cloud-progress", map[string]interface{}{
			"status":  "complete",
			"success": successCount,
			"skipped": skippedCount,
			"errors":  0,
		})
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

// DownloadCloudTabToLocal downloads a cloud tab to local storage
// IMPORTANT: This preserves existing metadata - does NOT re-parse the file
func (a *App) DownloadCloudTabToLocal(tabID string) error {
	tab, err := a.store.GetTab(tabID)
	if err != nil {
		return fmt.Errorf("failed to get tab: %w", err)
	}
	if tab == nil {
		return fmt.Errorf("tab not found")
	}
	if !tab.IsCloud {
		return fmt.Errorf("tab is not a cloud tab")
	}

	settings := a.store.GetSettings()
	if !settings.WebDAVEnabled {
		return fmt.Errorf("WebDAV is not enabled")
	}

	client := syncpkg.NewWebDAVClient(
		strings.TrimRight(settings.WebDAVURL, "/"),
		settings.WebDAVUser,
		settings.WebDAVPassword,
	)

	wailsRuntime.EventsEmit(a.ctx, "cloud-download-single", map[string]interface{}{
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
			wailsRuntime.EventsEmit(a.ctx, "cloud-download-single", map[string]interface{}{
				"status": "error",
				"tabId":  tabID,
				"error":  err.Error(),
			})
			return
		}
		tempPath := tempFile.Name()
		tempFile.Close()

		// Download file
		if err := client.DownloadFile(remotePath, tempPath); err != nil {
			a.logger.Error("Failed to download %s: %v", remotePath, err)
			os.Remove(tempPath)
			wailsRuntime.EventsEmit(a.ctx, "cloud-download-single", map[string]interface{}{
				"status": "error",
				"tabId":  tabID,
				"error":  err.Error(),
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
			wailsRuntime.EventsEmit(a.ctx, "cloud-download-single", map[string]interface{}{
				"status": "error",
				"tabId":  tabID,
				"error":  err.Error(),
			})
			return
		}
		os.Remove(tempPath)

		// CRITICAL: Do NOT call ProcessFile - preserve existing metadata
		// Only update the necessary state fields
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
			wailsRuntime.EventsEmit(a.ctx, "cloud-download-single", map[string]interface{}{
				"status": "error",
				"tabId":  tabID,
				"error":  err.Error(),
			})
			return
		}

		// Optionally fetch cover if not already present
		if tab.CoverPath == "" && tab.Artist != "" {
			a.fetchCoverAsync(*tab)
		}

		wailsRuntime.EventsEmit(a.ctx, "cloud-download-single", map[string]interface{}{
			"status": "complete",
			"tabId":  tabID,
			"tab":    tab,
		})
	}()

	return nil
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
