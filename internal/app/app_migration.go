package app

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// MigrationStatus holds information about a directory migration
type MigrationStatus struct {
	Count int   `json:"count"`
	Size  int64 `json:"size"`
}

// CheckMigration checks if a directory needs migration and returns the number of files and total size
func (a *App) CheckMigration(target string) (MigrationStatus, error) {
	var currentPath string
	if target == "storage" {
		currentPath = a.GetStorageDir()
	} else if target == "covers" {
		currentPath = a.GetCoversDir()
	} else {
		return MigrationStatus{}, fmt.Errorf("invalid target: %s", target)
	}

	var count int
	var size int64

	entries, err := os.ReadDir(currentPath)
	if err != nil {
		if os.IsNotExist(err) {
			return MigrationStatus{}, nil
		}
		return MigrationStatus{}, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		count++
		size += info.Size()
	}

	return MigrationStatus{Count: count, Size: size}, nil
}

// MigrateData migrates data from the current directory to the new path
func (a *App) MigrateData(target string, newPath string, copyOnly bool) error {
	var currentPath string
	settings := a.GetSettings()
	
	if target == "storage" {
		currentPath = a.GetStorageDir()
	} else if target == "covers" {
		currentPath = a.GetCoversDir()
	} else {
		return fmt.Errorf("invalid target: %s", target)
	}

	// Normalize paths to avoid migrating to the same place
	if strings.EqualFold(filepath.Clean(currentPath), filepath.Clean(newPath)) {
		// Just update settings and return
		if target == "storage" {
			settings.StoragePath = newPath
		} else {
			settings.CoversPath = newPath
		}
		return a.SaveSettings(settings)
	}

	// Ensure destination exists
	if err := os.MkdirAll(newPath, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Get total size to check space
	status, err := a.CheckMigration(target)
	if err != nil {
		return fmt.Errorf("failed to calculate source size: %w", err)
	}
	count := status.Count
	totalSize := status.Size

	if count > 0 {
		// Check free space on destination drive
		freeSpace, err := GetDiskFreeSpace(newPath)
		if err == nil && totalSize > 0 && freeSpace < uint64(totalSize)*2 {
			return fmt.Errorf("insufficient disk space on destination. Required: %d bytes, Available: %d bytes", totalSize, freeSpace)
		}

		// Pause Watcher if it's running and we're modifying related things
		if a.fileWatcher != nil {
			a.fileWatcher.Stop()
			defer a.initFileWatcher() // Restart after migration
		}
		
		// Stop sync tasks or mbworker if necessary, or just rely on file-level locks
		
		copiedFiles := []string{}
		
		// Copy files
		entries, err := os.ReadDir(currentPath)
		if err != nil {
			return fmt.Errorf("failed to read source directory: %w", err)
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			srcPath := filepath.Join(currentPath, entry.Name())
			destPath := filepath.Join(newPath, entry.Name())

			// Handle potential "file in use" errors during copy
			if err := a.safeCopyFile(srcPath, destPath); err != nil {
				// Rollback
				for _, cf := range copiedFiles {
					os.Remove(cf)
				}
				return fmt.Errorf("failed to copy %s: %w. Please ensure the file is not opened in another program", entry.Name(), err)
			}
			copiedFiles = append(copiedFiles, destPath)
		}

		// Verify size
		_, newTotalSize, _ := a.checkDirSize(newPath)
		if newTotalSize != totalSize {
			// Rollback
			for _, cf := range copiedFiles {
				os.Remove(cf)
			}
			return fmt.Errorf("migration size mismatch. Expected %d, got %d. Migration rolled back", totalSize, newTotalSize)
		}

		// Update DB logic? No, our DB stores relative paths for managed items,
		// or relies on the current GetStorageDir/GetCoversDir.
		// Wait, if it's `isManaged`, we store relative path (filename).
		// But OLD tabs might have absolute paths! We need to convert them to relative!
		if target == "storage" {
			tabs, _ := a.store.GetTabs()
			for _, t := range tabs {
				if t.IsManaged && filepath.IsAbs(t.FilePath) {
					// Make it relative
					t.FilePath = filepath.Base(t.FilePath)
					a.store.UpdateTab(t)
				}
			}
		} else if target == "covers" {
			tabs, _ := a.store.GetTabs()
			for _, t := range tabs {
				if t.CoverPath != "" && filepath.IsAbs(t.CoverPath) {
					t.CoverPath = filepath.Base(t.CoverPath)
					a.store.UpdateTab(t)
				}
			}
		}

		// Clean old files if not copyOnly
		if !copyOnly {
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				os.Remove(filepath.Join(currentPath, entry.Name()))
			}
			
			// Try to remove the old directory if it's empty
			os.Remove(currentPath)
		}
	}

	// Update Settings
	if target == "storage" {
		settings.StoragePath = newPath
	} else {
		settings.CoversPath = newPath
	}
	
	if err := a.SaveSettings(settings); err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}

	// Notify frontend to reload view
	time.Sleep(1 * time.Second) // Let file system settle
	wailsRuntime.EventsEmit(a.ctx, "migration-completed", target)

	return nil
}

func (a *App) safeCopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

func (a *App) checkDirSize(dir string) (int, int64, error) {
	var count int
	var size int64
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, 0, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		count++
		size += info.Size()
	}
	return count, size, nil
}
