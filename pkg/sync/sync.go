// Package sync provides file synchronization services for HAYA-TAB.
// It handles scanning directories, processing files, and managing tab entries.
package sync

import (
	"fmt"
	"haya-tab/pkg/coverpool"
	"haya-tab/pkg/logger"
	"haya-tab/pkg/metadata"
	"haya-tab/pkg/store"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// EventEmitter is an abstraction for emitting events to the frontend.
// This allows SyncService to be decoupled from wails runtime.
type EventEmitter interface {
	Emit(eventName string, data interface{})
}

// SyncResult contains the results of a sync operation
type SyncResult struct {
	Added   int
	Updated int
	Skipped int
	Errors  int
	Total   int
}

// SyncService handles file synchronization operations
type SyncService struct {
	store     *store.DBStore
	logger    *logger.Logger
	coverPool *coverpool.CoverPool
	emitter   EventEmitter
	appDir    string
}

// NewSyncService creates a new SyncService instance
func NewSyncService(
	store *store.DBStore,
	logger *logger.Logger,
	coverPool *coverpool.CoverPool,
	emitter EventEmitter,
	appDir string,
) *SyncService {
	return &SyncService{
		store:     store,
		logger:    logger,
		coverPool: coverPool,
		emitter:   emitter,
		appDir:    appDir,
	}
}

// TriggerSync scans configured sync paths and adds/updates tabs based on strategy
func (s *SyncService) TriggerSync() (string, error) {
	s.logger.Info("Starting TriggerSync...")
	settings := s.store.GetSettings()
	if len(settings.SyncPaths) == 0 {
		return "No sync paths configured", nil
	}

	result := SyncResult{}
	strategy := settings.SyncStrategy // "skip" or "overwrite"

	s.emitter.Emit("sync-started", nil)

	for _, root := range settings.SyncPaths {
		s.logger.Info("Scanning path: %s", root)
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				s.logger.Error("Error accessing path %s: %v", path, err)
				return nil // Skip unreadable
			}
			if info.IsDir() {
				return nil
			}

			// Check extension
			ext := strings.ToLower(filepath.Ext(path))
			if !s.isSupportedExtension(ext) {
				return nil
			}

			result.Total++
			// Emit progress for every file processed
			s.emitter.Emit("sync-progress", map[string]interface{}{
				"message":  fmt.Sprintf("Processing: %s", filepath.Base(path)),
				"count":    result.Total,
				"filePath": path,
			})

			// 1. Check if EXACT path exists using DB
			existingTab, err := s.store.GetTabByPath(path)
			if err == nil && existingTab != nil {
				return nil // Already exists
			}

			// 2. Parse Metadata to check Title conflict
			newTab := s.ProcessFile(path)

			// Check Title conflict using DB
			conflictTab, _ := s.store.GetTabByTitle(newTab.Title)

			if conflictTab != nil {
				switch strategy {
				case "skip":
					result.Skipped++
					return nil
				case "overwrite":
					// Non-destructive overwrite: Keep old file, rename new title
					uniqueTitle := s.generateUniqueTitle(newTab.Title)
					newTab.Title = uniqueTitle

					// Add as new tab with renamed title
					if err := s.store.AddTab(newTab); err == nil {
						result.Added++
						s.FetchCoverAsync(newTab)
					} else {
						result.Errors++
					}
					return nil
				}
			}

			// No conflict, add as new
			if err := s.store.AddTab(newTab); err == nil {
				result.Added++
				s.FetchCoverAsync(newTab)
			} else {
				result.Errors++
			}

			return nil
		})
		if err != nil {
			s.logger.Error("Error walking %s: %v", root, err)
		}
	}

	s.emitter.Emit("sync-completed", map[string]interface{}{
		"added":   result.Added,
		"updated": result.Updated,
		"skipped": result.Skipped,
		"errors":  result.Errors,
		"total":   result.Total,
	})

	// Update Last Sync Time
	settings.LastSyncTime = time.Now().Unix()
	s.store.UpdateSettings(settings)

	return fmt.Sprintf("Sync complete. Added: %d, Updated: %d, Skipped: %d, Errors: %d",
		result.Added, result.Updated, result.Skipped, result.Errors), nil
}

// ProcessFile takes a file path and returns a pre-filled Tab struct
func (s *SyncService) ProcessFile(path string) store.Tab {
	meta, err := metadata.ParseFile(path)
	if err != nil {
		s.logger.Error("Error parsing file metadata for %s: %v", path, err)
		meta = metadata.ParseFilename(path)
	}

	ext := strings.ToLower(filepath.Ext(path))
	typeStr := s.getFileType(ext)

	return store.Tab{
		ID:       fmt.Sprintf("%d", time.Now().UnixNano()),
		Title:    meta.Title,
		Artist:   meta.Artist,
		Album:    meta.Album,
		FilePath: path,
		Type:     typeStr,
	}
}

// FetchCoverAsync downloads album cover art asynchronously for a tab using worker pool
func (s *SyncService) FetchCoverAsync(tab store.Tab) {
	if tab.Artist == "" || (tab.Album == "" && tab.Title == "") {
		return // Not enough info to search for cover
	}

	coverFilename := tab.ID + ".jpg"
	coverPath := filepath.Join(s.appDir, "covers", coverFilename)

	s.coverPool.Submit(coverpool.CoverJob{
		TabID:     tab.ID,
		Artist:    tab.Artist,
		Album:     tab.Album,
		Title:     tab.Title,
		Country:   tab.Country,
		Language:  tab.Language,
		CoverPath: coverPath,
		OnComplete: func(tabID, coverPath string, err error) {
			if err == nil {
				s.logger.Info("Cover downloaded successfully to: %s", coverPath)
				currentTab, getErr := s.store.GetTab(tabID)
				if getErr != nil || currentTab == nil {
					s.logger.Error("Failed to get tab after cover download: %v", getErr)
					return
				}
				currentTab.CoverPath = coverPath
				s.store.AddTab(*currentTab)
				s.emitter.Emit("tab-updated", *currentTab)
			} else {
				s.logger.Error("Failed to download cover: %v", err)
			}
		},
	})
}

// generateUniqueTitle creates a unique title by appending _copy1, _copy2, etc.
func (s *SyncService) generateUniqueTitle(baseTitle string) string {
	copyNum := 1
	candidate := fmt.Sprintf("%s_copy%d", baseTitle, copyNum)

	for {
		existing, _ := s.store.GetTabByTitle(candidate)
		if existing == nil {
			return candidate
		}
		copyNum++
		candidate = fmt.Sprintf("%s_copy%d", baseTitle, copyNum)

		// Safety limit to prevent infinite loop
		if copyNum > 1000 {
			return fmt.Sprintf("%s_copy_%d", baseTitle, time.Now().UnixNano())
		}
	}
}

// isSupportedExtension checks if the file extension is supported
func (s *SyncService) isSupportedExtension(ext string) bool {
	switch ext {
	case ".pdf", ".gp", ".gp3", ".gp4", ".gp5", ".gpx":
		return true
	default:
		return false
	}
}

// getFileType returns the file type based on extension
func (s *SyncService) getFileType(ext string) string {
	switch ext {
	case ".pdf":
		return "pdf"
	case ".gp", ".gp3", ".gp4", ".gp5", ".gpx":
		return "gp"
	default:
		return "unknown"
	}
}
