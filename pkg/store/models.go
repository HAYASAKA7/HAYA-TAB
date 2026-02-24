package store

// Tab represents a guitar tab file and its metadata
type Tab struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Artist      string   `json:"artist"`
	Album       string   `json:"album"`
	FilePath    string   `json:"filePath"`    // Absolute path or relative to app
	Type        string   `json:"type"`        // "pdf" or "gp"
	IsManaged   bool     `json:"isManaged"`
	CoverPath   string   `json:"coverPath"`
	CategoryIDs []string `json:"categoryIds"` // List of Category IDs
	Country     string   `json:"country"`     // e.g. "US", "JP"
	Language    string   `json:"language"`    // e.g. "ja_jp"
	Tag         string   `json:"tag"`         // e.g. "Lead Guitar", "First Version"
	AddedAt     int64    `json:"addedAt"`     // Unix timestamp
	LastOpened  int64    `json:"lastOpened"`  // Unix timestamp
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
}
