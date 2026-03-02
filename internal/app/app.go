package app

import (
	"context"
	"fmt"
	"haya-tab/pkg/coverpool"
	"haya-tab/pkg/logger"
	"haya-tab/pkg/metadata"
	"haya-tab/pkg/store"
	syncpkg "haya-tab/pkg/sync"
	"haya-tab/pkg/watcher"
	"haya-tab/pkg/worker"
	"io"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// idCounter is used to ensure unique IDs even when called in rapid succession
var idCounter uint64

// getAppDir returns the directory where the database and logs should be stored.
// It is forced to the user's config directory so that it's accessible even if a custom storage drive is offline.
func getAppDir() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			cwd, _ := os.Getwd()
			appDataDir := filepath.Join(cwd, "data")
			os.MkdirAll(appDataDir, 0755)
			return cwd
		}
		appDataDir := filepath.Join(homeDir, ".haya-tab")
		os.MkdirAll(appDataDir, 0755)
		return appDataDir
	}
	appDataDir := filepath.Join(configDir, "HAYA-TAB")

	os.MkdirAll(appDataDir, 0755)

	return appDataDir
}

// GetStorageDir returns the directory for managed tabs.
func (a *App) GetStorageDir() string {
	if a.store != nil {
		settings := a.store.GetSettings()
		if settings.StoragePath != "" {
			os.MkdirAll(settings.StoragePath, 0755)
			return settings.StoragePath
		}
	}
	dir := filepath.Join(getAppDir(), "storage")
	os.MkdirAll(dir, 0755)
	return dir
}

// GetCoversDir returns the directory for cover images.
func (a *App) GetCoversDir() string {
	if a.store != nil {
		settings := a.store.GetSettings()
		if settings.CoversPath != "" {
			os.MkdirAll(settings.CoversPath, 0755)
			return settings.CoversPath
		}
	}
	dir := filepath.Join(getAppDir(), "covers")
	os.MkdirAll(dir, 0755)
	return dir
}

// ResolveTabPath converts a relative path to absolute using GetStorageDir.
func (a *App) ResolveTabPath(path string, isManaged bool) string {
	if !isManaged || path == "" {
		return path
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(a.GetStorageDir(), path)
}

// ResolveCoverPath converts a relative path to absolute using GetCoversDir.
func (a *App) ResolveCoverPath(path string) string {
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(a.GetCoversDir(), path)
}

// generateID generates a unique ID for tabs
func generateID() string {
	counter := atomic.AddUint64(&idCounter, 1)
	return fmt.Sprintf("tab_%d%d", time.Now().UnixNano(), counter)
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
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

// WailsEventEmitter adapts wails runtime to the EventEmitter interface
type WailsEventEmitter struct {
	ctx context.Context
}

// Emit sends an event to the frontend via wails runtime
func (e *WailsEventEmitter) Emit(eventName string, data interface{}) {
	if e.ctx != nil {
		wailsRuntime.EventsEmit(e.ctx, eventName, data)
	}
}

// emitEvent safely emits an event through the Wails runtime
// It checks if context is valid before emitting
func (a *App) emitEvent(eventName string, data interface{}) {
	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, eventName, data)
	}
}

// App struct holds all application dependencies and state
type App struct {
	ctx            context.Context
	store          *store.DBStore
	fileWatcher    *watcher.FileWatcher
	logger         *logger.Logger
	fileServerPort int
	coverPool      *coverpool.CoverPool
	syncService    *syncpkg.SyncService
	mbWorker       *worker.MBWorker
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

// Startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	appDir := getAppDir()

	// Init Logger
	a.logger = logger.NewLogger(appDir)
	a.logger.SetContext(ctx)
	a.logger.Info("App starting in directory: %s", appDir)

	// Ensure required directories exist for DB and logger
	requiredDirs := []string{
		filepath.Join(appDir, "data"),
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

	// Now that store is initialized, ensure storage and cover directories exist based on settings
	a.GetStorageDir()
	a.GetCoversDir()

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

	// Initialize MusicBrainz worker (1 request per second rate limit)
	a.mbWorker = worker.NewMBWorker(a.store, a.logger)
	a.mbWorker.Start()
	a.logger.Info("MusicBrainz worker started")

	// Initialize SyncService
	emitter := &WailsEventEmitter{ctx: a.ctx}
	a.syncService = syncpkg.NewSyncService(a.store, a.logger, a.coverPool, emitter, appDir, a.mbWorker)
	a.logger.Info("SyncService initialized")

	// Auto Sync Logic
	go a.runAutoSync()

	// Background backfill for legacy data: fetch origin_country for tabs with covers but no origin_country
	go a.backfillOriginCountry()

	// Background backfill for legacy data: calculate initials for tabs without them
	go a.backfillInitials()

	// Initialize file watcher if sync paths are configured
	a.initFileWatcher()
}

// runAutoSync handles automatic synchronization based on settings
func (a *App) runAutoSync() {
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
}

// backfillOriginCountry fetches origin_country for tabs with covers but no origin_country
func (a *App) backfillOriginCountry() {
	// Wait a bit longer to ensure the app is fully initialized
	time.Sleep(5 * time.Second)

	tabs, err := a.store.GetTabsNeedingOriginCountry()
	if err != nil {
		a.logger.Error("Failed to get tabs needing origin country: %v", err)
		return
	}

	if len(tabs) == 0 {
		a.logger.Info("No tabs need origin country backfill")
		return
	}

	a.logger.Info("Starting background backfill for %d tabs needing origin country", len(tabs))

	// Submit all tabs to the MusicBrainz worker queue
	// The worker will process them at 1 per second automatically
	for _, tab := range tabs {
		if tab.Artist != "" {
			a.mbWorker.Submit(worker.MBJob{
				TabID:      tab.ID,
				ArtistName: tab.Artist,
			})
		}
	}

	a.logger.Info("Queued %d tabs for origin country backfill", len(tabs))
}

// backfillInitials calculates initials for tabs without them
func (a *App) backfillInitials() {
	// Wait a bit to ensure the app is fully initialized
	time.Sleep(6 * time.Second)

	tabs, err := a.store.GetTabsNeedingInitials()
	if err != nil {
		a.logger.Error("Failed to get tabs needing initials: %v", err)
		return
	}

	if len(tabs) == 0 {
		a.logger.Info("No tabs need initials backfill")
		return
	}

	a.logger.Info("Starting background backfill for %d tabs needing initials", len(tabs))

	// Calculate and update initials for each tab
	updated := 0
	for _, tab := range tabs {
		az, kana := metadata.CalculateInitials(tab.Title, tab.OriginCountry)
		if err := a.store.UpdateTabInitials(tab.ID, az, kana); err != nil {
			a.logger.Error("Failed to update initials for tab %s: %v", tab.ID, err)
			continue
		}
		updated++
	}

	a.logger.Info("Successfully backfilled initials for %d/%d tabs", updated, len(tabs))
}

// initFileWatcher initializes the file watcher if sync paths are configured
func (a *App) initFileWatcher() {
	settings := a.store.GetSettings()
	if len(settings.SyncPaths) > 0 {
		a.fileWatcher = watcher.NewFileWatcher(func() {
			// Emit event to frontend when changes detected
			a.emitEvent("file-changes-detected", "Files have changed in sync directories")
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

// Shutdown is called when the app is closing
func (a *App) Shutdown(ctx context.Context) {
	// Stop MusicBrainz worker
	if a.mbWorker != nil {
		a.mbWorker.Stop()
	}

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
