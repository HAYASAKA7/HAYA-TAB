package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setupTestDB(t *testing.T) (*DBStore, string) {
	tmpDir, err := os.MkdirTemp("", "store-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")
	store := NewDBStore(dbPath)

	if err := store.Initialize(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to initialize store: %v", err)
	}

	return store, tmpDir
}

func cleanupTestDB(store *DBStore, tmpDir string) {
	if store != nil {
		store.Close()
	}
	os.RemoveAll(tmpDir)
}

func TestNewDBStore(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "store-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	store := NewDBStore(dbPath)

	if store == nil {
		t.Fatal("NewDBStore returned nil")
	}
	if store.dbPath != dbPath {
		t.Errorf("dbPath = %v, want %v", store.dbPath, dbPath)
	}
	if store.Settings.Theme != "system" {
		t.Errorf("Default theme = %v, want system", store.Settings.Theme)
	}
}

func TestDBStore_Initialize(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	// Verify database file was created
	if _, err := os.Stat(store.dbPath); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}

	// Verify database connection is working
	if store.db == nil {
		t.Error("Database connection is nil")
	}
}

func TestDBStore_AddAndGetTab(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	tab := Tab{
		ID:       "test-tab-1",
		Title:    "Test Song",
		Artist:   "Test Artist",
		Album:    "Test Album",
		FilePath: "/path/to/file.pdf",
		Type:     "pdf",
	}

	// Add tab
	err := store.AddTab(tab)
	if err != nil {
		t.Fatalf("AddTab() error = %v", err)
	}

	// Get tab
	retrieved, err := store.GetTab("test-tab-1")
	if err != nil {
		t.Fatalf("GetTab() error = %v", err)
	}
	if retrieved == nil {
		t.Fatal("GetTab() returned nil")
	}

	// Verify fields
	if retrieved.ID != tab.ID {
		t.Errorf("ID = %v, want %v", retrieved.ID, tab.ID)
	}
	if retrieved.Title != tab.Title {
		t.Errorf("Title = %v, want %v", retrieved.Title, tab.Title)
	}
	if retrieved.Artist != tab.Artist {
		t.Errorf("Artist = %v, want %v", retrieved.Artist, tab.Artist)
	}
}

func TestDBStore_UpdateTab(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	// Add initial tab
	tab := Tab{
		ID:     "test-tab-1",
		Title:  "Original Title",
		Artist: "Original Artist",
	}
	store.AddTab(tab)

	// Update tab
	tab.Title = "Updated Title"
	tab.Artist = "Updated Artist"
	err := store.AddTab(tab) // AddTab also updates if ID exists
	if err != nil {
		t.Fatalf("Update tab error = %v", err)
	}

	// Verify update
	retrieved, _ := store.GetTab("test-tab-1")
	if retrieved.Title != "Updated Title" {
		t.Errorf("Title = %v, want Updated Title", retrieved.Title)
	}
	if retrieved.Artist != "Updated Artist" {
		t.Errorf("Artist = %v, want Updated Artist", retrieved.Artist)
	}
}

func TestDBStore_DeleteTab(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	// Add tab
	tab := Tab{
		ID:    "test-tab-1",
		Title: "Test Song",
	}
	store.AddTab(tab)

	// Delete tab
	err := store.DeleteTab("test-tab-1")
	if err != nil {
		t.Fatalf("DeleteTab() error = %v", err)
	}

	// Verify deletion
	retrieved, _ := store.GetTab("test-tab-1")
	if retrieved != nil {
		t.Error("Tab should be deleted but was found")
	}
}

func TestDBStore_GetAllTabs(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	// Add multiple tabs
	tabs := []Tab{
		{ID: "tab-1", Title: "Song 1", Artist: "Artist 1"},
		{ID: "tab-2", Title: "Song 2", Artist: "Artist 2"},
		{ID: "tab-3", Title: "Song 3", Artist: "Artist 3"},
	}

	for _, tab := range tabs {
		store.AddTab(tab)
	}

	// Get all tabs
	retrieved, err := store.GetAllTabs()
	if err != nil {
		t.Fatalf("GetAllTabs() error = %v", err)
	}

	if len(retrieved) != 3 {
		t.Errorf("GetAllTabs() returned %d tabs, want 3", len(retrieved))
	}
}

func TestDBStore_SearchTabs(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	// Add tabs with searchable content
	tabs := []Tab{
		{ID: "tab-1", Title: "Stairway to Heaven", Artist: "Led Zeppelin"},
		{ID: "tab-2", Title: "Hotel California", Artist: "Eagles"},
		{ID: "tab-3", Title: "Bohemian Rhapsody", Artist: "Queen"},
	}

	for _, tab := range tabs {
		store.AddTab(tab)
	}

	// Search by title
	results, err := store.SearchTabs("Stairway")
	if err != nil {
		t.Fatalf("SearchTabs() error = %v", err)
	}
	if len(results) != 1 {
		t.Errorf("SearchTabs('Stairway') returned %d results, want 1", len(results))
	}
	if len(results) > 0 && results[0].Title != "Stairway to Heaven" {
		t.Errorf("SearchTabs() returned wrong tab: %v", results[0].Title)
	}

	// Search by artist
	results, err = store.SearchTabs("Queen")
	if err != nil {
		t.Fatalf("SearchTabs() error = %v", err)
	}
	if len(results) != 1 {
		t.Errorf("SearchTabs('Queen') returned %d results, want 1", len(results))
	}
}

func TestDBStore_AddAndGetCategory(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	category := Category{
		ID:   "cat-1",
		Name: "Rock",
	}

	// Add category
	err := store.AddCategory(category)
	if err != nil {
		t.Fatalf("AddCategory() error = %v", err)
	}

	// Get categories
	categories, err := store.GetCategories()
	if err != nil {
		t.Fatalf("GetCategories() error = %v", err)
	}

	found := false
	for _, cat := range categories {
		if cat.ID == "cat-1" && cat.Name == "Rock" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Category not found in GetCategories()")
	}
}

func TestDBStore_DeleteCategory(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	// Add category
	category := Category{
		ID:   "cat-1",
		Name: "Rock",
	}
	store.AddCategory(category)

	// Delete category
	err := store.DeleteCategory("cat-1")
	if err != nil {
		t.Fatalf("DeleteCategory() error = %v", err)
	}

	// Verify deletion
	categories, _ := store.GetCategories()
	for _, cat := range categories {
		if cat.ID == "cat-1" {
			t.Error("Category should be deleted but was found")
		}
	}
}

func TestDBStore_Settings(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	// Get default settings
	settings := store.GetSettings()
	if settings.Theme != "system" {
		t.Errorf("Default theme = %v, want system", settings.Theme)
	}

	// Update settings
	settings.Theme = "dark"
	settings.Language = "zh-CN"
	err := store.UpdateSettings(settings)
	if err != nil {
		t.Fatalf("UpdateSettings() error = %v", err)
	}

	// Verify update
	updated := store.GetSettings()
	if updated.Theme != "dark" {
		t.Errorf("Theme = %v, want dark", updated.Theme)
	}
	if updated.Language != "zh-CN" {
		t.Errorf("Language = %v, want zh-CN", updated.Language)
	}
}

func TestDBStore_HasData(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	// Initially should have no data
	if store.HasData() {
		t.Error("HasData() = true, want false for empty database")
	}

	// Add a tab
	tab := Tab{
		ID:    "test-tab",
		Title: "Test",
	}
	store.AddTab(tab)

	// Now should have data
	if !store.HasData() {
		t.Error("HasData() = false, want true after adding tab")
	}
}

func TestDBStore_UpdateTabOriginCountry(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	// Add tab
	tab := Tab{
		ID:     "test-tab",
		Title:  "Test Song",
		Artist: "Test Artist",
	}
	store.AddTab(tab)

	// Update origin country
	err := store.UpdateTabOriginCountry("test-tab", "JP")
	if err != nil {
		t.Fatalf("UpdateTabOriginCountry() error = %v", err)
	}

	// Verify update
	retrieved, _ := store.GetTab("test-tab")
	if retrieved.OriginCountry != "JP" {
		t.Errorf("OriginCountry = %v, want JP", retrieved.OriginCountry)
	}
}

func TestDBStore_UpdateTabInitials(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	// Add tab
	tab := Tab{
		ID:    "test-tab",
		Title: "Test Song",
	}
	store.AddTab(tab)

	// Update initials
	err := store.UpdateTabInitials("test-tab", "T", "て")
	if err != nil {
		t.Fatalf("UpdateTabInitials() error = %v", err)
	}

	// Verify update
	retrieved, _ := store.GetTab("test-tab")
	if retrieved.InitialAZ != "T" {
		t.Errorf("InitialAZ = %v, want T", retrieved.InitialAZ)
	}
	if retrieved.InitialKana != "て" {
		t.Errorf("InitialKana = %v, want て", retrieved.InitialKana)
	}
}

func TestDBStore_GetTabByPath(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	// Add tab with specific path
	tab := Tab{
		ID:       "test-tab",
		Title:    "Test Song",
		FilePath: "/unique/path/to/file.pdf",
	}
	store.AddTab(tab)

	// Get by path
	retrieved, err := store.GetTabByPath("/unique/path/to/file.pdf")
	if err != nil {
		t.Fatalf("GetTabByPath() error = %v", err)
	}
	if retrieved == nil {
		t.Fatal("GetTabByPath() returned nil")
	}
	if retrieved.ID != "test-tab" {
		t.Errorf("ID = %v, want test-tab", retrieved.ID)
	}
}

func TestDBStore_GetTabByTitle(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	// Add tab with specific title
	tab := Tab{
		ID:    "test-tab",
		Title: "Unique Song Title",
	}
	store.AddTab(tab)

	// Get by title
	retrieved, err := store.GetTabByTitle("Unique Song Title")
	if err != nil {
		t.Fatalf("GetTabByTitle() error = %v", err)
	}
	if retrieved == nil {
		t.Fatal("GetTabByTitle() returned nil")
	}
	if retrieved.ID != "test-tab" {
		t.Errorf("ID = %v, want test-tab", retrieved.ID)
	}
}

func TestDBStore_UpdateLastOpened(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	// Add tab
	tab := Tab{
		ID:    "test-tab",
		Title: "Test Song",
	}
	store.AddTab(tab)

	// Update last opened
	now := time.Now().Unix()
	err := store.UpdateLastOpened("test-tab")
	if err != nil {
		t.Fatalf("UpdateLastOpened() error = %v", err)
	}

	// Verify update
	retrieved, _ := store.GetTab("test-tab")
	if retrieved.LastOpened < now {
		t.Errorf("LastOpened = %v, want >= %v", retrieved.LastOpened, now)
	}
}

func TestDBStore_Close(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer os.RemoveAll(tmpDir)

	// Close should not error
	err := store.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Multiple closes should be safe
	err = store.Close()
	if err != nil {
		t.Errorf("Second Close() error = %v", err)
	}
}
