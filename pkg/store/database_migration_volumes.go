package store

import (
	"fmt"
	"strings"
)

// MigrateCloudTabsToVolumes migrates existing cloud tabs to use the volume system
// This is for backward compatibility with tabs created before the volume system was implemented
func (s *DBStore) MigrateCloudTabsToVolumes() error {
	// Find all cloud tabs without a volume_id
	query := `
		SELECT id, file_path, is_cloud
		FROM tabs
		WHERE is_cloud = 1 AND (volume_id IS NULL OR volume_id = '')
	`
	rows, err := s.db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to query cloud tabs: %w", err)
	}
	defer rows.Close()

	type legacyTab struct {
		id       string
		filePath string
		isCloud  int
	}

	var legacyTabs []legacyTab
	for rows.Next() {
		var tab legacyTab
		if err := rows.Scan(&tab.id, &tab.filePath, &tab.isCloud); err != nil {
			return fmt.Errorf("failed to scan tab: %w", err)
		}
		legacyTabs = append(legacyTabs, tab)
	}

	if len(legacyTabs) == 0 {
		fmt.Println("[Migration] No legacy cloud tabs to migrate")
		return nil
	}

	fmt.Printf("[Migration] Found %d legacy cloud tabs to migrate\n", len(legacyTabs))

	// Get all volumes
	volumes, err := s.GetAllVolumes()
	if err != nil {
		return fmt.Errorf("failed to get volumes: %w", err)
	}

	if len(volumes) == 0 {
		fmt.Println("[Migration] No volumes found. Legacy tabs will remain unmigrated until volumes are discovered.")
		return nil
	}

	migratedCount := 0
	unmatchedCount := 0

	// Try to match each legacy tab to a volume
	for _, tab := range legacyTabs {
		matched := false

		// Find the volume with the longest matching mount path
		longestMatch := ""
		var matchedVolumeID string

		for _, vol := range volumes {
			// Handle root volume specially
			if vol.MountPath == "/" {
				// For root volume, match if:
				// 1. Path doesn't start with any other volume's mount path
				// 2. Path is just a filename (no directory separators)
				// We'll check this after trying other volumes
				continue
			}

			// For non-root volumes, check if path starts with mount path
			if strings.HasPrefix(tab.filePath, vol.MountPath+"/") || tab.filePath == vol.MountPath {
				if len(vol.MountPath) > len(longestMatch) {
					longestMatch = vol.MountPath
					matchedVolumeID = vol.ID
					matched = true
				}
			}
		}

		// If no specific volume matched, try the root volume
		if !matched {
			for _, vol := range volumes {
				if vol.MountPath == "/" {
					// Root volume matches everything that didn't match a subdirectory
					longestMatch = vol.MountPath
					matchedVolumeID = vol.ID
					matched = true
					break
				}
			}
		}

		if matched {
			// Calculate relative path
			relativePath := strings.TrimPrefix(tab.filePath, longestMatch)
			relativePath = strings.TrimPrefix(relativePath, "/")

			// Update the tab
			updateQuery := `
				UPDATE tabs
				SET volume_id = ?, file_path = ?
				WHERE id = ?
			`
			if _, err := s.db.Exec(updateQuery, matchedVolumeID, relativePath, tab.id); err != nil {
				fmt.Printf("[Migration] Failed to migrate tab %s: %v\n", tab.id, err)
				continue
			}

			migratedCount++
			fmt.Printf("[Migration] Migrated tab %s to volume %s (path: %s -> %s)\n", tab.id, matchedVolumeID, tab.filePath, relativePath)
		} else {
			unmatchedCount++
			fmt.Printf("[Migration] Could not match tab %s (path: %s) to any volume\n", tab.id, tab.filePath)
		}
	}

	fmt.Printf("[Migration] Migration complete: %d migrated, %d unmatched\n", migratedCount, unmatchedCount)
	return nil
}

// CleanupOrphanedTabs removes tabs that reference non-existent volumes
func (s *DBStore) CleanupOrphanedTabs() (int, error) {
	// Find tabs with volume_id that don't exist in cloud_volumes
	query := `
		DELETE FROM tabs
		WHERE volume_id != ''
		AND volume_id NOT IN (SELECT id FROM cloud_volumes)
	`
	result, err := s.db.Exec(query)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup orphaned tabs: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		fmt.Printf("[Cleanup] Removed %d orphaned tabs\n", rowsAffected)
	}

	return int(rowsAffected), nil
}

// GetOrphanedTabsCount returns the count of tabs referencing non-existent volumes
func (s *DBStore) GetOrphanedTabsCount() (int, error) {
	query := `
		SELECT COUNT(*)
		FROM tabs
		WHERE volume_id != ''
		AND volume_id NOT IN (SELECT id FROM cloud_volumes)
	`
	var count int
	err := s.db.QueryRow(query).Scan(&count)
	return count, err
}

// EnsureCloudTabsHaveCloudCategory adds the syscloud category to all cloud tabs that don't have it
// This is a migration for tabs created before the automatic category assignment was implemented
func (s *DBStore) EnsureCloudTabsHaveCloudCategory() (int, error) {
	// First ensure the cloud category exists
	if err := s.EnsureCloudCategory(); err != nil {
		return 0, fmt.Errorf("failed to ensure cloud category: %w", err)
	}

	// Find all cloud tabs that don't have the syscloud category
	query := `
		SELECT id
		FROM tabs
		WHERE is_cloud = 1
		AND id NOT IN (
			SELECT tab_id
			FROM tab_categories
			WHERE category_id = ?
		)
	`
	rows, err := s.db.Query(query, SystemCloudCategoryID)
	if err != nil {
		return 0, fmt.Errorf("failed to query cloud tabs: %w", err)
	}
	defer rows.Close()

	var tabIDs []string
	for rows.Next() {
		var tabID string
		if err := rows.Scan(&tabID); err != nil {
			return 0, fmt.Errorf("failed to scan tab ID: %w", err)
		}
		tabIDs = append(tabIDs, tabID)
	}

	if len(tabIDs) == 0 {
		fmt.Println("[Migration] All cloud tabs already have the cloud category")
		return 0, nil
	}

	fmt.Printf("[Migration] Found %d cloud tabs without cloud category\n", len(tabIDs))

	// Add the cloud category to each tab
	addedCount := 0
	for _, tabID := range tabIDs {
		insertQuery := `
			INSERT OR IGNORE INTO tab_categories (tab_id, category_id, added_at)
			VALUES (?, ?, ?)
		`
		result, err := s.db.Exec(insertQuery, tabID, SystemCloudCategoryID, 0)
		if err != nil {
			fmt.Printf("[Migration] Failed to add cloud category to tab %s: %v\n", tabID, err)
			continue
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected > 0 {
			addedCount++
		}
	}

	fmt.Printf("[Migration] Added cloud category to %d tabs\n", addedCount)
	return addedCount, nil
}
