package sync

import (
	"haya-tab/pkg/coverpool"
	"haya-tab/pkg/logger"
	"haya-tab/pkg/store"
	"haya-tab/pkg/worker"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// mockEventEmitter implements EventEmitter for testing
type mockEventEmitter struct {
	events []mockEvent
}

type mockEvent struct {
	name string
	data interface{}
}

func (m *mockEventEmitter) Emit(eventName string, data interface{}) {
	m.events = append(m.events, mockEvent{name: eventName, data: data})
}

func setupTestSyncService(t *testing.T) (*SyncService, *store.DBStore, string, func()) {
	tmpDir, err := os.MkdirTemp("", "sync-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")
	testStore := store.NewDBStore(dbPath)
	if err := testStore.Initialize(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to initialize store: %v", err)
	}

	testLogger := logger.NewLogger(tmpDir)
	coverPool := coverpool.NewCoverPool(1, nil)
	emitter := &mockEventEmitter{}
	mbWorker := worker.NewMBWorker(testStore, testLogger)

	service := NewSyncService(testStore, testLogger, coverPool, emitter, tmpDir, mbWorker)

	cleanup := func() {
		testStore.Close()
		testLogger.Close()
		os.RemoveAll(tmpDir)
	}

	return service, testStore, tmpDir, cleanup
}

func TestNewSyncService(t *testing.T) {
	service, _, _, cleanup := setupTestSyncService(t)
	defer cleanup()

	if service == nil {
		t.Fatal("NewSyncService returned nil")
	}
	if service.store == nil {
		t.Error("SyncService.store is nil")
	}
	if service.logger == nil {
		t.Error("SyncService.logger is nil")
	}
	if service.coverPool == nil {
		t.Error("SyncService.coverPool is nil")
	}
	if service.emitter == nil {
		t.Error("SyncService.emitter is nil")
	}
}

func TestSyncService_ProcessFile(t *testing.T) {
	service, _, tmpDir, cleanup := setupTestSyncService(t)
	defer cleanup()

	tests := []struct {
		name     string
		filename string
		wantType string
	}{
		{
			name:     "PDF file",
			filename: "Test Artist - Test Song.pdf",
			wantType: "pdf",
		},
		{
			name:     "GP5 file",
			filename: "Artist - Album - Title.gp5",
			wantType: "gp",
		},
		{
			name:     "GPX file",
			filename: "Song.gpx",
			wantType: "gp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tmpDir, tt.filename)
			f, err := os.Create(testFile)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}
			f.Close()

			tab := service.ProcessFile(testFile)

			if tab.ID == "" {
				t.Error("ProcessFile() returned empty ID")
			}
			if tab.Title == "" {
				t.Error("ProcessFile() returned empty Title")
			}
			if tab.FilePath != testFile {
				t.Errorf("FilePath = %v, want %v", tab.FilePath, testFile)
			}
			if tab.Type != tt.wantType {
				t.Errorf("Type = %v, want %v", tab.Type, tt.wantType)
			}
		})
	}
}

func TestSyncService_isSupportedExtension(t *testing.T) {
	service, _, _, cleanup := setupTestSyncService(t)
	defer cleanup()

	tests := []struct {
		name string
		ext  string
		want bool
	}{
		{name: "PDF", ext: ".pdf", want: true},
		{name: "GP", ext: ".gp", want: true},
		{name: "GP3", ext: ".gp3", want: true},
		{name: "GP4", ext: ".gp4", want: true},
		{name: "GP5", ext: ".gp5", want: true},
		{name: "GPX", ext: ".gpx", want: true},
		{name: "XML", ext: ".xml", want: true},
		{name: "MusicXML", ext: ".musicxml", want: true},
		{name: "MXL", ext: ".mxl", want: true},
		{name: "TXT", ext: ".txt", want: false},
		{name: "DOC", ext: ".doc", want: false},
		{name: "Empty", ext: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := service.isSupportedExtension(tt.ext)
			if got != tt.want {
				t.Errorf("isSupportedExtension(%v) = %v, want %v", tt.ext, got, tt.want)
			}
		})
	}
}

func TestSyncService_getFileType(t *testing.T) {
	service, _, _, cleanup := setupTestSyncService(t)
	defer cleanup()

	tests := []struct {
		name string
		ext  string
		want string
	}{
		{name: "PDF", ext: ".pdf", want: "pdf"},
		{name: "GP", ext: ".gp", want: "gp"},
		{name: "GP5", ext: ".gp5", want: "gp"},
		{name: "GPX", ext: ".gpx", want: "gp"},
		{name: "XML", ext: ".xml", want: "gp"},
		{name: "Unknown", ext: ".txt", want: "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := service.getFileType(tt.ext)
			if got != tt.want {
				t.Errorf("getFileType(%v) = %v, want %v", tt.ext, got, tt.want)
			}
		})
	}
}

func TestSyncService_generateUniqueTitle(t *testing.T) {
	service, testStore, _, cleanup := setupTestSyncService(t)
	defer cleanup()

	baseTitle := "Test Song"

	// Add a tab with the base title
	testStore.AddTab(store.Tab{
		ID:    "tab-1",
		Title: baseTitle,
	})

	// Generate unique title
	uniqueTitle := service.generateUniqueTitle(baseTitle)

	if uniqueTitle == baseTitle {
		t.Error("generateUniqueTitle() returned same title")
	}
	if uniqueTitle != "Test Song_copy1" {
		t.Errorf("generateUniqueTitle() = %v, want Test Song_copy1", uniqueTitle)
	}

	// Add the copy1 title and generate another
	testStore.AddTab(store.Tab{
		ID:    "tab-2",
		Title: uniqueTitle,
	})

	uniqueTitle2 := service.generateUniqueTitle(baseTitle)
	if uniqueTitle2 != "Test Song_copy2" {
		t.Errorf("generateUniqueTitle() = %v, want Test Song_copy2", uniqueTitle2)
	}
}

func TestSyncService_TriggerSync_NoSyncPaths(t *testing.T) {
	service, testStore, _, cleanup := setupTestSyncService(t)
	defer cleanup()

	// Ensure no sync paths configured
	settings := testStore.GetSettings()
	settings.SyncPaths = []string{}
	testStore.UpdateSettings(settings)

	result, err := service.TriggerSync()
	if err != nil {
		t.Fatalf("TriggerSync() error = %v", err)
	}

	if result != "No sync paths configured" {
		t.Errorf("TriggerSync() = %v, want 'No sync paths configured'", result)
	}
}

func TestSyncService_TriggerSync_WithFiles(t *testing.T) {
	service, testStore, tmpDir, cleanup := setupTestSyncService(t)
	defer cleanup()

	// Create test directory with files
	syncDir := filepath.Join(tmpDir, "sync")
	os.MkdirAll(syncDir, 0755)

	// Create test files
	testFiles := []string{
		"Artist1 - Song1.pdf",
		"Artist2 - Song2.gp5",
		"Artist3 - Song3.gpx",
	}

	for _, filename := range testFiles {
		f, err := os.Create(filepath.Join(syncDir, filename))
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		f.Close()
	}

	// Configure sync path
	settings := testStore.GetSettings()
	settings.SyncPaths = []string{syncDir}
	settings.SyncStrategy = "skip"
	testStore.UpdateSettings(settings)

	// Trigger sync
	result, err := service.TriggerSync()
	if err != nil {
		t.Fatalf("TriggerSync() error = %v", err)
	}

	if result == "" {
		t.Error("TriggerSync() returned empty result")
	}

	// Verify tabs were added
	tabs, err := testStore.GetAllTabs()
	if err != nil {
		t.Fatalf("GetAllTabs() error = %v", err)
	}

	if len(tabs) != 3 {
		t.Errorf("Expected 3 tabs, got %d", len(tabs))
	}

	// Verify last sync time was updated
	updatedSettings := testStore.GetSettings()
	if updatedSettings.LastSyncTime == 0 {
		t.Error("LastSyncTime was not updated")
	}
}

func TestSyncService_TriggerSync_SkipStrategy(t *testing.T) {
	service, testStore, tmpDir, cleanup := setupTestSyncService(t)
	defer cleanup()

	// Add existing tab
	testStore.AddTab(store.Tab{
		ID:    "existing-tab",
		Title: "Existing Song",
	})

	// Create sync directory
	syncDir := filepath.Join(tmpDir, "sync")
	os.MkdirAll(syncDir, 0755)

	// Create file with same title
	testFile := filepath.Join(syncDir, "Artist - Existing Song.pdf")
	f, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	f.Close()

	// Configure sync with skip strategy
	settings := testStore.GetSettings()
	settings.SyncPaths = []string{syncDir}
	settings.SyncStrategy = "skip"
	testStore.UpdateSettings(settings)

	// Trigger sync
	service.TriggerSync()

	// Verify only one tab exists (skipped duplicate)
	tabs, _ := testStore.GetAllTabs()
	if len(tabs) != 1 {
		t.Errorf("Expected 1 tab (skipped duplicate), got %d", len(tabs))
	}
}

func TestSyncService_TriggerSync_OverwriteStrategy(t *testing.T) {
	service, testStore, tmpDir, cleanup := setupTestSyncService(t)
	defer cleanup()

	// Add existing tab
	testStore.AddTab(store.Tab{
		ID:    "existing-tab",
		Title: "Existing Song",
	})

	// Create sync directory
	syncDir := filepath.Join(tmpDir, "sync")
	os.MkdirAll(syncDir, 0755)

	// Create file with same title
	testFile := filepath.Join(syncDir, "Artist - Existing Song.pdf")
	f, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	f.Close()

	// Configure sync with overwrite strategy
	settings := testStore.GetSettings()
	settings.SyncPaths = []string{syncDir}
	settings.SyncStrategy = "overwrite"
	testStore.UpdateSettings(settings)

	// Trigger sync
	service.TriggerSync()

	// Verify two tabs exist (original + renamed copy)
	tabs, _ := testStore.GetAllTabs()
	if len(tabs) != 2 {
		t.Errorf("Expected 2 tabs (original + copy), got %d", len(tabs))
	}

	// Verify one has _copy suffix
	hasCopy := false
	for _, tab := range tabs {
		if tab.Title == "Existing Song_copy1" {
			hasCopy = true
			break
		}
	}
	if !hasCopy {
		t.Error("Expected to find tab with _copy1 suffix")
	}
}

func TestSyncService_TriggerSync_EmitsEvents(t *testing.T) {
	service, testStore, tmpDir, cleanup := setupTestSyncService(t)
	defer cleanup()

	emitter := service.emitter.(*mockEventEmitter)

	// Create sync directory with a file
	syncDir := filepath.Join(tmpDir, "sync")
	os.MkdirAll(syncDir, 0755)

	testFile := filepath.Join(syncDir, "Test.pdf")
	f, _ := os.Create(testFile)
	f.Close()

	// Configure sync path
	settings := testStore.GetSettings()
	settings.SyncPaths = []string{syncDir}
	testStore.UpdateSettings(settings)

	// Trigger sync
	service.TriggerSync()

	// Verify events were emitted
	if len(emitter.events) == 0 {
		t.Error("No events were emitted")
	}

	// Check for sync-started event
	hasStarted := false
	hasCompleted := false
	for _, event := range emitter.events {
		if event.name == "sync-started" {
			hasStarted = true
		}
		if event.name == "sync-completed" {
			hasCompleted = true
		}
	}

	if !hasStarted {
		t.Error("sync-started event was not emitted")
	}
	if !hasCompleted {
		t.Error("sync-completed event was not emitted")
	}
}

func TestSyncService_FetchCoverAsync(t *testing.T) {
	service, _, _, cleanup := setupTestSyncService(t)
	defer cleanup()

	tab := store.Tab{
		ID:     "test-tab",
		Title:  "Test Song",
		Artist: "Test Artist",
		Album:  "Test Album",
	}

	// This should not panic or error
	service.FetchCoverAsync(tab)

	// Wait a bit for async operation
	time.Sleep(100 * time.Millisecond)
}

func TestSyncService_FetchCoverAsync_NoArtist(t *testing.T) {
	service, _, _, cleanup := setupTestSyncService(t)
	defer cleanup()

	tab := store.Tab{
		ID:    "test-tab",
		Title: "Test Song",
		// No artist - should return early
	}

	// This should not panic or error
	service.FetchCoverAsync(tab)
}
