# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Development Commands

```bash
# Development (hot reload)
wails dev

# Production build (current platform)
wails build

# Cross-platform builds
wails build -platform windows/amd64
wails build -platform darwin/amd64      # macOS Intel
wails build -platform darwin/arm64      # macOS Apple Silicon
wails build -platform linux/amd64

# Frontend only (from frontend/)
npm install     # Install dependencies
npm run dev     # Dev server (rarely needed, use wails dev instead)
npm run build   # Build frontend only
```

## Architecture Overview

HAYA-TAB is a desktop music tab manager built with **Go + Wails v2** (backend) and **Vue 3 + TypeScript + Vite** (frontend).

### Backend Structure (Go)

- `app.go` - Wails bridge exposing all API methods to frontend (the main API surface)
- `main.go` - Entry point, sets up file server for serving local files to the viewer
- `pkg/store/` - SQLite database layer with FTS5 full-text search (models, migrations, CRUD)
- `pkg/metadata/` - Filename parsing and iTunes cover art fetching
- `pkg/sync/` - File sync engine and WebDAV client (custom HTTP transport for Chinese cloud services)
- `pkg/watcher/` - fsnotify-based file system watcher for sync folders
- `pkg/coverpool/` - Worker pool for concurrent cover art downloads
- `pkg/logger/` - Logging infrastructure

### Frontend Structure (Vue 3)

- `frontend/src/stores/` - Pinia stores: `tabs`, `settings`, `ui`, `viewers`
- `frontend/src/views/` - Main views: `HomeView`, `LibraryView`
- `frontend/src/components/` - UI components organized by feature (grid, modals, viewers, layout)
- `frontend/src/i18n/` - Translations for EN, ZH-CN, ZH-TW, JA

### Data Flow

1. Frontend calls Go methods via Wails bindings (auto-generated in `frontend/wailsjs/`)
2. Go methods in `app.go` interact with `pkg/store/` for database operations
3. Events flow back to frontend via `wailsRuntime.EventsEmit()`

### Key Patterns

- **Repository Pattern**: `DBStore` in `pkg/store/` abstracts all SQLite operations
- **Worker Pool**: `CoverPool` manages concurrent iTunes cover downloads
- **Event Emitter**: Wails runtime events notify frontend of backend changes (file sync, etc.)
- **WebDAV Client**: Custom implementation with anti-hotlinking headers for Baidu/Alibaba cloud compatibility

## Database

SQLite with FTS5, stored in `data/haya-tab.db`. Key tables:
- `tabs` - Tab metadata (title, artist, album, file_path, cover_path, categories)
- `categories` - Hierarchical folder structure
- `settings` - User preferences (theme, language, WebDAV credentials)
- `key_bindings` - Customizable keyboard shortcuts

## File Storage

- `storage/` - Uploaded tab files (managed internally)
- `covers/` - Downloaded cover art from iTunes
- `logs/` - Application logs

## Supported File Types

- PDF tabs (viewed with PDF.js)
- Guitar Pro: `.gp`, `.gp3`, `.gp4`, `.gp5`, `.gpx` (viewed with alphaTab)
- MusicXML: `.xml`, `.musicxml`, `.mxl` (viewed with alphaTab)
