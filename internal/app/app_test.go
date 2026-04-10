package app

import (
	"haya-tab/pkg/coverpool"
	"haya-tab/pkg/logger"
	"haya-tab/pkg/store"
	syncpkg "haya-tab/pkg/sync"
	"haya-tab/pkg/worker"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// setupTestApp creates a minimal App with a real DBStore for integration-style tests.
// It sets up isolated storage and covers directories within the temp directory.
func setupTestApp(t *testing.T) (*App, string) {
	t.Helper()
	tmpDir := t.TempDir()

	appDir := filepath.Join(tmpDir, "app")
	os.MkdirAll(appDir, 0755)

	dbPath := filepath.Join(appDir, "data", "test.db")
	os.MkdirAll(filepath.Dir(dbPath), 0755)

	s := store.NewDBStore(dbPath)
	if err := s.Initialize(); err != nil {
		t.Fatalf("Failed to initialize store: %v", err)
	}

	// Set up isolated storage and covers directories
	storagePath := filepath.Join(tmpDir, "storage")
	coversPath := filepath.Join(tmpDir, "covers")
	os.MkdirAll(storagePath, 0755)
	os.MkdirAll(coversPath, 0755)

	// Update settings to use isolated directories
	settings := s.GetSettings()
	settings.StoragePath = storagePath
	settings.CoversPath = coversPath
	s.UpdateSettings(settings)

	l := logger.NewLogger(appDir)
	cp := coverpool.NewCoverPool(1, nil) // No downloader for tests
	mb := worker.NewMBWorker(s, l)

	app := &App{
		store:     s,
		logger:    l,
		coverPool: cp,
		mbWorker:  mb,
	}

	emitter := &WailsEventEmitter{}
	app.syncService = syncpkg.NewSyncService(s, l, cp, emitter, appDir, mb, nil)
	app.volumeCache = syncpkg.NewVolumeCache(5 * time.Minute)

	return app, tmpDir
}

func cleanupTestApp(app *App) {
	if app != nil && app.logger != nil {
		app.logger.Close()
	}
	if app != nil && app.store != nil {
		app.store.Close()
	}
}

// --- Tests for pure functions ---

func TestGenerateID(t *testing.T) {
	id1 := generateID()
	id2 := generateID()

	if id1 == "" {
		t.Error("generateID() returned empty string")
	}
	if !strings.HasPrefix(id1, "tab_") {
		t.Errorf("generateID() = %q, want prefix 'tab_'", id1)
	}
	// Two consecutive IDs should differ (nano-precision)
	if id1 == id2 {
		t.Errorf("generateID() produced duplicate IDs: %q", id1)
	}
}

func TestParseMetadataFromFilename(t *testing.T) {
	tests := []struct {
		name       string
		filename   string
		wantTitle  string
		wantArtist string
	}{
		{
			name:       "Artist - Title format",
			filename:   "Led Zeppelin - Stairway to Heaven.gp5",
			wantTitle:  "Stairway to Heaven",
			wantArtist: "Led Zeppelin",
		},
		{
			name:       "Title only",
			filename:   "Bohemian Rhapsody.pdf",
			wantTitle:  "Bohemian Rhapsody",
			wantArtist: "",
		},
		{
			name:       "No extension",
			filename:   "No Extension",
			wantTitle:  "No Extension",
			wantArtist: "",
		},
		{
			name:       "Multiple dashes - only splits on first",
			filename:   "Artist - Title - Extra.gpx",
			wantTitle:  "Title - Extra",
			wantArtist: "Artist",
		},
		{
			name:       "Empty filename",
			filename:   "",
			wantTitle:  "",
			wantArtist: "",
		},
		{
			name:       "Just extension",
			filename:   ".gp5",
			wantTitle:  "",
			wantArtist: "",
		},
		{
			name:       "Spaces around dash",
			filename:   "  Queen  -  We Will Rock You  .gp",
			wantTitle:  "We Will Rock You",
			wantArtist: "Queen",
		},
		{
			name:       "Unicode artist and title",
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

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source file
	srcPath := filepath.Join(tmpDir, "source.txt")
	content := []byte("Hello, World!")
	if err := os.WriteFile(srcPath, content, 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Copy file
	dstPath := filepath.Join(tmpDir, "destination.txt")
	if err := copyFile(srcPath, dstPath); err != nil {
		t.Fatalf("copyFile() error = %v", err)
	}

	// Verify destination
	dstContent, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("Failed to read destination: %v", err)
	}
	if string(dstContent) != string(content) {
		t.Errorf("content = %q, want %q", string(dstContent), string(content))
	}
}

func TestCopyFile_SourceNotExist(t *testing.T) {
	tmpDir := t.TempDir()

	err := copyFile(filepath.Join(tmpDir, "nonexistent"), filepath.Join(tmpDir, "dst"))
	if err == nil {
		t.Error("Expected error when source doesn't exist")
	}
}

func TestCopyFile_InvalidDestination(t *testing.T) {
	tmpDir := t.TempDir()

	srcPath := filepath.Join(tmpDir, "source.txt")
	os.WriteFile(srcPath, []byte("data"), 0644)

	// Destination in a non-existent directory
	err := copyFile(srcPath, filepath.Join(tmpDir, "no-such-dir", "dst.txt"))
	if err == nil {
		t.Error("Expected error when destination directory doesn't exist")
	}
}

func TestCopyFile_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()

	srcPath := filepath.Join(tmpDir, "empty.txt")
	os.WriteFile(srcPath, []byte{}, 0644)

	dstPath := filepath.Join(tmpDir, "empty_copy.txt")
	if err := copyFile(srcPath, dstPath); err != nil {
		t.Fatalf("copyFile() error = %v", err)
	}

	info, err := os.Stat(dstPath)
	if err != nil {
		t.Fatalf("Failed to stat destination: %v", err)
	}
	if info.Size() != 0 {
		t.Errorf("Copied file size = %d, want 0", info.Size())
	}
}

func TestCopyFile_LargeFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a 1MB file
	data := make([]byte, 1024*1024)
	for i := range data {
		data[i] = byte(i % 256)
	}

	srcPath := filepath.Join(tmpDir, "large.bin")
	os.WriteFile(srcPath, data, 0644)

	dstPath := filepath.Join(tmpDir, "large_copy.bin")
	if err := copyFile(srcPath, dstPath); err != nil {
		t.Fatalf("copyFile() error = %v", err)
	}

	dstData, _ := os.ReadFile(dstPath)
	if len(dstData) != len(data) {
		t.Errorf("Copied file size = %d, want %d", len(dstData), len(data))
	}
}

// --- Tests for App methods that use the store ---

func TestNewApp(t *testing.T) {
	app := NewApp()
	if app == nil {
		t.Fatal("NewApp() returned nil")
	}
}

func TestApp_SetGetFileServerPort(t *testing.T) {
	app := NewApp()
	app.SetFileServerPort(8080)
	if got := app.GetFileServerPort(); got != 8080 {
		t.Errorf("GetFileServerPort() = %d, want 8080", got)
	}

	app.SetFileServerPort(0)
	if got := app.GetFileServerPort(); got != 0 {
		t.Errorf("GetFileServerPort() = %d, want 0", got)
	}
}

func TestApp_GetStorageDir(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app)
	_ = tmpDir // Ignore unused variable error

	dir := app.GetStorageDir()
	if dir == "" {
		t.Error("GetStorageDir() returned empty string")
	}

	// Should be a real directory
	info, err := os.Stat(dir)
	if err != nil {
		t.Errorf("GetStorageDir() returned non-existent path: %v", err)
	}
	if !info.IsDir() {
		t.Error("GetStorageDir() returned a non-directory path")
	}
}

func TestApp_GetStorageDir_CustomPath(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app)
	_ = tmpDir // Ignore unused variable error

	customPath := filepath.Join(tmpDir, "custom_storage")
	settings := app.store.GetSettings()
	settings.StoragePath = customPath
	app.store.UpdateSettings(settings)

	got := app.GetStorageDir()
	if got != customPath {
		t.Errorf("GetStorageDir() = %q, want %q", got, customPath)
	}

	// Should create the directory
	info, err := os.Stat(customPath)
	if err != nil {
		t.Fatalf("Custom storage dir not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("Custom storage path is not a directory")
	}
}

func TestApp_GetCoversDir(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app)
	_ = tmpDir // Ignore unused variable error

	dir := app.GetCoversDir()
	if dir == "" {
		t.Error("GetCoversDir() returned empty string")
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Errorf("GetCoversDir() returned non-existent path: %v", err)
	}
	if !info.IsDir() {
		t.Error("GetCoversDir() returned a non-directory path")
	}
}

func TestApp_GetCoversDir_CustomPath(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app)
	_ = tmpDir // Ignore unused variable error

	customPath := filepath.Join(tmpDir, "custom_covers")
	settings := app.store.GetSettings()
	settings.CoversPath = customPath
	app.store.UpdateSettings(settings)

	got := app.GetCoversDir()
	if got != customPath {
		t.Errorf("GetCoversDir() = %q, want %q", got, customPath)
	}
}

func TestApp_ResolveTabPath(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app)
	_ = tmpDir // Ignore unused variable error

	tests := []struct {
		name      string
		path      string
		isManaged bool
		wantAbs   bool
		wantSame  bool // Expect returned path == input path
	}{
		{
			name:      "non-managed returns path as-is",
			path:      "/some/absolute/path.gp5",
			isManaged: false,
			wantSame:  true,
		},
		{
			name:      "empty path returns empty",
			path:      "",
			isManaged: true,
			wantSame:  true,
		},
		{
			name:      "managed relative path gets resolved",
			path:      "tab_12345.gp5",
			isManaged: true,
			wantAbs:   true,
		},
		{
			name:      "managed absolute path stays absolute",
			path:      filepath.Join(tmpDir, "file.gp5"),
			isManaged: true,
			wantSame:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := app.ResolveTabPath(tt.path, tt.isManaged)
			if tt.wantSame && got != tt.path {
				t.Errorf("ResolveTabPath() = %q, want %q", got, tt.path)
			}
			if tt.wantAbs && !filepath.IsAbs(got) {
				t.Errorf("ResolveTabPath() = %q, want absolute path", got)
			}
			if tt.wantAbs && !strings.HasSuffix(got, tt.path) {
				t.Errorf("ResolveTabPath() = %q, should end with %q", got, tt.path)
			}
		})
	}
}

func TestApp_ResolveCoverPath(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app)
	_ = tmpDir // Ignore unused variable error

	tests := []struct {
		name     string
		path     string
		wantAbs  bool
		wantSame bool
	}{
		{
			name:     "empty path returns empty",
			path:     "",
			wantSame: true,
		},
		{
			name:     "absolute path stays absolute",
			path:     filepath.Join(tmpDir, "cover.jpg"),
			wantSame: true,
		},
		{
			name:    "relative path gets resolved",
			path:    "cover_123.jpg",
			wantAbs: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := app.ResolveCoverPath(tt.path)
			if tt.wantSame && got != tt.path {
				t.Errorf("ResolveCoverPath() = %q, want %q", got, tt.path)
			}
			if tt.wantAbs && !filepath.IsAbs(got) {
				t.Errorf("ResolveCoverPath() = %q, want absolute path", got)
			}
		})
	}
}

func TestApp_GetCover(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app)
	_ = tmpDir // Ignore unused variable error

	t.Run("empty path returns empty", func(t *testing.T) {
		result := app.GetCover("")
		if result != "" {
			t.Errorf("GetCover('') = %q, want empty", result)
		}
	})

	t.Run("non-existent file returns empty", func(t *testing.T) {
		result := app.GetCover(filepath.Join(tmpDir, "no.jpg"))
		if result != "" {
			t.Errorf("GetCover(nonexistent) = %q, want empty", result)
		}
	})

	t.Run("valid file returns base64", func(t *testing.T) {
		imgPath := filepath.Join(tmpDir, "test.jpg")
		os.WriteFile(imgPath, []byte{0xFF, 0xD8, 0xFF, 0xE0}, 0644)

		result := app.GetCover(imgPath)
		if result == "" {
			t.Error("GetCover(valid file) returned empty string")
		}
		// Base64 of 4 bytes should produce a specific output
		if len(result) == 0 {
			t.Error("Expected non-empty base64 output")
		}
	})
}

func TestApp_Shutdown_NilComponents(t *testing.T) {
	app := NewApp()
	// Shutdown with all nil components should not panic
	app.Shutdown()
}

func TestApp_GetAppDir(t *testing.T) {
	dir := getAppDir()
	if dir == "" {
		t.Error("getAppDir() returned empty string")
	}

	// Should be an absolute path
	if !filepath.IsAbs(dir) {
		t.Errorf("getAppDir() = %q, want absolute path", dir)
	}
}
