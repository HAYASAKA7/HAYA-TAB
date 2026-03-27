package app

import (
	apperrors "haya-tab/pkg/errors"
	"haya-tab/pkg/metadata"
	"haya-tab/pkg/store"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// TabsResponse represents a paginated response for tabs
type TabsResponse struct {
	Tabs     []store.Tab `json:"tabs"`
	Total    int         `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
	HasMore  bool        `json:"hasMore"`
}

// GetTabs returns the list of tabs (backward compatibility)
func (a *App) GetTabs() []store.Tab {
	tabs, err := a.store.GetTabs()
	if err != nil {
		a.logger.Error("Error getting tabs: %v", err)
		return []store.Tab{}
	}
	return tabs
}

// GetTabsPaginated returns a paginated list of tabs with optional search
func (a *App) GetTabsPaginated(categoryId string, page, pageSize int, searchQuery string, filterBy []string, isGlobal bool, sortBy string, sortDesc bool) TabsResponse {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}

	if len(filterBy) == 0 {
		filterBy = []string{"title"}
	}
	searchQuery = strings.ToLower(strings.TrimSpace(searchQuery))

	tabs, total, err := a.store.GetTabsPaginated(categoryId, page, pageSize, searchQuery, filterBy, isGlobal, sortBy, sortDesc)
	if err != nil {
		a.logger.Error("Error getting paginated tabs: %v", err)
		return TabsResponse{
			Tabs:     []store.Tab{},
			Total:    0,
			Page:     page,
			PageSize: pageSize,
			HasMore:  false,
		}
	}

	return TabsResponse{
		Tabs:     tabs,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		HasMore:  (page * pageSize) < total,
	}
}

// GetRecentTabs returns the list of recently accessed tabs
func (a *App) GetRecentTabs(limit int) []store.Tab {
	tabs, err := a.store.GetRecentTabs(limit)
	if err != nil {
		a.logger.Error("Error getting recent tabs: %v", err)
		return []store.Tab{}
	}
	return tabs
}

// SaveTab saves the tab. copyFile determines if we import it to internal storage.
// The passed tab should have the user-confirmed Metadata.
// Returns the saved tab on success.
func (a *App) SaveTab(tab store.Tab, shouldCopy bool) (*store.Tab, error) {
	// Check for duplicate file path before adding (for linked files)
	existingByPath, err := a.store.GetTabByPath(tab.FilePath)
	if err != nil {
		return nil, apperrors.WrapError(err, apperrors.ErrCodeDatabase, map[string]interface{}{"operation": "check_duplicate_path"})
	}
	if existingByPath != nil {
		return nil, apperrors.TabDuplicateError(existingByPath.Title)
	}

	// Check for duplicate title globally (catches uploaded files with same content)
	existingByTitle, err := a.store.GetTabByTitle(tab.Title)
	if err != nil {
		return nil, apperrors.WrapError(err, apperrors.ErrCodeDatabase, map[string]interface{}{"operation": "check_duplicate_title"})
	}
	if existingByTitle != nil {
		return nil, apperrors.TabDuplicateError(existingByTitle.Title)
	}

	// Handle File Copy
	if shouldCopy {
		ext := filepath.Ext(tab.FilePath)
		newFilename := tab.ID + ext
		storageDir := a.GetStorageDir()
		destPath := filepath.Join(storageDir, newFilename)

		src, err := os.Open(tab.FilePath)
		if err != nil {
			return nil, err
		}
		defer src.Close()

		dst, err := os.Create(destPath)
		if err != nil {
			return nil, err
		}
		defer dst.Close()

		if _, err = io.Copy(dst, src); err != nil {
			return nil, err
		}

		// Store relative path in DB for managed tabs to support portable storage directories
		tab.FilePath = newFilename
		tab.IsManaged = true
	} else {
		tab.IsManaged = false
	}

	if tab.AddedAt == 0 {
		tab.AddedAt = time.Now().Unix()
	}

	// Calculate initials for Quick Jump Bar
	tab.InitialAZ, tab.InitialKana = metadata.CalculateInitials(tab.Title, tab.OriginCountry)

	// Save initial version first
	if err := a.store.AddTab(tab); err != nil {
		return nil, err
	}

	// Handle Cover (Async)
	a.fetchCoverAsync(tab)

	return &tab, nil
}

// UpdateTab updates an existing tab's metadata
func (a *App) UpdateTab(tab store.Tab) error {
	// Check if tab exists
	existing, err := a.store.GetTab(tab.ID)
	if err != nil {
		return apperrors.WrapError(err, apperrors.ErrCodeDatabase, map[string]interface{}{"operation": "get_tab"})
	}
	if existing == nil {
		return apperrors.TabNotFoundError(tab.ID)
	}

	// Recalculate initials in case title or origin country changed
	tab.InitialAZ, tab.InitialKana = metadata.CalculateInitials(tab.Title, tab.OriginCountry)

	if err := a.store.AddTab(tab); err != nil {
		return err
	}

	// Update fingerprint if it's associated with a cloud volume
	if tab.IsCloud && tab.VolumeID != "" {
		go a.batchAddToFingerprint(tab.VolumeID, []store.Tab{tab})
	}

	// Notify frontend about the update
	a.emitEvent("tab-updated", tab)

	// Trigger Cover Update (Async)
	a.fetchCoverAsync(tab)

	return nil
}

// UpdateTabMetadata updates only the metadata fields (title, artist, album) for a tab.
// This is called by the frontend after AlphaTab parses the file's internal metadata.
// It implements a "smart update" strategy:
// - If no cover exists: prefer AlphaTab's data (more authoritative than filename parsing)
// - If cover exists: only update placeholder fields (existing data was good enough for cover search)
func (a *App) UpdateTabMetadata(id string, title string, artist string, album string) error {
	// Get current tab
	currentTab, err := a.store.GetTab(id)
	if err != nil {
		return apperrors.WrapError(err, apperrors.ErrCodeDatabase, map[string]interface{}{"operation": "get_tab"})
	}
	if currentTab == nil {
		return apperrors.TabNotFoundError(id)
	}

	needsUpdate := false
	noCoverYet := currentTab.CoverPath == ""

	// Helper to check if existing value is "placeholder" (empty or "Unknown")
	isPlaceholder := func(s string) bool {
		trimmed := strings.TrimSpace(s)
		return trimmed == "" || strings.EqualFold(trimmed, "Unknown")
	}

	// Helper to check if new value is meaningful
	isMeaningful := func(s string) bool {
		trimmed := strings.TrimSpace(s)
		if trimmed == "" {
			return false
		}
		// Skip generic placeholders
		lower := strings.ToLower(trimmed)
		if lower == "untitled" || lower == "unknown" || lower == "no title" {
			return false
		}
		return true
	}

	// Helper to check if values are different (case-insensitive)
	isDifferent := func(old, new string) bool {
		return !strings.EqualFold(strings.TrimSpace(old), strings.TrimSpace(new))
	}

	// Update title: if no cover AND AlphaTab has meaningful different data, prefer it
	// Otherwise only update if current is placeholder
	if isMeaningful(title) {
		if noCoverYet && isDifferent(currentTab.Title, title) {
			currentTab.Title = strings.TrimSpace(title)
			needsUpdate = true
			a.logger.Info("Updating title for tab %s (no cover, prefer AlphaTab): %s", id, title)
		} else if isPlaceholder(currentTab.Title) {
			currentTab.Title = strings.TrimSpace(title)
			needsUpdate = true
			a.logger.Info("Updating title for tab %s: %s", id, title)
		}
	}

	// Update artist: same logic
	if isMeaningful(artist) {
		if noCoverYet && isDifferent(currentTab.Artist, artist) {
			currentTab.Artist = strings.TrimSpace(artist)
			needsUpdate = true
			a.logger.Info("Updating artist for tab %s (no cover, prefer AlphaTab): %s", id, artist)
		} else if isPlaceholder(currentTab.Artist) {
			currentTab.Artist = strings.TrimSpace(artist)
			needsUpdate = true
			a.logger.Info("Updating artist for tab %s: %s", id, artist)
		}
	}

	// Update album: same logic
	if isMeaningful(album) {
		if noCoverYet && isDifferent(currentTab.Album, album) {
			currentTab.Album = strings.TrimSpace(album)
			needsUpdate = true
			a.logger.Info("Updating album for tab %s (no cover, prefer AlphaTab): %s", id, album)
		} else if isPlaceholder(currentTab.Album) {
			currentTab.Album = strings.TrimSpace(album)
			needsUpdate = true
			a.logger.Info("Updating album for tab %s: %s", id, album)
		}
	}

	if needsUpdate {
		// Recalculate initials in case title changed
		currentTab.InitialAZ, currentTab.InitialKana = metadata.CalculateInitials(currentTab.Title, currentTab.OriginCountry)

		if err := a.store.UpdateTab(*currentTab); err != nil {
			return apperrors.WrapError(err, apperrors.ErrCodeDatabase, map[string]interface{}{"operation": "update_tab"})
		}

		// Update fingerprint if it's associated with a cloud volume
		if currentTab.VolumeID != "" {
			go a.batchAddToFingerprint(currentTab.VolumeID, []store.Tab{*currentTab})
		}

		// Notify frontend about the update
		a.emitEvent("tab-updated", *currentTab)

		// If artist was updated and we have enough info, try fetching cover again
		if currentTab.Artist != "" && currentTab.CoverPath == "" {
			a.fetchCoverAsync(*currentTab)
		}
	}

	return nil
}

// DeleteTab deletes a tab and its managed file if applicable
func (a *App) DeleteTab(id string) error {
	// Find tab first to check for managed file
	targetTab, err := a.store.GetTab(id)
	if err != nil {
		return apperrors.WrapError(err, apperrors.ErrCodeDatabase, map[string]interface{}{"operation": "get_tab"})
	}
	if targetTab == nil {
		return apperrors.TabNotFoundError(id)
	}

	if targetTab.IsManaged {
		fullPath := a.ResolveTabPath(targetTab.FilePath, targetTab.IsManaged)
		// Try to delete the file, log error but proceed with DB deletion
		if err := os.Remove(fullPath); err != nil {
			a.logger.Error("Warning: Failed to delete managed file %s: %v", fullPath, err)
		}
		// Also delete cover
		if targetTab.CoverPath != "" {
			fullCoverPath := a.ResolveCoverPath(targetTab.CoverPath)
			os.Remove(fullCoverPath)
		}
	}

	// If it's a cloud tab, update the fingerprint file
	if targetTab.IsCloud && targetTab.VolumeID != "" {
		go a.removeFromFingerprint(targetTab.VolumeID, targetTab.FilePath)
	}

	if err := a.store.DeleteTab(id); err != nil {
		return err
	}

	// Emit event to notify frontend
	a.emitEvent("tab-deleted", id)
	return nil
}

// BatchDeleteTabs deletes multiple tabs at once
func (a *App) BatchDeleteTabs(ids []string) (int, error) {
	deleted := 0
	deletedIds := []string{}

	// Group cloud tabs by volume for batch fingerprint updates
	volumeFiles := make(map[string][]string)

	for _, id := range ids {
		targetTab, err := a.store.GetTab(id)
		if err != nil || targetTab == nil {
			continue
		}

		if targetTab.IsManaged {
			fullPath := a.ResolveTabPath(targetTab.FilePath, targetTab.IsManaged)
			// Try to delete the file
			if err := os.Remove(fullPath); err != nil {
				a.logger.Error("Warning: Failed to delete managed file %s: %v", fullPath, err)
			}
			// Also delete cover
			if targetTab.CoverPath != "" {
				fullCoverPath := a.ResolveCoverPath(targetTab.CoverPath)
				os.Remove(fullCoverPath)
			}
		}

		// Track cloud tabs for fingerprint updates
		if targetTab.IsCloud && targetTab.VolumeID != "" {
			volumeFiles[targetTab.VolumeID] = append(volumeFiles[targetTab.VolumeID], targetTab.FilePath)
		}

		if err := a.store.DeleteTab(id); err == nil {
			deleted++
			deletedIds = append(deletedIds, id)
		}
	}

	// Update fingerprints for all affected volumes
	for volumeID, files := range volumeFiles {
		go a.batchRemoveFromFingerprint(volumeID, files)
	}

	// Emit event to notify frontend
	if deleted > 0 {
		a.emitEvent("tabs-deleted", deletedIds)
	}

	return deleted, nil
}

// BatchMoveTabs moves multiple tabs to a category at once (replaces existing categories)
func (a *App) BatchMoveTabs(ids []string, categoryID string) (int, error) {
	moved := 0
	baseTime := time.Now().Unix()

	// Group cloud tabs by volume for batch fingerprint updates
	volumeTabs := make(map[string][]store.Tab)

	for i, id := range ids {
		// Increment added time slightly to preserve order
		// For backward compatibility, "Move" implies setting the single category
		cats := []string{}
		if categoryID != "" {
			cats = append(cats, categoryID)
		}
		if err := a.store.SetTabCategories(id, cats, baseTime+int64(i)); err == nil {
			moved++

			// Track cloud tabs for fingerprint updates (even if local)
			if tab, _ := a.store.GetTab(id); tab != nil && tab.IsCloud && tab.VolumeID != "" {
				volumeTabs[tab.VolumeID] = append(volumeTabs[tab.VolumeID], *tab)
			}
		}
	}

	// Update fingerprints for all affected volumes
	for volumeID, tabs := range volumeTabs {
		go a.batchAddToFingerprint(volumeID, tabs)
	}

	return moved, nil
}

// BatchAddTabsToCategory adds multiple tabs to a category
func (a *App) BatchAddTabsToCategory(ids []string, categoryID string) (int, error) {
	added := 0
	baseTime := time.Now().Unix()

	// Group cloud tabs by volume for batch fingerprint updates
	volumeTabs := make(map[string][]store.Tab)

	for i, id := range ids {
		// Get existing tab to check for duplicates
		tab, err := a.store.GetTab(id)
		if err != nil || tab == nil {
			continue
		}

		// Check if already in category
		exists := false
		for _, c := range tab.CategoryIDs {
			if c == categoryID {
				exists = true
				break
			}
		}
		if exists {
			continue
		}

		newCats := append(tab.CategoryIDs, categoryID)
		if err := a.store.SetTabCategories(id, newCats, baseTime+int64(i)); err == nil {
			added++

			// Track cloud tabs for fingerprint updates (even if local)
			if updatedTab, _ := a.store.GetTab(id); updatedTab != nil && updatedTab.VolumeID != "" {
				volumeTabs[updatedTab.VolumeID] = append(volumeTabs[updatedTab.VolumeID], *updatedTab)
			}
		}
	}

	// Update fingerprints for all affected volumes
	for volumeID, tabs := range volumeTabs {
		go a.batchAddToFingerprint(volumeID, tabs)
	}

	return added, nil
}

// MoveTab updates the category of a tab (replaces existing categories with this one)
func (a *App) MoveTab(tabID, categoryID string) error {
	cats := []string{}
	if categoryID != "" {
		cats = append(cats, categoryID)
	}
	err := a.store.SetTabCategories(tabID, cats, time.Now().Unix())
	if err == nil {
		if tab, _ := a.store.GetTab(tabID); tab != nil && tab.IsCloud && tab.VolumeID != "" {
			go a.batchAddToFingerprint(tab.VolumeID, []store.Tab{*tab})
		}
	}
	return err
}

// UpdateTabCategories updates the categories for a tab
func (a *App) UpdateTabCategories(tabID string, categoryIDs []string) error {
	err := a.store.SetTabCategories(tabID, categoryIDs, time.Now().Unix())
	if err == nil {
		if tab, _ := a.store.GetTab(tabID); tab != nil && tab.IsCloud && tab.VolumeID != "" {
			go a.batchAddToFingerprint(tab.VolumeID, []store.Tab{*tab})
		}
	}
	return err
}

// AddTabToCategory adds a tab to a category without removing it from others
func (a *App) AddTabToCategory(tabID, categoryID string) error {
	tab, err := a.store.GetTab(tabID)
	if err != nil {
		return apperrors.WrapError(err, apperrors.ErrCodeDatabase, map[string]interface{}{"operation": "get_tab"})
	}
	if tab == nil {
		return apperrors.TabNotFoundError(tabID)
	}

	// Check if already in category
	for _, c := range tab.CategoryIDs {
		if c == categoryID {
			return nil // Already in category
		}
	}

	newCats := append(tab.CategoryIDs, categoryID)
	err = a.store.SetTabCategories(tabID, newCats, time.Now().Unix())
	if err == nil {
		if updatedTab, _ := a.store.GetTab(tabID); updatedTab != nil && updatedTab.VolumeID != "" {
			go a.batchAddToFingerprint(tab.VolumeID, []store.Tab{*updatedTab})
		}
	}
	return err
}

// RemoveTabFromCategory removes a tab from a category
func (a *App) RemoveTabFromCategory(tabID, categoryID string) error {
	tab, err := a.store.GetTab(tabID)
	if err != nil {
		return apperrors.WrapError(err, apperrors.ErrCodeDatabase, map[string]interface{}{"operation": "get_tab"})
	}
	if tab == nil {
		return apperrors.TabNotFoundError(tabID)
	}

	// Prevent removing cloud tabs from the cloud storage category
	// Cloud tabs can only be removed from this category when downloaded to local
	if tab.IsCloud && categoryID == store.SystemCloudCategoryID {
		return apperrors.NewAppError("errors.cannotRemoveCloudCategory", "cannot remove cloud tab from cloud storage category", nil, nil)
	}

	newCats := []string{}
	found := false
	for _, c := range tab.CategoryIDs {
		if c == categoryID {
			found = true
			continue
		}
		newCats = append(newCats, c)
	}

	if !found {
		return nil
	}

	err = a.store.SetTabCategories(tabID, newCats, time.Now().Unix())
	if err == nil {
		if updatedTab, _ := a.store.GetTab(tabID); updatedTab != nil && updatedTab.VolumeID != "" {
			go a.batchAddToFingerprint(tab.VolumeID, []store.Tab{*updatedTab})
		}
	}
	return err
}

// OpenTab opens the file using system default
func (a *App) OpenTab(id string) error {
	targetTab, err := a.store.GetTab(id)
	if err != nil {
		return apperrors.WrapError(err, apperrors.ErrCodeDatabase, map[string]interface{}{"operation": "get_tab"})
	}
	if targetTab == nil {
		return apperrors.TabNotFoundError(id)
	}

	// Update LastOpened
	targetTab.LastOpened = time.Now().Unix()
	a.store.UpdateTab(*targetTab)

	var cmd *exec.Cmd
	path := a.ResolveTabPath(targetTab.FilePath, targetTab.IsManaged)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return apperrors.FileNotFoundError(path, err)
	}

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default: // linux
		cmd = exec.Command("xdg-open", path)
	}
	return cmd.Start()
}

// MarkAsOpened updates the LastOpened timestamp for a tab without opening it
func (a *App) MarkAsOpened(id string) error {
	targetTab, err := a.store.GetTab(id)
	if err != nil {
		return apperrors.WrapError(err, apperrors.ErrCodeDatabase, map[string]interface{}{"operation": "get_tab"})
	}
	if targetTab == nil {
		return apperrors.TabNotFoundError(id)
	}

	targetTab.LastOpened = time.Now().Unix()
	return a.store.UpdateTab(*targetTab)
}

// SaveTabAnnotations saves page-level annotation JSON data for a tab.
func (a *App) SaveTabAnnotations(tabID string, pageNumber int, jsonData string) error {
	if err := a.store.SaveTabAnnotation(tabID, pageNumber, jsonData); err != nil {
		return err
	}

	// Best-effort cloud sync: store annotations under fingerprint metadata directory.
	go a.syncAnnotationToWebDAV(tabID, pageNumber, jsonData)
	return nil
}

// GetTabAnnotations returns page-level annotation JSON data for a tab.
// Returns "[]" when no annotation exists for this page.
func (a *App) GetTabAnnotations(tabID string, pageNumber int) (string, error) {
	jsonData, err := a.store.GetTabAnnotation(tabID, pageNumber)
	if err != nil {
		return "", err
	}

	// Lazy pull from WebDAV when local cache is empty.
	if strings.TrimSpace(jsonData) == "[]" {
		if remoteData, ok := a.fetchAnnotationFromWebDAV(tabID, pageNumber); ok {
			return remoteData, nil
		}
	}

	return jsonData, nil
}

// RecalculateAllInitials forces recalculation of initials for all tabs
func (a *App) RecalculateAllInitials() (int, error) {
	tabs, err := a.store.GetTabs()
	if err != nil {
		return 0, err
	}

	updated := 0
	for _, tab := range tabs {
		az, kana := metadata.CalculateInitials(tab.Title, tab.OriginCountry)
		if err := a.store.UpdateTabInitials(tab.ID, az, kana); err != nil {
			if a.logger != nil {
				a.logger.Error("Failed to update initials for tab %s: %v", tab.ID, err)
			}
			continue
		}
		updated++
	}

	if a.logger != nil {
		a.logger.Info("Recalculated initials for %d/%d tabs", updated, len(tabs))
	}
	return updated, nil
}

// ExportTab copies the tab file to a destination folder
func (a *App) ExportTab(id string, destFolder string) error {
	targetTab, err := a.store.GetTab(id)
	if err != nil {
		return apperrors.WrapError(err, apperrors.ErrCodeDatabase, map[string]interface{}{"operation": "get_tab"})
	}
	if targetTab == nil {
		return apperrors.TabNotFoundError(id)
	}

	srcPath := a.ResolveTabPath(targetTab.FilePath, targetTab.IsManaged)
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return apperrors.FileAccessError(srcPath, err)
	}
	defer srcFile.Close()

	fileName := filepath.Base(srcPath)
	destPath := filepath.Join(destFolder, fileName)

	destFile, err := os.Create(destPath)
	if err != nil {
		return apperrors.FileAccessError(destPath, err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	return err
}

// ProcessFile delegates to SyncService for file processing
func (a *App) ProcessFile(path string) store.Tab {
	return a.syncService.ProcessFile(path)
}
