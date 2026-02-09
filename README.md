# HAYA-TAB

A lightweight music tab manager for guitarists and musicians, built with Go and Wails.

![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20macOS%20%7C%20Linux-blue)
![Version](https://img.shields.io/badge/version-1.1.0-green)
![License](https://img.shields.io/badge/license-MIT-yellow)

## ✨ Features

- **Tab Management** - Organize your PDF and Guitar Pro (.gp, .gp5, .gpx) tabs in one place
- **Upload or Link** - Upload tabs to internal storage or link existing files from your filesystem
- **Folder Sync** - Automatically sync tabs from specified folders on startup
- **Smart Metadata** - Auto-parse artist, album, and song info from filenames
- **Album Artwork** - Automatic cover art fetching from iTunes
- **Categories** - Organize tabs into virtual folders with drag-and-drop support
- **Batch Operations** - Select and move/delete multiple tabs at once
- **Built-in PDF Viewer** - View PDF tabs without leaving the app
- **Dark/Light Theme** - System-aware theme with manual override
- **Duplicate Detection** - Prevents adding the same tab twice

## 📦 Installation

### Pre-built Binary
Download the latest release from the [Releases](https://github.com/HAYASAKA7/HAYA-TAB/releases) page.

### Build from Source
1. Ensure you have [Go](https://go.dev/) and [Wails](https://wails.io/) installed
2. Clone this repository
3. Run the development server:
   ```bash
   wails dev
   ```
4. To build for production:
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
├── frontend/           # UI (HTML/CSS/JS)
│   ├── app.js          # Frontend application logic
│   ├── style.css       # Styles
│   └── pdfjs/          # PDF.js viewer
├── pkg/                # Internal packages
│   ├── store/          # Database storage
│   └── metadata/       # Metadata parsing & cover art
├── storage/            # Uploaded tabs (managed files)
├── covers/             # Downloaded cover art
└── data/               # SQLite database
```

## 🛠️ Tech Stack

- **Backend**: Go + Wails v2
- **Frontend**: Vanilla HTML/CSS/JS
- **Database**: SQLite
- **PDF Viewer**: PDF.js

## 📄 License

MIT License - see [LICENSE](LICENSE) for details.

## 👤 Author

**HAYASAKA7** - [cyanluxury267@gmail.com](mailto:cyanluxury267@gmail.com)
