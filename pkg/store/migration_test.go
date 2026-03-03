package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestMigrateFromJSON_NoFile(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	// Test with non-existent file
	err := MigrateFromJSON(store, filepath.Join(tmpDir, "nonexistent.json"))
	if err != nil {
		t.Errorf("MigrateFromJSON() with non-existent file should not error, got: %v", err)
	}
}

func TestMigrateFromJSON_FullData(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	// Create test JSON file with full data
	jsonPath := filepath.Join(tmpDir, "test-data.json")
	testData := struct {
		Tabs       []Tab      `json:"tabs"`
		Categories []Category `json:"categories"`
		Settings   Settings   `json:"settings"`
	}{
		Tabs: []Tab{
			{ID: "tab-1", Title: "Song 1", Artist: "Artist 1"},
			{ID: "tab-2", Title: "Song 2", Artist: "Artist 2"},
		},
		Categories: []Category{
			{ID: "cat-1", Name: "Rock"},
			{ID: "cat-2", Name: "Jazz"},
		},
		Settings: Settings{
			Theme:      "dark",
			Language:   "zh-CN",
			OpenMethod: "default",
		},
	}

	data, err := json.Marshal(testData)
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}

	if err := os.WriteFile(jsonPath, data, 0644); err != nil {
		t.Fatalf("Failed to write test JSON: %v", err)
	}

	// Migrate data
	err = MigrateFromJSON(store, jsonPath)
	if err != nil {
		t.Fatalf("MigrateFromJSON() error = %v", err)
	}

	// Verify tabs were migrated
	tabs, err := store.GetAllTabs()
	if err != nil {
		t.Fatalf("GetAllTabs() error = %v", err)
	}
	if len(tabs) != 2 {
		t.Errorf("Expected 2 tabs, got %d", len(tabs))
	}

	// Verify categories were migrated
	categories, err := store.GetCategories()
	if err != nil {
		t.Fatalf("GetCategories() error = %v", err)
	}
	if len(categories) != 2 {
		t.Errorf("Expected 2 categories, got %d", len(categories))
	}

	// Verify settings were migrated
	settings := store.GetSettings()
	if settings.Theme != "dark" {
		t.Errorf("Theme = %v, want dark", settings.Theme)
	}
	if settings.Language != "zh-CN" {
		t.Errorf("Language = %v, want zh-CN", settings.Language)
	}

	// Verify backup file was created
	backupPath := jsonPath + ".bak"
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Error("Backup file was not created")
	}
}

func TestMigrateFromJSON_TabsOnlyArray(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	// Create test JSON file with old format (array of tabs)
	jsonPath := filepath.Join(tmpDir, "test-tabs.json")
	testTabs := []Tab{
		{ID: "tab-1", Title: "Song 1", Artist: "Artist 1"},
		{ID: "tab-2", Title: "Song 2", Artist: "Artist 2"},
	}

	data, err := json.Marshal(testTabs)
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}

	if err := os.WriteFile(jsonPath, data, 0644); err != nil {
		t.Fatalf("Failed to write test JSON: %v", err)
	}

	// Migrate data
	err = MigrateFromJSON(store, jsonPath)
	if err != nil {
		t.Fatalf("MigrateFromJSON() error = %v", err)
	}

	// Verify tabs were migrated
	tabs, err := store.GetAllTabs()
	if err != nil {
		t.Fatalf("GetAllTabs() error = %v", err)
	}
	if len(tabs) != 2 {
		t.Errorf("Expected 2 tabs, got %d", len(tabs))
	}

	// Verify backup file was created
	backupPath := jsonPath + ".bak"
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Error("Backup file was not created")
	}
}

func TestMigrateFromJSON_InvalidJSON(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	// Create invalid JSON file
	jsonPath := filepath.Join(tmpDir, "invalid.json")
	if err := os.WriteFile(jsonPath, []byte("invalid json {{{"), 0644); err != nil {
		t.Fatalf("Failed to write invalid JSON: %v", err)
	}

	// Migration should fail
	err := MigrateFromJSON(store, jsonPath)
	if err == nil {
		t.Error("MigrateFromJSON() should error with invalid JSON")
	}
}

func TestMigrateFromJSON_EmptySettings(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	// Create test JSON with empty settings
	jsonPath := filepath.Join(tmpDir, "test-empty-settings.json")
	testData := struct {
		Tabs       []Tab      `json:"tabs"`
		Categories []Category `json:"categories"`
		Settings   Settings   `json:"settings"`
	}{
		Tabs: []Tab{
			{ID: "tab-1", Title: "Song 1"},
		},
		Settings: Settings{}, // Empty settings
	}

	data, err := json.Marshal(testData)
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}

	if err := os.WriteFile(jsonPath, data, 0644); err != nil {
		t.Fatalf("Failed to write test JSON: %v", err)
	}

	// Migrate data
	err = MigrateFromJSON(store, jsonPath)
	if err != nil {
		t.Fatalf("MigrateFromJSON() error = %v", err)
	}

	// Verify tabs were migrated
	tabs, err := store.GetAllTabs()
	if err != nil {
		t.Fatalf("GetAllTabs() error = %v", err)
	}
	if len(tabs) != 1 {
		t.Errorf("Expected 1 tab, got %d", len(tabs))
	}

	// Settings should remain default since they were empty
	settings := store.GetSettings()
	if settings.Theme != "system" {
		t.Errorf("Theme should remain default 'system', got %v", settings.Theme)
	}
}
