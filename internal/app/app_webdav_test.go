package app

import (
	"haya-tab/pkg/store"
	"testing"
)

func TestApp_WebDAVTestConnection_EmptyURL(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app, tmpDir)

	err := app.WebDAVTestConnection("", "user", "pass")
	if err == nil {
		t.Error("Expected error for empty URL")
	}
}

func TestApp_WebDAVTestConnection_InvalidURL(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app, tmpDir)

	// Test with an invalid/unreachable URL
	err := app.WebDAVTestConnection("http://invalid.local.test:9999", "user", "pass")
	if err == nil {
		t.Error("Expected error for invalid URL")
	}
}

func TestApp_WebDAVTestConnection_URLNormalization(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app, tmpDir)

	// Even if connection fails, the URL normalization should happen
	// The error message should not contain trailing slashes
	err := app.WebDAVTestConnection("http://nonexistent.test///", "user", "pass")
	if err == nil {
		t.Error("Expected error for non-existent host")
	}
}

func TestApp_WebDAVCheckStatus_NotEnabled(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app, tmpDir)

	// WebDAV is not enabled by default
	status := app.WebDAVCheckStatus()
	if status {
		t.Error("WebDAV status should be false when not enabled")
	}
}

func TestApp_WebDAVCheckStatus_EnabledButInvalidURL(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app, tmpDir)

	settings := app.GetSettings()
	settings.WebDAVEnabled = true
	settings.WebDAVURL = "http://invalid.local.test:9999"
	settings.WebDAVUser = "test"
	settings.WebDAVPassword = "test"
	app.store.UpdateSettings(settings)

	status := app.WebDAVCheckStatus()
	if status {
		t.Error("WebDAV status should be false when connection fails")
	}
}

func TestApp_WebDAVCheckStatus_NoURL(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app, tmpDir)

	settings := app.GetSettings()
	settings.WebDAVEnabled = true
	settings.WebDAVURL = ""
	app.store.UpdateSettings(settings)

	status := app.WebDAVCheckStatus()
	if status {
		t.Error("WebDAV status should be false when URL is empty")
	}
}

func TestApp_DownloadCloudTabToLocal_NotCloud(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app, tmpDir)

	// Add a non-cloud tab
	app.store.AddTab(store.Tab{
		ID:      "tab1",
		Title:   "Local Tab",
		IsCloud: false,
	})

	err := app.DownloadCloudTabToLocal("tab1")
	if err == nil {
		t.Error("Expected error for non-cloud tab")
	}
}

func TestApp_DownloadCloudTabToLocal_NotFound(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app, tmpDir)

	err := app.DownloadCloudTabToLocal("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent tab")
	}
}

func TestApp_DownloadCloudTabToLocal_WebDAVDisabled(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app, tmpDir)

	// Add a cloud tab
	app.store.AddTab(store.Tab{
		ID:       "cloud_tab",
		Title:    "Cloud Tab",
		IsCloud:  true,
		FilePath: "/remote/path/file.gp5",
	})

	// WebDAV is disabled by default
	err := app.DownloadCloudTabToLocal("cloud_tab")
	if err == nil {
		t.Error("Expected error when WebDAV is not enabled")
	}
}

func TestParseMetadataFromFilename_WebDAV(t *testing.T) {
	tests := []struct {
		name       string
		filename   string
		wantTitle  string
		wantArtist string
	}{
		{
			name:       "Unicode characters",
			filename:   "YUI - Again.gp5",
			wantTitle:  "Again",
			wantArtist: "YUI",
		},
		{
			name:       "Chinese filename",
			filename:   "周杰伦 - 晴天.pdf",
			wantTitle:  "晴天",
			wantArtist: "周杰伦",
		},
		{
			name:       "Japanese filename",
			filename:   "YOASOBI - 夜に駆ける.gp5",
			wantTitle:  "夜に駆ける",
			wantArtist: "YOASOBI",
		},
		{
			name:       "Korean filename",
			filename:   "BTS - Dynamite.gp5",
			wantTitle:  "Dynamite",
			wantArtist: "BTS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTitle, gotArtist := parseMetadataFromFilename(tt.filename)
			if gotTitle != tt.wantTitle {
				t.Errorf("title = %q, want %q", gotTitle, tt.wantTitle)
			}
			if gotArtist != tt.wantArtist {
				t.Errorf("artist = %q, want %q", gotArtist, tt.wantArtist)
			}
		})
	}
}

func TestApp_WebDAVScanRemoteFiles_InvalidURL(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app, tmpDir)

	_, err := app.WebDAVScanRemoteFiles("http://invalid.local.test:9999", "user", "pass", "/")
	if err == nil {
		t.Error("Expected error for invalid URL")
	}
}

func TestApp_WebDAVListRemoteDirectories_InvalidURL(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app, tmpDir)

	_, err := app.WebDAVListRemoteDirectories("http://invalid.local.test:9999", "user", "pass", "/")
	if err == nil {
		t.Error("Expected error for invalid URL")
	}
}

func TestApp_WebDAVListDir_InvalidURL(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app, tmpDir)

	_, err := app.WebDAVListDir("http://invalid.local.test:9999", "user", "pass", "/")
	if err == nil {
		t.Error("Expected error for invalid URL")
	}
}
