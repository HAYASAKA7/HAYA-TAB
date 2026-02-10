# HAYA-TAB

A lightweight music tab manager for guitarists and musicians, built with Go and Wails.

![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20macOS%20%7C%20Linux-blue)
![Version](https://img.shields.io/badge/version-1.3.1-green)
![License](https://img.shields.io/badge/license-MIT-yellow)

## ✨ Features

- **Tab Management** - Organize your PDF and Guitar Pro (.gp, .gp5, .gpx) tabs in one place
- **Upload or Link** - Upload tabs to internal storage or link existing files from your filesystem
- **Advanced Search** - Filter by range (Category/Global) and type (Song/Artist/Album/Tag) with a smart collapsible interface
- **Real-time Sync** - Automatically watches synced folders for file changes (add/delete/rename)
- **Smart Metadata** - Auto-parse artist, album, and song info from filenames
- **Tag Support** - Add version/part tags to tabs (e.g., "Lead Guitar", "Bass", "First Version")
- **Album Artwork** - Automatic cover art fetching from iTunes (now also for synced tabs)
- **Categories** - Organize tabs into virtual folders with drag-and-drop support
- **Batch Operations** - Select and move/delete multiple tabs at once
- **Internal Viewer** - Built-in viewer for both PDF and Guitar Pro files
- **Dark/Light Theme** - System-aware theme with manual override
- **Duplicate Detection** - Prevents adding the same tab twice

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
2. **Organize**: Create categories and drag tabs into them
3. **Sync Folders**: Go to Settings → Add sync paths to auto-import tabs from folders
4. **View Tabs**: Click a tab to open with system default, or right-click → "Open with Inner Viewer"

## 📁 Project Structure

```
├── app.go              # Backend logic (Tab management, File ops)
├── main.go             # Application entry point
├── frontend/           # UI (Vue 3 + Vite)
│   ├── src/            # Frontend source code
│   ├── index.html      # Entry point
│   └── vite.config.ts  # Build config
├── pkg/                # Internal packages
│   ├── store/          # Database storage
│   ├── metadata/       # Metadata parsing & cover art
│   └── watcher/        # File system watcher
├── storage/            # Uploaded tabs (managed files)
├── covers/             # Downloaded cover art
└── data/               # SQLite database
```

## 🛠️ Tech Stack

- **Backend**: Go + Wails v2
- **Frontend**: Vue 3 + TypeScript + Vite
- **Database**: SQLite
- **Viewer Engine**: PDF.js & alphaTab

## 📄 License

MIT License - see [LICENSE](LICENSE) for details.

## 👤 Author

**HAYASAKA7** - [cyanluxury267@gmail.com](mailto:cyanluxury267@gmail.com)
