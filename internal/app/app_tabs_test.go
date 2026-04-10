package app

import (
	"haya-tab/pkg/store"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestApp_GetTabs(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app)
	_ = tmpDir // Ignore unused variable error

	// Initially empty
	tabs := app.GetTabs()
	if len(tabs) != 0 {
		t.Errorf("Expected 0 tabs, got %d", len(tabs))
	}

	// Add tabs
	app.store.AddTab(store.Tab{ID: "tab1", Title: "Song 1"})
	app.store.AddTab(store.Tab{ID: "tab2", Title: "Song 2"})

	tabs = app.GetTabs()
	if len(tabs) != 2 {
		t.Errorf("Expected 2 tabs, got %d", len(tabs))
	}
}

func TestApp_GetTabsPaginated(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app)
	_ = tmpDir // Ignore unused variable error

	// Add 15 tabs
	for i := 1; i <= 15; i++ {
		app.store.AddTab(store.Tab{
			ID:    generateID(),
			Title: "Song " + string(rune('A'+i-1)),
		})
	}

	t.Run("first page", func(t *testing.T) {
		resp := app.GetTabsPaginated("", 1, 10, "", nil, false, "", false)
		if resp.Total != 15 {
			t.Errorf("Total = %d, want 15", resp.Total)
		}
		if len(resp.Tabs) != 10 {
			t.Errorf("Got %d tabs, want 10", len(resp.Tabs))
		}
		if !resp.HasMore {
			t.Error("HasMore should be true")
		}
	})

	t.Run("second page", func(t *testing.T) {
		resp := app.GetTabsPaginated("", 2, 10, "", nil, false, "", false)
		if len(resp.Tabs) != 5 {
			t.Errorf("Got %d tabs, want 5", len(resp.Tabs))
		}
		if resp.HasMore {
			t.Error("HasMore should be false on last page")
		}
	})

	t.Run("search filter", func(t *testing.T) {
		resp := app.GetTabsPaginated("", 1, 10, "Song B", nil, false, "", false)
		if resp.Total == 0 {
			t.Error("Expected matching results for 'Song B'")
		}
	})

	t.Run("invalid page defaults", func(t *testing.T) {
		resp := app.GetTabsPaginated("", 0, 0, "", nil, false, "", false)
		if resp.Page != 1 {
			t.Errorf("Page = %d, want 1", resp.Page)
		}
		if resp.PageSize < 1 || resp.PageSize > 200 {
			t.Errorf("PageSize = %d, want between 1 and 200", resp.PageSize)
		}
	})
}

func TestApp_GetRecentTabs(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app)
	_ = tmpDir // Ignore unused variable error

	// Add tabs with different last opened times
	now := time.Now().Unix()
	app.store.AddTab(store.Tab{ID: "tab1", Title: "Old", LastOpened: now - 100})
	app.store.AddTab(store.Tab{ID: "tab2", Title: "Recent", LastOpened: now})
	app.store.AddTab(store.Tab{ID: "tab3", Title: "Middle", LastOpened: now - 50})

	tabs := app.GetRecentTabs(2)
	if len(tabs) != 2 {
		t.Fatalf("Expected 2 tabs, got %d", len(tabs))
	}
	// Should be ordered by LastOpened descending
	if tabs[0].ID != "tab2" {
		t.Errorf("First tab should be 'tab2' (most recent), got %q", tabs[0].ID)
	}
}

func TestApp_DeleteTab(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app)
	_ = tmpDir // Ignore unused variable error

	t.Run("delete non-existent tab", func(t *testing.T) {
		err := app.DeleteTab("nonexistent")
		if err == nil {
			t.Error("Expected error for non-existent tab")
		}
	})

	t.Run("delete unmanaged tab", func(t *testing.T) {
		tab := store.Tab{ID: "tab1", Title: "Test", IsManaged: false}
		app.store.AddTab(tab)

		err := app.DeleteTab("tab1")
		if err != nil {
			t.Fatalf("DeleteTab() error = %v", err)
		}

		retrieved, _ := app.store.GetTab("tab1")
		if retrieved != nil {
			t.Error("Tab should be deleted")
		}
	})

	t.Run("delete managed tab with file", func(t *testing.T) {
		// Create a managed file
		storageDir := app.GetStorageDir()
		filePath := filepath.Join(storageDir, "tab_managed.gp5")
		os.WriteFile(filePath, []byte("data"), 0644)

		tab := store.Tab{
			ID:        "tab_managed",
			Title:     "Managed Tab",
			FilePath:  "tab_managed.gp5",
			IsManaged: true,
		}
		app.store.AddTab(tab)

		err := app.DeleteTab("tab_managed")
		if err != nil {
			t.Fatalf("DeleteTab() error = %v", err)
		}

		// File should be deleted
		if _, err := os.Stat(filePath); !os.IsNotExist(err) {
			t.Error("Managed file should be deleted")
		}
	})
}

func TestApp_BatchDeleteTabs(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app)
	_ = tmpDir // Ignore unused variable error

	// Add tabs
	app.store.AddTab(store.Tab{ID: "tab1", Title: "Tab 1"})
	app.store.AddTab(store.Tab{ID: "tab2", Title: "Tab 2"})
	app.store.AddTab(store.Tab{ID: "tab3", Title: "Tab 3"})

	deleted, err := app.BatchDeleteTabs([]string{"tab1", "tab2", "nonexistent"})
	if err != nil {
		t.Fatalf("BatchDeleteTabs() error = %v", err)
	}
	if deleted != 2 {
		t.Errorf("Deleted = %d, want 2", deleted)
	}

	// tab3 should still exist
	tab3, _ := app.store.GetTab("tab3")
	if tab3 == nil {
		t.Error("tab3 should still exist")
	}
}

func TestApp_MoveTab(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app)
	_ = tmpDir // Ignore unused variable error

	app.store.AddTab(store.Tab{ID: "tab1", Title: "Tab 1"})
	app.store.AddCategory(store.Category{ID: "cat1", Name: "Category 1"})

	err := app.MoveTab("tab1", "cat1")
	if err != nil {
		t.Fatalf("MoveTab() error = %v", err)
	}

	tab, _ := app.store.GetTab("tab1")
	if len(tab.CategoryIDs) != 1 || tab.CategoryIDs[0] != "cat1" {
		t.Errorf("Tab categories = %v, want [cat1]", tab.CategoryIDs)
	}
}

func TestApp_BatchMoveTabs(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app)
	_ = tmpDir // Ignore unused variable error

	app.store.AddTab(store.Tab{ID: "tab1", Title: "Tab 1"})
	app.store.AddTab(store.Tab{ID: "tab2", Title: "Tab 2"})
	app.store.AddCategory(store.Category{ID: "cat1", Name: "Category 1"})

	moved, err := app.BatchMoveTabs([]string{"tab1", "tab2"}, "cat1")
	if err != nil {
		t.Fatalf("BatchMoveTabs() error = %v", err)
	}
	if moved != 2 {
		t.Errorf("Moved = %d, want 2", moved)
	}

	// Verify both tabs are in the category
	tab1, _ := app.store.GetTab("tab1")
	tab2, _ := app.store.GetTab("tab2")
	if len(tab1.CategoryIDs) != 1 || tab1.CategoryIDs[0] != "cat1" {
		t.Errorf("Tab1 categories = %v, want [cat1]", tab1.CategoryIDs)
	}
	if len(tab2.CategoryIDs) != 1 || tab2.CategoryIDs[0] != "cat1" {
		t.Errorf("Tab2 categories = %v, want [cat1]", tab2.CategoryIDs)
	}
}

func TestApp_AddTabToCategory(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app)
	_ = tmpDir // Ignore unused variable error

	app.store.AddTab(store.Tab{ID: "tab1", Title: "Tab 1"})
	app.store.AddCategory(store.Category{ID: "cat1", Name: "Category 1"})
	app.store.AddCategory(store.Category{ID: "cat2", Name: "Category 2"})
	// Set initial category
	app.store.SetTabCategories("tab1", []string{"cat1"}, time.Now().Unix())

	t.Run("add to new category", func(t *testing.T) {
		err := app.AddTabToCategory("tab1", "cat2")
		if err != nil {
			t.Fatalf("AddTabToCategory() error = %v", err)
		}

		tab, _ := app.store.GetTab("tab1")
		if len(tab.CategoryIDs) != 2 {
			t.Errorf("Tab should have 2 categories, got %d", len(tab.CategoryIDs))
		}
	})

	t.Run("add to existing category is idempotent", func(t *testing.T) {
		err := app.AddTabToCategory("tab1", "cat1")
		if err != nil {
			t.Fatalf("AddTabToCategory() error = %v", err)
		}

		tab, _ := app.store.GetTab("tab1")
		// Should still have only 2 categories
		if len(tab.CategoryIDs) != 2 {
			t.Errorf("Tab should have 2 categories, got %d", len(tab.CategoryIDs))
		}
	})

	t.Run("add to non-existent tab returns error", func(t *testing.T) {
		err := app.AddTabToCategory("nonexistent", "cat1")
		if err == nil {
			t.Error("Expected error for non-existent tab")
		}
	})
}

func TestApp_RemoveTabFromCategory(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app)
	_ = tmpDir // Ignore unused variable error

	app.store.AddTab(store.Tab{ID: "tab1", Title: "Tab 1"})
	app.store.AddCategory(store.Category{ID: "cat1", Name: "Category 1"})
	app.store.AddCategory(store.Category{ID: "cat2", Name: "Category 2"})
	// Set initial categories
	app.store.SetTabCategories("tab1", []string{"cat1", "cat2"}, time.Now().Unix())

	t.Run("remove from category", func(t *testing.T) {
		err := app.RemoveTabFromCategory("tab1", "cat1")
		if err != nil {
			t.Fatalf("RemoveTabFromCategory() error = %v", err)
		}

		tab, _ := app.store.GetTab("tab1")
		if len(tab.CategoryIDs) != 1 {
			t.Errorf("Tab should have 1 category, got %d", len(tab.CategoryIDs))
		}
		if tab.CategoryIDs[0] != "cat2" {
			t.Errorf("Tab should be in cat2, got %v", tab.CategoryIDs)
		}
	})

	t.Run("remove from non-existing category is no-op", func(t *testing.T) {
		err := app.RemoveTabFromCategory("tab1", "nonexistent")
		if err != nil {
			t.Fatalf("RemoveTabFromCategory() error = %v", err)
		}
	})

	t.Run("non-existent tab returns error", func(t *testing.T) {
		err := app.RemoveTabFromCategory("nonexistent", "cat1")
		if err == nil {
			t.Error("Expected error for non-existent tab")
		}
	})
}

func TestApp_UpdateTabCategories(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app)
	_ = tmpDir // Ignore unused variable error

	app.store.AddTab(store.Tab{ID: "tab1", Title: "Tab 1"})
	app.store.AddCategory(store.Category{ID: "cat1", Name: "Category 1"})
	app.store.AddCategory(store.Category{ID: "cat2", Name: "Category 2"})

	err := app.UpdateTabCategories("tab1", []string{"cat1", "cat2"})
	if err != nil {
		t.Fatalf("UpdateTabCategories() error = %v", err)
	}

	tab, _ := app.store.GetTab("tab1")
	if len(tab.CategoryIDs) != 2 {
		t.Errorf("Tab should have 2 categories, got %d", len(tab.CategoryIDs))
	}
}

func TestApp_BatchAddTabsToCategory(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app)
	_ = tmpDir // Ignore unused variable error

	app.store.AddTab(store.Tab{ID: "tab1", Title: "Tab 1"})
	app.store.AddTab(store.Tab{ID: "tab2", Title: "Tab 2", CategoryIDs: []string{"cat1"}})
	app.store.AddCategory(store.Category{ID: "cat1", Name: "Category 1"})

	added, err := app.BatchAddTabsToCategory([]string{"tab1", "tab2"}, "cat1")
	if err != nil {
		t.Fatalf("BatchAddTabsToCategory() error = %v", err)
	}
	// tab1 should be added, tab2 already has cat1 so not added
	if added != 1 {
		t.Errorf("Added = %d, want 1", added)
	}
}

func TestApp_ExportTab(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app)
	_ = tmpDir // Ignore unused variable error

	// Create a managed file
	storageDir := app.GetStorageDir()
	srcPath := filepath.Join(storageDir, "export_test.gp5")
	os.WriteFile(srcPath, []byte("test data"), 0644)

	app.store.AddTab(store.Tab{
		ID:        "export_tab",
		Title:     "Export Test",
		FilePath:  "export_test.gp5",
		IsManaged: true,
	})

	exportDir := filepath.Join(tmpDir, "export")
	os.MkdirAll(exportDir, 0755)

	t.Run("successful export", func(t *testing.T) {
		err := app.ExportTab("export_tab", exportDir)
		if err != nil {
			t.Fatalf("ExportTab() error = %v", err)
		}

		exportedPath := filepath.Join(exportDir, "export_test.gp5")
		data, err := os.ReadFile(exportedPath)
		if err != nil {
			t.Fatalf("Failed to read exported file: %v", err)
		}
		if string(data) != "test data" {
			t.Errorf("Exported content = %q, want %q", string(data), "test data")
		}
	})

	t.Run("export non-existent tab", func(t *testing.T) {
		err := app.ExportTab("nonexistent", exportDir)
		if err == nil {
			t.Error("Expected error for non-existent tab")
		}
	})
}

func TestApp_MarkAsOpened(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app)
	_ = tmpDir // Ignore unused variable error

	initialTime := time.Now().Add(-time.Hour).Unix()
	app.store.AddTab(store.Tab{ID: "tab1", Title: "Tab 1", LastOpened: initialTime})

	err := app.MarkAsOpened("tab1")
	if err != nil {
		t.Fatalf("MarkAsOpened() error = %v", err)
	}

	tab, _ := app.store.GetTab("tab1")
	if tab.LastOpened <= initialTime {
		t.Error("LastOpened should be updated to a more recent time")
	}
}

func TestApp_RecalculateAllInitials(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app)
	_ = tmpDir // Ignore unused variable error

	app.store.AddTab(store.Tab{ID: "tab1", Title: "Stairway to Heaven"})
	app.store.AddTab(store.Tab{ID: "tab2", Title: "Bohemian Rhapsody"})

	updated, err := app.RecalculateAllInitials()
	if err != nil {
		t.Fatalf("RecalculateAllInitials() error = %v", err)
	}
	if updated != 2 {
		t.Errorf("Updated = %d, want 2", updated)
	}

	tab1, _ := app.store.GetTab("tab1")
	if tab1.InitialAZ == "" {
		t.Error("InitialAZ should be set")
	}
}

func TestApp_SaveTab(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app)
	_ = tmpDir // Ignore unused variable error

	t.Run("save tab with invalid file", func(t *testing.T) {
		tab := store.Tab{Title: "Invalid", FilePath: "/nonexistent/file.gp5"}
		_, err := app.SaveTab(tab, true)
		if err == nil {
			t.Error("Expected error for non-existent file")
		}
	})

	t.Run("save tab with valid file", func(t *testing.T) {
		// Create a source file
		srcPath := filepath.Join(tmpDir, "source.gp5")
		os.WriteFile(srcPath, []byte("data"), 0644)

		tab := store.Tab{ID: "test-tab-1", Title: "Valid", FilePath: srcPath}
		savedTab, err := app.SaveTab(tab, true)
		if err != nil {
			t.Fatalf("SaveTab() error = %v", err)
		}

		if savedTab.ID == "" {
			t.Error("Saved tab should have an ID")
		}
		if !savedTab.IsManaged {
			t.Error("Saved tab should be managed")
		}
		
		// Verify file was copied to storage
		storagePath := filepath.Join(app.GetStorageDir(), savedTab.FilePath)
		if _, err := os.Stat(storagePath); os.IsNotExist(err) {
			t.Error("File should be copied to storage")
		}
	})
}

func TestApp_UpdateTab(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app)
	_ = tmpDir // Ignore unused variable error

	// Add initial tab
	initialTab := store.Tab{ID: "tab1", Title: "Initial", Artist: "Artist"}
	app.store.AddTab(initialTab)

	t.Run("update non-existent tab", func(t *testing.T) {
		err := app.UpdateTab(store.Tab{ID: "nonexistent", Title: "New"})
		if err == nil {
			t.Error("Expected error for non-existent tab")
		}
	})

	t.Run("update existing tab", func(t *testing.T) {
		err := app.UpdateTab(store.Tab{ID: "tab1", Title: "Updated", Artist: "New Artist"})
		if err != nil {
			t.Fatalf("UpdateTab() error = %v", err)
		}

		updated, _ := app.store.GetTab("tab1")
		if updated.Title != "Updated" || updated.Artist != "New Artist" {
			t.Errorf("Updated tab = %v, want Updated/New Artist", updated)
		}
	})
}

func TestApp_UpdateTabMetadata(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app)
	_ = tmpDir // Ignore unused variable error

	app.store.AddTab(store.Tab{ID: "tab1", Title: "Old Title"})

	err := app.UpdateTabMetadata("tab1", "New Title", "New Artist", "New Album")
	if err != nil {
		t.Fatalf("UpdateTabMetadata() error = %v", err)
	}

	updated, _ := app.store.GetTab("tab1")
	if updated.Title != "New Title" || updated.Artist != "New Artist" {
		t.Error("Metadata not updated correctly")
	}
}

func TestApp_RecalculateAllInitials_Empty(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app)
	_ = tmpDir // Ignore unused variable error

	count, err := app.RecalculateAllInitials()
	if err != nil {
		t.Fatalf("RecalculateAllInitials() error = %v", err)
	}
	if count != 0 {
		t.Errorf("Updated = %d, want 0", count)
	}
}
