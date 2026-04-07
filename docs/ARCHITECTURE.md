# HAYA-TAB Architecture

HAYA-TAB is a cross-platform desktop application built using the [Wails](https://wails.io/) framework, combining a Go backend with a modern web frontend (Vue 3 + Vite).

## System Overview

The system architecture is divided into two primary layers:
1. **Frontend (Presentation Layer):** A responsive, web-based UI rendering views, handling user interactions, and maintaining client-side state.
2. **Backend (Application Layer):** A native Go application managing the business logic, database operations, file system interactions, synchronization, and the Wails application lifecycle.

Communication between the frontend and backend happens via Wails' IPC (Inter-Process Communication) bridge.

## Frontend Architecture

- **Framework:** Vue 3 via Vite.
- **State Management:** Pinia stores (`src/stores`) manage application state including tabs, settings, UI states, and viewer configurations.
- **Routing:** Handled via Vue's dynamic component rendering or a minimal router setup depending on state (e.g., `HomeView` vs `LibraryView`).
- **Styling:** Custom CSS with a responsive grid layout.
- **Viewers:**
  - **PDF.js:** For rendering standard PDF documents.
    - A custom non-destructive annotation overlay is injected on top of PDF pages and persisted as JSON.
  - **alphaTab:** A music notation and guitar tablature rendering engine for playing `.gp`, `.gp5`, `.gpx`, and MusicXML files.
- **Components:** Modular components organized into grids, layouts, modals, viewers, and common UI elements.

## Backend Architecture

The Go backend is structured into distinct, modular packages:

- **`internal/app`:** The core application logic. It initializes the Wails app lifecycle and exposes methods bound to the frontend (e.g., managing tabs, categories, file dialogs, settings, migrations, and WebDAV operations).
  - `app_webdav.go` - Main WebDAV entry points (initialize, discover volumes, health checks, reconnect)
  - `app_webdav_helpers.go` - Fingerprint batch operations (add/remove files from fingerprints)
  - `app_annotations.go` - Annotation remote path mapping and lightweight WebDAV sync helpers
  - `app_tabs.go` - Tab CRUD, batch operations (with cloud tab support)
  - `app_settings.go` - Settings persistence (tracks WebDAV configuration changes)
  
- **`pkg/store`:** The data persistence layer. Uses SQLite (via `modernc.org/sqlite`) for local storage, implementing Full-Text Search (FTS5) for fast querying.
  - Core database operations in `database.go`
  - Annotation persistence in `database_annotations.go` (`tab_annotations` table)
  - Cloud volume operations in `database_volumes.go` (add, update, delete, query volumes)
  - Volume migration in `database_migration_volumes.go` (legacy tab migration, orphan cleanup)
  - Data models including `CloudVolume` struct in `models.go`
  
- **`pkg/sync`:** File synchronization and cloud operations:
  - `sync.go` - Core sync engine for local folder scanning
  - `webdav.go` - WebDAV client wrapper (connection, file operations, directory listing)
  - `volume.go` - Fingerprint file management (read, write, create volume fingerprints)
  - `volume_sync.go` - Volume discovery, registration, multi-device sync logic
  
- **`pkg/metadata`:** Parses metadata from filenames and tab files (like Guitar Pro). Includes MusicBrainz API integration (`musicbrainz.go`) for retrieving missing album artwork and information.

- **`pkg/coverpool`:** A worker pool for concurrent, high-performance downloading of cover images without blocking the main application flow.

- **`pkg/watcher`:** File system watcher for detecting changes in configured sync folders (non-destructive import).

- **`pkg/logger`:** Structured logging to trace application events and errors.

- **`pkg/worker`:** Background processing queues for long-running tasks, keeping the UI responsive.

## WebDAV Volume System Architecture (v2.3.0+)

The volume system enables seamless multi-device synchronization by treating each cloud drive (or WebDAV directory) as a **volume** with a unique fingerprint file:

### Volume Components

1. **CloudVolume Model** (`pkg/store/models.go`)
   - Unique ID (UUID) identifying the volume across devices
   - Mount path (e.g., `/gdrive/`, `/onedrive/`)
   - Fingerprint path (`.haya-volume-fingerprint` file location)
   - Creation and last-seen timestamps
   - Availability status

2. **Metadata Directory** (`haya-metadata/`)
   - Stores volume metadata and fingerprint buckets (`bucket-00.json` ... `bucket-15.json`)
   - Contains metadata (ID, name, creation date, app version, device name)
   - Stores file records uploaded via HAYA-TAB with metadata
   - Enables other devices to discover and sync the volume autonomously
   - Also stores non-destructive PDF annotation JSON under `haya-metadata/annotations/...`

3. **Database Operations** (`pkg/store/database_volumes.go`, `pkg/store/database_migration_volumes.go`)
   - `cloud_volumes` table stores volume information
   - Tab records include `volume_id` for cloud tabs (foreign key)
   - Relative file paths (not absolute WebDAV paths) enable portability
   - Migration functions handle legacy cloud tabs and orphaned records

4. **WebDAV Operations** (`pkg/sync/volume.go`, `pkg/sync/volume_sync.go`)
   - `ReadVolumeFingerprint()` - Read volume metadata from WebDAV
   - `UpdateVolumeFingerprint()` - Write file records to fingerprint
   - `FingerprintExists()` - Check if a directory has a fingerprint
   - `ScanVolumes()` - Discover all volumes in WebDAV root
   - `DiscoverAndRegisterVolumes()` - Register discovered volumes with auto-create for new directories

5. **App-Level Integration** (`internal/app/app_webdav.go`, `internal/app/app_webdav_helpers.go`, `internal/app/app_annotations.go`)
  - `WebDAVInitialize()` - Initialize volume system on startup
  - `WebDAVDiscoverVolumes()` - Scan and register all volumes
  - `WebDAVCheckVolumeHealth()` - Monitor volume accessibility
   - `WebDAVMigrateCloudTabs()` - Migrate legacy cloud tabs
  - `WebDAVReconnect()` - Reconnect and reinitialize on connection restore
  - `monitorWebDAVConnection()` - Background monitor (checks every 30 seconds)
  - Fingerprint batch operations for efficient uploads/deletions
  - Annotation sync uses relative paths and stores JSON in metadata subdirectory

### Multi-Device Sync Flow

```
Device A discovers WebDAV and enables it:
  ├─ WebDAVDiscoverVolumes() scans /gdrive/, /onedrive/, etc.
  ├─ Creates fingerprints for volumes without them
  ├─ Registers volumes in local database
  └─ Stores relative paths for cloud tabs

Device A uploads a file:
  ├─ File uploaded to /gdrive/folder/file.gp5
  ├─ Metadata extracted (title, artist, album)
  ├─ Fingerprint updated with file record
  └─ Relative path "folder/file.gp5" stored (not absolute)

Device B later accesses the same WebDAV:
  ├─ WebDAVDiscoverVolumes() scans /gdrive/ (same mount path)
  ├─ Finds existing volume (fingerprint exists)
  ├─ Reads fingerprint, discovers Device A's uploaded file
  ├─ Imports file record with metadata to local database
  └─ User can now play/view without re-parsing
```

## Plugin System Architecture (v2.4.0+)

The plugin system allows extending the core functionality of HAYA-TAB through custom JavaScript scripts evaluated securely at runtime.

### Plugin Components

1. **JavaScript Engine (`dop251/goja`)**
   - Embedded ECMAScript 5.1+ runtime purely in Go.
   - Secure execution environment with controlled access to host functions.

2. **Plugin Manager (`internal/app/plugin_manager.go`)**
   - On app startup, resolves runtime plugin directory as `<os.UserConfigDir()>/HAYA-TAB/plugins` and loads plugins from there.
   - Manages plugin lifecycles and provides a set of injected API functions (e.g., logging, HTTP requests).
   - Reads plugin-local `config.json` and exposes it to JavaScript as global `config`.
   - Supports enable/disable state via `__enabled` in plugin config.

3. **Plugin Structure**
   - Runtime path: `<os.UserConfigDir()>/HAYA-TAB/plugins/<plugin-id>/`
   - Repository source path: `internal/app/plugins/<plugin-id>/` (for built-in/distributed plugins)
   - `manifest.json`: Defines the plugin ID, name, version, entry point (`index.js`), hooks, permissions, and optional settings schema.
   - `index.js`: JavaScript entry script using `module.exports`.
   - Optional `config.json`: plugin-local runtime config (string key/value map).

4. **Hook System**
   - `metadata` hook: called during sync; plugin exports `enhanceMetadata(tab)` and returns modified tab data.
   - `cover` hook: called during cover lookup; plugin exports `getCoverUrl(artist, album, title, country, lang)` and returns a URL or `null`.
   - Plugins registered for a hook are invoked only when enabled.

5. **Injected Runtime APIs**
   - `log(message)` for plugin-scoped logging.
   - `fetch(url)` for simple GET requests.
   - `httpRequest({ method, url, headers, body })` for advanced HTTP use cases.
   - `config` global populated from `config.json`.

### Built-in Plugins

- **AI Metadata (`ai-metadata`)**: An AI-powered plugin that integrates with an OpenAI-compatible API to automatically infer, complete, and enhance missing tab metadata (title, artist, album, tags) based on the filename or contextual clues.

## Data Flow

### Standard Tab Management
1. **User Action:** The user interacts with the Vue interface (e.g., uploading a tab).
2. **Wails Binding:** The frontend calls a bound Go method asynchronously over IPC.
3. **Business Logic:** The Go `app` package processes the request, validating inputs and determining necessary actions (e.g., calculating metadata, checking for duplicates).
4. **Data Persistence:** The `store` package inserts or updates records in the SQLite database. If new metadata needs fetching, it might queue a task to `worker` and `coverpool`.
5. **Response:** The Go method returns a result or an error back over the IPC bridge.
6. **State Update:** The frontend's Pinia store updates its local state and triggers a re-render of the Vue components.

### WebDAV Multi-Device Sync Flow
1. **Connection Enable:** User enables WebDAV in settings.
2. **Volume Discovery:** `WebDAVInitialize()` → `WebDAVDiscoverVolumes()` scans WebDAV root for volumes.
3. **Registration:** Existing volumes found → Registered with mount paths. New directories → Auto-create fingerprints.
4. **File Sync:** For each discovered volume:
   - Read fingerprint to get metadata of files uploaded via the app
   - Import file records to local database with relative paths
   - User can access files without re-downloading/re-parsing
5. **Upload/Download:** When user uploads files:
   - File streamed to WebDAV
   - Metadata extracted and added to fingerprint
   - Relative path stored (enables portability across devices)
6. **Connection Monitoring:** Background monitor checks connection every 30 seconds → Auto-reconnect on restore.

## Deployment & Build Process

Wails bundles the frontend static assets directly into the Go binary. During `wails3 task build`, Vite compiles the Vue application into standard HTML/JS/CSS, and Go compiles the backend along with the embedded frontend assets to produce a single native executable for the target platform (Windows, macOS, or Linux).
