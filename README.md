# HAYA-TAB

A lightweight music tab manager for guitarists and musicians, built with Go and Wails.

![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20macOS%20%7C%20Linux-blue)
![Version](https://img.shields.io/badge/version-2.2.8-green)
![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)

## ✨ Features

- **Tab Management** - Organize your PDF, Guitar Pro (.gp, .gp5, .gpx), and MusicXML (.xml, .musicxml, .mxl) tabs in one place
- **Upload or Link** - Upload tabs to internal storage or link existing files from your filesystem
- **Advanced Search** - **Instant Full-Text Search (FTS5)** across titles, artists, and albums with fuzzy matching
- **Real-time Sync** - Automatically watches synced folders for changes; **Non-destructive** import (renames duplicates)
- **Cloud Sync** - **WebDAV** integration for on-demand cloud file access and uploads
- **Smart Metadata** - Auto-parse info from filenames; **Bi-directional sync** with Guitar Pro internal metadata
- **Tag Support** - Add version/part tags (e.g., "Lead Guitar", "Bass", "First Version")
- **Customizable Environment** - Change storage locations for managed tabs and covers to customize your setup
- **Album Artwork** - Automatic cover art fetching from iTunes; **High-performance** concurrent downloads
- **Categories** - Organize tabs into virtual folders
- **Batch Operations** - Select and move/delete multiple tabs at once
- **Rich Internal Viewer**:
  - **PDF:** Built-in viewer with **Auto-Scroll** (variable speed)
  - **Guitar Pro:** alphaTab engine with **Looping**, **Section Playback**, **Speed Control**, and **Floating Toolbar**
- **Internationalization** - Full support for **English, Chinese (Simplified/Traditional), and Japanese**; **Auto-detects system language** on first launch
- **Modern UI** - Dark/Light theme, **Auto-saving settings**, and responsive Grid/List views

## 📦 Installation

### Pre-built Binary
Download the latest release from the [Releases](https://github.com/HAYASAKA7/HAYA-TAB/releases) page.

### Build from Source
1. Ensure you have [Go](https://go.dev/), [Node.js](https://nodejs.org/) (npm), and [Wails](https://wails.io/) installed
2. Clone this repository
3. Install frontend dependencies:
   ```bash
   cd frontend
   npm install
   cd ..
   ```
4. Run the development server:
   ```bash
   wails dev
   ```
5. To build for production:
   ```bash
   # Build for current platform
   wails build
   
   # Cross-platform builds
   wails build -platform windows/amd64
   wails build -platform darwin/amd64     # macOS Intel
   wails build -platform darwin/arm64     # macOS Apple Silicon
   wails build -platform linux/amd64
   ```

## 🚀 Usage

1. **Add Tabs**: Right-click on empty space → "Upload TAB" or "Link Local TAB"
2. **Organize**: Create categories and move tabs into them
3. **Sync Folders**: Go to Settings → Add sync paths to auto-import tabs from folders
4. **Cloud Sync**: Configure WebDAV in Settings to access your cloud library. See [WebDAV Guide](docs/WEBDAV.md).
5. **View Tabs**: Click a tab to open with system default, or right-click → "Open with Inner Viewer"
6. **Key Bindings**: Customize viewer controls (Loop, Auto-scroll, etc.) in Settings

## 📁 Project Structure

```
├── main.go                           # Application entry point
├── go.mod & go.sum                   # Go module dependencies
├── wails.json                        # Wails framework configuration
│
├── internal/                         # Internal packages (not importable)
│   └── app/                          # Core application logic
│       ├── app.go                    # App struct, lifecycle management
│       ├── app_tabs.go               # Tab CRUD operations
│       ├── app_categories.go         # Category management
│       ├── app_files.go              # File dialogs & sync triggers
│       ├── app_settings.go           # Settings persistence
│       ├── app_migration.go          # Data migration utilities
│       ├── app_webdav.go             # WebDAV cloud operations
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
│   │   └── wailsjs/                  # Auto-generated Wails bindings
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
│   │   ├── models.go                 # Data models
│   │   ├── migration.go              # Schema migrations
│   │   ├── crypto.go                 # Encryption utilities
│   │   └── locale_*.go               # Platform-specific locale detection
│   ├── sync/                         # File synchronization
│   │   ├── sync.go                   # Sync engine
│   │   └── webdav.go                 # WebDAV operations
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
│   └── WEBDAV.md                     # WebDAV setup guide
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

- **Backend**: Go + Wails v2
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
