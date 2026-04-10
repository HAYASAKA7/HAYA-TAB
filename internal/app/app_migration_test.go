package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApp_MigrateData_InvalidTarget(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app)
	_ = tmpDir // Ignore unused variable error

	err := app.MigrateData("invalid", "/some/path", false)
	if err == nil {
		t.Error("Expected error for invalid target")
	}
}

func TestApp_MigrateData_SamePath(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app)
	_ = tmpDir // Ignore unused variable error

	storageDir := app.GetStorageDir()

	// Migrate to same path should just update settings
	err := app.MigrateData("storage", storageDir, false)
	if err != nil {
		t.Fatalf("MigrateData() error = %v", err)
	}

	settings := app.GetSettings()
	if settings.StoragePath != storageDir {
		t.Errorf("StoragePath = %q, want %q", settings.StoragePath, storageDir)
	}
}

func TestApp_MigrateData_StorageMove(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app)
	_ = tmpDir // Ignore unused variable error

	storageDir := app.GetStorageDir()

	// Create some files in storage
	os.WriteFile(filepath.Join(storageDir, "file1.gp5"), []byte("data1"), 0644)
	os.WriteFile(filepath.Join(storageDir, "file2.pdf"), []byte("data22"), 0644)

	newPath := filepath.Join(tmpDir, "new_storage")

	err := app.MigrateData("storage", newPath, false)
	if err != nil {
		t.Fatalf("MigrateData() error = %v", err)
	}

	// Check files exist in new location
	if _, err := os.Stat(filepath.Join(newPath, "file1.gp5")); err != nil {
		t.Error("file1.gp5 should exist in new location")
	}
	if _, err := os.Stat(filepath.Join(newPath, "file2.pdf")); err != nil {
		t.Error("file2.pdf should exist in new location")
	}

	// Check files removed from old location (since copyOnly=false)
	if _, err := os.Stat(filepath.Join(storageDir, "file1.gp5")); !os.IsNotExist(err) {
		t.Error("file1.gp5 should be removed from old location")
	}

	// Check settings updated
	settings := app.GetSettings()
	if settings.StoragePath != newPath {
		t.Errorf("StoragePath = %q, want %q", settings.StoragePath, newPath)
	}
}

func TestApp_MigrateData_StorageCopy(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app)
	_ = tmpDir // Ignore unused variable error

	storageDir := app.GetStorageDir()

	// Create some files in storage
	os.WriteFile(filepath.Join(storageDir, "file1.gp5"), []byte("data1"), 0644)

	newPath := filepath.Join(tmpDir, "copy_storage")

	err := app.MigrateData("storage", newPath, true) // copyOnly=true
	if err != nil {
		t.Fatalf("MigrateData() error = %v", err)
	}

	// Check files exist in new location
	if _, err := os.Stat(filepath.Join(newPath, "file1.gp5")); err != nil {
		t.Error("file1.gp5 should exist in new location")
	}

	// Check files still exist in old location (since copyOnly=true)
	if _, err := os.Stat(filepath.Join(storageDir, "file1.gp5")); err != nil {
		t.Error("file1.gp5 should still exist in old location (copyOnly)")
	}
}

func TestApp_MigrateData_CoversMigration(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app)
	_ = tmpDir // Ignore unused variable error

	coversDir := app.GetCoversDir()

	// Create some cover files
	os.WriteFile(filepath.Join(coversDir, "cover1.jpg"), []byte("jpeg data"), 0644)

	newPath := filepath.Join(tmpDir, "new_covers")

	err := app.MigrateData("covers", newPath, false)
	if err != nil {
		t.Fatalf("MigrateData() error = %v", err)
	}

	// Check files exist in new location
	if _, err := os.Stat(filepath.Join(newPath, "cover1.jpg")); err != nil {
		t.Error("cover1.jpg should exist in new location")
	}

	// Check settings updated
	settings := app.GetSettings()
	if settings.CoversPath != newPath {
		t.Errorf("CoversPath = %q, want %q", settings.CoversPath, newPath)
	}
}

func TestApp_MigrateData_EmptyDirectory(t *testing.T) {
	app, tmpDir := setupTestApp(t)
	defer cleanupTestApp(app)
	_ = tmpDir // Ignore unused variable error

	newPath := filepath.Join(tmpDir, "empty_migration")

	// Migrate empty storage should work
	err := app.MigrateData("storage", newPath, false)
	if err != nil {
		t.Fatalf("MigrateData() error = %v", err)
	}

	// Settings should still be updated
	settings := app.GetSettings()
	if settings.StoragePath != newPath {
		t.Errorf("StoragePath = %q, want %q", settings.StoragePath, newPath)
	}
}

func TestMigrationStatus(t *testing.T) {
	status := MigrationStatus{
		Count: 5,
		Size:  1024,
	}

	if status.Count != 5 {
		t.Errorf("Count = %d, want 5", status.Count)
	}
	if status.Size != 1024 {
		t.Errorf("Size = %d, want 1024", status.Size)
	}
}
