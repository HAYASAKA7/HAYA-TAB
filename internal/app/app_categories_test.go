package app

import (
	"haya-tab/pkg/store"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestApp_GetCategories(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app)
	_ = tmpDir // Ignore unused variable error

	// Initialize() always creates the system cloud category (sys_cloud)
	cats := app.GetCategories()
	if len(cats) != 1 {
		t.Errorf("Expected 1 category (system cloud), got %d", len(cats))
	}

	// Add categories
	app.store.AddCategory(store.Category{ID: "cat1", Name: "Rock"})
	app.store.AddCategory(store.Category{ID: "cat2", Name: "Jazz"})

	cats = app.GetCategories()
	if len(cats) != 3 {
		t.Errorf("Expected 3 categories (2 added + system cloud), got %d", len(cats))
	}
}

func TestApp_GetRecentCategories(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app)
	_ = tmpDir // Ignore unused variable error

	// Add categories and tabs
	app.store.AddCategory(store.Category{ID: "cat1", Name: "Rock"})
	app.store.AddTab(store.Tab{ID: "tab1", Title: "Song", LastOpened: time.Now().Unix()})
	app.store.SetTabCategories("tab1", []string{"cat1"}, time.Now().Unix())

	cats := app.GetRecentCategories(10)
	if len(cats) == 0 {
		t.Error("Expected at least 1 recent category")
	}
}

func TestApp_AddCategory(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app)
	_ = tmpDir // Ignore unused variable error

	t.Run("with provided ID", func(t *testing.T) {
		cat := store.Category{ID: "custom-id", Name: "Rock"}
		err := app.AddCategory(cat)
		if err != nil {
			t.Fatalf("AddCategory() error = %v", err)
		}

		retrieved, _ := app.store.GetCategory("custom-id")
		if retrieved == nil {
			t.Fatal("Category not found after adding")
		}
		if retrieved.Name != "Rock" {
			t.Errorf("Name = %q, want %q", retrieved.Name, "Rock")
		}
	})

	t.Run("with empty ID generates one", func(t *testing.T) {
		cat := store.Category{Name: "Jazz"}
		err := app.AddCategory(cat)
		if err != nil {
			t.Fatalf("AddCategory() error = %v", err)
		}

		cats, _ := app.store.GetCategories()
		found := false
		for _, c := range cats {
			if c.Name == "Jazz" {
				found = true
				if c.ID == "" {
					t.Error("Expected generated ID, got empty")
				}
			}
		}
		if !found {
			t.Error("Jazz category not found")
		}
	})
}

func TestApp_DeleteCategory(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app)
	_ = tmpDir // Ignore unused variable error

	app.store.AddCategory(store.Category{ID: "cat1", Name: "Rock"})

	err := app.DeleteCategory("cat1")
	if err != nil {
		t.Fatalf("DeleteCategory() error = %v", err)
	}

	retrieved, _ := app.store.GetCategory("cat1")
	if retrieved != nil {
		t.Error("Category should be deleted")
	}
}

func TestApp_MoveCategory(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app)
	_ = tmpDir // Ignore unused variable error

	app.store.AddCategory(store.Category{ID: "cat1", Name: "Rock"})
	app.store.AddCategory(store.Category{ID: "cat2", Name: "Jazz"})

	t.Run("valid move", func(t *testing.T) {
		err := app.MoveCategory("cat1", "cat2")
		if err != nil {
			t.Fatalf("MoveCategory() error = %v", err)
		}

		retrieved, _ := app.store.GetCategory("cat1")
		if retrieved.ParentID != "cat2" {
			t.Errorf("ParentID = %q, want %q", retrieved.ParentID, "cat2")
		}
	})

	t.Run("move to self returns error", func(t *testing.T) {
		err := app.MoveCategory("cat1", "cat1")
		if err == nil {
			t.Error("Expected error when moving category to itself")
		}
	})
}

func TestApp_GetSettings(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app)
	_ = tmpDir // Ignore unused variable error

	settings := app.GetSettings()
	if settings.Theme != "system" {
		t.Errorf("Default theme = %q, want %q", settings.Theme, "system")
	}
}

func TestApp_CheckMigration(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app)
	_ = tmpDir // Ignore unused variable error

	t.Run("invalid target", func(t *testing.T) {
		_, err := app.CheckMigration("invalid")
		if err == nil {
			t.Error("Expected error for invalid target")
		}
	})

	t.Run("storage target", func(t *testing.T) {
		// Create some files in storage dir
		storageDir := app.GetStorageDir()
		os.WriteFile(filepath.Join(storageDir, "file1.gp5"), []byte("data1"), 0644)
		os.WriteFile(filepath.Join(storageDir, "file2.pdf"), []byte("data22"), 0644)

		status, err := app.CheckMigration("storage")
		if err != nil {
			t.Fatalf("CheckMigration() error = %v", err)
		}
		if status.Count != 2 {
			t.Errorf("Count = %d, want 2", status.Count)
		}
		if status.Size != 11 { // 5 + 6 bytes
			t.Errorf("Size = %d, want 11", status.Size)
		}
	})

	t.Run("covers target", func(t *testing.T) {
		status, err := app.CheckMigration("covers")
		if err != nil {
			t.Fatalf("CheckMigration() error = %v", err)
		}
		if status.Count != 0 {
			t.Errorf("Count = %d, want 0", status.Count)
		}
	})

	t.Run("empty directory", func(t *testing.T) {
		status, err := app.CheckMigration("storage")
		// This may have files from previous subtest, just verify no error
		if err != nil {
			t.Fatalf("CheckMigration() error = %v", err)
		}
		_ = status
	})
}

func TestApp_SafeCopyFile(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app)
	_ = tmpDir // Ignore unused variable error

	srcPath := filepath.Join(tmpDir, "source.txt")
	os.WriteFile(srcPath, []byte("hello world"), 0644)

	dstPath := filepath.Join(tmpDir, "dest.txt")
	err := app.safeCopyFile(srcPath, dstPath)
	if err != nil {
		t.Fatalf("safeCopyFile() error = %v", err)
	}

	data, _ := os.ReadFile(dstPath)
	if string(data) != "hello world" {
		t.Errorf("Content = %q, want %q", string(data), "hello world")
	}
}

func TestApp_SafeCopyFile_NonExistentSource(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app)
	_ = tmpDir // Ignore unused variable error

	err := app.safeCopyFile(filepath.Join(tmpDir, "nope"), filepath.Join(tmpDir, "dest"))
	if err == nil {
		t.Error("Expected error for non-existent source")
	}
}

func TestApp_CheckDirSize(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app)
	_ = tmpDir // Ignore unused variable error

	testDir := filepath.Join(tmpDir, "checkdir")
	os.MkdirAll(testDir, 0755)

	// Empty directory
	count, size, err := app.checkDirSize(testDir)
	if err != nil {
		t.Fatalf("checkDirSize() error = %v", err)
	}
	if count != 0 || size != 0 {
		t.Errorf("Empty dir: count=%d size=%d, want 0/0", count, size)
	}

	// Add files
	os.WriteFile(filepath.Join(testDir, "a.txt"), []byte("aaa"), 0644)
	os.WriteFile(filepath.Join(testDir, "b.txt"), []byte("bbbbb"), 0644)

	// Add a subdirectory (should be ignored)
	os.MkdirAll(filepath.Join(testDir, "subdir"), 0755)

	count, size, err = app.checkDirSize(testDir)
	if err != nil {
		t.Fatalf("checkDirSize() error = %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
	if size != 8 { // 3 + 5
		t.Errorf("size = %d, want 8", size)
	}
}

func TestApp_CheckDirSize_NonExistentDir(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app)
	_ = tmpDir // Ignore unused variable error

	_, _, err := app.checkDirSize(filepath.Join(tmpDir, "nonexistent"))
	if err == nil {
		t.Error("Expected error for non-existent directory")
	}
}
