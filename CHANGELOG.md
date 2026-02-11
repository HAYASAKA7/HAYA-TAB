# Changelog

All notable changes to this project will be documented in this file.

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
