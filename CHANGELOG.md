# Changelog

All notable changes to this project will be documented in this file.

## [2.3.8] - 2026-03-12

### Added
- **True Virtual Grid Optimization:** Refactored the `LibraryView` to use `vue-virtual-scroller`, resolving severe initial render blocking and scrolling lag when dealing with thousands of items.
- **Improved A-Z Quick Jump Bar:** Re-implemented the alphabetical jump bar navigation logic to calculate dynamic indices, ensuring flawless synchronization with the newly virtualized DOM structure.

## [2.3.7] - 2026-03-11

### Added
- Centralized service layer in the frontend (`CategoryService`, `CloudService`, `FileService`, `SettingsService`, `TabService`) to encapsulate backend logic and improve code maintainability.
- Decoupled components and stores from direct Wails backend calls, leading to cleaner and more testable frontend code.

### Changed
- Refactored multiple frontend components and Pinia stores to use the new service layer.
- Updated documentation to reflect version 2.3.7.

## [2.3.6] - 2026-03-10

### Added
- **WebDAV Fingerprint Tombstones**: Added tombstone management to `FingerprintCache` to properly handle file deletions during concurrent syncs.
- **Concurrent Conflict Resolution**: Implemented "Read-Merge-Write" pattern in `FingerprintCache.Flush()` to prevent data loss when multiple devices update the same bucket.
- **LRU Cache Persistence**: Optimized `FingerprintCache` to keep data in memory after flushing, only clearing dirty flags.

### Improved
- **Sync Reliability**: Enhanced `FingerprintCache` to merge remote changes with local state during flush, ensuring eventual consistency across devices.

## [2.3.5] - 2026-03-09

### Added
- **Enhanced Settings Tests**: Added comprehensive test coverage for WebDAV settings changes with initialization verification
- **FileWatcher Lifecycle Tests**: Added tests for FileWatcher initialization and cleanup when sync paths change
- **Tab Validation**: Added tab existence checks in `UpdateTab()` to prevent silent failures with non-existent tabs
- **Tab Operation Tests**: Added new test cases for `SaveTab()` with invalid/valid files and storage verification
- **Metadata Update Tests**: Added dedicated tests for `UpdateTab()` and `UpdateTabMetadata()` operations

### Improved
- **Test Suite Refactoring**: Refactored `setupTestApp()` to properly initialize all dependencies (Logger, CoverPool, MBWorker, SyncService, VolumeCache)
- **Integration Tests**: Enhanced test setup to use real database initialization for more realistic testing scenarios
- **Error Handling**: Added explicit error checking in `UpdateTab()` for missing tabs with descriptive error messages

## [2.3.4] - 2026-03-09

### Added
- **Extended Format Support**: Added support for Guitar Pro versions 3 and 4 (.gp3, .gp4) format detection for webDAV sync and viewer loading
- **MusicXML Format Support**: Added full support for MusicXML standard formats (.xml, .musicxml) and compressed MusicXML (.mxl) for both webDAV sync and viewer loading
- **Content Type Headers**: Proper HTTP content-type headers for all supported music formats
  - Guitar Pro: `application/x-guitar-pro` (for .gp, .gp3, .gp4, .gp5, .gpx)
  - MusicXML: `application/vnd.recordare.musicxml+xml` (for .xml, .musicxml)
  - MusicXML (compressed): `application/vnd.recordare.musicxml` (for .mxl)

### Improved
- **WebDAV File Detection**: Enhanced WebDAV client to recognize and sync all new music formats
- **Format Detection**: Updated file extension checks to prioritize .pdf and properly order other format detection

## [2.3.3] - 2026-03-07

### Changed
- **Initialization Optimization**: Moved WebDAV initialization from Startup to DomReady hook for better timing and UI responsiveness.
- **WebDAV Connection Monitoring**: Removed the 5-second sleep delay in connection monitor for faster status checks.
- **HTTP Transport**: Added strict concurrency limits (MaxConnsPerHost = 5) to prevent overwhelming WebDAV servers.

### Improved
- **Memory Management**: Added AudioContext cleanup in alphaTab viewer to prevent memory leaks and exceeding hardware limits.
- **AlphaTab Disposal**: Enhanced cleanup logic for audio synthesis context with proper error handling.

## [2.3.2] - 2026-03-07

### Added
- **WebDAV Status**: Added a floating bar for better visibility of WebDAV sync status.

### Changed
- **WebDAV Optimizations**: Improved WebDAV integration and optimized the volume system.
- **Fingerprint Logic**: Optimized fingerprint operations by introducing a hash bucket mechanism.

### Fixed
- **Sync View**: Fixed an issue where the WebDAV sync would not correctly update the view.
- **Sync Scores**: Fixed an issue with initial letters handling when processing sync scores.

## [2.3.1] - 2026-03-04

### Added
- **Internationalization (i18n) for Volume System**: Added i18n strings for all new WebDAV volume features introduced in v2.3.0
  - `cloud.discoveringVolumes` — status message while scanning for volumes
  - `cloud.volumesDiscovered` — confirmation after volume discovery completes
  - `cloud.connectionRestored` — notification when WebDAV auto-reconnects
  - `cloud.connectionLost` — notification when WebDAV connection drops
  - `cloud.volumeUnavailable` — notification for inaccessible cloud volumes
  - `gpViewer.volumeUnavailable` — viewer error when cloud volume is offline
  - `gpViewer.volumeNotFound` — viewer error when volume record is missing
  - All strings available in English, Japanese, Simplified Chinese, and Traditional Chinese

### Fixed
- **Cloud Viewer Error Messages**: Improved error detection in AlphaTab loader for volume-specific 500 errors
  - Distinguishes "volume unavailable" from "volume not found" from generic server errors
  - Users now see actionable messages (e.g., "reconnect WebDAV") instead of generic server error text

## [2.3.0] - 2026-03-04

### Added
- **Multi-Device WebDAV Sync**: New volume system for managing cloud drives with fingerprint files
  - Each volume (cloud drive) has a unique fingerprint file (`.haya-volume-fingerprint`) containing metadata
  - Fingerprints track uploaded files, their metadata (title, artist, album), and upload source device
  - Enables seamless discovery and synchronization across multiple devices
- **WebDAV Volume Discovery**: `WebDAVDiscoverVolumes()` automatically scans WebDAV root for all volumes
  - Supports both existing volumes and auto-creates fingerprints for new directories
  - Handles multi-device scenarios where volumes are discovered after initial setup
- **Volume Health Checking**: `WebDAVCheckVolumeHealth()` monitors volume accessibility
  - Tracks which cloud drives are currently available
  - Updates availability status in database
- **Cloud Tab Migration**: Automatic migration of existing cloud tabs to use the new volume system
  - `WebDAVMigrateCloudTabs()` relinks legacy tabs to their respective volumes
  - `WebDAVCleanupOrphanedTabs()` removes tabs referencing non-existent volumes
- **Connection Monitoring**: Automatic WebDAV reconnection when connection is restored
  - `WebDAVReconnect()` reinitializes volume system after network restore
  - Background monitor checks connection every 30 seconds
- **Volume Creation API**: `WebDAVCreateVolume()` allows creating new cloud volumes programmatically
- **Fingerprint Management**: Direct fingerprint file access for upload/download operations
  - Tracks file relationships to volumes using relative paths
  - Updates fingerprints when files are uploaded or deleted
- **Enhanced Tab Management**: Tab records now include `volume_id` and `fingerprint_path` fields
  - Cloud tabs store relative path within volume (not absolute WebDAV path)
  - Enables portable multi-device support
- **Database Schema Updates**:
  - New `cloud_volumes` table to store volume metadata
  - Added `volume_id` column to tabs table
  - Migration functions for legacy data

### Improved
- **WebDAV Upload**: Files are now added to volume fingerprints automatically upon upload
  - Tracks metadata (title, artist, album, type) in fingerprint
  - Records upload timestamp and device name
- **Cloud Tab Addition**: Enhanced `WebDAVAddOnlineFiles()` with improved volume matching
  - Uses volume mount paths to correctly associate files with volumes
  - Logs detailed debug info for troubleshooting
  - Batch updates fingerprints for better performance
- **File Deletion**: Removes files from volume fingerprints when deleted from app
  - Single and batch deletions automatically update fingerprints
  - Maintains consistency between app database and WebDAV
- **App Startup**: WebDAV system initializes automatically on startup if enabled
  - Discovers volumes, checks health, and migrates legacy data
  - Non-blocking initialization prevents UI freeze

### Fixed
- **Cloud File Streaming**: Fixed path resolution for cloud tab playback
  - Correctly resolves relative paths using volume information
  - Proper error handling for unavailable volumes

## [2.2.20] - 2026-03-04

### Added
- **Grid UI Enhancement:** Improved TabCard component with better visual feedback

## [2.2.19] - 2026-03-04

### Added
- **WebDAV UI Enhancement:** Disabled cloud picker button when WebDAV is not connected
- **Style Improvements:** Added disabled button state styling
- **Localization:** Added offline state messages for all supported languages (English, Japanese, Simplified Chinese, Traditional Chinese)

### Improved
- **User Experience:** Better visual feedback for unavailable cloud features when WebDAV is disconnected

## [2.2.18] - 2026-03-04

### Added
- **pkg/errors:** New error handling package for standardized error management

### Improved
- **WebDAV Integration:** Enhanced sync.go and webdav.go with improved stability and test coverage

## [2.2.17] - 2026-03-04

### Fixed
- **Logger Multi-Writer Safety:** Implemented `safeMultiWriter` to handle Windows GUI stdout errors
  - Custom writer ignores stdout write errors to prevent panic on Windows Wails GUI
  - Maintains backward compatibility while improving stability

## [2.2.16] - 2026-03-04

### Improved
- **Database Query Optimization:** Enhanced `GetRecentTabs` query to use secondary sort by `added_at` for improved consistency
  - Changed: `ORDER BY last_opened DESC` → `ORDER BY last_opened DESC, added_at DESC`
  - Ensures stable and predictable ordering when multiple tabs have the same last_opened timestamp

## [2.2.15] - 2026-03-03

### Security
- **Enhanced Encryption System:**
  - Migrated from machine-derived encryption keys (PBKDF2) to OS keyring-based master keys
  - Uses OS-native credential managers: Windows Credential Manager, macOS Keychain, Linux Secret Service
  - Implements cryptographically random 32-byte keys with v2 encryption prefix
  - Provides OS-level encryption and access control for sensitive credentials

### Improved
- **Backward Compatibility:**
  - Legacy encrypted data (without v2: prefix) automatically detected and decrypted
  - Seamless migration path for existing users with zero data loss
  - Automatic cache management for both new and legacy encryption keys

### Dependencies
- Added `github.com/zalando/go-keyring` for cross-platform OS keyring integration

## [2.2.14] - 2026-03-03

### Added
- **UI/UX Improvements:**
  - Added MenuIcon component for context menus
  - Enhanced ContextMenu with improved icon support

### Improved
- **Component Updates:**
  - CategoryCard with better visual feedback
  - TabCard with enhanced interaction patterns
  - TabGrid with improved rendering performance
  - SidebarTabItem with better styling
  - HomeView and LibraryView with improved layout
  - Type definitions for better type safety

## [2.2.13] - 2026-03-02

### Added
- **Comprehensive Test Coverage:** Added extensive test suites across multiple packages
  - `pkg/coverpool`: Worker pool concurrency and error handling tests
  - `pkg/logger`: Structured logging functionality tests
  - `pkg/metadata`: Metadata parsing and initial calculation tests
  - `pkg/store`: Database operations, encryption, and migration tests
  - `pkg/sync`: File synchronization and WebDAV integration tests
  - `pkg/watcher`: File system watcher tests
  - `pkg/worker`: MusicBrainz worker tests
  - `internal/app`: Application logic tests for tabs, categories, settings, WebDAV, and migration
- **Developer Documentation:** Added comprehensive documentation for contributors
  - `docs/API.md`: Complete API reference for backend methods
  - `docs/ARCHITECTURE.md`: System architecture and design patterns
  - `docs/CONTRIBUTING.md`: Contribution guidelines and development workflow

### Improved
- **Code Quality:** Achieved significantly higher test coverage across the codebase
- **Maintainability:** Better documentation makes it easier for new contributors to understand the project

### Removed
- **Dead Code Cleanup:** Removed unused code to improve maintainability
  - Removed `pkg/metadata/gp_binary.go` (legacy binary parsing)
  - Removed `pkg/metadata/parser_gpx.go` (unused GPX parser)
  - Cleaned up unused WebDAV functions in `pkg/sync/webdav.go`

## [2.2.12] - 2026-03-02

### Fixed
- **Initial Letter Sorting:** Fixed issue where title initial letters were not recalculated when metadata was updated through backfill mechanisms
  - Manual metadata updates now correctly recalculate initials when title changes
  - MusicBrainz worker now recalculates initials after updating origin_country
  - Ensures consistent sorting in Quick Jump Bar across all metadata update paths

## [2.2.11] - 2026-02-28

### Changed
- **Wails Info:** Updated wails.json with Windows executable version information
  - Added companyName, productName, productVersion, copyright, and comments fields
  - Windows builds now display proper version details in file properties

## [2.2.10] - 2026-02-28

### Added
- **Category Management:** Enhanced context menu for tab cards with category operations
  - "Add to Category" option to move tabs between categories
  - "Remove from Category" option to remove tabs from non-system categories
  - Backend protection prevents removal from cloud storage category

## [2.2.9] - 2026-02-28

### Changed
- **Database Package Refactor:** Split monolithic `database.go` into focused modules for better maintainability
  - `database_tabs.go`: Tab CRUD operations and search functions (FTS5, pagination)
  - `database_categories.go`: Category management and relationships
  - `database_settings.go`: Settings CRUD and persistence
  - Core database initialization, connection management, and shared utilities remain in `database.go`
  - No logic changes; purely organizational improvement

### Fixed
- **Whitespace:** Removed trailing whitespace in FTS5 trigger definitions for cleaner SQL formatting

## [2.2.8] - 2026-02-28

### Added
- **XSS Prevention:** Added `safeHtml` utility function to sanitize user-controlled data in dialog messages
  - CategoryCard.vue and TabCard.vue now use `safeHtml` to prevent XSS attacks in confirmation dialogs
  - Dialog messages with user-provided category names and tab titles are now properly escaped

### Improved
- **Encryption Security:** Replaced hardcoded encryption key with machine-specific PBKDF2-derived key
  - Encryption key is now derived from machine hostname and username using PBKDF2 (100,000 iterations)
  - Different machines and users will have different encryption keys
  - Provides better security than hardcoded key while maintaining backward compatibility
  - Uses fixed salt to ensure consistent decryption across app restarts
- **Dependencies:** Moved `golang.org/x/crypto` from indirect to direct dependency for explicit security support

## [2.2.7] - 2026-02-28

### Fixed
- **Toast Notifications:** Unified toast styling for consistent appearance across all notification types
  - Toast boxes now size themselves based on content (max-width: 400px, min-width: 200px)
  - Removed excessive height and fixed dimensions for better visual consistency
  - All toast types (info, success, warning) use theme color; only error toasts display red
  - Added hover effect for better user interaction feedback

## [2.2.6] - 2026-02-28

### Added
- **WebDAV Improvements:** Enhanced WebDAV functionality with improved settings integration and user experience
- **Settings UI Updates:** Refined settings interface for better WebDAV configuration
- **i18n:** Updated translations for WebDAV-related features across all supported languages

### Improved
- **WebDAV Modal:** Better error handling and user feedback in cloud file operations
- **Settings Persistence:** Improved settings storage and migration for WebDAV credentials

## [2.2.5] - 2026-02-28

### Added
- **Customizable Data Path**: Users can now change the storage directory for managed tabs and covers via Settings
- **Data Migration**: Automatically prompts users to migrate existing files to the new directory when changing paths
- **i18n**: Added translation keys related to data storage and data migration features across all supported languages

## [2.2.4] - 2026-02-27

### Fixed
- **Cloud Add-Online Toast:** Adding cloud tabs to library (online) now shows the correct success/error message instead of incorrectly displaying "Download failed" or "Download complete"
- **i18n:** Added missing `addOnlineComplete` translation key for all supported languages (EN, ZH-CN, ZH-TW, JA)

## [2.2.3] - 2026-02-27

### Added
- **Auto-detect System Language:** The app now initializes with the user's OS language on first launch, instead of defaulting to English
  - Supports Windows (via `GetUserDefaultLocaleName` API), macOS, and Linux (via `LANG`/`LC_ALL` env vars)
  - Maps system locale to the nearest supported language: English, 简体中文, 繁體中文, 日本語
  - Falls back to English if the system language is not in the supported set
  - Existing users are unaffected — their saved language preference is preserved

## [2.2.2] - 2026-02-27

### Fixed
- **Database Concurrency:** Removed over-synchronization in SQLite operations to fully leverage WAL mode
  - Eliminated Go-level mutex locks from all pure database operations (CRUD methods)
  - Database operations now rely solely on `database/sql` connection pool + SQLite WAL mode for concurrency
  - Narrowed mutex scope to protect only in-memory Settings cache (using `sync.RWMutex`)
  - Preserved lifecycle locks for `Initialize()` only; removed unnecessary lock from `Close()`
- **Performance:** Background sync writes no longer block frontend read requests, significantly improving UI responsiveness during heavy sync operations

## [2.2.1] - 2026-02-27

### Changed
- **Code Architecture Refactor:** Reorganized backend code structure for better maintainability
  - Moved `App` struct and all related methods from `package main` to `internal/app` package
  - Split monolithic app.go into focused modules: `app.go`, `app_tabs.go`, `app_categories.go`, `app_files.go`, `app_settings.go`, `app_webdav.go`, `server.go`
  - Extracted HTTP file server logic into dedicated `server.go`
  - Updated Wails bindings from `window.go.main.App` to `window.go.app.App`
- **No Logic Changes:** This is a pure structural refactor with no changes to application behavior

## [2.2.0] - 2026-02-27

### Added
- **A-Z Quick Jump Bar:** Alphabet index navigation on the right side of Singles view for quick scrolling to letter groups (similar to iOS contacts). Supports click and drag/touch to scroll.
- **MusicBrainz Integration:** Automatic artist origin country lookup via MusicBrainz API with rate-limited background worker (1 req/sec).
- **Multi-language Title Sorting:** Intelligent initial calculation based on artist origin country:
  - Chinese (CN/TW/HK): Pinyin-based A-Z sorting
  - Japanese (JP): Gojūon (あかさたな) rows for JA UI, Romaji A-Z for EN/ZH UI
  - Latin/English: Standard A-Z sorting
- **Japanese Reading Support:** Integrated Kagome tokenizer (IPA dictionary) for accurate Japanese title readings (e.g., "青春" → "セイシュン" → "S/さ行").
- **Background Backfill:** Automatic migration for existing tabs to calculate initials and fetch origin countries on startup.

### Improved
- **Cloud Category Protection:** Tabs can no longer be manually added to the "Cloud Storage" category - only cloud-synced files appear there.

## [2.1.2] - 2026-02-27

### Added
- **A-Z Quick Jump Bar:** Added alphabet index navigation on the right side of the Singles view for quick scrolling to letter groups (similar to iOS contacts). Supports click and drag/touch to scroll.

## [2.1.1] - 2026-02-26

### Fixed
- **Singles Search:** Fixed critical bug where search functionality in Singles view was completely broken due to missing `is_cloud` column in FTS query
- **Cloud Category Protection:** Cloud tabs can no longer be manually removed from the "Cloud Storage" category - they are only removed automatically when downloaded to local storage

## [2.1.0] - 2026-02-26

### Added
- **MusicXML Support:** Added support for MusicXML score files (.xml, .musicxml, .mxl) in the viewer using alphaTab
- **Improved File Type Display:** Tab cards now show cleaner file type badges (PDF, XML, MXL, GP) instead of raw file extensions
- **Better Visual Feedback:** Newly added tabs are now highlighted and automatically scrolled into view for better user awareness

### Improved
- **In-Place Tab Operations:** Tab additions and deletions now update the UI in-place without full page refresh, preserving scroll position and improving UX
- **API Enhancement:** `SaveTab` now returns the saved tab object, enabling better frontend state management

### Fixed
- **Scroll Position Preservation:** Fixed issue where adding or deleting tabs would reset scroll position to the top
- **Event Handling:** Removed redundant event listeners that could cause unnecessary data refreshes

## [2.0.1] - 2026-02-26

### Improved
- **Download Progress Indicator:** Added a visual progress bar when loading Guitar Pro files, showing real-time download progress percentage

## [2.0.0] - 2026-02-26

### Added
- **WebDAV Integration:** Full WebDAV support for cloud storage sync with any standard WebDAV server (Nextcloud, ownCloud, etc.)
  - Browse and download files directly from your cloud library
  - Upload local tabs to WebDAV server with folder selection
  - Secure credential storage with encryption
- **Online Viewer for Cloud Files:** View WebDAV files directly in the internal viewer without downloading to local storage
- **Cloud Library Browser:** New modal for browsing, selecting, and downloading remote files

### Improved
- **Large File Handling:** Significantly improved loading performance for large files, especially when using Chinese cloud services (Baidu Netdisk, Alibaba Cloud, etc.)
- **UI Polish:** Various UI improvements and refinements across the application

### Fixed
- Fixed folder icon display issues in cloud browser
- Fixed WebDAV file adding and score loading issues
- Fixed status update and locale synchronization issues
- General stability improvements and bug fixes

## [1.5.6] - 2026-02-24

### Improved
- **Settings Auto-Save:** Settings now save automatically when changed, eliminating the need for a manual "Save" button. Key bindings save when closing the modal via "Done" button.

## [1.5.5] - 2026-02-24

### Fixed
- **PDF Viewer i18n:** Fixed PDF.js toolbar not following app language settings. The viewer now correctly uses the configured language instead of falling back to system locale.

## [1.5.4] - 2026-02-24

### Added
- **Internationalization (i18n):** Full multi-language support (English, Chinese, Japanese, etc.).
- **Language Settings:** Users can now switch interface language in the Settings menu.

### Improved
- **UI Localization:** Migrated hardcoded strings to localization keys across the application components (Modals, Settings, Context Menus).
- **Store Updates:** Updated `Settings` store and database models to persist user language preference.

## [1.5.3] - 2026-02-24

### Improved
- **Cover Image Loading:** Migrated cover image loading from Base64 IPC to HTTP streaming, reducing memory overhead and improving grid scroll performance

## [1.5.2] - 2026-02-24

### Improved
- **Library View UX:** Streamlined the UI based on view mode context:
  - "Upload TAB" and "Link Local TAB" buttons/menu items now only appear in Singles mode
  - "New Category" button/menu item now only appears in root Categories view (not inside a playlist)
  - Search filters are hidden in root Categories view for a cleaner interface
- **Category Search:** Added fuzzy search support for category names in Categories view

### Fixed
- **Search Scope Removal:** Removed the redundant "Range" (Inside Category / Global) filter from the search bar, simplifying the search interface

## [1.5.0] - 2026-02-22

### Fixed
- **Singles View Logic:** The "Singles" view now correctly displays *all* tabs in the library, regardless of whether they are in a category. Previously, it only showed uncategorized tabs, acting more like an "Unsorted" folder. This change aligns the view with its intended purpose as a complete "All Songs" list, while Categories function as playlists/tags.

## [1.4.17] - 2026-02-21

### Fixed
- **Edit Tab Modal:** Updated the metadata editing interface to use a single-column layout, resolving overlap issues and improving readability.
- **Modal Scrolling:** Added a vertical scrollbar to modal windows when content exceeds the viewport height.

## [1.4.10] - 2026-02-13

### Added
- **Auto-Scroll Feature:** New auto-scroll functionality for PDF viewer with adjustable speed control
  - Toggle auto-scroll with configurable key binding (default: `N`)
  - Adjust scroll speed with configurable key bindings (default: `,` to decrease, `.` to increase)
  - Native-looking toolbar integration with speed input and visual feedback
  - Automatically stops when reaching the end of the document

### Improved
- **Key Bindings:** Added three new configurable key bindings for auto-scroll controls (toggle, speed up, speed down)
- **Settings Persistence:** Auto-scroll key bindings are now saved to the database and persist across sessions

## [1.4.9] - 2026-02-12

### Improved
- **File Watcher:** Optimized internal debounce logic for file system events to improve stability and performance.

## [1.4.8] - 2026-02-12

### Added
- **Auto-Dodge Toolbar:** The floating toolbar in the Guitar Pro viewer now automatically fades out when scrolling or when idle for 3 seconds, reducing visual clutter.
- **SoundFont Loading Indicator:** Added a dedicated loading spinner that persists until the SoundFont is fully loaded, providing better feedback during initialization.

### Improved
- **SoundFont Optimization:** Switched to `sonivox.sf3` for potentially better performance and compatibility.
- **Viewer Initialization:** Refactored `GpViewer` event registration to occur before score loading, preventing race conditions with fast-loading files.

## [1.4.7] - 2026-02-12

### Improved
- **UI Refinement:** Unified modal styles for a more consistent and compact look across the application.
- **Sidebar Polish:** Improved sidebar item spacing, alignment, and hover transitions for a smoother feel.
- **Key Binding Modal:** Optimized the layout and spacing of the key binding settings for better readability and compactness.

## [1.4.6] - 2026-02-12

### Added
- **Shift+Drag Selection**: Hold `Shift` while dragging to enter "Section Playback Mode" (shows toolbar). Normal drag is now visual-only.
- **Section Playback Mode**:
  - Protects selection from accidental clicks (must use `Esc` or Clear button).
  - Right-click an active selection to enter this mode.
- **Menu Positioning**: Selection menu is now clamped to the viewport (won't go off-screen).

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
