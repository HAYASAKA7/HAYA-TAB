package app

import (
	"haya-tab/pkg/store"
	"testing"
)

func TestApp_SaveSettings(t *testing.T) {
	app, _ := setupTestApp(t)
	defer cleanupTestApp(app)

	settings := app.GetSettings()
	originalTheme := settings.Theme

	t.Run("update theme", func(t *testing.T) {
		settings.Theme = "dark"
		err := app.SaveSettings(settings)
		if err != nil {
			t.Fatalf("SaveSettings() error = %v", err)
		}

		updated := app.GetSettings()
		if updated.Theme != "dark" {
			t.Errorf("Theme = %q, want %q", updated.Theme, "dark")
		}
	})

	t.Run("restore original theme", func(t *testing.T) {
		settings.Theme = originalTheme
		err := app.SaveSettings(settings)
		if err != nil {
			t.Fatalf("SaveSettings() error = %v", err)
		}
	})
}

func TestApp_SaveSettings_SyncPaths(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app)

	settings := app.GetSettings()

	t.Run("add sync paths", func(t *testing.T) {
		settings.SyncPaths = []string{tmpDir}
		err := app.SaveSettings(settings)
		if err != nil {
			t.Fatalf("SaveSettings() error = %v", err)
		}

		updated := app.GetSettings()
		if len(updated.SyncPaths) != 1 {
			t.Errorf("SyncPaths length = %d, want 1", len(updated.SyncPaths))
		}
	})

	t.Run("clear sync paths", func(t *testing.T) {
		settings.SyncPaths = []string{}
		err := app.SaveSettings(settings)
		if err != nil {
			t.Fatalf("SaveSettings() error = %v", err)
		}

		updated := app.GetSettings()
		if len(updated.SyncPaths) != 0 {
			t.Errorf("SyncPaths length = %d, want 0", len(updated.SyncPaths))
		}
	})
}

func TestApp_SaveSettings_AutoSync(t *testing.T) {
	app, _ := setupTestApp(t)
	defer cleanupTestApp(app)

	settings := app.GetSettings()

	t.Run("enable auto sync", func(t *testing.T) {
		settings.AutoSyncEnabled = true
		settings.AutoSyncFrequency = "weekly"
		err := app.SaveSettings(settings)
		if err != nil {
			t.Fatalf("SaveSettings() error = %v", err)
		}

		updated := app.GetSettings()
		if !updated.AutoSyncEnabled {
			t.Error("AutoSyncEnabled should be true")
		}
		if updated.AutoSyncFrequency != "weekly" {
			t.Errorf("AutoSyncFrequency = %q, want %q", updated.AutoSyncFrequency, "weekly")
		}
	})

	t.Run("disable auto sync", func(t *testing.T) {
		settings.AutoSyncEnabled = false
		err := app.SaveSettings(settings)
		if err != nil {
			t.Fatalf("SaveSettings() error = %v", err)
		}

		updated := app.GetSettings()
		if updated.AutoSyncEnabled {
			t.Error("AutoSyncEnabled should be false")
		}
	})
}

func TestApp_SaveSettings_StoragePaths(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app)

	settings := app.GetSettings()

	t.Run("update storage path", func(t *testing.T) {
		newPath := tmpDir + "/new_storage"
		settings.StoragePath = newPath
		err := app.SaveSettings(settings)
		if err != nil {
			t.Fatalf("SaveSettings() error = %v", err)
		}

		updated := app.GetSettings()
		if updated.StoragePath != newPath {
			t.Errorf("StoragePath = %q, want %q", updated.StoragePath, newPath)
		}
	})

	t.Run("update covers path", func(t *testing.T) {
		newPath := tmpDir + "/new_covers"
		settings.CoversPath = newPath
		err := app.SaveSettings(settings)
		if err != nil {
			t.Fatalf("SaveSettings() error = %v", err)
		}

		updated := app.GetSettings()
		if updated.CoversPath != newPath {
			t.Errorf("CoversPath = %q, want %q", updated.CoversPath, newPath)
		}
	})

	t.Run("WebDAV settings change", func(t *testing.T) {
		settings.WebDAVEnabled = true
		settings.WebDAVURL = "http://localhost:8080"
		settings.WebDAVUser = "user"
		settings.WebDAVPassword = "pass"
		err := app.SaveSettings(settings)
		if err != nil {
			t.Fatalf("SaveSettings() error = %v", err)
		}
		
		// Verification would involve checking if WebDAVInitialize was called, 
		// but since it's in a goroutine, we just check it doesn't panic.
	})

	t.Run("FileWatcher path change", func(t *testing.T) {
		settings.SyncPaths = []string{tmpDir}
		err := app.SaveSettings(settings)
		if err != nil {
			t.Fatalf("SaveSettings() error = %v", err)
		}
		
		if app.fileWatcher == nil {
			t.Error("fileWatcher should be initialized when sync paths are added")
		}
		
		// Clear sync paths should stop watcher
		settings.SyncPaths = []string{}
		err = app.SaveSettings(settings)
		if err != nil {
			t.Fatalf("SaveSettings() error = %v", err)
		}
		
		if app.fileWatcher != nil {
			t.Error("fileWatcher should be nil when sync paths are cleared")
		}
	})
}

func TestApp_Settings_DefaultValues(t *testing.T) {
	app, _ := setupTestApp(t)
	defer cleanupTestApp(app)

	settings := app.GetSettings()

	// Test default values
	if settings.Theme != "system" {
		t.Errorf("Default theme = %q, want %q", settings.Theme, "system")
	}

	if settings.Language != "" && settings.Language != store.DetectSystemLocale() {
		// Language should be either empty or detected system locale
		t.Logf("Language = %q (detected = %q)", settings.Language, store.DetectSystemLocale())
	}
}
