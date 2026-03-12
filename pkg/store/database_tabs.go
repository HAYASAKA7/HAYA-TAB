package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// === Tab Operations ===

func (s *DBStore) GetTabs() ([]Tab, error) {
	rows, err := s.db.Query(`
		SELECT id, title, artist, album, file_path, COALESCE(cloud_path, ''), type, is_managed, is_cloud, cover_path, category_id, country, language, COALESCE(tag, ''), COALESCE(origin_country, ''), added_at, last_opened, COALESCE(initial_az, '#'), COALESCE(initial_kana, '#')
		FROM tabs
	`)
	if err != nil {
		return []Tab{}, err
	}
	defer rows.Close()

	tabs := []Tab{}
	tabMap := make(map[string]*Tab) // Pointer map for easy update

	for rows.Next() {
		var t Tab
		var isManaged, isCloud int
		var legacyCatID sql.NullString // Handle legacy or null category_id
		if err := rows.Scan(&t.ID, &t.Title, &t.Artist, &t.Album, &t.FilePath, &t.CloudPath, &t.Type, &isManaged, &isCloud, &t.CoverPath, &legacyCatID, &t.Country, &t.Language, &t.Tag, &t.OriginCountry, &t.AddedAt, &t.LastOpened, &t.InitialAZ, &t.InitialKana); err != nil {
			return nil, err
		}
		t.IsManaged = isManaged == 1
		t.IsCloud = isCloud == 1
		t.CategoryIDs = []string{} // Initialize
		tabs = append(tabs, t)
		tabMap[t.ID] = &tabs[len(tabs)-1]
	}

	// Fetch all categories
	catRows, err := s.db.Query("SELECT tab_id, category_id FROM tab_categories")
	if err != nil {
		return nil, err
	}
	defer catRows.Close()

	for catRows.Next() {
		var tID, cID string
		if err := catRows.Scan(&tID, &cID); err == nil {
			if tab, ok := tabMap[tID]; ok {
				tab.CategoryIDs = append(tab.CategoryIDs, cID)
			}
		}
	}

	return tabs, nil
}

func (s *DBStore) GetTabsPaginated(categoryId string, page, pageSize int, searchQuery string, filterBy []string, isGlobal bool, sortBy string, sortDesc bool) ([]Tab, int, error) {
	// Use FTS5 for search if query is provided
	if searchQuery != "" && len(filterBy) > 0 {
		return s.getTabsPaginatedFTS(categoryId, page, pageSize, searchQuery, filterBy, isGlobal, sortBy, sortDesc)
	}

	// Standard query without search
	var whereClauses []string
	var args []interface{}
	var joins []string

	// Category Filter
	if !isGlobal {
		if categoryId != "" {
			// Specific Category
			joins = append(joins, "JOIN tab_categories tc ON tabs.id = tc.tab_id")
			whereClauses = append(whereClauses, "tc.category_id = ?")
			args = append(args, categoryId)
		}
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}
	joinSQL := strings.Join(joins, " ")

	// Count Total
	countQuery := fmt.Sprintf("SELECT COUNT(DISTINCT tabs.id) FROM tabs %s %s", joinSQL, whereSQL)
	var total int
	if err := s.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Get Page Data
	offset := (page - 1) * pageSize
	limit := pageSize

	orderBy := "tabs.title ASC"
	direction := "ASC"
	if sortDesc {
		direction = "DESC"
	}

	switch sortBy {
	case "added_at":
		orderBy = "tabs.added_at " + direction
	case "last_opened":
		orderBy = "tabs.last_opened " + direction
	case "title":
		orderBy = "tabs.title " + direction
	default:
		orderBy = "tabs.title " + direction
	}

	query := fmt.Sprintf(`
		SELECT tabs.id, tabs.title, tabs.artist, tabs.album, tabs.file_path, COALESCE(tabs.cloud_path, ''), tabs.type, tabs.is_managed, COALESCE(tabs.is_cloud, 0), tabs.cover_path, tabs.category_id, tabs.country, tabs.language, COALESCE(tabs.tag, ''), COALESCE(tabs.origin_country, ''), tabs.added_at, tabs.last_opened, COALESCE(tabs.initial_az, '#'), COALESCE(tabs.initial_kana, '#')
		FROM tabs
		%s
		%s
		ORDER BY %s
		LIMIT ? OFFSET ?
	`, joinSQL, whereSQL, orderBy)

	queryArgs := append(args, limit, offset)

	rows, err := s.db.Query(query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	tabs := []Tab{}
	tabIDs := []interface{}{}
	tabMap := make(map[string]*Tab)

	for rows.Next() {
		var t Tab
		var isManaged, isCloud int
		var legacyCatID sql.NullString
		if err := rows.Scan(&t.ID, &t.Title, &t.Artist, &t.Album, &t.FilePath, &t.CloudPath, &t.Type, &isManaged, &isCloud, &t.CoverPath, &legacyCatID, &t.Country, &t.Language, &t.Tag, &t.OriginCountry, &t.AddedAt, &t.LastOpened, &t.InitialAZ, &t.InitialKana); err != nil {
			return nil, 0, err
		}
		t.IsManaged = isManaged == 1
		t.IsCloud = isCloud == 1
		t.CategoryIDs = []string{}
		tabs = append(tabs, t)
		tabIDs = append(tabIDs, t.ID)
		tabMap[t.ID] = &tabs[len(tabs)-1]
	}

	if len(tabs) > 0 {
		// Fetch categories for these tabs
		placeholders := strings.Repeat("?,", len(tabIDs))
		placeholders = placeholders[:len(placeholders)-1]
		catQuery := fmt.Sprintf("SELECT tab_id, category_id FROM tab_categories WHERE tab_id IN (%s)", placeholders)

		catRows, err := s.db.Query(catQuery, tabIDs...)
		if err != nil {
			return nil, 0, err
		}
		defer catRows.Close()

		for catRows.Next() {
			var tID, cID string
			if err := catRows.Scan(&tID, &cID); err == nil {
				if tab, ok := tabMap[tID]; ok {
					tab.CategoryIDs = append(tab.CategoryIDs, cID)
				}
			}
		}
	}

	return tabs, total, nil
}

// getTabsPaginatedFTS uses FTS5 for fast full-text search
func (s *DBStore) getTabsPaginatedFTS(categoryId string, page, pageSize int, searchQuery string, filterBy []string, isGlobal bool, sortBy string, sortDesc bool) ([]Tab, int, error) {
	// Build FTS5 match query with column filters
	// FTS5 supports column filters like: title:query OR artist:query
	var ftsTerms []string
	for _, field := range filterBy {
		switch field {
		case "title", "artist", "album", "tag":
			// Escape special FTS5 characters and add wildcards for prefix matching
			escapedQuery := strings.ReplaceAll(searchQuery, "\"", "\"\"")
			ftsTerms = append(ftsTerms, fmt.Sprintf("%s:\"%s\"*", field, escapedQuery))
		}
	}

	if len(ftsTerms) == 0 {
		return nil, 0, fmt.Errorf("no valid filter fields")
	}

	ftsQuery := strings.Join(ftsTerms, " OR ")

	// Build category filter
	var catWhere string
	var catJoin string
	var catArgs []interface{}

	if !isGlobal {
		if categoryId != "" {
			catJoin = " JOIN tab_categories tc ON tabs.id = tc.tab_id"
			catWhere = " AND tc.category_id = ?"
			catArgs = append(catArgs, categoryId)
		}
	}

	// Count total with FTS5 join
	countQuery := fmt.Sprintf(`
		SELECT COUNT(DISTINCT tabs.id)
		FROM tabs
		INNER JOIN tabs_fts ON tabs.rowid = tabs_fts.rowid
		%s
		WHERE tabs_fts MATCH ?%s
	`, catJoin, catWhere)

	countArgs := append([]interface{}{ftsQuery}, catArgs...)
	var total int
	if err := s.db.QueryRow(countQuery, countArgs...).Scan(&total); err != nil {
		// Fallback to LIKE query if FTS fails (e.g., special characters)
		return s.getTabsPaginatedLike(categoryId, page, pageSize, searchQuery, filterBy, isGlobal, sortBy, sortDesc)
	}

	// Get paginated results
	offset := (page - 1) * pageSize
	limit := pageSize

	orderBy := "bm25(tabs_fts), tabs.title ASC"
	direction := "ASC"
	if sortDesc {
		direction = "DESC"
	}

	switch sortBy {
	case "added_at":
		orderBy = "tabs.added_at " + direction
	case "last_opened":
		orderBy = "tabs.last_opened " + direction
	case "title":
		orderBy = "tabs.title " + direction
	}

	query := fmt.Sprintf(`
		SELECT tabs.id, tabs.title, tabs.artist, tabs.album, tabs.file_path, COALESCE(tabs.cloud_path, ''), tabs.type,
			   tabs.is_managed, COALESCE(tabs.is_cloud, 0), tabs.cover_path, tabs.category_id, tabs.country, tabs.language,
			   COALESCE(tabs.tag, ''), COALESCE(tabs.origin_country, ''), tabs.added_at, tabs.last_opened, COALESCE(tabs.initial_az, '#'), COALESCE(tabs.initial_kana, '#')
		FROM tabs
		INNER JOIN tabs_fts ON tabs.rowid = tabs_fts.rowid
		%s
		WHERE tabs_fts MATCH ?%s
		ORDER BY %s
		LIMIT ? OFFSET ?
	`, catJoin, catWhere, orderBy)

	queryArgs := append([]interface{}{ftsQuery}, catArgs...)
	queryArgs = append(queryArgs, limit, offset)

	rows, err := s.db.Query(query, queryArgs...)
	if err != nil {
		// Fallback to LIKE query if FTS fails
		return s.getTabsPaginatedLike(categoryId, page, pageSize, searchQuery, filterBy, isGlobal, sortBy, sortDesc)
	}
	defer rows.Close()

	tabs := []Tab{}
	tabIDs := []interface{}{}
	tabMap := make(map[string]*Tab)

	for rows.Next() {
		var t Tab
		var isManaged, isCloud int
		var legacyCatID sql.NullString
		if err := rows.Scan(&t.ID, &t.Title, &t.Artist, &t.Album, &t.FilePath, &t.CloudPath, &t.Type, &isManaged, &isCloud, &t.CoverPath, &legacyCatID, &t.Country, &t.Language, &t.Tag, &t.OriginCountry, &t.AddedAt, &t.LastOpened, &t.InitialAZ, &t.InitialKana); err != nil {
			return nil, 0, err
		}
		t.IsManaged = isManaged == 1
		t.IsCloud = isCloud == 1
		t.CategoryIDs = []string{}
		tabs = append(tabs, t)
		tabIDs = append(tabIDs, t.ID)
		tabMap[t.ID] = &tabs[len(tabs)-1]
	}

	if len(tabs) > 0 {
		// Fetch categories for these tabs
		placeholders := strings.Repeat("?,", len(tabIDs))
		placeholders = placeholders[:len(placeholders)-1]
		catQuery := fmt.Sprintf("SELECT tab_id, category_id FROM tab_categories WHERE tab_id IN (%s)", placeholders)

		catRows, err := s.db.Query(catQuery, tabIDs...)
		if err != nil {
			return nil, 0, err
		}
		defer catRows.Close()

		for catRows.Next() {
			var tID, cID string
			if err := catRows.Scan(&tID, &cID); err == nil {
				if tab, ok := tabMap[tID]; ok {
					tab.CategoryIDs = append(tab.CategoryIDs, cID)
				}
			}
		}
	}

	return tabs, total, nil
}

// getTabsPaginatedLike is the fallback using LIKE (for special cases or when FTS fails)
func (s *DBStore) getTabsPaginatedLike(categoryId string, page, pageSize int, searchQuery string, filterBy []string, isGlobal bool, sortBy string, sortDesc bool) ([]Tab, int, error) {
	var whereClauses []string
	var args []interface{}
	var joins []string

	// Category Filter
	if !isGlobal {
		if categoryId != "" {
			joins = append(joins, "JOIN tab_categories tc ON tabs.id = tc.tab_id")
			whereClauses = append(whereClauses, "tc.category_id = ?")
			args = append(args, categoryId)
		}
	}

	// Search Filter with LIKE
	var searchConditions []string
	term := "%" + searchQuery + "%"
	for _, field := range filterBy {
		switch field {
		case "title", "artist", "album", "tag":
			searchConditions = append(searchConditions, fmt.Sprintf("%s LIKE ?", field))
			args = append(args, term)
		}
	}
	if len(searchConditions) > 0 {
		whereClauses = append(whereClauses, "("+strings.Join(searchConditions, " OR ")+")")
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}
	joinSQL := strings.Join(joins, " ")

	// Count Total
	countQuery := fmt.Sprintf("SELECT COUNT(DISTINCT tabs.id) FROM tabs %s %s", joinSQL, whereSQL)
	var total int
	if err := s.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Get Page Data
	offset := (page - 1) * pageSize
	limit := pageSize

	orderBy := "title ASC"
	direction := "ASC"
	if sortDesc {
		direction = "DESC"
	}

	switch sortBy {
	case "added_at":
		orderBy = "added_at " + direction
	case "last_opened":
		orderBy = "last_opened " + direction
	case "title":
		orderBy = "title " + direction
	}

	query := fmt.Sprintf(`
		SELECT tabs.id, tabs.title, tabs.artist, tabs.album, tabs.file_path, COALESCE(tabs.cloud_path, ''), tabs.type, tabs.is_managed, COALESCE(tabs.is_cloud, 0), tabs.cover_path, tabs.category_id, tabs.country, tabs.language, COALESCE(tabs.tag, ''), COALESCE(tabs.origin_country, ''), tabs.added_at, tabs.last_opened, COALESCE(tabs.initial_az, '#'), COALESCE(tabs.initial_kana, '#')
		FROM tabs
		%s
		%s
		ORDER BY %s
		LIMIT ? OFFSET ?
	`, joinSQL, whereSQL, orderBy)

	queryArgs := append(args, limit, offset)

	rows, err := s.db.Query(query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	tabs := []Tab{}
	tabIDs := []interface{}{}
	tabMap := make(map[string]*Tab)

	for rows.Next() {
		var t Tab
		var isManaged, isCloud int
		var legacyCatID sql.NullString
		if err := rows.Scan(&t.ID, &t.Title, &t.Artist, &t.Album, &t.FilePath, &t.CloudPath, &t.Type, &isManaged, &isCloud, &t.CoverPath, &legacyCatID, &t.Country, &t.Language, &t.Tag, &t.OriginCountry, &t.AddedAt, &t.LastOpened, &t.InitialAZ, &t.InitialKana); err != nil {
			return nil, 0, err
		}
		t.IsManaged = isManaged == 1
		t.IsCloud = isCloud == 1
		t.CategoryIDs = []string{}
		tabs = append(tabs, t)
		tabIDs = append(tabIDs, t.ID)
		tabMap[t.ID] = &tabs[len(tabs)-1]
	}

	if len(tabs) > 0 {
		// Fetch categories for these tabs
		placeholders := strings.Repeat("?,", len(tabIDs))
		placeholders = placeholders[:len(placeholders)-1]
		catQuery := fmt.Sprintf("SELECT tab_id, category_id FROM tab_categories WHERE tab_id IN (%s)", placeholders)

		catRows, err := s.db.Query(catQuery, tabIDs...)
		if err != nil {
			return nil, 0, err
		}
		defer catRows.Close()

		for catRows.Next() {
			var tID, cID string
			if err := catRows.Scan(&tID, &cID); err == nil {
				if tab, ok := tabMap[tID]; ok {
					tab.CategoryIDs = append(tab.CategoryIDs, cID)
				}
			}
		}
	}

	return tabs, total, nil
}

func (s *DBStore) GetTab(id string) (*Tab, error) {
	var t Tab
	var isManaged, isCloud int
	var legacyCatID sql.NullString
	var volumeID sql.NullString
	err := s.db.QueryRow(`
		SELECT id, title, artist, album, file_path, COALESCE(cloud_path, ''), type, is_managed, COALESCE(is_cloud, 0), cover_path, category_id, country, language, COALESCE(tag, ''), COALESCE(origin_country, ''), added_at, last_opened, COALESCE(initial_az, '#'), COALESCE(initial_kana, '#'), COALESCE(volume_id, '')
		FROM tabs WHERE id = ?
	`, id).Scan(&t.ID, &t.Title, &t.Artist, &t.Album, &t.FilePath, &t.CloudPath, &t.Type, &isManaged, &isCloud, &t.CoverPath, &legacyCatID, &t.Country, &t.Language, &t.Tag, &t.OriginCountry, &t.AddedAt, &t.LastOpened, &t.InitialAZ, &t.InitialKana, &volumeID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	t.IsManaged = isManaged == 1
	t.IsCloud = isCloud == 1
	t.VolumeID = volumeID.String
	t.CategoryIDs = []string{}

	// Fetch categories
	rows, err := s.db.Query("SELECT category_id FROM tab_categories WHERE tab_id = ?", id)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var cID string
			if err := rows.Scan(&cID); err == nil {
				t.CategoryIDs = append(t.CategoryIDs, cID)
			}
		}
	}

	return &t, nil
}

func (s *DBStore) AddTab(tab Tab) error {
	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	isManaged := 0
	if tab.IsManaged {
		isManaged = 1
	}
	isCloud := 0
	if tab.IsCloud {
		isCloud = 1
	}

	// For backward compatibility or if we decide to keep a "primary" category, we could use the first one.
	// For now, let's just use empty string for category_id in tabs table
	primaryCatID := ""
	if len(tab.CategoryIDs) > 0 {
		primaryCatID = tab.CategoryIDs[0]
	}

	_, err = tx.Exec(`
		INSERT OR REPLACE INTO tabs (id, title, artist, album, file_path, cloud_path, volume_id, type, is_managed, is_cloud, cover_path, category_id, country, language, tag, origin_country, added_at, last_opened, initial_az, initial_kana)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, tab.ID, tab.Title, tab.Artist, tab.Album, tab.FilePath, tab.CloudPath, tab.VolumeID, tab.Type, isManaged, isCloud, tab.CoverPath, primaryCatID, tab.Country, tab.Language, tab.Tag, tab.OriginCountry, tab.AddedAt, tab.LastOpened, tab.InitialAZ, tab.InitialKana)
	if err != nil {
		return err
	}

	// Update categories: Delete old ones and insert new ones
	_, err = tx.Exec("DELETE FROM tab_categories WHERE tab_id = ?", tab.ID)
	if err != nil {
		return err
	}

	if len(tab.CategoryIDs) > 0 {
		stmt, err := tx.Prepare("INSERT INTO tab_categories (tab_id, category_id, added_at) VALUES (?, ?, ?)")
		if err != nil {
			return err
		}
		defer stmt.Close()

		for _, catID := range tab.CategoryIDs {
			if _, err := stmt.Exec(tab.ID, catID, tab.AddedAt); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

// BatchAddTabs adds multiple tabs in a single transaction for better performance
// Returns the number of successfully added tabs and any error encountered
func (s *DBStore) BatchAddTabs(tabs []Tab) (int, error) {
	if len(tabs) == 0 {
		return 0, nil
	}

	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// Prepare statements for reuse
	tabStmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO tabs (id, title, artist, album, file_path, cloud_path, volume_id, type, is_managed, is_cloud, cover_path, category_id, country, language, tag, origin_country, added_at, last_opened, initial_az, initial_kana)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return 0, err
	}
	defer tabStmt.Close()

	catDeleteStmt, err := tx.Prepare("DELETE FROM tab_categories WHERE tab_id = ?")
	if err != nil {
		return 0, err
	}
	defer catDeleteStmt.Close()

	catInsertStmt, err := tx.Prepare("INSERT INTO tab_categories (tab_id, category_id, added_at) VALUES (?, ?, ?)")
	if err != nil {
		return 0, err
	}
	defer catInsertStmt.Close()

	successCount := 0

	// Process each tab
	for _, tab := range tabs {
		isManaged := 0
		if tab.IsManaged {
			isManaged = 1
		}
		isCloud := 0
		if tab.IsCloud {
			isCloud = 1
		}

		primaryCatID := ""
		if len(tab.CategoryIDs) > 0 {
			primaryCatID = tab.CategoryIDs[0]
		}

		// Insert tab
		_, err = tabStmt.Exec(
			tab.ID, tab.Title, tab.Artist, tab.Album, tab.FilePath, tab.CloudPath, tab.VolumeID,
			tab.Type, isManaged, isCloud, tab.CoverPath, primaryCatID,
			tab.Country, tab.Language, tab.Tag, tab.OriginCountry,
			tab.AddedAt, tab.LastOpened, tab.InitialAZ, tab.InitialKana,
		)
		if err != nil {
			// Log error but continue with other tabs
			fmt.Printf("[Warning] Failed to insert tab %s: %v\n", tab.ID, err)
			continue
		}

		// Delete old categories
		_, err = catDeleteStmt.Exec(tab.ID)
		if err != nil {
			fmt.Printf("[Warning] Failed to delete old categories for tab %s: %v\n", tab.ID, err)
			continue
		}

		// Insert new categories
		if len(tab.CategoryIDs) > 0 {
			for _, catID := range tab.CategoryIDs {
				if _, err := catInsertStmt.Exec(tab.ID, catID, tab.AddedAt); err != nil {
					fmt.Printf("[Warning] Failed to insert category %s for tab %s: %v\n", catID, tab.ID, err)
				}
			}
		}

		successCount++
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return successCount, nil
}

func (s *DBStore) UpdateTab(tab Tab) error {
	return s.AddTab(tab) // INSERT OR REPLACE handles update
}

func (s *DBStore) DeleteTab(id string) error {
	_, err := s.db.Exec("DELETE FROM tabs WHERE id = ?", id)
	return err
}

func (s *DBStore) SetTabCategories(id string, categoryIDs []string, addedAt int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update legacy category_id (primary category)
	primaryCatID := ""
	if len(categoryIDs) > 0 {
		primaryCatID = categoryIDs[0]
	}
	if _, err := tx.Exec("UPDATE tabs SET category_id = ? WHERE id = ?", primaryCatID, id); err != nil {
		return err
	}

	// Delete existing associations
	if _, err := tx.Exec("DELETE FROM tab_categories WHERE tab_id = ?", id); err != nil {
		return err
	}

	// Insert new associations
	stmt, err := tx.Prepare("INSERT INTO tab_categories (tab_id, category_id, added_at) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, cID := range categoryIDs {
		if _, err := stmt.Exec(id, cID, addedAt); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *DBStore) GetTabByPath(filePath string) (*Tab, error) {
	var t Tab
	var isManaged, isCloud int
	var legacyCatID sql.NullString
	err := s.db.QueryRow(`
		SELECT id, title, artist, album, file_path, COALESCE(cloud_path, ''), type, is_managed, COALESCE(is_cloud, 0), cover_path, category_id, country, language, COALESCE(tag, ''), COALESCE(origin_country, ''), added_at, last_opened, COALESCE(initial_az, '#'), COALESCE(initial_kana, '#')
		FROM tabs WHERE file_path = ?
	`, filePath).Scan(&t.ID, &t.Title, &t.Artist, &t.Album, &t.FilePath, &t.CloudPath, &t.Type, &isManaged, &isCloud, &t.CoverPath, &legacyCatID, &t.Country, &t.Language, &t.Tag, &t.OriginCountry, &t.AddedAt, &t.LastOpened, &t.InitialAZ, &t.InitialKana)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	t.IsManaged = isManaged == 1
	t.IsCloud = isCloud == 1
	t.CategoryIDs = []string{}

	// Fetch categories
	rows, err := s.db.Query("SELECT category_id FROM tab_categories WHERE tab_id = ?", t.ID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var cID string
			if err := rows.Scan(&cID); err == nil {
				t.CategoryIDs = append(t.CategoryIDs, cID)
			}
		}
	}

	return &t, nil
}

func (s *DBStore) GetTabByTitle(title string) (*Tab, error) {
	var t Tab
	var isManaged, isCloud int
	var legacyCatID sql.NullString
	err := s.db.QueryRow(`
		SELECT id, title, artist, album, file_path, COALESCE(cloud_path, ''), type, is_managed, COALESCE(is_cloud, 0), cover_path, category_id, country, language, COALESCE(tag, ''), COALESCE(origin_country, ''), added_at, last_opened, COALESCE(initial_az, '#'), COALESCE(initial_kana, '#')
		FROM tabs WHERE title = ?
	`, title).Scan(&t.ID, &t.Title, &t.Artist, &t.Album, &t.FilePath, &t.CloudPath, &t.Type, &isManaged, &isCloud, &t.CoverPath, &legacyCatID, &t.Country, &t.Language, &t.Tag, &t.OriginCountry, &t.AddedAt, &t.LastOpened, &t.InitialAZ, &t.InitialKana)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	t.IsManaged = isManaged == 1
	t.IsCloud = isCloud == 1
	t.CategoryIDs = []string{}

	// Fetch categories
	rows, err := s.db.Query("SELECT category_id FROM tab_categories WHERE tab_id = ?", t.ID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var cID string
			if err := rows.Scan(&cID); err == nil {
				t.CategoryIDs = append(t.CategoryIDs, cID)
			}
		}
	}

	return &t, nil
}

func (s *DBStore) GetRecentTabs(limit int) ([]Tab, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := s.db.Query(`
		SELECT id, title, artist, album, file_path, COALESCE(cloud_path, ''), type, is_managed, COALESCE(is_cloud, 0), cover_path, category_id, country, language, COALESCE(tag, ''), COALESCE(origin_country, ''), added_at, last_opened, COALESCE(initial_az, '#'), COALESCE(initial_kana, '#')
		FROM tabs
		WHERE last_opened > 0
		ORDER BY last_opened DESC, added_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return []Tab{}, err
	}
	defer rows.Close()

	tabs := []Tab{}
	tabMap := make(map[string]*Tab)

	for rows.Next() {
		var t Tab
		var isManaged, isCloud int
		var legacyCatID sql.NullString
		if err := rows.Scan(&t.ID, &t.Title, &t.Artist, &t.Album, &t.FilePath, &t.CloudPath, &t.Type, &isManaged, &isCloud, &t.CoverPath, &legacyCatID, &t.Country, &t.Language, &t.Tag, &t.OriginCountry, &t.AddedAt, &t.LastOpened, &t.InitialAZ, &t.InitialKana); err != nil {
			return nil, err
		}
		t.IsManaged = isManaged == 1
		t.IsCloud = isCloud == 1
		t.CategoryIDs = []string{}
		tabs = append(tabs, t)
		tabMap[t.ID] = &tabs[len(tabs)-1]
	}

	if len(tabs) > 0 {
		// Fetch categories for these tabs
		placeholders := strings.Repeat("?,", len(tabs))
		placeholders = placeholders[:len(placeholders)-1]
		ids := make([]interface{}, len(tabs))
		for i, t := range tabs {
			ids[i] = t.ID
		}

		catRows, err := s.db.Query(fmt.Sprintf("SELECT tab_id, category_id FROM tab_categories WHERE tab_id IN (%s)", placeholders), ids...)
		if err == nil {
			defer catRows.Close()
			for catRows.Next() {
				var tID, cID string
				if err := catRows.Scan(&tID, &cID); err == nil {
					if tab, ok := tabMap[tID]; ok {
						tab.CategoryIDs = append(tab.CategoryIDs, cID)
					}
				}
			}
		}
	}

	return tabs, nil
}

// GetTabsNeedingOriginCountry returns tabs that have a cover but no origin_country set
// Used for background backfill of MusicBrainz data
func (s *DBStore) GetTabsNeedingOriginCountry() ([]Tab, error) {
	rows, err := s.db.Query(`
		SELECT id, title, artist, album, file_path, COALESCE(cloud_path, ''), type, is_managed, COALESCE(is_cloud, 0), cover_path, category_id, country, language, COALESCE(tag, ''), COALESCE(origin_country, ''), added_at, last_opened, COALESCE(initial_az, '#'), COALESCE(initial_kana, '#')
		FROM tabs
		WHERE cover_path != '' AND (origin_country IS NULL OR origin_country = '') AND artist != ''
		ORDER BY added_at DESC
	`)
	if err != nil {
		return []Tab{}, err
	}
	defer rows.Close()

	tabs := []Tab{}
	for rows.Next() {
		var t Tab
		var isManaged, isCloud int
		var legacyCatID sql.NullString
		if err := rows.Scan(&t.ID, &t.Title, &t.Artist, &t.Album, &t.FilePath, &t.CloudPath, &t.Type, &isManaged, &isCloud, &t.CoverPath, &legacyCatID, &t.Country, &t.Language, &t.Tag, &t.OriginCountry, &t.AddedAt, &t.LastOpened, &t.InitialAZ, &t.InitialKana); err != nil {
			return nil, err
		}
		t.IsManaged = isManaged == 1
		t.IsCloud = isCloud == 1
		t.CategoryIDs = []string{}
		tabs = append(tabs, t)
	}

	return tabs, nil
}

// UpdateTabOriginCountry updates only the origin_country field for a tab
func (s *DBStore) UpdateTabOriginCountry(tabID, originCountry string) error {
	_, err := s.db.Exec("UPDATE tabs SET origin_country = ? WHERE id = ?", originCountry, tabID)
	return err
}

// GetTabsNeedingInitials returns tabs that need initial_az/initial_kana backfill
// Used for background backfill of legacy data
func (s *DBStore) GetTabsNeedingInitials() ([]Tab, error) {
	rows, err := s.db.Query(`
		SELECT id, title, artist, album, file_path, COALESCE(cloud_path, ''), type, is_managed, COALESCE(is_cloud, 0), cover_path, category_id, country, language, COALESCE(tag, ''), COALESCE(origin_country, ''), added_at, last_opened, COALESCE(initial_az, '#'), COALESCE(initial_kana, '#')
		FROM tabs
		WHERE initial_az IS NULL OR initial_az = '' OR initial_az = '#'
		ORDER BY added_at DESC
	`)
	if err != nil {
		return []Tab{}, err
	}
	defer rows.Close()

	tabs := []Tab{}
	for rows.Next() {
		var t Tab
		var isManaged, isCloud int
		var legacyCatID sql.NullString
		if err := rows.Scan(&t.ID, &t.Title, &t.Artist, &t.Album, &t.FilePath, &t.CloudPath, &t.Type, &isManaged, &isCloud, &t.CoverPath, &legacyCatID, &t.Country, &t.Language, &t.Tag, &t.OriginCountry, &t.AddedAt, &t.LastOpened, &t.InitialAZ, &t.InitialKana); err != nil {
			return nil, err
		}
		t.IsManaged = isManaged == 1
		t.IsCloud = isCloud == 1
		t.CategoryIDs = []string{}
		tabs = append(tabs, t)
	}

	return tabs, nil
}

// UpdateTabInitials updates only the initial_az and initial_kana fields for a tab
func (s *DBStore) UpdateTabInitials(tabID, initialAZ, initialKana string) error {
	_, err := s.db.Exec("UPDATE tabs SET initial_az = ?, initial_kana = ? WHERE id = ?", initialAZ, initialKana, tabID)
	return err
}

// GetAllTabs is an alias for GetTabs for backward compatibility
func (s *DBStore) GetAllTabs() ([]Tab, error) {
	return s.GetTabs()
}

// SearchTabs performs a simple search across title, artist, album, and tag fields
func (s *DBStore) SearchTabs(query string) ([]Tab, error) {
	// Use FTS5 for better search performance
	searchTerm := "%" + query + "%"

	rows, err := s.db.Query(`
		SELECT DISTINCT tabs.id, tabs.title, tabs.artist, tabs.album, tabs.file_path, COALESCE(tabs.cloud_path, ''), tabs.type,
		       tabs.is_managed, COALESCE(tabs.is_cloud, 0), tabs.cover_path, tabs.category_id,
		       tabs.country, tabs.language, COALESCE(tabs.tag, ''), COALESCE(tabs.origin_country, ''),
		       tabs.added_at, tabs.last_opened, COALESCE(tabs.initial_az, '#'), COALESCE(tabs.initial_kana, '#')
		FROM tabs
		WHERE tabs.title LIKE ? OR tabs.artist LIKE ? OR tabs.album LIKE ? OR tabs.tag LIKE ?
		ORDER BY tabs.title ASC
	`, searchTerm, searchTerm, searchTerm, searchTerm)

	if err != nil {
		return []Tab{}, err
	}
	defer rows.Close()

	tabs := []Tab{}
	tabMap := make(map[string]*Tab)

	for rows.Next() {
		var t Tab
		var isManaged, isCloud int
		var legacyCatID sql.NullString
		if err := rows.Scan(&t.ID, &t.Title, &t.Artist, &t.Album, &t.FilePath, &t.CloudPath, &t.Type, &isManaged, &isCloud, &t.CoverPath, &legacyCatID, &t.Country, &t.Language, &t.Tag, &t.OriginCountry, &t.AddedAt, &t.LastOpened, &t.InitialAZ, &t.InitialKana); err != nil {
			return nil, err
		}
		t.IsManaged = isManaged == 1
		t.IsCloud = isCloud == 1
		t.CategoryIDs = []string{}
		tabs = append(tabs, t)
		tabMap[t.ID] = &tabs[len(tabs)-1]
	}

	// Fetch categories for these tabs
	if len(tabs) > 0 {
		placeholders := strings.Repeat("?,", len(tabs))
		placeholders = placeholders[:len(placeholders)-1]
		ids := make([]interface{}, len(tabs))
		for i, t := range tabs {
			ids[i] = t.ID
		}

		catRows, err := s.db.Query(fmt.Sprintf("SELECT tab_id, category_id FROM tab_categories WHERE tab_id IN (%s)", placeholders), ids...)
		if err == nil {
			defer catRows.Close()
			for catRows.Next() {
				var tID, cID string
				if err := catRows.Scan(&tID, &cID); err == nil {
					if tab, ok := tabMap[tID]; ok {
						tab.CategoryIDs = append(tab.CategoryIDs, cID)
					}
				}
			}
		}
	}

	return tabs, nil
}

// UpdateLastOpened updates the last_opened timestamp for a tab
func (s *DBStore) UpdateLastOpened(tabID string) error {
	_, err := s.db.Exec("UPDATE tabs SET last_opened = ? WHERE id = ?", sql.NullInt64{Int64: int64(time.Now().Unix()), Valid: true}, tabID)
	return err
}

