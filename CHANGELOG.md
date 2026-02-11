# Changelog

All notable changes to this project will be documented in this file.

## [1.3.2] - 2026-02-11

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
