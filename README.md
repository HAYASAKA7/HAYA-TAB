# HAYA-TAB

A lightweight music tab manager for guitarists and musicians, built with Go and Wails.

![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20macOS%20%7C%20Linux%20%7C%20iOS-blue)
![Version](https://img.shields.io/badge/version-3.1.7-green)
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

## 🚀 Usage

1. **Add Tabs**: Right-click on empty space → "Upload TAB" or "Link Local TAB"
2. **Organize**: Create categories and move tabs into them
3. **Sync Folders**: Go to Settings → Add sync paths to auto-import tabs from folders
4. **Cloud Sync**: Configure WebDAV in Settings to access your cloud library. See [WebDAV Guide](docs/WEBDAV.md).
5. **PDF Annotation**: In the PDF viewer, open the annotation menu from the toolbar, choose tools (selection/pen/highlighter/eraser), and draw on the non-destructive overlay layer.
6. **View Tabs**: Click a tab to open with system default, or right-click → "Open with Inner Viewer"
7. **Key Bindings**: Customize viewer controls (Loop, Auto-scroll, etc.) in Settings

## 🛠️ For Developers & Advanced Users

Information about building from source, testing, project architecture, and the tech stack has been moved to our developer documentation:

- [Development Guide](docs/DEVELOPMENT.md)
- [Mobile Development Guide](docs/MOBILE_DEVELOPMENT.md)
- [Architecture Overview](docs/ARCHITECTURE.md)
- [Contributing Guidelines](docs/CONTRIBUTING.md)
- [WebDAV Guide](docs/WEBDAV.md)

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

![Stats from the past 12 weeks](https://repopulse-l41y.onrender.com/api/status?repo=HAYASAKA7/HAYA-TAB&period=weekly&count=12)
