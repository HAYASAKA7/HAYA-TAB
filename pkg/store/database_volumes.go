package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// AddVolume adds a new cloud volume to the database
func (s *DBStore) AddVolume(volume CloudVolume) error {
	query := `
		INSERT INTO cloud_volumes (id, name, mount_path, fingerprint_path, created_at, last_seen_at, is_available)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	// Retry logic for database lock errors
	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		_, err := s.db.Exec(query,
			volume.ID,
			volume.Name,
			volume.MountPath,
			volume.FingerprintPath,
			volume.CreatedAt,
			volume.LastSeenAt,
			boolToInt(volume.IsAvailable),
		)
		if err == nil {
			return nil
		}

		// Check if it's a database lock error
		if strings.Contains(err.Error(), "database is locked") || strings.Contains(err.Error(), "SQLITE_BUSY") {
			// Exponential backoff: 10ms, 20ms, 40ms, 80ms, 160ms
			time.Sleep(time.Duration(10*(1<<uint(i))) * time.Millisecond)
			continue
		}

		// Non-lock error, return immediately
		return err
	}

	return fmt.Errorf("failed to add volume after %d retries: database is locked", maxRetries)
}

// GetVolume retrieves a volume by ID
func (s *DBStore) GetVolume(id string) (*CloudVolume, error) {
	query := `
		SELECT id, name, mount_path, fingerprint_path, created_at, last_seen_at, is_available
		FROM cloud_volumes
		WHERE id = ?
	`
	var volume CloudVolume
	var isAvailable int
	err := s.db.QueryRow(query, id).Scan(
		&volume.ID,
		&volume.Name,
		&volume.MountPath,
		&volume.FingerprintPath,
		&volume.CreatedAt,
		&volume.LastSeenAt,
		&isAvailable,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	volume.IsAvailable = intToBool(isAvailable)
	return &volume, nil
}

// GetVolumeByMountPath retrieves a volume by its mount path
func (s *DBStore) GetVolumeByMountPath(mountPath string) (*CloudVolume, error) {
	query := `
		SELECT id, name, mount_path, fingerprint_path, created_at, last_seen_at, is_available
		FROM cloud_volumes
		WHERE mount_path = ?
	`
	var volume CloudVolume
	var isAvailable int
	err := s.db.QueryRow(query, mountPath).Scan(
		&volume.ID,
		&volume.Name,
		&volume.MountPath,
		&volume.FingerprintPath,
		&volume.CreatedAt,
		&volume.LastSeenAt,
		&isAvailable,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	volume.IsAvailable = intToBool(isAvailable)
	return &volume, nil
}

// GetAllVolumes retrieves all cloud volumes
func (s *DBStore) GetAllVolumes() ([]CloudVolume, error) {
	query := `
		SELECT id, name, mount_path, fingerprint_path, created_at, last_seen_at, is_available
		FROM cloud_volumes
		ORDER BY name
	`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var volumes []CloudVolume
	for rows.Next() {
		var volume CloudVolume
		var isAvailable int
		if err := rows.Scan(
			&volume.ID,
			&volume.Name,
			&volume.MountPath,
			&volume.FingerprintPath,
			&volume.CreatedAt,
			&volume.LastSeenAt,
			&isAvailable,
		); err != nil {
			return nil, err
		}
		volume.IsAvailable = intToBool(isAvailable)
		volumes = append(volumes, volume)
	}
	return volumes, nil
}

// UpdateVolume updates an existing volume
func (s *DBStore) UpdateVolume(volume CloudVolume) error {
	query := `
		UPDATE cloud_volumes
		SET name = ?, mount_path = ?, fingerprint_path = ?, last_seen_at = ?, is_available = ?
		WHERE id = ?
	`
	_, err := s.db.Exec(query,
		volume.Name,
		volume.MountPath,
		volume.FingerprintPath,
		volume.LastSeenAt,
		boolToInt(volume.IsAvailable),
		volume.ID,
	)
	return err
}

// UpdateVolumeMountPath updates the mount path of a volume (when WebDAV root changes)
func (s *DBStore) UpdateVolumeMountPath(volumeID, newMountPath string) error {
	query := `
		UPDATE cloud_volumes
		SET mount_path = ?, last_seen_at = ?
		WHERE id = ?
	`
	_, err := s.db.Exec(query, newMountPath, time.Now().Unix(), volumeID)
	return err
}

// MarkVolumeAvailable marks a volume as available or unavailable
func (s *DBStore) MarkVolumeAvailable(volumeID string, available bool) error {
	query := `
		UPDATE cloud_volumes
		SET is_available = ?, last_seen_at = ?
		WHERE id = ?
	`
	_, err := s.db.Exec(query, boolToInt(available), time.Now().Unix(), volumeID)
	return err
}

// DeleteVolume deletes a volume and all associated tabs
func (s *DBStore) DeleteVolume(volumeID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete all tabs associated with this volume
	if _, err := tx.Exec("DELETE FROM tabs WHERE volume_id = ?", volumeID); err != nil {
		return err
	}

	// Delete the volume
	if _, err := tx.Exec("DELETE FROM cloud_volumes WHERE id = ?", volumeID); err != nil {
		return err
	}

	return tx.Commit()
}

// GetTabsByVolume retrieves all tabs for a specific volume
func (s *DBStore) GetTabsByVolume(volumeID string) ([]Tab, error) {
	query := `
		SELECT id, title, artist, album, file_path, COALESCE(volume_id, ''), type, is_managed, is_cloud,
		       cover_path, country, language, origin_country, tag, added_at, last_opened,
		       initial_az, initial_kana
		FROM tabs
		WHERE volume_id = ?
		ORDER BY title
	`
	rows, err := s.db.Query(query, volumeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tabs []Tab
	for rows.Next() {
		var tab Tab
		var isManaged, isCloud int
		if err := rows.Scan(
			&tab.ID, &tab.Title, &tab.Artist, &tab.Album, &tab.FilePath, &tab.VolumeID,
			&tab.Type, &isManaged, &isCloud, &tab.CoverPath, &tab.Country, &tab.Language,
			&tab.OriginCountry, &tab.Tag, &tab.AddedAt, &tab.LastOpened,
			&tab.InitialAZ, &tab.InitialKana,
		); err != nil {
			return nil, err
		}
		tab.IsManaged = intToBool(isManaged)
		tab.IsCloud = intToBool(isCloud)

		// Load category IDs
		categoryIDs, err := s.getTabCategoryIDs(tab.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to load categories for tab %s: %w", tab.ID, err)
		}
		tab.CategoryIDs = categoryIDs

		tabs = append(tabs, tab)
	}
	return tabs, nil
}

// GetTabByVolumeAndPath retrieves a tab by volume ID and relative path
func (s *DBStore) GetTabByVolumeAndPath(volumeID, relativePath string) (*Tab, error) {
	query := `
		SELECT id, title, artist, album, file_path, COALESCE(volume_id, ''), type, is_managed, is_cloud,
		       cover_path, country, language, origin_country, tag, added_at, last_opened,
		       initial_az, initial_kana
		FROM tabs
		WHERE volume_id = ? AND file_path = ?
	`
	var tab Tab
	var isManaged, isCloud int
	err := s.db.QueryRow(query, volumeID, relativePath).Scan(
		&tab.ID, &tab.Title, &tab.Artist, &tab.Album, &tab.FilePath, &tab.VolumeID,
		&tab.Type, &isManaged, &isCloud, &tab.CoverPath, &tab.Country, &tab.Language,
		&tab.OriginCountry, &tab.Tag, &tab.AddedAt, &tab.LastOpened,
		&tab.InitialAZ, &tab.InitialKana,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	tab.IsManaged = intToBool(isManaged)
	tab.IsCloud = intToBool(isCloud)

	// Load category IDs
	categoryIDs, err := s.getTabCategoryIDs(tab.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load categories for tab %s: %w", tab.ID, err)
	}
	tab.CategoryIDs = categoryIDs

	return &tab, nil
}

// getTabCategoryIDs retrieves all category IDs for a tab
func (s *DBStore) getTabCategoryIDs(tabID string) ([]string, error) {
	query := "SELECT category_id FROM tab_categories WHERE tab_id = ?"
	rows, err := s.db.Query(query, tabID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categoryIDs []string
	for rows.Next() {
		var catID string
		if err := rows.Scan(&catID); err != nil {
			return nil, err
		}
		categoryIDs = append(categoryIDs, catID)
	}
	return categoryIDs, nil
}

// Helper function to convert bool to int for SQLite
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// Helper function to convert int to bool from SQLite
func intToBool(i int) bool {
	return i != 0
}
