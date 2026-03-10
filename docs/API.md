# HAYA-TAB Go API Documentation

> Generated from Go code using `go doc` command. Updated for v2.3.6

## Package: haya-tab/internal/app

```go
package app // import "haya-tab/internal/app"

VARIABLES

var AppVersion = "2.3.6"
    AppVersion is the application version. Can be set via ldflags during build: 
    -ldflags "-X haya-tab/internal/app.AppVersion=2.3.6"
FUNCTIONS

func GetDiskFreeSpace(path string) (uint64, error)
    GetDiskFreeSpace returns the free space of the drive containing the path in
    bytes.


TYPES

type App struct {
	// Has unexported fields.
}
    App struct holds all application dependencies and state

func NewApp() *App
    NewApp creates a new App application struct

func (a *App) AddCategory(cat store.Category) error
    AddCategory adds a new category

func (a *App) AddTabToCategory(tabID, categoryID string) error
    AddTabToCategory adds a tab to a category without removing it from others

func (a *App) BatchAddTabsToCategory(ids []string, categoryID string) (int, error)
    BatchAddTabsToCategory adds multiple tabs to a category

func (a *App) BatchDeleteTabs(ids []string) (int, error)
    BatchDeleteTabs deletes multiple tabs at once

func (a *App) BatchMoveTabs(ids []string, categoryID string) (int, error)
    BatchMoveTabs moves multiple tabs to a category at once (replaces existing
    categories)

func (a *App) CheckMigration(target string) (MigrationStatus, error)
    CheckMigration checks if a directory needs migration and returns the number
    of files and total size

func (a *App) DeleteCategory(id string) error
    DeleteCategory deletes a category

func (a *App) DeleteTab(id string) error
    DeleteTab deletes a tab and its managed file if applicable

func (a *App) DownloadCloudTabToLocal(tabID string) error
    DownloadCloudTabToLocal downloads a cloud tab to local storage IMPORTANT:
    This preserves existing metadata - does NOT re-parse the file

func (a *App) ExportTab(id string, destFolder string) error
    ExportTab copies the tab file to a destination folder

func (a *App) GetCategories() []store.Category
    GetCategories returns the list of categories

func (a *App) GetCover(path string) string
    GetCover returns the base64 encoded image

func (a *App) GetCoversDir() string
    GetCoversDir returns the directory for cover images.

func (a *App) GetFileServerPort() int
    GetFileServerPort returns the port of the local file server

func (a *App) GetRecentCategories(limit int) []store.Category
    GetRecentCategories returns the list of recently accessed categories

func (a *App) GetRecentTabs(limit int) []store.Tab
    GetRecentTabs returns the list of recently accessed tabs

func (a *App) GetSettings() store.Settings
    GetSettings returns the current settings

func (a *App) GetStorageDir() string
    GetStorageDir returns the directory for managed tabs.

func (a *App) GetTabs() []store.Tab
    GetTabs returns the list of tabs (backward compatibility)

func (a *App) GetTabsPaginated(categoryId string, page, pageSize int, searchQuery string, filterBy []string, isGlobal bool, sortBy string, sortDesc bool) TabsResponse
    GetTabsPaginated returns a paginated list of tabs with optional search

func (a *App) MarkAsOpened(id string) error
    MarkAsOpened updates the LastOpened timestamp for a tab without opening it

func (a *App) MigrateData(target string, newPath string, copyOnly bool) error
    MigrateData migrates data from the current directory to the new path

func (a *App) MoveCategory(id, newParentID string) error
    MoveCategory moves a category into another category

func (a *App) MoveTab(tabID, categoryID string) error
    MoveTab updates the category of a tab (replaces existing categories with
    this one)

func (a *App) OpenTab(id string) error
    OpenTab opens the file using system default

func (a *App) ProcessFile(path string) store.Tab
    ProcessFile delegates to SyncService for file processing

func (a *App) RecalculateAllInitials() (int, error)
    RecalculateAllInitials forces recalculation of initials for all tabs

func (a *App) RemoveTabFromCategory(tabID, categoryID string) error
    RemoveTabFromCategory removes a tab from a category

func (a *App) ResolveCoverPath(path string) string
    ResolveCoverPath converts a relative path to absolute using GetCoversDir.

func (a *App) ResolveTabPath(path string, isManaged bool) string
    ResolveTabPath converts a relative path to absolute using GetStorageDir.

func (a *App) SaveSettings(s store.Settings) error
    SaveSettings updates the settings

func (a *App) SaveTab(tab store.Tab, shouldCopy bool) (*store.Tab, error)
    SaveTab saves the tab. copyFile determines if we import it to internal
    storage. The passed tab should have the user-confirmed Metadata. Returns the
    saved tab on success.

func (a *App) SelectFiles() []string
    SelectFiles opens a file dialog and returns the selected file paths

func (a *App) SelectFolder() string
    SelectFolder opens a folder selection dialog

func (a *App) SelectImage() string
    SelectImage opens a file dialog for selecting images

func (a *App) SetFileServerPort(port int)
    SetFileServerPort sets the port of the local file server

func (a *App) Shutdown(ctx context.Context)
    Shutdown is called when the app is closing

func (a *App) StartFileServer() (int, error)
    StartFileServer starts a local HTTP server to serve files

func (a *App) Startup(ctx context.Context)
    Startup is called when the app starts. The context is saved so we can call
    the runtime methods

func (a *App) TriggerSync() (string, error)
    TriggerSync delegates to SyncService for file synchronization

func (a *App) UpdateTab(tab store.Tab) error
    UpdateTab updates an existing tab's metadata

func (a *App) UpdateTabCategories(tabID string, categoryIDs []string) error
    UpdateTabCategories updates the categories for a tab

func (a *App) UpdateTabMetadata(id string, title string, artist string, album string) error
    UpdateTabMetadata updates only the metadata fields (title, artist, album)
    for a tab. This is called by the frontend after AlphaTab parses the file's
    internal metadata. It implements a "smart update" strategy: - If no cover
    exists: prefer AlphaTab's data (more authoritative than filename parsing)
    - If cover exists: only update placeholder fields (existing data was good
    enough for cover search)

func (a *App) WebDAVAddOnlineFiles(url, user, password string, remotePaths []string) error
    WebDAVAddOnlineFiles adds cloud files to library without downloading (lazy
    loading)

func (a *App) WebDAVCheckStatus() bool
    WebDAVCheckStatus checks if WebDAV connection is available

func (a *App) WebDAVCheckVolumeHealth() (map[string]bool, error)
    WebDAVCheckVolumeHealth checks the health of all registered volumes

func (a *App) WebDAVCleanupOrphanedTabs() (int, error)
    WebDAVCleanupOrphanedTabs removes tabs that reference non-existent volumes

func (a *App) WebDAVCreateVolume(volumeName, remotePath string) (*store.CloudVolume, error)
    WebDAVCreateVolume creates a new volume with a fingerprint file

func (a *App) WebDAVDiscoverVolumes() ([]store.CloudVolume, error)
    WebDAVDiscoverVolumes scans WebDAV for all volumes and registers them.
    This is the entry point for multi-device sync.

func (a *App) WebDAVDownloadFiles(url, user, password string, remotePaths []string) error
    WebDAVDownloadFiles downloads selected files and processes them

func (a *App) WebDAVGetOrphanedTabsCount() (int, error)
    WebDAVGetOrphanedTabsCount returns the count of tabs referencing
    non-existent volumes

func (a *App) WebDAVInitialize() error
    WebDAVInitialize initializes the WebDAV volume system. This should be called
    on app startup if WebDAV is enabled.

func (a *App) WebDAVListDir(url, user, password, dir string) ([]store.RemoteFile, error)
    WebDAVListDir lists files and directories in a remote path (non-recursive)

func (a *App) WebDAVListRemoteDirectories(url, user, password, dir string) ([]string, error)
    WebDAVListRemoteDirectories lists directories in a remote path

func (a *App) WebDAVMigrateCloudTabs() error
    WebDAVMigrateCloudTabs migrates existing cloud tabs to use the volume system

func (a *App) WebDAVReconnect() error
    WebDAVReconnect attempts to reconnect and reinitialize WebDAV. This should be
    called when connection is restored after being lost.

func (a *App) WebDAVScanRemoteFiles(url, user, password, dir string) ([]store.RemoteFile, error)
    WebDAVScanRemoteFiles scans a remote directory

func (a *App) WebDAVTestConnection(url, user, password string) error
    WebDAVTestConnection tests the WebDAV connection

func (a *App) WebDAVUploadFiles(url, user, password string, localPaths []string, remoteDir string) error
    WebDAVUploadFiles uploads local files to a remote directory

type FileHandler struct {
	// Has unexported fields.
}
    FileHandler handles HTTP requests for streaming files

func NewFileHandler(app *App) *FileHandler
    NewFileHandler creates a new file handler

func (h *FileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request)
    ServeHTTP implements http.Handler for streaming files

type MigrationStatus struct {
	Count int   `json:"count"`
	Size  int64 `json:"size"`
}
    MigrationStatus holds information about a directory migration

type TabsResponse struct {
	Tabs     []store.Tab `json:"tabs"`
	Total    int         `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
	HasMore  bool        `json:"hasMore"`
}
    TabsResponse represents a paginated response for tabs

type WailsEventEmitter struct {
	// Has unexported fields.
}
    WailsEventEmitter adapts wails runtime to the EventEmitter interface

func (e *WailsEventEmitter) Emit(eventName string, data interface{})
    Emit sends an event to the frontend via wails runtime

```

## Package: haya-tab/pkg/coverpool

```go
package coverpool // import "haya-tab/pkg/coverpool"


TYPES

type CoverJob struct {
	TabID      string
	Artist     string
	Album      string
	Title      string
	Country    string
	Language   string
	CoverPath  string
	OnComplete func(tabID, coverPath string, err error)
}
    CoverJob represents a cover download task

type CoverPool struct {
	// Has unexported fields.
}
    CoverPool manages concurrent cover download workers

func NewCoverPool(workers int, downloadFn func(artist, album, title, country, lang, dstPath string) error) *CoverPool
    NewCoverPool creates a new worker pool with the specified number of workers

func (p *CoverPool) QueueSize() int
    QueueSize returns the current number of pending jobs

func (p *CoverPool) Start()
    Start launches the worker goroutines

func (p *CoverPool) Stop()
    Stop gracefully shuts down the worker pool

func (p *CoverPool) Submit(job CoverJob)
    Submit adds a new job to the queue

func (p *CoverPool) SubmitAsync(job CoverJob) bool
    SubmitAsync adds a job without blocking (drops if queue is full)

```

## Package: haya-tab/pkg/logger

```go
package logger // import "haya-tab/pkg/logger"


TYPES

type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelError
)
type Logger struct {
	// Has unexported fields.
}

func NewLogger(appDir string) *Logger

func (l *Logger) Close()

func (l *Logger) Debug(format string, args ...interface{})

func (l *Logger) Error(format string, args ...interface{})

func (l *Logger) Info(format string, args ...interface{})

func (l *Logger) SetContext(ctx context.Context)

```

## Package: haya-tab/pkg/metadata

```go
package metadata // import "haya-tab/pkg/metadata"

Package metadata provides music metadata parsing and fetching utilities.

FUNCTIONS

func CalculateInitials(title string, originCountry string) (az string, kana string)
    CalculateInitials computes both A-Z and Kana initials for a tab title based
    on the artist's origin country (from MusicBrainz).

    Returns:
      - az: A-Z initial for EN/ZH UI (Pinyin/Romaji mapped to A-Z)
      - kana: Kana initial for JA UI (銇傘亱銇曘仧銇?.. or A-Z for Latin)

    Logic:
      - Chinese (CN/TW/HK): az = Pinyin uppercase, kana = "#"
      - Japanese (JP): az = Romaji uppercase, kana = Goj奴on row
      - Latin/English: az = uppercase A-Z, kana = same uppercase A-Z
      - Fallback: Heuristic detection or "#" for both

func DownloadCover(artist, album, title, country, lang, dstPath string) error
    DownloadCover searches iTunes and saves the cover to dstPath. Falls back to
    US/en_us if specific country/lang returns no results.


TYPES

type ItunesResponse struct {
	ResultCount int `json:"resultCount"`
	Results     []struct {
		ArtworkUrl100 string `json:"artworkUrl100"`
	} `json:"results"`
}

type Metadata struct {
	Title  string `json:"title"`
	Artist string `json:"artist"`
	Album  string `json:"album"`
}

func ParseFile(path string) (Metadata, error)
    ParseFile extracts metadata from the filename only. Binary parsing has
    been removed for stability and performance. The frontend (AlphaTab) handles
    accurate metadata extraction and writes it back.

func ParseFilename(filename string) Metadata
    ParseFilename attempts to extract Artist - Album - Song from filename
    Enhanced with multiple pattern recognition: 1. "Artist - Album - Title.ext"
    2. "Artist - Title.ext" 3. "01. Artist - Title.ext" (with track number) 4.
    "[Artist] Title.ext" (bracket format) 5. "Artist - Title (Key).ext" (with
    key signature) 6. "Title.ext" (fallback)

type MusicBrainzArtistInfo struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	SortName       string `json:"sort-name"`
	Country        string `json:"country"`
	Type           string `json:"type"`
	Score          int    `json:"score"`
	Disambiguation string `json:"disambiguation"`
}
    MusicBrainzArtistInfo represents an artist entry from MusicBrainz

type MusicBrainzArtistResponse struct {
	Created string                  `json:"created"`
	Count   int                     `json:"count"`
	Offset  int                     `json:"offset"`
	Artists []MusicBrainzArtistInfo `json:"artists"`
}
    MusicBrainzArtistResponse represents the response from MusicBrainz artist
    search

type MusicBrainzClient struct {
	// Has unexported fields.
}
    MusicBrainzClient handles requests to the MusicBrainz API

func NewMusicBrainzClient() *MusicBrainzClient
    NewMusicBrainzClient creates a new MusicBrainz API client

func (c *MusicBrainzClient) SearchArtistCountry(artistName string) (string, error)
    SearchArtistCountry searches for an artist and returns their origin country
    code Returns empty string if not found or on error

```

## Package: haya-tab/pkg/store

```go
package store // import "haya-tab/pkg/store"


CONSTANTS

const (
	SystemCloudCategoryID = "sys_cloud"
)
    System category IDs


FUNCTIONS

func Decrypt(cryptoText string) (string, error)
func DetectSystemLocale() string
    DetectSystemLocale detects the OS language and maps it to a supported app
    locale. Returns "en" if the system language is not in the supported set.

func Encrypt(text string) (string, error)
func MigrateFromJSON(s *DBStore, jsonPath string) error
    MigrateFromJSON migrates data from old JSON file to database


TYPES

type Category struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	ParentID           string `json:"parentId"`           // Empty if root
	CoverPath          string `json:"coverPath"`          // Custom cover path (raw)
	EffectiveCoverPath string `json:"effectiveCoverPath"` // Derived or custom
}
    Category represents a grouping of tabs

type CloudVolume struct {
	ID              string `json:"id"`              // UUID generated on first scan
	Name            string `json:"name"`            // User-friendly name (e.g., "Google Drive", "OneDrive")
	MountPath       string `json:"mountPath"`       // Current mount path in WebDAV (e.g., "/gdrive")
	FingerprintPath string `json:"fingerprintPath"` // Path to fingerprint file (e.g., "/gdrive/.haya-volume-fingerprint")
	CreatedAt       int64  `json:"createdAt"`       // Unix timestamp when volume was first registered
	LastSeenAt      int64  `json:"lastSeenAt"`      // Unix timestamp when volume was last detected
	IsAvailable     bool   `json:"isAvailable"`     // True if volume is currently accessible
}
    CloudVolume represents a virtual volume (physical cloud drive) mounted via
    WebDAV

type DBStore struct {
	Settings Settings
	// Has unexported fields.
}

func NewDBStore(dbPath string) *DBStore

func (s *DBStore) AddCategory(cat Category) error

func (s *DBStore) AddTab(tab Tab) error

func (s *DBStore) AddVolume(volume CloudVolume) error
    AddVolume adds a new cloud volume to the database

func (s *DBStore) CleanupOrphanedTabs() (int, error)
    CleanupOrphanedTabs removes tabs that reference non-existent volumes

func (s *DBStore) Close() error
    Close closes the database connection

func (s *DBStore) DeleteCategory(id string) error

func (s *DBStore) DeleteTab(id string) error

func (s *DBStore) DeleteVolume(volumeID string) error
    DeleteVolume deletes a volume and all associated tabs

func (s *DBStore) EnsureCloudCategory() error
    EnsureCloudCategory creates or updates the system cloud category

func (s *DBStore) EnsureCloudTabsHaveCloudCategory() (int, error)
    EnsureCloudTabsHaveCloudCategory adds the syscloud category to all cloud
    tabs that don't have it. This is a migration for tabs created before the
    automatic category assignment was implemented.

func (s *DBStore) GetAllTabs() ([]Tab, error)
    GetAllTabs is an alias for GetTabs for backward compatibility

func (s *DBStore) GetAllVolumes() ([]CloudVolume, error)
    GetAllVolumes retrieves all cloud volumes

func (s *DBStore) GetCategories() ([]Category, error)

func (s *DBStore) GetCategory(id string) (*Category, error)
    GetCategory retrieves a single category by ID

func (s *DBStore) GetOrphanedTabsCount() (int, error)
    GetOrphanedTabsCount returns the count of tabs referencing non-existent
    volumes

func (s *DBStore) GetRecentCategories(limit int) ([]Category, error)

func (s *DBStore) GetRecentTabs(limit int) ([]Tab, error)

func (s *DBStore) GetSettings() Settings

func (s *DBStore) GetTab(id string) (*Tab, error)

func (s *DBStore) GetTabByPath(filePath string) (*Tab, error)

func (s *DBStore) GetTabByTitle(title string) (*Tab, error)

func (s *DBStore) GetTabByVolumeAndPath(volumeID, relativePath string) (*Tab, error)
    GetTabByVolumeAndPath retrieves a tab by volume ID and relative path

func (s *DBStore) GetTabs() ([]Tab, error)

func (s *DBStore) GetTabsByVolume(volumeID string) ([]Tab, error)
    GetTabsByVolume retrieves all tabs for a specific volume

func (s *DBStore) GetTabsNeedingInitials() ([]Tab, error)
    GetTabsNeedingInitials returns tabs that need initial_az/initial_kana
    backfill. Used for background backfill of legacy data.

func (s *DBStore) GetTabsNeedingOriginCountry() ([]Tab, error)
    GetTabsNeedingOriginCountry returns tabs that have a cover but no
    origin_country set. Used for background backfill of MusicBrainz data.

func (s *DBStore) GetTabsPaginated(categoryId string, page, pageSize int, searchQuery string, filterBy []string, isGlobal bool, sortBy string, sortDesc bool) ([]Tab, int, error)

func (s *DBStore) GetVolume(id string) (*CloudVolume, error)
    GetVolume retrieves a volume by ID

func (s *DBStore) GetVolumeByMountPath(mountPath string) (*CloudVolume, error)
    GetVolumeByMountPath retrieves a volume by its mount path

func (s *DBStore) HasData() bool
    HasData checks if the database has any data

func (s *DBStore) Initialize() error
    Initialize creates the database and tables

func (s *DBStore) MarkVolumeAvailable(volumeID string, available bool) error
    MarkVolumeAvailable marks a volume as available or unavailable

func (s *DBStore) MigrateCloudTabsToVolumes() error
    MigrateCloudTabsToVolumes migrates existing cloud tabs to use the volume
    system. This is for backward compatibility with tabs created before the
    volume system was implemented.

func (s *DBStore) MoveCategory(id, newParentID string) error

func (s *DBStore) SearchTabs(query string) ([]Tab, error)
    SearchTabs performs a simple search across title, artist, album, and tag
    fields

func (s *DBStore) SetTabCategories(id string, categoryIDs []string, addedAt int64) error

func (s *DBStore) UpdateLastOpened(tabID string) error
    UpdateLastOpened updates the last_opened timestamp for a tab

func (s *DBStore) UpdateSettings(settings Settings) error

func (s *DBStore) UpdateTab(tab Tab) error

func (s *DBStore) UpdateTabInitials(tabID, initialAZ, initialKana string) error
    UpdateTabInitials updates only the initial_az and initial_kana fields for a
    tab

func (s *DBStore) UpdateTabOriginCountry(tabID, originCountry string) error
    UpdateTabOriginCountry updates only the origin_country field for a tab

func (s *DBStore) UpdateVolume(volume CloudVolume) error
    UpdateVolume updates an existing volume

func (s *DBStore) UpdateVolumeMountPath(volumeID, newMountPath string) error
    UpdateVolumeMountPath updates the mount path of a volume (when WebDAV root
    changes)

type KeyBindings struct {
	ScrollDown      string `json:"scrollDown"`
	ScrollUp        string `json:"scrollUp"`
	Metronome       string `json:"metronome"`
	PlayPause       string `json:"playPause"`
	Stop            string `json:"stop"`
	BpmPlus         string `json:"bpmPlus"`
	BpmMinus        string `json:"bpmMinus"`
	ToggleLoop      string `json:"toggleLoop"`
	ClearSelection  string `json:"clearSelection"`
	JumpToBar       string `json:"jumpToBar"`
	JumpToStart     string `json:"jumpToStart"`
	AutoScroll      string `json:"autoScroll"`
	ScrollSpeedUp   string `json:"scrollSpeedUp"`
	ScrollSpeedDown string `json:"scrollSpeedDown"`
}
    KeyBindings represents user-configurable keyboard shortcuts

type RemoteFile struct {
	Name  string `json:"name"`
	Path  string `json:"path"` // Full remote path
	Size  int64  `json:"size"`
	IsDir bool   `json:"isDir"`
}

type Settings struct {
	Theme             string      `json:"theme"`        // "dark", "light", "system"
	Language          string      `json:"language"`     // "en", "zh-CN", "zh-TW", "ja"
	Background        string      `json:"background"`   // URL or path
	BgType            string      `json:"bgType"`       // "url", "local"
	OpenMethod        string      `json:"openMethod"`   // "system", "inner"
	OpenGpMethod      string      `json:"openGpMethod"` // "system", "inner"
	AudioDevice       string      `json:"audioDevice"`  // Device ID for audio output
	SyncPaths         []string    `json:"syncPaths"`
	SyncStrategy      string      `json:"syncStrategy"` // "skip", "overwrite"
	AutoSyncEnabled   bool        `json:"autoSyncEnabled"`
	AutoSyncFrequency string      `json:"autoSyncFrequency"` // "startup", "weekly", "monthly", "yearly"
	LastSyncTime      int64       `json:"lastSyncTime"`      // Unix timestamp
	KeyBindings       KeyBindings `json:"keyBindings"`
	StoragePath       string      `json:"storagePath"` // Custom storage path
	CoversPath        string      `json:"coversPath"`  // Custom covers path

	// WebDAV Settings
	WebDAVEnabled  bool   `json:"webdavEnabled"`
	WebDAVURL      string `json:"webdavUrl"`
	WebDAVUser     string `json:"webdavUser"`
	WebDAVPassword string `json:"webdavPassword"`
}
    Settings represents application configuration

type Tab struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Artist        string   `json:"artist"`
	Album         string   `json:"album"`
	FilePath      string   `json:"filePath"`      // For local: absolute path. For cloud: relative path within volume
	VolumeID      string   `json:"volumeId"`      // Cloud volume ID (empty for local files)
	Type          string   `json:"type"`          // "pdf" or "gp"
	IsManaged     bool     `json:"isManaged"`
	IsCloud       bool     `json:"isCloud"`       // True if this is a cloud/online tab (not downloaded)
	CoverPath     string   `json:"coverPath"`
	CategoryIDs   []string `json:"categoryIds"`   // List of Category IDs
	Country       string   `json:"country"`       // e.g. "US", "JP" (user's preferred search country)
	Language      string   `json:"language"`      // e.g. "ja_jp"
	OriginCountry string   `json:"originCountry"` // Artist's origin country from MusicBrainz (e.g. "JP", "CN", "US")
	Tag           string   `json:"tag"`           // e.g. "Lead Guitar", "First Version"
	AddedAt       int64    `json:"addedAt"`       // Unix timestamp
	LastOpened    int64    `json:"lastOpened"`    // Unix timestamp
	InitialAZ     string   `json:"initialAz"`     // A-Z initial for EN/ZH UI (Pinyin/Romaji mapped to A-Z)
	InitialKana   string   `json:"initialKana"`   // Kana initial for JA UI (銇傘亱銇曘仧銇?.. or A-Z for Latin)
}
    Tab represents a guitar tab file and its metadata

```

## Package: haya-tab/pkg/sync

```go
package sync // import "haya-tab/pkg/sync"

Package sync provides file synchronization services for HAYA-TAB. It handles
scanning directories, processing files, and managing tab entries.

CONSTANTS

const BrowserUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
    BrowserUserAgent is a Chrome-like User-Agent to avoid 403 blocks from WebDAV
    servers


TYPES

type EventEmitter interface {
	Emit(eventName string, data interface{})
}
    EventEmitter is an abstraction for emitting events to the frontend. This
    allows SyncService to be decoupled from wails runtime.

type SyncResult struct {
	Added   int
	Updated int
	Skipped int
	Errors  int
	Total   int
}
    SyncResult contains the results of a sync operation

type SyncService struct {
	// Has unexported fields.
}
    SyncService handles file synchronization operations

func NewSyncService(
	store *store.DBStore,
	logger *logger.Logger,
	coverPool *coverpool.CoverPool,
	emitter EventEmitter,
	appDir string,
	mbWorker *worker.MBWorker,
) *SyncService
    NewSyncService creates a new SyncService instance

func (s *SyncService) FetchCoverAsync(tab store.Tab)
    FetchCoverAsync downloads album cover art asynchronously for a tab using
    worker pool

func (s *SyncService) ProcessFile(path string) store.Tab
    ProcessFile takes a file path and returns a pre-filled Tab struct

func (s *SyncService) TriggerSync() (string, error)
    TriggerSync scans configured sync paths and adds/updates tabs based on
    strategy

type WebDAVClient struct {
	// Has unexported fields.
}

func NewWebDAVClient(serverURL, user, password string) *WebDAVClient

func (c *WebDAVClient) DownloadFile(remotePath, localPath string) error
    DownloadFile downloads a single file to the local destination

func (c *WebDAVClient) GetFileInfo(remotePath string) (os.FileInfo, error)
    GetFileInfo returns file info for a remote path

func (c *WebDAVClient) GetHTTPClient() *http.Client
    GetHTTPClient returns the underlying HTTP client for advanced operations

func (c *WebDAVClient) GetURL() string
    GetURL returns the base WebDAV URL

func (c *WebDAVClient) ListDir(dir string) ([]store.RemoteFile, error)
    ListDir returns a list of files and directories in the given path
    (non-recursive)

func (c *WebDAVClient) ListRemoteDirectories(dir string) ([]string, error)
    ListRemoteDirectories returns a list of directories in the given path
    (non-recursive)

func (c *WebDAVClient) ReadStream(remotePath string) (io.ReadCloser, error)
    ReadStream returns a read stream for the remote file (for streaming/proxy)

func (c *WebDAVClient) ScanRemoteFiles(dir string) ([]store.RemoteFile, error)
    ScanRemoteFiles recursively scans the remote directory for supported files

func (c *WebDAVClient) TestConnection() error

func (c *WebDAVClient) UploadFile(localPath, remoteDir string) error
    UploadFile uploads a single file to the remote directory

```

## Package: haya-tab/pkg/watcher

```go
package watcher // import "haya-tab/pkg/watcher"


TYPES

type FileWatcher struct {
	// Has unexported fields.
}
    FileWatcher watches directories for file changes

func NewFileWatcher(onChange func()) *FileWatcher
    NewFileWatcher creates a new file watcher

func (w *FileWatcher) AddPath(path string) error
    AddPath adds a path to watch

func (w *FileWatcher) GetPaths() []string
    GetPaths returns the currently watched paths

func (w *FileWatcher) IsRunning() bool
    IsRunning returns whether the watcher is running

func (w *FileWatcher) RemovePath(path string) error
    RemovePath removes a path from watching

func (w *FileWatcher) SetLogger(l Logger)
    SetLogger sets the logger

func (w *FileWatcher) SetPaths(paths []string) error
    SetPaths sets all paths to watch (replaces existing)

func (w *FileWatcher) Start() error
    Start initializes and starts the file watcher

func (w *FileWatcher) Stop()
    Stop stops the file watcher

type Logger interface {
	Info(format string, args ...interface{})
	Error(format string, args ...interface{})
}
    Logger interface for dependency injection

```

## Package: haya-tab/pkg/worker

```go
package worker // import "haya-tab/pkg/worker"

Package worker provides background job processing utilities.

TYPES

type Logger interface {
	Info(format string, args ...interface{})
	Error(format string, args ...interface{})
}
    Logger interface for logging (to avoid circular dependency with logger
    package)

type MBJob struct {
	TabID      string
	ArtistName string
}
    MBJob represents a job to fetch artist origin country from MusicBrainz

type MBWorker struct {
	// Has unexported fields.
}
    MBWorker is a single-threaded worker that processes MusicBrainz requests
    with strict rate limiting (1 request per second) to comply with API rules

func NewMBWorker(store *store.DBStore, logger Logger) *MBWorker
    NewMBWorker creates a new MusicBrainz worker

func (w *MBWorker) QueueSize() int
    QueueSize returns the current number of jobs in the queue

func (w *MBWorker) Start()
    Start begins the worker goroutine

func (w *MBWorker) Stop()
    Stop gracefully shuts down the worker

func (w *MBWorker) Submit(job MBJob)
    Submit adds a job to the queue (non-blocking, drops if queue is full)

```

