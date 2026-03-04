package app

import (
	"haya-tab/pkg/store"
	"haya-tab/pkg/watcher"
)

// GetSettings returns the current settings
func (a *App) GetSettings() store.Settings {
	return a.store.GetSettings()
}

// SaveSettings updates the settings
func (a *App) SaveSettings(s store.Settings) error {
	// Update file watcher paths if they changed
	oldSettings := a.store.GetSettings()
	if err := a.store.UpdateSettings(s); err != nil {
		return err
	}

	// Check if WebDAV settings changed
	webdavChanged := oldSettings.WebDAVEnabled != s.WebDAVEnabled ||
		oldSettings.WebDAVURL != s.WebDAVURL ||
		oldSettings.WebDAVUser != s.WebDAVUser ||
		oldSettings.WebDAVPassword != s.WebDAVPassword

	// If WebDAV was just enabled or connection info changed, initialize it
	if webdavChanged && s.WebDAVEnabled && s.WebDAVURL != "" {
		a.logger.Info("WebDAV settings changed, initializing...")
		go func() {
			if err := a.WebDAVInitialize(); err != nil {
				a.logger.Error("WebDAV initialization failed: %v", err)
			}
		}()
	}

	// Update file watcher if sync paths changed (only in app context with logger)
	if len(s.SyncPaths) > 0 && a.logger != nil {
		if a.fileWatcher == nil {
			// Create new watcher with context-aware callback
			a.fileWatcher = watcher.NewFileWatcher(func() {
				a.emitEvent("file-changes-detected", "Files have changed in sync directories")
			})
			a.fileWatcher.SetLogger(a.logger)

			if err := a.fileWatcher.Start(); err != nil {
				a.logger.Error("Failed to start file watcher: %v", err)
			}
		}

		// Update watched paths
		if a.fileWatcher != nil && a.fileWatcher.IsRunning() {
			if err := a.fileWatcher.SetPaths(s.SyncPaths); err != nil {
				a.logger.Error("Failed to update watcher paths: %v", err)
			}
		}
	} else if a.fileWatcher != nil {
		// No sync paths, stop watcher
		a.fileWatcher.Stop()
		a.fileWatcher = nil
	}

	// Check if paths changed to emit notification
	pathsChanged := len(oldSettings.SyncPaths) != len(s.SyncPaths)
	if !pathsChanged {
		for i := range oldSettings.SyncPaths {
			if oldSettings.SyncPaths[i] != s.SyncPaths[i] {
				pathsChanged = true
				break
			}
		}
	}

	if pathsChanged && len(s.SyncPaths) > 0 && a.logger != nil {
		a.logger.Info("File watcher updated with %d paths", len(s.SyncPaths))
	}

	return nil
}
