package store

import (
	"encoding/json"
	"fmt"
	"os"
)

// MigrateFromJSON migrates data from old JSON file to database
func MigrateFromJSON(s *DBStore, jsonPath string) error {
	// Check if JSON file exists
	data, err := os.ReadFile(jsonPath)
	if os.IsNotExist(err) {
		return nil // No migration needed
	}
	if err != nil {
		return fmt.Errorf("failed to read JSON file: %w", err)
	}

	// Try to parse as PersistenceData
	var pData struct {
		Tabs       []Tab      `json:"tabs"`
		Categories []Category `json:"categories"`
		Settings   Settings   `json:"settings"`
	}

	if err := json.Unmarshal(data, &pData); err != nil {
		// Fallback for old data format (array of tabs)
		var tabs []Tab
		if err2 := json.Unmarshal(data, &tabs); err2 == nil {
			pData.Tabs = tabs
		} else {
			return fmt.Errorf("failed to parse JSON data: %w", err)
		}
	}

	// Migrate tabs
	for _, tab := range pData.Tabs {
		if err := s.AddTab(tab); err != nil {
			return fmt.Errorf("failed to migrate tab %s: %w", tab.ID, err)
		}
	}

	// Migrate categories
	for _, cat := range pData.Categories {
		if err := s.AddCategory(cat); err != nil {
			return fmt.Errorf("failed to migrate category %s: %w", cat.ID, err)
		}
	}

	// Migrate settings
	if pData.Settings.Theme != "" || pData.Settings.OpenMethod != "" {
		if err := s.UpdateSettings(pData.Settings); err != nil {
			return fmt.Errorf("failed to migrate settings: %w", err)
		}
	}

	// Rename old JSON file to backup
	backupPath := jsonPath + ".bak"
	if err := os.Rename(jsonPath, backupPath); err != nil {
		return fmt.Errorf("failed to backup JSON file: %w", err)
	}

	fmt.Printf("Migration complete. Old data backed up to: %s\n", backupPath)
	return nil
}
