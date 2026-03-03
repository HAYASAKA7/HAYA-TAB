package store

import (
	"fmt"
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

func TestDBStore_GetRecentCategories(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	// Add categories
	cat1 := Category{ID: "cat-1", Name: "Rock"}
	cat2 := Category{ID: "cat-2", Name: "Jazz"}
	store.AddCategory(cat1)
	store.AddCategory(cat2)

	// Add tabs with categories
	tab1 := Tab{ID: "tab-1", Title: "Song 1", LastOpened: time.Now().Unix()}
	tab1.CategoryIDs = []string{"cat-1"}
	store.AddTab(tab1)
	store.SetTabCategories("tab-1", []string{"cat-1"}, time.Now().Unix())

	tab2 := Tab{ID: "tab-2", Title: "Song 2", LastOpened: time.Now().Unix() - 100}
	tab2.CategoryIDs = []string{"cat-2"}
	store.AddTab(tab2)
	store.SetTabCategories("tab-2", []string{"cat-2"}, time.Now().Unix())

	// Get recent categories
	categories, err := store.GetRecentCategories(10)
	if err != nil {
		t.Fatalf("GetRecentCategories() error = %v", err)
	}

	// Should return categories that have tabs
	if len(categories) == 0 {
		t.Error("Expected categories with tabs, got 0")
	}
}

func TestDBStore_MoveCategory(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	// Add categories
	cat1 := Category{ID: "cat-1", Name: "Rock"}
	cat2 := Category{ID: "cat-2", Name: "Jazz"}
	store.AddCategory(cat1)
	store.AddCategory(cat2)

	// Move category
	err := store.MoveCategory("cat-1", "cat-2")
	if err != nil {
		t.Fatalf("MoveCategory() error = %v", err)
	}

	// Verify parent changed
	retrieved, _ := store.GetCategory("cat-1")
	if retrieved.ParentID != "cat-2" {
		t.Errorf("ParentID = %v, want cat-2", retrieved.ParentID)
	}
}

func TestDBStore_EnsureCloudCategory(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	// Ensure cloud category
	err := store.EnsureCloudCategory()
	if err != nil {
		t.Fatalf("EnsureCloudCategory() error = %v", err)
	}

	// Verify cloud category exists
	cloudCat, err := store.GetCategory(SystemCloudCategoryID)
	if err != nil {
		t.Fatalf("GetCategory() error = %v", err)
	}
	if cloudCat == nil {
		t.Error("Cloud category was not created")
	}
	if cloudCat != nil && cloudCat.Name != "Cloud Storage" {
		t.Errorf("Cloud category name = %v, want Cloud Storage", cloudCat.Name)
	}

	// Calling again should not error
	err = store.EnsureCloudCategory()
	if err != nil {
		t.Errorf("Second EnsureCloudCategory() error = %v", err)
	}
}

func TestDBStore_GetCategory(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	// Add category
	cat := Category{ID: "cat-1", Name: "Rock"}
	store.AddCategory(cat)

	// Get category
	retrieved, err := store.GetCategory("cat-1")
	if err != nil {
		t.Fatalf("GetCategory() error = %v", err)
	}
	if retrieved == nil {
		t.Fatal("GetCategory() returned nil")
	}
	if retrieved.Name != "Rock" {
		t.Errorf("Name = %v, want Rock", retrieved.Name)
	}

	// Get non-existent category (returns nil, not error)
	nonExistent, err := store.GetCategory("nonexistent")
	if err != nil {
		t.Errorf("GetCategory() for non-existent should not error, got: %v", err)
	}
	if nonExistent != nil {
		t.Error("Expected nil for non-existent category")
	}
}

func TestDBStore_GetTabsPaginated(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	// Add test tabs
	for i := 0; i < 25; i++ {
		tab := Tab{
			ID:     fmt.Sprintf("tab-%d", i),
			Title:  fmt.Sprintf("Song %d", i),
			Artist: "Test Artist",
		}
		store.AddTab(tab)
	}

	// Test pagination
	tabs, total, err := store.GetTabsPaginated("", 1, 10, "", []string{"title"}, false, "title", false)
	if err != nil {
		t.Fatalf("GetTabsPaginated() error = %v", err)
	}

	if len(tabs) != 10 {
		t.Errorf("Expected 10 tabs, got %d", len(tabs))
	}
	if total != 25 {
		t.Errorf("Total = %d, want 25", total)
	}

	// Test second page
	tabs, _, err = store.GetTabsPaginated("", 2, 10, "", []string{"title"}, false, "title", false)
	if err != nil {
		t.Fatalf("GetTabsPaginated() page 2 error = %v", err)
	}
	if len(tabs) != 10 {
		t.Errorf("Expected 10 tabs on page 2, got %d", len(tabs))
	}

	// Test search
	tabs, total, err = store.GetTabsPaginated("", 1, 10, "Song 1", []string{"title"}, false, "title", false)
	if err != nil {
		t.Fatalf("GetTabsPaginated() with search error = %v", err)
	}
	if total == 0 {
		t.Error("Search should return results")
	}
}

func TestDBStore_UpdateTab2(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	// Add tab
	tab := Tab{
		ID:     "tab-1",
		Title:  "Original",
		Artist: "Artist",
	}
	store.AddTab(tab)

	// Update tab
	tab.Title = "Updated"
	err := store.UpdateTab(tab)
	if err != nil {
		t.Fatalf("UpdateTab() error = %v", err)
	}

	// Verify update
	retrieved, _ := store.GetTab("tab-1")
	if retrieved.Title != "Updated" {
		t.Errorf("Title = %v, want Updated", retrieved.Title)
	}
}

func TestDBStore_SetTabCategories(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	// Add tab and categories
	tab := Tab{ID: "tab-1", Title: "Song"}
	store.AddTab(tab)

	cat1 := Category{ID: "cat-1", Name: "Rock"}
	cat2 := Category{ID: "cat-2", Name: "Jazz"}
	store.AddCategory(cat1)
	store.AddCategory(cat2)

	// Set categories
	err := store.SetTabCategories("tab-1", []string{"cat-1", "cat-2"}, time.Now().Unix())
	if err != nil {
		t.Fatalf("SetTabCategories() error = %v", err)
	}

	// Verify categories were set
	retrieved, _ := store.GetTab("tab-1")
	if len(retrieved.CategoryIDs) != 2 {
		t.Errorf("Expected 2 categories, got %d", len(retrieved.CategoryIDs))
	}
}

func TestDBStore_GetRecentTabs(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	// Add tabs with different last opened times
	for i := 0; i < 5; i++ {
		tab := Tab{
			ID:         fmt.Sprintf("tab-%d", i),
			Title:      fmt.Sprintf("Song %d", i),
			LastOpened: time.Now().Unix() - int64(i*100),
		}
		store.AddTab(tab)
	}

	// Get recent tabs
	tabs, err := store.GetRecentTabs(3)
	if err != nil {
		t.Fatalf("GetRecentTabs() error = %v", err)
	}

	if len(tabs) != 3 {
		t.Errorf("Expected 3 tabs, got %d", len(tabs))
	}

	// Verify they are sorted by last opened (most recent first)
	if len(tabs) >= 2 && tabs[0].LastOpened < tabs[1].LastOpened {
		t.Error("Tabs are not sorted by last opened time")
	}
}

func TestDBStore_GetTabsNeedingOriginCountry(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	// Add tab with cover but no origin country
	tab := Tab{
		ID:        "tab-1",
		Title:     "Song",
		Artist:    "Artist",
		CoverPath: "cover.jpg",
	}
	store.AddTab(tab)

	// Get tabs needing origin country
	tabs, err := store.GetTabsNeedingOriginCountry()
	if err != nil {
		t.Fatalf("GetTabsNeedingOriginCountry() error = %v", err)
	}

	if len(tabs) != 1 {
		t.Errorf("Expected 1 tab, got %d", len(tabs))
	}
}

func TestDBStore_GetTabsNeedingInitials(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	// Add tab without initials
	tab := Tab{
		ID:    "tab-1",
		Title: "Song",
	}
	store.AddTab(tab)

	// Get tabs needing initials
	tabs, err := store.GetTabsNeedingInitials()
	if err != nil {
		t.Fatalf("GetTabsNeedingInitials() error = %v", err)
	}

	if len(tabs) != 1 {
		t.Errorf("Expected 1 tab, got %d", len(tabs))
	}
}

func TestDBStore_SearchTabs_MultipleResults(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	// Add test tabs
	tabs := []Tab{
		{ID: "tab-1", Title: "Stairway to Heaven", Artist: "Led Zeppelin"},
		{ID: "tab-2", Title: "Hotel California", Artist: "Eagles"},
		{ID: "tab-3", Title: "Bohemian Rhapsody", Artist: "Queen"},
	}

	for _, tab := range tabs {
		store.AddTab(tab)
	}

	// Search for "Heaven"
	results, err := store.SearchTabs("Heaven")
	if err != nil {
		t.Fatalf("SearchTabs() error = %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected search results")
	}
}

func TestDBStore_GetTabsPaginated_Sorting(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	// Add tabs with different timestamps
	now := time.Now().Unix()
	tabs := []Tab{
		{ID: "tab-1", Title: "A Song", AddedAt: now - 100},
		{ID: "tab-2", Title: "B Song", AddedAt: now - 50},
		{ID: "tab-3", Title: "C Song", AddedAt: now},
	}

	for _, tab := range tabs {
		store.AddTab(tab)
	}

	// Sort by added_at descending
	results, _, err := store.GetTabsPaginated("", 1, 10, "", []string{"title"}, false, "added_at", true)
	if err != nil {
		t.Fatalf("GetTabsPaginated() error = %v", err)
	}

	if len(results) < 3 {
		t.Fatalf("Expected at least 3 results, got %d", len(results))
	}
}

func TestDBStore_HasData_AfterAdding(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	// Add a tab
	tab := Tab{ID: "tab-1", Title: "Song"}
	store.AddTab(tab)

	// Now should have data
	hasData := store.HasData()
	if !hasData {
		t.Error("Expected to have data after adding tab")
	}
}

func TestDBStore_DeleteCategory_WithSubcategories(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	// Add parent and child categories
	parent := Category{ID: "parent", Name: "Parent"}
	child := Category{ID: "child", Name: "Child", ParentID: "parent"}
	store.AddCategory(parent)
	store.AddCategory(child)

	// Delete parent
	err := store.DeleteCategory("parent")
	if err != nil {
		t.Fatalf("DeleteCategory() error = %v", err)
	}

	// Child should now have empty parent
	childCat, _ := store.GetCategory("child")
	if childCat != nil && childCat.ParentID != "" {
		t.Errorf("Child category should have empty parent, got %v", childCat.ParentID)
	}
}

func TestDBStore_GetTabsPaginatedLike(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	// Add test tabs with various fields
	tabs := []Tab{
		{ID: "tab-1", Title: "Stairway to Heaven", Artist: "Led Zeppelin", Album: "Led Zeppelin IV", Tag: "rock"},
		{ID: "tab-2", Title: "Hotel California", Artist: "Eagles", Album: "Hotel California", Tag: "rock"},
		{ID: "tab-3", Title: "Bohemian Rhapsody", Artist: "Queen", Album: "A Night at the Opera", Tag: "rock"},
		{ID: "tab-4", Title: "Imagine", Artist: "John Lennon", Album: "Imagine", Tag: "pop"},
		{ID: "tab-5", Title: "Yesterday", Artist: "The Beatles", Album: "Help!", Tag: "pop"},
	}

	for _, tab := range tabs {
		store.AddTab(tab)
	}

	// Add categories
	cat1 := Category{ID: "cat-1", Name: "Rock"}
	cat2 := Category{ID: "cat-2", Name: "Pop"}
	store.AddCategory(cat1)
	store.AddCategory(cat2)

	// Assign tabs to categories
	store.SetTabCategories("tab-1", []string{"cat-1"}, time.Now().Unix())
	store.SetTabCategories("tab-2", []string{"cat-1"}, time.Now().Unix())
	store.SetTabCategories("tab-3", []string{"cat-1"}, time.Now().Unix())
	store.SetTabCategories("tab-4", []string{"cat-2"}, time.Now().Unix())
	store.SetTabCategories("tab-5", []string{"cat-2"}, time.Now().Unix())

	tests := []struct {
		name         string
		categoryId   string
		page         int
		pageSize     int
		searchQuery  string
		filterBy     []string
		isGlobal     bool
		sortBy       string
		sortDesc     bool
		wantCount    int
		wantTotal    int
		checkResults func(t *testing.T, tabs []Tab)
	}{
		{
			name:        "Search by title - global",
			categoryId:  "",
			page:        1,
			pageSize:    10,
			searchQuery: "Heaven",
			filterBy:    []string{"title"},
			isGlobal:    true,
			sortBy:      "title",
			sortDesc:    false,
			wantCount:   1, // Only "Stairway to Heaven" contains "Heaven"
			wantTotal:   1,
		},
		{
			name:        "Search by artist - global",
			categoryId:  "",
			page:        1,
			pageSize:    10,
			searchQuery: "Queen",
			filterBy:    []string{"artist"},
			isGlobal:    true,
			sortBy:      "title",
			sortDesc:    false,
			wantCount:   1,
			wantTotal:   1,
		},
		{
			name:        "Search by album - global",
			categoryId:  "",
			page:        1,
			pageSize:    10,
			searchQuery: "Imagine",
			filterBy:    []string{"album"},
			isGlobal:    true,
			sortBy:      "title",
			sortDesc:    false,
			wantCount:   1,
			wantTotal:   1,
		},
		{
			name:        "Search by tag - global",
			categoryId:  "",
			page:        1,
			pageSize:    10,
			searchQuery: "rock",
			filterBy:    []string{"tag"},
			isGlobal:    true,
			sortBy:      "title",
			sortDesc:    false,
			wantCount:   3,
			wantTotal:   3,
		},
		{
			name:        "Search multiple fields - global",
			categoryId:  "",
			page:        1,
			pageSize:    10,
			searchQuery: "Led",
			filterBy:    []string{"title", "artist", "album"},
			isGlobal:    true,
			sortBy:      "title",
			sortDesc:    false,
			wantCount:   1, // "Led Zeppelin" in artist and album
			wantTotal:   1,
		},
		{
			name:        "Search in specific category",
			categoryId:  "cat-1",
			page:        1,
			pageSize:    10,
			searchQuery: "Heaven",
			filterBy:    []string{"title"},
			isGlobal:    false,
			sortBy:      "title",
			sortDesc:    false,
			wantCount:   1, // Only "Stairway to Heaven" in cat-1
			wantTotal:   1,
		},
		{
			name:        "Pagination - page 1",
			categoryId:  "",
			page:        1,
			pageSize:    2,
			searchQuery: "",
			filterBy:    []string{"title"},
			isGlobal:    true,
			sortBy:      "title",
			sortDesc:    false,
			wantCount:   2,
			wantTotal:   5,
		},
		{
			name:        "Pagination - page 2",
			categoryId:  "",
			page:        2,
			pageSize:    2,
			searchQuery: "",
			filterBy:    []string{"title"},
			isGlobal:    true,
			sortBy:      "title",
			sortDesc:    false,
			wantCount:   2,
			wantTotal:   5,
		},
		{
			name:        "Sort by title descending",
			categoryId:  "",
			page:        1,
			pageSize:    10,
			searchQuery: "",
			filterBy:    []string{"title"},
			isGlobal:    true,
			sortBy:      "title",
			sortDesc:    true,
			wantCount:   5,
			wantTotal:   5,
			checkResults: func(t *testing.T, tabs []Tab) {
				if len(tabs) >= 2 && tabs[0].Title < tabs[1].Title {
					t.Error("Tabs should be sorted by title descending")
				}
			},
		},
		{
			name:        "Empty search query",
			categoryId:  "",
			page:        1,
			pageSize:    10,
			searchQuery: "",
			filterBy:    []string{"title"},
			isGlobal:    true,
			sortBy:      "title",
			sortDesc:    false,
			wantCount:   5,
			wantTotal:   5,
		},
		{
			name:        "No results",
			categoryId:  "",
			page:        1,
			pageSize:    10,
			searchQuery: "NonExistentSong",
			filterBy:    []string{"title"},
			isGlobal:    true,
			sortBy:      "title",
			sortDesc:    false,
			wantCount:   0,
			wantTotal:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, total, err := store.getTabsPaginatedLike(
				tt.categoryId,
				tt.page,
				tt.pageSize,
				tt.searchQuery,
				tt.filterBy,
				tt.isGlobal,
				tt.sortBy,
				tt.sortDesc,
			)

			if err != nil {
				t.Fatalf("getTabsPaginatedLike() error = %v", err)
			}

			if len(results) != tt.wantCount {
				t.Errorf("getTabsPaginatedLike() returned %d results, want %d", len(results), tt.wantCount)
			}

			if total != tt.wantTotal {
				t.Errorf("getTabsPaginatedLike() total = %d, want %d", total, tt.wantTotal)
			}

			if tt.checkResults != nil {
				tt.checkResults(t, results)
			}
		})
	}
}

func TestDBStore_GetTabsPaginatedLike_EdgeCases(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	// Test with empty database
	results, total, err := store.getTabsPaginatedLike("", 1, 10, "test", []string{"title"}, true, "title", false)
	if err != nil {
		t.Fatalf("getTabsPaginatedLike() on empty DB error = %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results on empty DB, got %d", len(results))
	}
	if total != 0 {
		t.Errorf("Expected total 0 on empty DB, got %d", total)
	}

	// Add a tab
	tab := Tab{ID: "tab-1", Title: "Test Song", Artist: "Test Artist"}
	store.AddTab(tab)

	// Test with invalid filter field (should be ignored)
	results, total, err = store.getTabsPaginatedLike("", 1, 10, "Test", []string{"invalid_field"}, true, "title", false)
	if err != nil {
		t.Fatalf("getTabsPaginatedLike() with invalid filter error = %v", err)
	}
	// Should return all tabs since no valid filter was applied
	if total != 1 {
		t.Errorf("Expected total 1, got %d", total)
	}

	// Test with page beyond available data
	results, total, err = store.getTabsPaginatedLike("", 10, 10, "", []string{"title"}, true, "title", false)
	if err != nil {
		t.Fatalf("getTabsPaginatedLike() with page beyond data error = %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results for page beyond data, got %d", len(results))
	}
	if total != 1 {
		t.Errorf("Expected total 1, got %d", total)
	}
}

