package store

// System category IDs
const (
	SystemCloudCategoryID = "sys_cloud"
)

// CloudVolume represents a virtual volume (physical cloud drive) mounted via WebDAV
type CloudVolume struct {
	ID              string `json:"id"`              // UUID generated on first scan
	Name            string `json:"name"`            // User-friendly name (e.g., "Google Drive", "OneDrive")
	MountPath       string `json:"mountPath"`       // Current mount path in WebDAV (e.g., "/gdrive")
	FingerprintPath string `json:"fingerprintPath"` // Path to fingerprint file (e.g., "/gdrive/.haya-volume-fingerprint")
	CreatedAt       int64  `json:"createdAt"`       // Unix timestamp when volume was first registered
	LastSeenAt      int64  `json:"lastSeenAt"`      // Unix timestamp when volume was last detected
	IsAvailable     bool   `json:"isAvailable"`     // True if volume is currently accessible
}

// Tab represents a guitar tab file and its metadata
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
	InitialKana   string   `json:"initialKana"`   // Kana initial for JA UI (あかさたな... or A-Z for Latin)
}

// Category represents a grouping of tabs
type Category struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	ParentID           string `json:"parentId"`           // Empty if root
	CoverPath          string `json:"coverPath"`          // Custom cover path (raw)
	EffectiveCoverPath string `json:"effectiveCoverPath"` // Derived or custom
}

// KeyBindings represents user-configurable keyboard shortcuts
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

// Settings represents application configuration
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
	StoragePath       string      `json:"storagePath"`       // Custom storage path
	CoversPath        string      `json:"coversPath"`        // Custom covers path

	// WebDAV Settings
	WebDAVEnabled  bool   `json:"webdavEnabled"`
	WebDAVURL      string `json:"webdavUrl"`
	WebDAVUser     string `json:"webdavUser"`
	WebDAVPassword string `json:"webdavPassword"`
}

type RemoteFile struct {
	Name string `json:"name"`
	Path string `json:"path"` // Full remote path
	Size int64  `json:"size"`
	IsDir bool  `json:"isDir"`
}

