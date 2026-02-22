package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"haya-tab/pkg/coverpool"
	"haya-tab/pkg/logger"
	"haya-tab/pkg/metadata"
	"haya-tab/pkg/store"
	syncpkg "haya-tab/pkg/sync"
	"haya-tab/pkg/watcher"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// TabsResponse represents a paginated response for tabs
type TabsResponse struct {
	Tabs     []store.Tab `json:"tabs"`
	Total    int         `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
	HasMore  bool        `json:"hasMore"`
}

// getAppDir returns the directory where the executable is located
// This is more reliable than os.Getwd() for built applications
func getAppDir() string {
	// Check if running in Dev mode (project root contains wails.json)
	if cwd, err := os.Getwd(); err == nil {
		if _, err := os.Stat(filepath.Join(cwd, "wails.json")); err == nil {
			return cwd
		}
	}

	exePath, err := os.Executable()
	if err != nil {
		// Fallback to working directory
		cwd, _ := os.Getwd()
		return cwd
	}
	// Resolve symlinks to get the real path
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		cwd, _ := os.Getwd()
		return cwd
	}
	return filepath.Dir(exePath)
}

// WailsEventEmitter adapts wails runtime to the EventEmitter interface
type WailsEventEmitter struct {
	ctx context.Context
}

// Emit sends an event to the frontend via wails runtime
func (e *WailsEventEmitter) Emit(eventName string, data interface{}) {
	wailsRuntime.EventsEmit(e.ctx, eventName, data)
}

// App struct
type App struct {
	ctx            context.Context
	store          *store.DBStore
	fileWatcher    *watcher.FileWatcher
	logger         *logger.Logger
	fileServerPort int
	coverPool      *coverpool.CoverPool
	syncService    *syncpkg.SyncService
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// SetFileServerPort sets the port of the local file server
func (a *App) SetFileServerPort(port int) {
	a.fileServerPort = port
}

// GetFileServerPort returns the port of the local file server
func (a *App) GetFileServerPort() int {
	return a.fileServerPort
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	appDir := getAppDir()

	// Init Logger
	a.logger = logger.NewLogger(appDir)
	a.logger.SetContext(ctx)
	a.logger.Info("App starting in directory: %s", appDir)

	// Ensure required directories exist
	requiredDirs := []string{
		filepath.Join(appDir, "data"),
		filepath.Join(appDir, "storage"),
		filepath.Join(appDir, "covers"),
	}
	for _, dir := range requiredDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			a.logger.Error("Error creating directory %s: %v", dir, err)
		} else {
			a.logger.Info("Directory ensured: %s", dir)
		}
	}

	dbPath := filepath.Join(appDir, "data", "haya-tab.db")
	jsonPath := filepath.Join(appDir, "data", "tabs.json")
	a.logger.Info("Database path: %s", dbPath)

	a.store = store.NewDBStore(dbPath)
	if err := a.store.Initialize(); err != nil {
		a.logger.Error("Error initializing database: %v", err)
		return
	}

	// Migrate from JSON if database is empty and JSON exists
	if !a.store.HasData() {
		if err := store.MigrateFromJSON(a.store, jsonPath); err != nil {
			a.logger.Error("Error migrating from JSON: %v", err)
		}
	}

	// Initialize cover download worker pool (3 concurrent downloads max)
	a.coverPool = coverpool.NewCoverPool(3, metadata.DownloadCover)
	a.coverPool.Start()
	a.logger.Info("Cover download pool started with 3 workers")

	// Initialize SyncService
	emitter := &WailsEventEmitter{ctx: a.ctx}
	a.syncService = syncpkg.NewSyncService(a.store, a.logger, a.coverPool, emitter, appDir)
	a.logger.Info("SyncService initialized")

	// Auto Sync Logic
	go func() {
		// Small delay to ensure UI is ready
		time.Sleep(1 * time.Second)

		settings := a.store.GetSettings()
		if !settings.AutoSyncEnabled {
			return
		}

		shouldSync := false
		now := time.Now()
		lastSync := time.Unix(settings.LastSyncTime, 0)

		switch settings.AutoSyncFrequency {
		case "startup":
			shouldSync = true
		case "weekly":
			y1, w1 := lastSync.ISOWeek()
			y2, w2 := now.ISOWeek()
			if y1 != y2 || w1 != w2 {
				shouldSync = true
			}
		case "monthly":
			if lastSync.Month() != now.Month() || lastSync.Year() != now.Year() {
				shouldSync = true
			}
		case "yearly":
			if lastSync.Year() != now.Year() {
				shouldSync = true
			}
		default: // Fallback
			shouldSync = true
		}

		if shouldSync {
			a.logger.Info("Auto-sync triggered due to schedule.")
			a.TriggerSync()
		}
	}()

	// Initialize file watcher if sync paths are configured
	settings := a.store.GetSettings()
	if len(settings.SyncPaths) > 0 {
		a.fileWatcher = watcher.NewFileWatcher(func() {
			// Emit event to frontend when changes detected
			wailsRuntime.EventsEmit(a.ctx, "file-changes-detected", "Files have changed in sync directories")
		})
		a.fileWatcher.SetLogger(a.logger)

		if err := a.fileWatcher.Start(); err != nil {
			a.logger.Error("Failed to start file watcher: %v", err)
		} else {
			// Add all sync paths to watcher
			for _, path := range settings.SyncPaths {
				if err := a.fileWatcher.AddPath(path); err != nil {
					a.logger.Error("Failed to watch path %s: %v", path, err)
				}
			}
		}
	}
}

// shutdown is called when the app is closing
func (a *App) shutdown(ctx context.Context) {
	// Stop cover download pool
	if a.coverPool != nil {
		a.coverPool.Stop()
	}

	// Stop file watcher
	if a.fileWatcher != nil {
		a.fileWatcher.Stop()
	}

	if a.store != nil {
		a.store.Close()
	}

	if a.logger != nil {
		a.logger.Close()
	}
}

// GetSettings returns the current settings
func (a *App) GetSettings() store.Settings {
	return a.store.GetSettings()
}

// SaveSettings updates the settings
func (a *App) SaveSettings(s store.Settings) error {
	// Update file watcher paths if they changed
	oldSettings := a.store.GetSettings()
	if err := a.store.UpdateSettings(s); err != nil {
		return err
	}

	// Update file watcher if sync paths changed
	if len(s.SyncPaths) > 0 {
		if a.fileWatcher == nil {
			// Create new watcher
			a.fileWatcher = watcher.NewFileWatcher(func() {
				wailsRuntime.EventsEmit(a.ctx, "file-changes-detected", "Files have changed in sync directories")
			})
			a.fileWatcher.SetLogger(a.logger)

			if err := a.fileWatcher.Start(); err != nil {
				a.logger.Error("Failed to start file watcher: %v", err)
			}
		}

		// Update watched paths
		if a.fileWatcher != nil && a.fileWatcher.IsRunning() {
			if err := a.fileWatcher.SetPaths(s.SyncPaths); err != nil {
				a.logger.Error("Failed to update watcher paths: %v", err)
			}
		}
	} else if a.fileWatcher != nil {
		// No sync paths, stop watcher
		a.fileWatcher.Stop()
		a.fileWatcher = nil
	}

	// Check if paths changed to emit notification
	pathsChanged := len(oldSettings.SyncPaths) != len(s.SyncPaths)
	if !pathsChanged {
		for i := range oldSettings.SyncPaths {
			if oldSettings.SyncPaths[i] != s.SyncPaths[i] {
				pathsChanged = true
				break
			}
		}
	}

	if pathsChanged && len(s.SyncPaths) > 0 {
		a.logger.Info("File watcher updated with %d paths", len(s.SyncPaths))
	}

	return nil
}

// TriggerSync delegates to SyncService for file synchronization
func (a *App) TriggerSync() (string, error) {
	return a.syncService.TriggerSync()
}

// fetchCoverAsync delegates to SyncService for async cover download
func (a *App) fetchCoverAsync(tab store.Tab) {
	a.syncService.FetchCoverAsync(tab)
}

// GetTabs returns the list of tabs (backward compatibility)
func (a *App) GetTabs() []store.Tab {
	tabs, err := a.store.GetTabs()
	if err != nil {
		a.logger.Error("Error getting tabs: %v", err)
		return []store.Tab{}
	}
	return tabs
}

// GetTabsPaginated returns a paginated list of tabs with optional search
func (a *App) GetTabsPaginated(categoryId string, page, pageSize int, searchQuery string, filterBy []string, isGlobal bool, sortBy string, sortDesc bool) TabsResponse {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}

	if len(filterBy) == 0 {
		filterBy = []string{"title"}
	}
	searchQuery = strings.ToLower(strings.TrimSpace(searchQuery))

	tabs, total, err := a.store.GetTabsPaginated(categoryId, page, pageSize, searchQuery, filterBy, isGlobal, sortBy, sortDesc)
	if err != nil {
		a.logger.Error("Error getting paginated tabs: %v", err)
		return TabsResponse{
			Tabs:     []store.Tab{},
			Total:    0,
			Page:     page,
			PageSize: pageSize,
			HasMore:  false,
		}
	}

	return TabsResponse{
		Tabs:     tabs,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		HasMore:  (page * pageSize) < total,
	}
}

// ProcessFile delegates to SyncService for file processing
func (a *App) ProcessFile(path string) store.Tab {
	return a.syncService.ProcessFile(path)
}

// GetCategories returns the list of categories
func (a *App) GetCategories() []store.Category {
	categories, err := a.store.GetCategories()
	if err != nil {
		a.logger.Error("Error getting categories: %v", err)
		return []store.Category{}
	}
	return categories
}

// GetRecentCategories returns the list of recently accessed categories
func (a *App) GetRecentCategories(limit int) []store.Category {
	categories, err := a.store.GetRecentCategories(limit)
	if err != nil {
		a.logger.Error("Error getting recent categories: %v", err)
		return []store.Category{}
	}
	return categories
}

// GetRecentTabs returns the list of recently accessed tabs
func (a *App) GetRecentTabs(limit int) []store.Tab {
	tabs, err := a.store.GetRecentTabs(limit)
	if err != nil {
		a.logger.Error("Error getting recent tabs: %v", err)
		return []store.Tab{}
	}
	return tabs
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
			a.logger.Error("Warning: Failed to delete managed file %s: %v", targetTab.FilePath, err)
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
				a.logger.Error("Warning: Failed to delete managed file %s: %v", targetTab.FilePath, err)
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

// BatchMoveTabs moves multiple tabs to a category at once (replaces existing categories)
func (a *App) BatchMoveTabs(ids []string, categoryID string) (int, error) {
	moved := 0
	baseTime := time.Now().Unix()
	for i, id := range ids {
		// Increment added time slightly to preserve order
		// For backward compatibility, "Move" implies setting the single category
		cats := []string{}
		if categoryID != "" {
			cats = append(cats, categoryID)
		}
		if err := a.store.SetTabCategories(id, cats, baseTime+int64(i)); err == nil {
			moved++
		}
	}
	return moved, nil
}

// BatchAddTabsToCategory adds multiple tabs to a category
func (a *App) BatchAddTabsToCategory(ids []string, categoryID string) (int, error) {
	added := 0
	baseTime := time.Now().Unix()
	for i, id := range ids {
		// Get existing tab to check for duplicates
		tab, err := a.store.GetTab(id)
		if err != nil || tab == nil {
			continue
		}

		// Check if already in category
		exists := false
		for _, c := range tab.CategoryIDs {
			if c == categoryID {
				exists = true
				break
			}
		}
		if exists {
			continue
		}

		newCats := append(tab.CategoryIDs, categoryID)
		if err := a.store.SetTabCategories(id, newCats, baseTime+int64(i)); err == nil {
			added++
		}
	}
	return added, nil
}

// MoveTab updates the category of a tab (replaces existing categories with this one)
func (a *App) MoveTab(tabID, categoryID string) error {
	cats := []string{}
	if categoryID != "" {
		cats = append(cats, categoryID)
	}
	return a.store.SetTabCategories(tabID, cats, time.Now().Unix())
}

// UpdateTabCategories updates the categories for a tab
func (a *App) UpdateTabCategories(tabID string, categoryIDs []string) error {
	return a.store.SetTabCategories(tabID, categoryIDs, time.Now().Unix())
}

// AddTabToCategory adds a tab to a category without removing it from others
func (a *App) AddTabToCategory(tabID, categoryID string) error {
	tab, err := a.store.GetTab(tabID)
	if err != nil {
		return err
	}
	if tab == nil {
		return fmt.Errorf("tab not found")
	}

	// Check if already in category
	for _, c := range tab.CategoryIDs {
		if c == categoryID {
			return nil // Already in category
		}
	}

	newCats := append(tab.CategoryIDs, categoryID)
	return a.store.SetTabCategories(tabID, newCats, time.Now().Unix())
}

// RemoveTabFromCategory removes a tab from a category
func (a *App) RemoveTabFromCategory(tabID, categoryID string) error {
	tab, err := a.store.GetTab(tabID)
	if err != nil {
		return err
	}
	if tab == nil {
		return fmt.Errorf("tab not found")
	}

	newCats := []string{}
	found := false
	for _, c := range tab.CategoryIDs {
		if c == categoryID {
			found = true
			continue
		}
		newCats = append(newCats, c)
	}

	if !found {
		return nil
	}

	return a.store.SetTabCategories(tabID, newCats, time.Now().Unix())
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
	// Check for duplicate file path before adding (for linked files)
	existingByPath, err := a.store.GetTabByPath(tab.FilePath)
	if err != nil {
		return fmt.Errorf("failed to check for duplicate path: %w", err)
	}
	if existingByPath != nil {
		return fmt.Errorf("a tab with this file already exists: %s", existingByPath.Title)
	}

	// Check for duplicate title globally (catches uploaded files with same content)
	existingByTitle, err := a.store.GetTabByTitle(tab.Title)
	if err != nil {
		return fmt.Errorf("failed to check for duplicate title: %w", err)
	}
	if existingByTitle != nil {
		return fmt.Errorf("a tab with title '%s' already exists", existingByTitle.Title)
	}

	appDir := getAppDir()

	// 1. Handle File Copy
	if shouldCopy {
		ext := filepath.Ext(tab.FilePath)
		newFilename := tab.ID + ext
		destPath := filepath.Join(appDir, "storage", newFilename)

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

	if tab.AddedAt == 0 {
		tab.AddedAt = time.Now().Unix()
	}

	// Save initial version first
	if err := a.store.AddTab(tab); err != nil {
		return err
	}

	// 2. Handle Cover (Async)
	a.fetchCoverAsync(tab)

	return nil
}

// UpdateTab updates an existing tab's metadata
func (a *App) UpdateTab(tab store.Tab) error {
	// Let's just update the store.
	if err := a.store.AddTab(tab); err != nil {
		return err
	}

	// Trigger Cover Update (Async)
	a.fetchCoverAsync(tab)

	return nil
}

// UpdateTabMetadata updates only the metadata fields (title, artist, album) for a tab.
// This is called by the frontend after AlphaTab parses the file's internal metadata.
// It implements a "smart update" strategy:
// - If no cover exists: prefer AlphaTab's data (more authoritative than filename parsing)
// - If cover exists: only update placeholder fields (existing data was good enough for cover search)
func (a *App) UpdateTabMetadata(id string, title string, artist string, album string) error {
	// Get current tab
	currentTab, err := a.store.GetTab(id)
	if err != nil {
		return fmt.Errorf("failed to get tab: %w", err)
	}
	if currentTab == nil {
		return fmt.Errorf("tab not found: %s", id)
	}

	needsUpdate := false
	noCoverYet := currentTab.CoverPath == ""

	// Helper to check if existing value is "placeholder" (empty or "Unknown")
	isPlaceholder := func(s string) bool {
		trimmed := strings.TrimSpace(s)
		return trimmed == "" || strings.EqualFold(trimmed, "Unknown")
	}

	// Helper to check if new value is meaningful
	isMeaningful := func(s string) bool {
		trimmed := strings.TrimSpace(s)
		if trimmed == "" {
			return false
		}
		// Skip generic placeholders
		lower := strings.ToLower(trimmed)
		if lower == "untitled" || lower == "unknown" || lower == "no title" {
			return false
		}
		return true
	}

	// Helper to check if values are different (case-insensitive)
	isDifferent := func(old, new string) bool {
		return !strings.EqualFold(strings.TrimSpace(old), strings.TrimSpace(new))
	}

	// Update title: if no cover AND AlphaTab has meaningful different data, prefer it
	// Otherwise only update if current is placeholder
	if isMeaningful(title) {
		if noCoverYet && isDifferent(currentTab.Title, title) {
			currentTab.Title = strings.TrimSpace(title)
			needsUpdate = true
			a.logger.Info("Updating title for tab %s (no cover, prefer AlphaTab): %s", id, title)
		} else if isPlaceholder(currentTab.Title) {
			currentTab.Title = strings.TrimSpace(title)
			needsUpdate = true
			a.logger.Info("Updating title for tab %s: %s", id, title)
		}
	}

	// Update artist: same logic
	if isMeaningful(artist) {
		if noCoverYet && isDifferent(currentTab.Artist, artist) {
			currentTab.Artist = strings.TrimSpace(artist)
			needsUpdate = true
			a.logger.Info("Updating artist for tab %s (no cover, prefer AlphaTab): %s", id, artist)
		} else if isPlaceholder(currentTab.Artist) {
			currentTab.Artist = strings.TrimSpace(artist)
			needsUpdate = true
			a.logger.Info("Updating artist for tab %s: %s", id, artist)
		}
	}

	// Update album: same logic
	if isMeaningful(album) {
		if noCoverYet && isDifferent(currentTab.Album, album) {
			currentTab.Album = strings.TrimSpace(album)
			needsUpdate = true
			a.logger.Info("Updating album for tab %s (no cover, prefer AlphaTab): %s", id, album)
		} else if isPlaceholder(currentTab.Album) {
			currentTab.Album = strings.TrimSpace(album)
			needsUpdate = true
			a.logger.Info("Updating album for tab %s: %s", id, album)
		}
	}

	if needsUpdate {
		if err := a.store.UpdateTab(*currentTab); err != nil {
			return fmt.Errorf("failed to update tab metadata: %w", err)
		}

		// Notify frontend about the update
		wailsRuntime.EventsEmit(a.ctx, "tab-updated", *currentTab)

		// If artist was updated and we have enough info, try fetching cover again
		if currentTab.Artist != "" && currentTab.CoverPath == "" {
			a.fetchCoverAsync(*currentTab)
		}
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

	// Update LastOpened
	targetTab.LastOpened = time.Now().Unix()
	a.store.UpdateTab(*targetTab)

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

// MarkAsOpened updates the LastOpened timestamp for a tab without opening it
func (a *App) MarkAsOpened(id string) error {
	targetTab, err := a.store.GetTab(id)
	if err != nil {
		return fmt.Errorf("failed to get tab: %w", err)
	}
	if targetTab == nil {
		return fmt.Errorf("tab not found")
	}

	targetTab.LastOpened = time.Now().Unix()
	return a.store.UpdateTab(*targetTab)
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
