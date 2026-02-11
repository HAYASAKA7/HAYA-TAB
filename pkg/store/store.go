package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type Tab struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Artist     string `json:"artist"`
	Album      string `json:"album"`
	FilePath   string `json:"filePath"` // Absolute path or relative to app
	Type       string `json:"type"`     // "pdf" or "gp"
	IsManaged  bool   `json:"isManaged"`
	CoverPath  string `json:"coverPath"`
	CategoryID string `json:"categoryId"` // Virtual folder ID
	Country    string `json:"country"`    // e.g. "US", "JP"
	Language   string `json:"language"`   // e.g. "ja_jp"
	Tag        string `json:"tag"`        // e.g. "Lead Guitar", "First Version"
}

type Category struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ParentID string `json:"parentId"` // Empty if root
}

type Settings struct {
	Theme             string   `json:"theme"`        // "dark", "light", "system"
	Background        string   `json:"background"`   // URL or path
	BgType            string   `json:"bgType"`       // "url", "local"
	OpenMethod        string   `json:"openMethod"`   // "system", "inner"
	OpenGpMethod      string   `json:"openGpMethod"` // "system", "inner"
	AudioDevice       string   `json:"audioDevice"`  // Device ID for audio output
	SyncPaths         []string `json:"syncPaths"`
	SyncStrategy      string   `json:"syncStrategy"` // "skip", "overwrite"
	AutoSyncEnabled   bool     `json:"autoSyncEnabled"`
	AutoSyncFrequency string   `json:"autoSyncFrequency"` // "startup", "weekly", "monthly", "yearly"
	LastSyncTime      int64    `json:"lastSyncTime"`      // Unix timestamp
}

// Deprecated: Use DBStore instead
type Store struct {
	mu         sync.Mutex
	Tabs       []Tab
	Categories []Category
	Settings   Settings
	DataPath   string
}

// Deprecated: Use DBStore instead
type PersistenceData struct {
	Tabs       []Tab      `json:"tabs"`
	Categories []Category `json:"categories"`
	Settings   Settings   `json:"settings"`
}

// Deprecated: Use NewDBStore instead
func NewStore(dataPath string) *Store {
	return &Store{
		DataPath:   dataPath,
		Tabs:       []Tab{},
		Categories: []Category{},
		Settings: Settings{
			Theme:             "system",
			OpenMethod:        "inner",
			OpenGpMethod:      "system",
			SyncStrategy:      "skip",
			SyncPaths:         []string{},
			AutoSyncEnabled:   false,
			AutoSyncFrequency: "startup",
			LastSyncTime:      0,
		},
	}
}

// Deprecated: Use DBStore instead
func (s *Store) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.DataPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	var pData PersistenceData
	if err := json.Unmarshal(data, &pData); err != nil {
		// Fallback for old data format (array of tabs)
		// Try unmarshalling as []Tab just in case
		var tabs []Tab
		if err2 := json.Unmarshal(data, &tabs); err2 == nil {
			s.Tabs = tabs
			return nil
		}
		return err
	}
	s.Tabs = pData.Tabs
	s.Categories = pData.Categories
	s.Settings = pData.Settings
	return nil
}

// Deprecated: Use DBStore instead
func (s *Store) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	pData := PersistenceData{
		Tabs:       s.Tabs,
		Categories: s.Categories,
		Settings:   s.Settings,
	}

	data, err := json.MarshalIndent(pData, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(s.DataPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(s.DataPath, data, 0644)
}

func (s *Store) UpdateSettings(settings Settings) error {
	s.mu.Lock()
	s.Settings = settings
	s.mu.Unlock()
	return s.Save()
}

func (s *Store) AddTab(tab Tab) error {
	s.mu.Lock()
	// Check if ID exists, if so update
	found := false
	for i, t := range s.Tabs {
		if t.ID == tab.ID {
			s.Tabs[i] = tab
			found = true
			break
		}
	}
	if !found {
		s.Tabs = append(s.Tabs, tab)
	}
	s.mu.Unlock() // Save acquires lock, so unlock first
	return s.Save()
}

func (s *Store) AddCategory(cat Category) error {
	s.mu.Lock()
	found := false
	for i, c := range s.Categories {
		if c.ID == cat.ID {
			s.Categories[i] = cat
			found = true
			break
		}
	}
	if !found {
		s.Categories = append(s.Categories, cat)
	}
	s.mu.Unlock()
	return s.Save()
}

func (s *Store) DeleteCategory(id string) error {
	s.mu.Lock()
	// Filter out the category
	newCats := []Category{}
	for _, c := range s.Categories {
		if c.ID != id {
			newCats = append(newCats, c)
		}
	}
	s.Categories = newCats

	// Optional: Move children tabs to root or parent?
	// For simplicity, let's move tabs in this category to root ("")
	for i := range s.Tabs {
		if s.Tabs[i].CategoryID == id {
			s.Tabs[i].CategoryID = ""
		}
	}
	// Note: We are not recursively deleting sub-categories here for simplicity,
	// but strictly speaking we should.
	// Let's also move sub-categories to root.
	for i := range s.Categories {
		if s.Categories[i].ParentID == id {
			s.Categories[i].ParentID = ""
		}
	}

	s.mu.Unlock()
	return s.Save()
}

func (s *Store) MoveCategory(id, newParentID string) error {
	s.mu.Lock()
	found := false
	for i, c := range s.Categories {
		if c.ID == id {
			s.Categories[i].ParentID = newParentID
			found = true
			break
		}
	}
	s.mu.Unlock()
	if !found {
		return os.ErrNotExist // Or custom error
	}
	return s.Save()
}

func (s *Store) DeleteTab(id string) error {
	s.mu.Lock()
	newTabs := []Tab{}
	for _, t := range s.Tabs {
		if t.ID != id {
			newTabs = append(newTabs, t)
		}
	}
	s.Tabs = newTabs
	s.mu.Unlock()
	return s.Save()
}
