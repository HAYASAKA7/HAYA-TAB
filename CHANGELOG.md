# Changelog

All notable changes to this project will be documented in this file.

## [1.4.5] - 2026-02-12

### Added
- **"Jump to Start" Shortcut:** New keybinding (Default: `I`) to instantly jump to the first measure.

### Improved
- **Focus Management:** Score area now auto-focuses after track switching, enabling immediate use of keyboard shortcuts.
- **Playback Control:** "Play Selection" and "Jump to Bar" actions now precisely position the playback cursor, ensuring playback starts exactly where expected.

### Fixed
- **Multi-line Selection:** Fixed visual artifacts when selecting measures across multiple lines; highlights are now correctly segmented per line.

## [1.4.4] - 2026-02-11

### Changed
- **Architecture Refactor:** Extracted sync logic into dedicated `SyncService` (`pkg/sync/sync.go`). The `App` struct now serves purely as a bridge between frontend and backend services.
- **EventEmitter Interface:** Introduced `EventEmitter` abstraction to decouple sync logic from wails runtime, improving testability.

### Improved
- **Code Organization:** Reduced `app.go` from 890 to 728 lines by moving sync-related code to its own package.
- **Separation of Concerns:** `TriggerSync`, `ProcessFile`, `FetchCoverAsync`, and `generateUniqueTitle` now live in `SyncService`.
- **Maintainability:** Services are now injected via constructor, making dependencies explicit and easier to mock for testing.

## [1.4.3] - 2026-02-11

### Added
- **Non-destructive Sync:** When a duplicate title is detected during sync, the new file is now added with a `_copy1`, `_copy2`, etc. suffix instead of overwriting or deleting the existing file. This prevents accidental data loss.
- **Enhanced Sync Progress UI:** Added an animated progress bar, spinner on the Sync button, and real-time file count display ("X files processed") during synchronization.
- **Real-time File Feedback:** Backend now emits progress for every file being processed (not just every 10th), showing "Processing: [filename]" to indicate the app is active.

### Improved
- **Smart Metadata Updates:** When opening a GP file without cover art, AlphaTab's parsed metadata now overwrites filename-parsed data (considered more authoritative). If cover art already exists, only placeholder fields are updated.
- **Sync Strategy Label:** Updated the "Overwrite" option label to "Add as Copy (Rename new files)" to better reflect its non-destructive behavior.

## [1.4.2] - 2026-02-11

### Added
- **Cover Download Worker Pool:** Implemented a queuing mechanism that limits concurrent cover downloads to 3 workers. This prevents IP bans and system lag when syncing thousands of files.
- **FTS5 Full-Text Search:** Replaced LIKE queries with SQLite's FTS5 module for microsecond-level search performance. Search results are now ranked by relevance using BM25 scoring.

### Improved
- **SQLite WAL Mode:** Enabled Write-Ahead Logging for the database, allowing simultaneous reading and writing. The UI now remains smooth while background sync operations write to the database.
- **Database Performance:** Added optimized SQLite pragmas (64MB cache, memory temp store, normal synchronous mode) for faster overall database operations.
- **Search Fallback:** FTS5 search gracefully falls back to LIKE queries for special characters or edge cases.

## [1.4.1] - 2026-02-11

### Changed
- **Filename-First Parsing Strategy:** Completely removed backend binary parsing for GP files. Metadata is now extracted purely from filenames during import/sync, eliminating crash risks and encoding issues. Scanning speed improved by ~100x.

### Added
- **Frontend Reverse Write-back:** When a user opens a Guitar Pro file, AlphaTab parses the internal metadata (title, artist, album) and silently sends it back to the backend. The database becomes increasingly accurate as the user naturally uses the app.
- **Smart Metadata Updates:** New `UpdateTabMetadata` API intelligently updates only placeholder values, preserving user-edited metadata.

### Improved
- **Stability:** Removed complex binary header parsing that was prone to crashes on malformed or unusual GP files.
- **Cover Art Fetching:** Cover art is now automatically re-attempted when artist information becomes available via write-back.

## [1.4.0] - 2026-02-11

### Added
- **Guitar Pro Viewer Enhancements:**
  - **Floating Toolbar:** Added a quick-access toolbar for viewer tools.
  - **Context Menu:** Right-click menu for selection-based actions (Play Selection, Loop).
  - **Selection & Looping:** Users can now select a range of bars and loop playback.
  - **Jump to Bar:** Navigate quickly to a specific measure with visual highlighting.
- **Key Bindings:**
  - Added configurable key bindings for "Toggle Loop", "Clear Selection", and "Jump to Bar".
  - Updated the Settings UI to support customizing these new controls.
- **Store Updates:** Persisted key binding preferences in the database.

## [1.3.7] - 2026-02-11

### Improved
- **Sync Performance:** Optimized synchronization logic to use direct database queries instead of loading all tabs into memory. This significantly reduces memory usage when managing large libraries (10,000+ files).
- **Sync Feedback:** Added real-time progress updates during synchronization. The settings UI now displays the current file being scanned and the total count.

## [1.3.6] - 2026-02-11

### Fixed
- **Cover Display:** Fixed a bug where downloaded cover art would not appear on the tab card until the application was reloaded. The UI now reactively updates as soon as the cover is available.
- **Guitar Pro Parsing:** Fixed a critical issue where metadata (Title, Artist, etc.) could not be parsed from legacy Guitar Pro files (GP3, GP4, GP5) due to incorrect string length handling. This ensures cover art can now be correctly fetched for these files.
- **GPX Parsing:** Improved robustness of `.gpx` (GP7+) file parsing to handle case variations and subdirectories in the archive structure.

## [1.3.5] - 2026-02-11

### Changed
- **Core Architecture:** Refactored `pkg/store` to deprecate legacy JSON-based storage. The application now exclusively uses SQLite (`DBStore`) for data persistence.
- **Migration Logic:** Extracted data migration logic into a standalone module (`pkg/store/migration.go`) to decouple it from the main database logic.

### Fixed
- **Startup Stability:** Improved error handling during the file server startup. The application now catches port binding errors and logs them appropriately instead of failing silently or returning invalid ports.

## [1.3.1] - 2026-02-10

### Added
- **Advanced Search Component:** New collapsible search bar with detailed filtering options.
  - Added "Range" filter (Inside Category vs. Global).
  - Added "Type" filter (Song Name, Artist, Album, Tag).
  - Implemented "Click Outside" to collapse functionality.
- **SVG Icons:** Replaced font icons with SVG icons in the search component for better rendering.

### Changed
- **Sidebar Behavior:** Left sidebar is now collapsed by default on application start.
- **Search Logic:** Searching now hides category folders in the grid view to purely focus on tab results.
- **UI Styling:** Updated search component styles to match the application's card/board theme exactly.

## [1.3.0] - 2026-02-10

### Added
- **Modern Frontend Stack:** Migrated from Vanilla JS to Vue 3 + TypeScript + Vite for better performance and maintainability.
- **Internal Viewer:** Added internal support for viewing PDF and Guitar Pro files directly within the application.
- **File Watcher:** Integrated a file system watcher to automatically detect changes in synced directories.
- **Pagination:** Implemented paginated data loading for improved performance with large libraries (`GetTabsPaginated`).
- **Category Management:** Added support for moving categories and batch moving tabs between categories.
- **Image Selection:** Added native dialog for selecting custom cover images.

### Changed
- **Build System:** Updated Wails configuration to use `npm` based build pipeline.
- **Architecture:** Refactored backend `App` struct to support the new features and better state management.
- **UI/UX:** Complete overhaul of the user interface with modern components and styling.

### Fixed
- Fixed icon display issues.
- Improved Guitar Pro tab rendering consistency.
