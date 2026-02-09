# HAYA-TAB

Music Tab Manager built with Go and Wails.

## Features
- Manage local PDF/Guitar Pro tabs.
- Upload tabs (copies to storage) or Link existing paths.
- Automatic metadata guessing (Artist - Album - Song).
- iTunes Cover Art integration (Offline support).
- Dark Mode UI.

## How to Run

1. Ensure you have Go and Wails installed.
2. Run the development server:
   ```bash
   wails dev
   ```
3. To build for production:
   ```bash
   wails build
   ```

## Structure
- `app.go`: Backend logic (Tab management, File ops).
- `frontend/`: UI (HTML/CSS/JS).
- `pkg/`: Internal packages (Storage, Metadata).
- `storage/`: Uploaded tabs.
- `covers/`: Downloaded cover art.
- `data/`: JSON database.
