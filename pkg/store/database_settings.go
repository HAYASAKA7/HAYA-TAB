package store

import (
	"encoding/json"
	"fmt"
	"strings"
)

// === Settings Operations ===

func (s *DBStore) GetSettings() Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Settings
}

func (s *DBStore) UpdateSettings(settings Settings) error {
	// Encrypt password before DB operations
	encryptedPass, err := Encrypt(settings.WebDAVPassword)
	if err != nil {
		return err
	}

	midiSettingsJSON, _ := json.Marshal(settings.MidiSettings)

	// Save each setting to DB (no lock needed - relies on database/sql pool + WAL)
	settingsMap := map[string]string{
		"theme":                       settings.Theme,
		"language":                    settings.Language,
		"background":                  settings.Background,
		"bgType":                      settings.BgType,
		"openMethod":                  settings.OpenMethod,
		"openGpMethod":                settings.OpenGpMethod,
		"audioDevice":                 settings.AudioDevice,
		"autoSyncEnabled":             fmt.Sprintf("%v", settings.AutoSyncEnabled),
		"autoSyncFrequency":           settings.AutoSyncFrequency,
		"lastSyncTime":                fmt.Sprintf("%d", settings.LastSyncTime),
		"syncStrategy":                settings.SyncStrategy,
		"syncPaths":                   strings.Join(settings.SyncPaths, "|"),
		"storagePath":                 settings.StoragePath,
		"coversPath":                  settings.CoversPath,
		"updateCheckEnabled":          fmt.Sprintf("%v", settings.UpdateCheckEnabled),
		"lastUpdateCheckTime":         fmt.Sprintf("%d", settings.LastUpdateCheckTime),
		"latestVersion":               settings.LatestVersion,
		"webdavEnabled":               fmt.Sprintf("%v", settings.WebDAVEnabled),
		"webdavUrl":                   settings.WebDAVURL,
		"webdavUser":                  settings.WebDAVUser,
		"webdavPassword":              encryptedPass,
		"keyBindings.scrollDown":      settings.KeyBindings.ScrollDown,
		"keyBindings.scrollUp":        settings.KeyBindings.ScrollUp,
		"keyBindings.metronome":       settings.KeyBindings.Metronome,
		"keyBindings.playPause":       settings.KeyBindings.PlayPause,
		"keyBindings.stop":            settings.KeyBindings.Stop,
		"keyBindings.bpmPlus":         settings.KeyBindings.BpmPlus,
		"keyBindings.bpmMinus":        settings.KeyBindings.BpmMinus,
		"keyBindings.toggleLoop":      settings.KeyBindings.ToggleLoop,
		"keyBindings.clearSelection":  settings.KeyBindings.ClearSelection,
		"keyBindings.jumpToBar":       settings.KeyBindings.JumpToBar,
		"keyBindings.jumpToStart":     settings.KeyBindings.JumpToStart,
		"keyBindings.autoScroll":      settings.KeyBindings.AutoScroll,
		"keyBindings.scrollSpeedUp":   settings.KeyBindings.ScrollSpeedUp,
		"keyBindings.scrollSpeedDown": settings.KeyBindings.ScrollSpeedDown,
		"midiSettings":                string(midiSettingsJSON),
	}

	for key, value := range settingsMap {
		if _, err := s.db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", key, value); err != nil {
			return err
		}
	}

	// Lock only for memory assignment
	s.mu.Lock()
	s.Settings = settings
	s.mu.Unlock()

	return nil
}
