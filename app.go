package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"haya-tab/pkg/metadata"
	"haya-tab/pkg/store"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx   context.Context
	store *store.DBStore
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	// Init store
	cwd, _ := os.Getwd()
	dbPath := filepath.Join(cwd, "data", "haya-tab.db")
	jsonPath := filepath.Join(cwd, "data", "tabs.json")

	a.store = store.NewDBStore(dbPath)
	if err := a.store.Initialize(); err != nil {
		fmt.Printf("Error initializing database: %v\n", err)
		return
	}

	// Migrate from JSON if database is empty and JSON exists
	if !a.store.HasData() {
		if err := a.store.MigrateFromJSON(jsonPath); err != nil {
			fmt.Printf("Error migrating from JSON: %v\n", err)
		}
	}

	// Auto Sync on Startup
	go func() {
		// Small delay to ensure UI is ready if we want to emit events (optional)
		time.Sleep(1 * time.Second)
		a.TriggerSync()
	}()
}

// shutdown is called when the app is closing
func (a *App) shutdown(ctx context.Context) {
	if a.store != nil {
		a.store.Close()
	}
}

// GetSettings returns the current settings
func (a *App) GetSettings() store.Settings {
	return a.store.GetSettings()
}

// SaveSettings updates the settings
func (a *App) SaveSettings(s store.Settings) error {
	return a.store.UpdateSettings(s)
}

// TriggerSync scans the sync paths and adds/updates tabs based on strategy
func (a *App) TriggerSync() (string, error) {
	fmt.Println("Starting TriggerSync...")
	settings := a.store.GetSettings()
	if len(settings.SyncPaths) == 0 {
		return "No sync paths configured", nil
	}

	added := 0
	updated := 0
	skipped := 0
	errors := 0

	strategy := settings.SyncStrategy // "skip" or "overwrite"

	// Get all tabs once for comparison
	allTabs, err := a.store.GetTabs()
	if err != nil {
		return "", fmt.Errorf("failed to get tabs: %w", err)
	}

	for _, root := range settings.SyncPaths {
		fmt.Printf("Scanning path: %s\n", root)
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				fmt.Printf("Error accessing path %s: %v\n", path, err)
				return nil // Skip unreadable
			}
			if info.IsDir() {
				return nil
			}

			// check extension
			ext := strings.ToLower(filepath.Ext(path))
			if ext != ".pdf" && ext != ".gp" && ext != ".gp5" && ext != ".gpx" {
				return nil
			}

			// 1. Check if EXACT path exists
			for _, t := range allTabs {
				if t.FilePath == path {
					return nil // Already exists
				}
			}

			// 2. Parse Metadata to check Title conflict
			newTab := a.ProcessFile(path) // This creates a Tab struct with parsed info

			var conflictTab *store.Tab
			for i, t := range allTabs {
				// Compare Titles (or maybe normalize them?)
				// Using Title as "same name" indicator
				if t.Title == newTab.Title {
					conflictTab = &allTabs[i]
					break
				}
			}

			if conflictTab != nil {
				if strategy == "skip" {
					skipped++
					return nil
				} else if strategy == "overwrite" {
					// Handle Overwrite

					// If old one was managed (uploaded), delete the file
					if conflictTab.IsManaged {
						os.Remove(conflictTab.FilePath) // Ignore error
						conflictTab.IsManaged = false
					}

					// Update path
					conflictTab.FilePath = path
					// Update Metadata? Maybe keep old custom metadata?
					// Prompt implies replacing, so let's update basic fields but keep ID
					// Actually, let's keep Category, ID, Cover. Update FilePath and maybe Type.
					conflictTab.Type = newTab.Type

					// Save
					if err := a.store.AddTab(*conflictTab); err == nil {
						updated++
					}
					return nil
				}
			}

			// No conflict, add as new
			if err := a.store.AddTab(newTab); err == nil {
				added++
			} else {
				errors++
			}

			return nil
		})
		if err != nil {
			fmt.Printf("Error walking %s: %v\n", root, err)
		}
	}

	resultMsg := fmt.Sprintf("Sync complete: %d added, %d updated, %d skipped.", added, updated, skipped)
	wailsRuntime.EventsEmit(a.ctx, "sync-complete", resultMsg)
	return resultMsg, nil
}

// GetTabs returns the list of tabs
func (a *App) GetTabs() []store.Tab {
	tabs, err := a.store.GetTabs()
	if err != nil {
		fmt.Printf("Error getting tabs: %v\n", err)
		return []store.Tab{}
	}
	return tabs
}

// ProcessFile takes a file path and returns a pre-filled Tab struct
func (a *App) ProcessFile(path string) store.Tab {
	meta := metadata.ParseFilename(path)
	ext := strings.ToLower(filepath.Ext(path))
	typeStr := "unknown"
	if ext == ".pdf" {
		typeStr = "pdf"
	} else if ext == ".gp" || ext == ".gp5" || ext == ".gpx" {
		typeStr = "gp"
	}

	return store.Tab{
		ID:       fmt.Sprintf("%d", time.Now().UnixNano()), // Simple ID
		Title:    meta.Title,
		Artist:   meta.Artist,
		Album:    meta.Album,
		FilePath: path,
		Type:     typeStr,
	}
}

// GetCategories returns the list of categories
func (a *App) GetCategories() []store.Category {
	categories, err := a.store.GetCategories()
	if err != nil {
		fmt.Printf("Error getting categories: %v\n", err)
		return []store.Category{}
	}
	return categories
}

// AddCategory adds a new category
func (a *App) AddCategory(cat store.Category) error {
	// Generate ID if missing (though frontend might handle it, safer here or ensure uniqueness)
	if cat.ID == "" {
		cat.ID = fmt.Sprintf("cat_%d", time.Now().UnixNano())
	}
	return a.store.AddCategory(cat)
}

// DeleteCategory deletes a category
func (a *App) DeleteCategory(id string) error {
	return a.store.DeleteCategory(id)
}

// DeleteTab deletes a tab and its managed file if applicable
func (a *App) DeleteTab(id string) error {
	// Find tab first to check for managed file
	targetTab, err := a.store.GetTab(id)
	if err != nil {
		return fmt.Errorf("failed to get tab: %w", err)
	}
	if targetTab == nil {
		return fmt.Errorf("tab not found")
	}

	if targetTab.IsManaged {
		// Try to delete the file, log error but proceed with DB deletion
		if err := os.Remove(targetTab.FilePath); err != nil {
			fmt.Printf("Warning: Failed to delete managed file %s: %v\n", targetTab.FilePath, err)
		}
		// Also delete cover?
		if targetTab.CoverPath != "" {
			os.Remove(targetTab.CoverPath)
		}
	}

	return a.store.DeleteTab(id)
}

// BatchDeleteTabs deletes multiple tabs at once
func (a *App) BatchDeleteTabs(ids []string) (int, error) {
	deleted := 0
	for _, id := range ids {
		targetTab, err := a.store.GetTab(id)
		if err != nil || targetTab == nil {
			continue
		}

		if targetTab.IsManaged {
			// Try to delete the file
			if err := os.Remove(targetTab.FilePath); err != nil {
				fmt.Printf("Warning: Failed to delete managed file %s: %v\n", targetTab.FilePath, err)
			}
			// Also delete cover
			if targetTab.CoverPath != "" {
				os.Remove(targetTab.CoverPath)
			}
		}

		if err := a.store.DeleteTab(id); err == nil {
			deleted++
		}
	}
	return deleted, nil
}

// BatchMoveTabs moves multiple tabs to a category at once
func (a *App) BatchMoveTabs(ids []string, categoryID string) (int, error) {
	moved := 0
	for _, id := range ids {
		if err := a.store.MoveTab(id, categoryID); err == nil {
			moved++
		}
	}
	return moved, nil
}

// MoveTab updates the category of a tab
func (a *App) MoveTab(tabID, categoryID string) error {
	return a.store.MoveTab(tabID, categoryID)
}

// MoveCategory moves a category into another category
func (a *App) MoveCategory(id, newParentID string) error {
	if id == newParentID {
		return fmt.Errorf("cannot move category into itself")
	}
	// Note: A robust implementation should also check for circular dependency
	return a.store.MoveCategory(id, newParentID)
}

// ExportTab copies the tab file to a destination folder
func (a *App) ExportTab(id string, destFolder string) error {
	targetTab, err := a.store.GetTab(id)
	if err != nil {
		return fmt.Errorf("failed to get tab: %w", err)
	}
	if targetTab == nil {
		return fmt.Errorf("tab not found")
	}

	srcFile, err := os.Open(targetTab.FilePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	fileName := filepath.Base(targetTab.FilePath)
	destPath := filepath.Join(destFolder, fileName)

	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	return err
}

// SelectFolder opens a folder selection dialog
func (a *App) SelectFolder() string {
	selection, err := wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Select Destination Folder",
	})
	if err != nil {
		return ""
	}
	return selection
}

// SaveTab saves the tab. copyFile determines if we import it to internal storage.
// The passed tab should have the user-confirmed Metadata.
func (a *App) SaveTab(tab store.Tab, shouldCopy bool) error {
	cwd, _ := os.Getwd()

	// 1. Handle File Copy
	if shouldCopy {
		ext := filepath.Ext(tab.FilePath)
		newFilename := tab.ID + ext
		destPath := filepath.Join(cwd, "storage", newFilename)

		src, err := os.Open(tab.FilePath)
		if err != nil {
			return err
		}
		defer src.Close()

		dst, err := os.Create(destPath)
		if err != nil {
			return err
		}
		defer dst.Close()

		if _, err = io.Copy(dst, src); err != nil {
			return err
		}

		tab.FilePath = destPath
		tab.IsManaged = true
	} else {
		tab.IsManaged = false
	}

	// 2. Handle Cover (Async-ish)
	coverFilename := tab.ID + ".jpg"
	coverPath := filepath.Join(cwd, "covers", coverFilename)

	// If artist/album exists
	if tab.Artist != "" && tab.Album != "" {
		tabID := tab.ID // Capture ID for goroutine
		go func() {
			fmt.Printf("Attempting to download cover for: %s - %s\n", tab.Artist, tab.Album)
			err := metadata.DownloadCover(tab.Artist, tab.Album, tab.Country, tab.Language, coverPath)
			if err == nil {
				fmt.Printf("Cover downloaded successfully to: %s\n", coverPath)
				// Fetch current tab state from DB and update cover path
				currentTab, getErr := a.store.GetTab(tabID)
				if getErr != nil || currentTab == nil {
					fmt.Printf("Failed to get tab after cover download: %v\n", getErr)
					return
				}
				currentTab.CoverPath = coverPath
				a.store.AddTab(*currentTab)
				wailsRuntime.EventsEmit(a.ctx, "tab-updated", *currentTab) // Notify frontend
			} else {
				fmt.Printf("Failed to download cover: %v\n", err)
			}
		}()
	}

	// Save initial version
	return a.store.AddTab(tab)
}

// UpdateTab updates an existing tab's metadata
func (a *App) UpdateTab(tab store.Tab) error {
	// Let's just update the store.
	if err := a.store.AddTab(tab); err != nil {
		return err
	}

	// Trigger Cover Update
	cwd, _ := os.Getwd()
	coverFilename := tab.ID + ".jpg"
	coverPath := filepath.Join(cwd, "covers", coverFilename)

	if tab.Artist != "" && tab.Album != "" {
		tabID := tab.ID // Capture ID for goroutine
		go func() {
			fmt.Printf("Attempting to download cover for: %s - %s\n", tab.Artist, tab.Album)
			err := metadata.DownloadCover(tab.Artist, tab.Album, tab.Country, tab.Language, coverPath)
			if err == nil {
				fmt.Printf("Cover downloaded successfully to: %s\n", coverPath)
				// Fetch current tab state from DB and update cover path
				currentTab, getErr := a.store.GetTab(tabID)
				if getErr != nil || currentTab == nil {
					fmt.Printf("Failed to get tab after cover download: %v\n", getErr)
					return
				}
				currentTab.CoverPath = coverPath
				a.store.AddTab(*currentTab)
				wailsRuntime.EventsEmit(a.ctx, "tab-updated", *currentTab)
			} else {
				// Failed
				fmt.Printf("Failed to download cover for '%s': %v\n", tab.Title, err)
				wailsRuntime.EventsEmit(a.ctx, "cover-error", fmt.Sprintf("Failed to update cover for '%s': %v", tab.Title, err))
			}
		}()
	}

	return nil
}

// OpenTab opens the file using system default
func (a *App) OpenTab(id string) error {
	targetTab, err := a.store.GetTab(id)
	if err != nil {
		return fmt.Errorf("failed to get tab: %w", err)
	}
	if targetTab == nil {
		return fmt.Errorf("tab not found")
	}

	var cmd *exec.Cmd
	path := targetTab.FilePath

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default: // linux
		cmd = exec.Command("xdg-open", path)
	}
	return cmd.Start()
}

// GetCover returns the base64 encoded image
func (a *App) GetCover(path string) string {
	if path == "" {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(data)
}

// GetTabContent returns the base64 encoded content of the tab file for the internal viewer
func (a *App) GetTabContent(id string) (string, error) {
	targetTab, err := a.store.GetTab(id)
	if err != nil {
		return "", fmt.Errorf("failed to get tab: %w", err)
	}
	if targetTab == nil {
		return "", fmt.Errorf("tab not found")
	}

	data, err := os.ReadFile(targetTab.FilePath)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// SelectFiles opens a file dialog and returns the selected file paths
func (a *App) SelectFiles() []string {
	selection, err := wailsRuntime.OpenMultipleFilesDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Select Tab Files",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "Tabs (*.pdf;*.gp;*.gp5;*.gpx)", Pattern: "*.pdf;*.gp;*.gp5;*.gpx"},
		},
	})

	if err != nil {
		return nil
	}
	return selection
}

// SelectImage opens a file dialog for selecting images
func (a *App) SelectImage() string {
	selection, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Select Image",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "Images (*.jpg;*.png;*.jpeg;*.webp)", Pattern: "*.jpg;*.png;*.jpeg;*.webp"},
		},
	})

	if err != nil {
		return ""
	}
	return selection
}

// ReadPDF reads a PDF file and returns its base64 encoded content
func (a *App) ReadPDF(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(data)
}
