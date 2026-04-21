# HAYA-TAB

A lightweight music tab manager for guitarists and musicians, built with Go and Wails.

![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20macOS%20%7C%20Linux%20%7C%20iOS-blue)
![Version](https://img.shields.io/badge/version-3.1.4-green)
![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)

## ✨ Features

- **Tab Management** - Organize your PDF, Guitar Pro (.gp, .gp5, .gpx), and MusicXML (.xml, .musicxml, .mxl) tabs in one place
- **Upload or Link** - Upload tabs to internal storage or link existing files from your filesystem
- **Advanced Search** - **Instant Full-Text Search (FTS5)** across titles, artists, and albums with fuzzy matching
- **Real-time Sync** - Automatically watches synced folders for changes; **Non-destructive** import (renames duplicates)
- **Cloud Sync** - **WebDAV** integration with multi-device support via volume fingerprints; on-demand cloud file access and uploads with automatic metadata tracking
- **PDF Annotation Layer** - Non-destructive transparent canvas annotations (pen/highlighter/eraser) stored as lightweight JSON, without modifying original PDF files; includes a compact toolbar menu with current-tool icon indicator
- **Smart Metadata** - Auto-parse info from filenames; **Bi-directional sync** with Guitar Pro internal metadata
- **Tag Support** - Add version/part tags (e.g., "Lead Guitar", "Bass", "First Version")
- **Plugin System** - Extend functionality with JavaScript plugins (e.g., AI Metadata Enhancer)
- **Customizable Environment** - Change storage locations for managed tabs and covers to customize your setup
- **Album Artwork** - Automatic cover art fetching from iTunes; **High-performance** concurrent downloads
- **Categories** - Organize tabs into virtual folders
- **Batch Operations** - Select and move/delete multiple tabs at once
- **Rich Internal Viewer**:
  - **PDF:** Built-in viewer with **Auto-Scroll** and non-destructive **Annotation Layer** (pen/highlighter/eraser)
  - **Guitar Pro:** alphaTab engine with **Looping**, **Section Playback**, **Speed Control**, and **Floating Toolbar**
- **MIDI Pedal Support** - Control page turning, playback (Play/Pause), and **Smooth Scrolling** (via Expression Pedal) with any standard MIDI foot controller; includes **MIDI Learn** for easy mapping.
- **Internationalization** - Full support for **English, Chinese (Simplified/Traditional), and Japanese**; **Auto-detects system language** on first launch
- **Modern UI** - Dark/Light theme, **Auto-saving settings**, and responsive Grid/List views

## 📦 Installation

### Pre-built Binary
Download the latest release from the [Releases](https://github.com/HAYASAKA7/HAYA-TAB/releases) page.

#### 🍎 macOS Users (Unverified Developer)
Since the application is not signed with an Apple Developer account, macOS may show a warning like "cannot be opened because the developer cannot be verified" or "app is damaged". 

To fix this:
1. Go to **System Settings** -> **Privacy & Security**.
2. Scroll down to the Security section and click **"Open Anyway"** for HAYA-TAB.
3. Alternatively, run the following command in your terminal to remove the quarantine attribute (assuming you moved the app to `/Applications`):
   ```bash
   xattr -cr /Applications/HAYA-TAB.app
   ```

### Build from Source
1. Ensure you have [Go](https://go.dev/), [Node.js](https://nodejs.org/) (npm), and [Wails](https://v3.wails.io/getting-started/installation/) (v3) installed
2. Clone this repository
3. Install frontend dependencies:
   ```bash
   cd frontend
   npm install
   cd ..
   ```
4. Run the development server:
   ```bash
   wails3 task dev
   ```
5. To build for production:
   ```bash
   # Build for current platform
   wails3 task build
   
   # Cross-platform builds
   wails3 task windows:build ARCH=amd64
   wails3 task darwin:build ARCH=amd64     # macOS Intel
   wails3 task darwin:build ARCH=arm64     # macOS Apple Silicon
   wails3 task linux:build ARCH=amd64
   ```
   Note: when cross-compiling from a different host OS, run `wails3 task setup:docker` once first.

### Testing (E2E)
End-to-End UI integration tests are configured via Playwright. Note that your environment must support Node.js.
```bash
# Run headlessly against the Vite + Wails dev server
npx @go-task/cli test:e2e

# Run with Playwright UI visualizer for debugging
npx @go-task/cli test:e2e-ui
```

## 🚀 Usage

1. **Add Tabs**: Right-click on empty space → "Upload TAB" or "Link Local TAB"
2. **Organize**: Create categories and move tabs into them
3. **Sync Folders**: Go to Settings → Add sync paths to auto-import tabs from folders
4. **Cloud Sync**: Configure WebDAV in Settings to access your cloud library. See [WebDAV Guide](docs/WEBDAV.md).
5. **PDF Annotation**: In the PDF viewer, open the annotation menu from the toolbar, choose tools (selection/pen/highlighter/eraser), and draw on the non-destructive overlay layer.
6. **View Tabs**: Click a tab to open with system default, or right-click → "Open with Inner Viewer"
7. **Key Bindings**: Customize viewer controls (Loop, Auto-scroll, etc.) in Settings

## 📁 Project Structure

```
├── assets_embed.go                   # Embedded frontend assets package
├── cmd/
│   └── haya-tab/                     # Application entry package
│       ├── main.go                   # Application entry point
│       ├── main_ios.go               # iOS entry shim
│       ├── main_android.go           # Android entry shim
│       ├── app_options_ios.go        # iOS runtime options
│       └── app_options_default.go    # Non-iOS runtime options
├── go.mod & go.sum                   # Go module dependencies
├── Taskfile.yml                      # Top-level task entrypoint
│
├── internal/                         # Internal packages (not importable)
│   └── app/                          # Core application logic
│       ├── app.go                    # App struct, lifecycle management
│       ├── app_tabs.go               # Tab CRUD operations
│       ├── app_categories.go         # Category management
│       ├── app_files.go              # File dialogs & sync triggers
│       ├── app_settings.go           # Settings persistence
│       ├── app_migration.go          # Data migration utilities
│       ├── app_plugins.go            # Plugin management APIs
│       ├── app_webdav.go             # WebDAV cloud operations & volume system
│       ├── app_annotations.go        # Annotation WebDAV path mapping & sync helpers
│       ├── app_webdav_helpers.go     # WebDAV fingerprint helpers (batch operations)
│       ├── plugin_manager.go         # JS plugin runtime loader/executor
│       ├── plugins/                  # Built-in/distributed plugin source
│       ├── server.go                 # HTTP file server
│       ├── disk_unix.go              # Unix disk operations
│       └── disk_windows.go           # Windows disk operations
│
├── frontend/                         # Vue 3 + TypeScript + Vite frontend
│   ├── src/
│   │   ├── App.vue                   # Root component
│   │   ├── main.ts                   # Frontend entry point
│   │   ├── vite-env.d.ts             # Vite type definitions
│   │   ├── assets/                   # Styles & icons
│   │   ├── components/               # Reusable UI components
│   │   │   ├── BatchActionBar.vue    # Batch operation controls
│   │   │   ├── SettingsView.vue      # Settings panel
│   │   │   ├── common/               # Generic components (ContextMenu, SearchBar, Toast)
│   │   │   ├── grid/                 # Grid view components (TabCard, CategoryCard, TabGrid)
│   │   │   ├── layout/               # Layout components (AppSidebar, SidebarTabItem)
│   │   │   ├── modals/               # Modal dialogs (CloudPicker, WebDAV, Category, etc.)
│   │   │   └── viewers/              # File viewers (PDF, Guitar Pro, MusicXML)
│   │   ├── composables/              # Vue composables (useAlphaTab, useContextMenu, useToast)
│   │   ├── stores/                   # Pinia state management
│   │   │   ├── tabs.ts               # Tab state
│   │   │   ├── settings.ts           # Application settings
│   │   │   ├── ui.ts                 # UI state
│   │   │   ├── viewers.ts            # Viewer state
│   │   │   └── index.ts              # Store configuration
│   │   ├── views/                    # Page-level components
│   │   │   ├── HomeView.vue          # Landing page
│   │   │   └── LibraryView.vue       # Main library interface
│   │   ├── types/                    # TypeScript type definitions
│   │   ├── i18n/                     # Internationalization setup
│   │   │   └── locales/              # Translation files (EN, ZH-CN, ZH-TW, JA)
│   │   └── bindings/                 # Auto-generated Wails v3 bindings
│   ├── public/                       # Static assets
│   │   ├── alphatab/                 # alphaTab library & soundfonts
│   │   └── pdfjs/                    # PDF.js library & viewer
│   ├── index.html                    # HTML entry point
│   ├── package.json                  # Frontend dependencies
│   ├── tsconfig.json                 # TypeScript configuration
│   └── vite.config.ts                # Vite build configuration
│
├── pkg/                              # Shared packages
│   ├── coverpool/                    # Concurrent download worker pool
│   ├── logger/                       # Structured logging
│   ├── metadata/                     # Tab metadata parsing
│   │   ├── metadata.go               # Core metadata operations
│   │   ├── parser_gpx.go             # Guitar Pro file parser
│   │   ├── musicbrainz.go            # MusicBrainz API client
│   │   ├── initial.go                # Title initial calculation
│   │   └── gp_binary.go              # Binary format handlers
│   ├── store/                        # SQLite database layer
│   │   ├── database.go               # DB connection & management
│   │   ├── database_annotations.go   # PDF annotation persistence
│   │   ├── database_volumes.go       # Cloud volume operations
│   │   ├── database_migration_volumes.go    # Volume system migrations
│   │   ├── models.go                 # Data models (includes CloudVolume)
│   │   ├── migration.go              # Schema migrations
│   │   ├── crypto.go                 # Encryption utilities
│   │   └── locale_*.go               # Platform-specific locale detection
│   ├── sync/                         # File synchronization & WebDAV volume system
│   │   ├── sync.go                   # Sync engine
│   │   ├── webdav.go                 # WebDAV client & operations
│   │   ├── volume.go                 # Volume fingerprinting & metadata tracking
│   │   └── volume_sync.go            # Multi-device volume discovery & migration
│   ├── watcher/                      # File system watcher
│   │   └── watcher.go                # Watch folders for changes
│   └── worker/                       # Background workers
│       └── mb_worker.go              # MusicBrainz async worker
│
├── build/                            # Wails build output & assets
│   ├── darwin/                       # macOS build resources
│   └── windows/                      # Windows build resources
│
├── docs/                             # Documentation
│   ├── WEBDAV.md                     # WebDAV setup guide
│   ├── API.md                        # Backend API reference
│   ├── ARCHITECTURE.md               # System architecture overview
│   └── CONTRIBUTING.md               # Contribution guidelines
│
├── README.md                         # This file
├── CHANGELOG.md                      # Version history
├── LICENSE                           # Apache 2.0 License
├── NOTICE                            # Attribution notices
│
├── .github/                          # GitHub configuration (workflows, etc.)
├── .git/                             # Git repository metadata
├── .gitignore & .gitattributes       # Version control settings
└── frontend/package-lock.json        # Frozen frontend dependencies
```

## 🛠️ Tech Stack

- **Backend**: Go + Wails v3
- **Frontend**: Vue 3 + TypeScript + Vite
- **State Management**: Pinia
- **Internationalization**: vue-i18n
- **Database**: SQLite (via modernc.org/sqlite) + FTS5
- **Viewer Engine**: PDF.js & alphaTab

## ⚖️ License & Legal Notice

HAYA-TAB is open-sourced software licensed under the **Apache License 2.0**.

### Terms and Conditions
This project is free for personal and commercial use, modification, and distribution, provided that:
1. **License & Copyright**: You include a copy of the Apache 2.0 license and the original copyright notice in any substantial portion of the software.
2. **State Changes**: You explicitly state significant changes made to the files.
3. **No Liability**: The software is provided "as is", without warranty of any kind.

See the [LICENSE](LICENSE) file for the full legal text and [NOTICE](NOTICE) for attribution requirements.

## 👤 Author

**HAYASAKA7** - [cyanluxury267@gmail.com](mailto:cyanluxury267@gmail.com)
