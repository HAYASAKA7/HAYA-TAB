package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type Tab struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Artist    string `json:"artist"`
	Album     string `json:"album"`
	FilePath  string `json:"filePath"` // Absolute path or relative to app
	Type      string `json:"type"`     // "pdf" or "gp"
	IsManaged bool   `json:"isManaged"`
	CoverPath string `json:"coverPath"`
}

type Store struct {
	mu       sync.Mutex
	Tabs     []Tab
	DataPath string
}

func NewStore(dataPath string) *Store {
	return &Store{
		DataPath: dataPath,
		Tabs:     []Tab{},
	}
}

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

	return json.Unmarshal(data, &s.Tabs)
}

func (s *Store) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(s.Tabs, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(s.DataPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(s.DataPath, data, 0644)
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
