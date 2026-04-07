# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

HAYA-TAB is a cross-platform desktop music tab manager built with Go (backend) and Vue 3 (frontend) using the Wails v3 framework. It manages PDF, Guitar Pro (.gp, .gp5, .gpx), and MusicXML tabs with features including WebDAV cloud sync, PDF annotations, full-text search, and a JavaScript plugin system.

**Tech Stack:**
- Backend: Go 1.25.4 + Wails v3
- Frontend: Vue 3 + TypeScript + Vite + Pinia
- Database: SQLite (modernc.org/sqlite) with FTS5 full-text search
- Key Libraries: alphaTab (music notation), PDF.js, goja (JS runtime), fsnotify

## Development Commands

### Setup
```bash
# Install frontend dependencies
cd frontend && npm install && cd ..
```

### Development
```bash
# Run with hot-reload (starts both backend and frontend dev servers)
wails3 task dev

# Frontend only (for UI work)
cd frontend && npm run dev
```

### Testing
```bash
# Run all tests
go test ./...

# Run with race detection
go test -race ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./internal/app/...
go test ./pkg/store/...
```

### Building
```bash
# Build for current platform
wails3 task build

# Cross-platform builds
wails3 task windows:build ARCH=amd64
wails3 task darwin:build ARCH=amd64     # macOS Intel
wails3 task darwin:build ARCH=arm64     # macOS Apple Silicon
wails3 task linux:build ARCH=amd64
```

Build output: `bin/`

### Code Quality
```bash
# Format code
gofmt -w .
goimports -w .

# Static analysis
go vet ./...

# Security scanning (if installed)
gosec ./...
```

## Architecture Overview

### Layered Architecture
```
Frontend (Vue 3 + Pinia)
    ↓ Wails IPC Bridge
App Layer (internal/app)
    ↓ Dependency Injection
Service Layer (pkg/*)
    ↓ Data Access
Data Layer (pkg/store - SQLite)
```

### Backend Structure

**internal/app/** - Core application layer exposing methods to frontend via Wails bindings

Key files:
- `app.go` - App lifecycle, initialization, dependency injection
- `app_tabs.go` - Tab CRUD operations, batch operations
- `app_categories.go` - Category management
- `app_settings.go` - Settings persistence (theme, language, sync paths)
- `app_webdav.go` - WebDAV operations, volume system, multi-device sync
- `app_webdav_helpers.go` - Fingerprint batch operations for efficient uploads
- `app_annotations.go` - PDF annotation WebDAV sync
- `app_plugins.go` - Plugin management APIs
- `plugin_manager.go` - JavaScript plugin runtime (goja)
- `server.go` - HTTP file server for streaming tabs/covers

**pkg/store/** - SQLite data persistence with FTS5 full-text search

Key files:
- `database.go` - DB connection, WAL mode, migrations
- `database_volumes.go` - Cloud volume operations
- `database_annotations.go` - PDF annotation persistence
- `models.go` - Data models (Tab, Category, CloudVolume, Settings)
- `migration.go` - Schema migrations

Tables: `tabs`, `categories`, `tab_categories`, `settings`, `cloud_volumes`, `tab_annotations`, `tabs_fts` (FTS5)

**Database Performance Tuning:**
```go
PRAGMA journal_mode=WAL       // Write-Ahead Logging for read/write concurrency
PRAGMA synchronous=NORMAL     // Balance safety and performance
PRAGMA cache_size=-64000      // 64MB cache
PRAGMA temp_store=MEMORY      // In-memory temp tables
PRAGMA foreign_keys=ON        // Enforce referential integrity
```

**FTS5 Full-Text Search:**
- Virtual table `tabs_fts` synced via triggers (`tabs_ai`, `tabs_ad`, `tabs_au`)
- Rebuild command: `INSERT INTO tabs_fts(tabs_fts) VALUES('rebuild')`
- Automatic sync on INSERT, UPDATE, DELETE operations

**pkg/sync/** - File synchronization and WebDAV cloud operations

Key files:
- `sync.go` - Core sync engine (folder scanning, metadata extraction)
- `webdav.go` - WebDAV client wrapper
- `volume.go` - Volume fingerprint management
- `volume_sync.go` - Multi-device volume discovery and sync

**pkg/metadata/** - Tab metadata parsing from filenames and Guitar Pro files

Supported patterns:
- "Artist - Album - Title.ext"
- "Artist - Title.ext"
- "01. Artist - Title.ext"
- "[Artist] Title.ext"
- "Artist - Title (Key).ext"

**pkg/coverpool/** - Concurrent worker pool for cover art downloads

**pkg/watcher/** - File system monitoring with fsnotify

**pkg/worker/** - Background job processing (MusicBrainz API worker)

### Frontend Structure

**src/stores/** (Pinia state management)
- `tabs.ts` - Tab state
- `settings.ts` - Application settings
- `ui.ts` - UI state (modals, toasts)
- `viewers.ts` - Viewer state (PDF, alphaTab)

**src/components/**
- `grid/` - Grid view components
- `layout/` - Layout components (sidebar, etc.)
- `modals/` - Modal dialogs
- `viewers/` - File viewers (PDF.js, alphaTab)
- `common/` - Reusable UI components

**src/views/**
- `HomeView.vue` - Landing page
- `LibraryView.vue` - Main library interface

## Key Architectural Patterns

### Dependency Injection
Services are injected via constructor functions:
```go
func NewSyncService(
    store *store.DBStore,
    logger *logger.Logger,
    coverPool *coverpool.CoverPool,
    emitter EventEmitter,
    appDir string,
    mbWorker *worker.MBWorker,
    enhanceMetadata EnhanceMetadataFunc,
) *SyncService
```

### Event-Driven Communication
- Frontend → Backend: Wails bound method calls (async over IPC)
- Backend → Frontend: `wailsRuntime.EventsEmit()` for real-time updates
- File changes: fsnotify watcher → callback → frontend event

### Error Handling (`pkg/errors`)
The codebase uses a structured `AppError` type for all user-facing errors:
- `AppError` carries both a technical message (for logging) and an i18n key (for frontend translation)
- Specialized constructors: `WebDAVError()`, `TabNotFoundError()`, `DatabaseError()`, `FileNotFoundError()`, etc.
- `ToPayload(err)` serializes any error into `ErrorPayload` for API responses — frontend receives i18n key + args
- `WrapError()` converts plain errors into `AppError`, preserving existing `AppError` if already wrapped
- `Unwrap()` supports Go error chaining (`errors.As`, `errors.Is`)

### Concurrency Patterns
- `sync.RWMutex` protects settings and plugin access (DB operations rely on `database/sql` pool + WAL)
- Worker pools: `coverpool` (concurrent cover downloads), `mb_worker` (rate-limited MusicBrainz)
- Semaphore-limited concurrent operations (volume bucket reading)
- `sync.WaitGroup` for goroutine coordination
- Background goroutines for auto-sync, WebDAV monitoring, and backfill tasks

## WebDAV Volume System

The volume system enables multi-device sync by treating each cloud directory as a **volume** with a unique fingerprint.

### Key Concepts
- Each volume has a unique UUID stored in `haya-metadata/` directory
- **Bucket-based storage**: 16 bucket files (`bucket-00.json` to `bucket-15.json`) distribute file metadata using MD5 hash of relative path
- Relative file paths enable portability across devices
- Volume discovery happens automatically on WebDAV connection
- Legacy `.haya-volume-fingerprint` file supported for migration

### WebDAV Client Implementation (`pkg/sync/webdav.go`)
Uses a **dual-client strategy** for optimal performance:
- `metadataClient`: Keep-Alive enabled for metadata operations (directory listing, fingerprint reads)
- `streamClient`: Keep-Alive disabled for file transfers (uploads, downloads)

Custom transport injects browser-like headers to bypass anti-hotlinking on cloud drives (AliyunDrive, Baidu, Quark, 115, 189, 123Pan, etc.)

### Sync Flow
1. Device A uploads file → Metadata added to appropriate bucket (based on MD5 hash)
2. Device B connects → Discovers volume → Reads all buckets → Imports metadata
3. Both devices can access files without re-parsing

### Key Functions
- `WebDAVInitialize()` - Initialize volume system on startup
- `WebDAVDiscoverVolumes()` - Scan and register all volumes
- `WebDAVCheckVolumeHealth()` - Monitor volume accessibility
- Background monitor checks connection every 30 seconds

## Plugin System

JavaScript plugins extend functionality using the goja runtime (ECMAScript 5.1+).

**Plugin Location:**
- Runtime: `<UserConfigDir>/HAYA-TAB/plugins/<plugin-id>/`
- Repository: `internal/app/plugins/<plugin-id>/` (built-in plugins)

**Plugin Structure:**
```
<plugin-id>/
  manifest.json       # Plugin metadata, hooks, permissions
  index.js           # Entry point (module.exports)
  config.json        # Runtime configuration (optional)
```

**Available Hooks:**
- `metadata` - Enhance tab metadata during sync: `enhanceMetadata(tab)`
- `cover` - Provide cover art URL: `getCoverUrl(artist, album, title, country, lang)`

**Runtime APIs:**
- `log(message)` - Plugin logging with `[Plugin name]` prefix
- `fetch(url)` - Simple GET requests (returns response body as string)
- `httpRequest({ method, url, headers, body })` - Advanced HTTP (returns `{ status, headers, body }`)
- `config` - Global object from config.json (includes `__enabled` field for enable/disable state)

**Plugin Lifecycle:**
- Plugins loaded at startup from `<UserConfigDir>/HAYA-TAB/plugins/`
- `StartSyncRun()` resets per-run counters for metadata hooks
- Thread-safe access via `sync.RWMutex` in plugin manager
- Config stored in `config.json` within plugin directory

**Plugin Development:**
- Keep plugins self-contained and deterministic
- Fail gracefully, return null/original data on errors
- Never hardcode secrets (use config.json)
- Validate and sanitize external data
- Only request needed permissions in manifest.json
- Use `module.exports` for hook functions

**Syncing Plugins to Plugin Repository:**
```bash
# One-time setup
git remote add plugins-repo https://github.com/HAYASAKA7/HAYA-TAB-Plugins.git

# Push plugins subtree
git subtree push --prefix=internal/app/plugins plugins-repo main
```

## Testing Patterns

### Table-Driven Tests
Use table-driven patterns with setup helpers:
```go
func setupTestApp(t *testing.T) (*App, string) {
    t.Helper()  // Mark as test helper for better error traces
    tempDir := t.TempDir()
    // Initialize DBStore with isolated database
    // Set up isolated storage/covers directories
    // Return App and cleanup path (t.TempDir() auto-cleans)
}

func TestApp_Operation(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"valid input", "input", "expected", false},
        {"invalid input", "", "", true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Test Isolation
- Use `t.TempDir()` for automatic cleanup of temporary directories
- Each test gets isolated SQLite database in temp directory
- `ctx` set to `nil` to avoid Wails runtime dependency
- Integration-style tests with real database operations (not mocks)
- Use `t.Helper()` marker for setup functions

### Coverage Requirements
- Aim for comprehensive coverage of business logic
- Test error paths and edge cases
- Use race detector for concurrency testing: `go test -race ./...`

## Database Migrations

### Migration Pattern
Migrations run automatically in `runMigrations()` during DB initialization.

**Idempotent Schema Changes:**
```go
// Add column (idempotent)
_, err = s.db.Exec("ALTER TABLE tabs ADD COLUMN tag TEXT DEFAULT ''")
if err != nil && !strings.Contains(err.Error(), "duplicate column name") {
    return fmt.Errorf("failed to add column: %w", err)
}

// Create index (idempotent)
_, err = s.db.Exec("CREATE INDEX IF NOT EXISTS idx_tabs_volume ON tabs(volume_id)")
```

**Data Migration:**
```go
// Migrate existing data after schema change
_, err = s.db.Exec(`
    INSERT INTO tab_categories (tab_id, category_id, added_at)
    SELECT id, category_id, added_at FROM tabs
    WHERE category_id != '' AND ...
`)
```

**FTS Rebuild:**
```go
// Rebuild full-text search index
_, err = s.db.Exec("INSERT INTO tabs_fts(tabs_fts) VALUES('rebuild')")
```

### Schema Evolution Guidelines
- Use `ALTER TABLE` for column additions
- Handle "duplicate column name" errors gracefully
- Use `IF NOT EXISTS` for indexes
- Migrate data after schema changes
- Rebuild FTS index when needed

## Common Development Tasks

### Adding a New Tab Field
1. Update `Tab` struct in `pkg/store/models.go`
2. Add migration in `pkg/store/migration.go` (ALTER TABLE)
3. Update FTS virtual table if searchable
4. Update frontend TypeScript types in `src/types/`
5. Update UI components to display/edit field

### Adding a New WebDAV Operation
1. Add method to `WebDAVClient` in `pkg/sync/webdav.go`
2. Add app-level method in `internal/app/app_webdav.go`
3. Bind method to frontend in `app.go`
4. Call from frontend via Wails runtime

### Adding a New Plugin Hook
1. Define hook interface in `plugin_manager.go`
2. Add hook registration in `LoadPlugins()`
3. Invoke hook at appropriate point in app logic
4. Document hook in plugin development guide

### Modifying Database Schema
1. Add migration function in `pkg/store/migration.go`
2. Call migration in `runMigrations()`
3. Test with existing database (idempotency)
4. Update models and queries as needed

## Important Notes

- **Wails IPC**: All frontend-backend communication goes through Wails bindings
- **WAL Mode**: SQLite uses WAL mode for read/write concurrency
- **Relative Paths**: Cloud tabs use relative paths for device portability
- **Non-Destructive**: PDF annotations stored separately, never modify original files
- **Rate Limiting**: MusicBrainz API limited to 1 req/sec
- **Concurrency**: Use worker pools for parallel operations (cover downloads, etc.)
- **Context**: Pass context through for lifecycle management and cancellation

## Version Information

Current version: 3.0.1 (set via ldflags during build)

```bash
# Set version during build
go build -ldflags "-X haya-tab/internal/app.AppVersion=3.0.1"
```
