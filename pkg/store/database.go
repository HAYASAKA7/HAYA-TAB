package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	_ "modernc.org/sqlite"
)

type DBStore struct {
	mu       sync.Mutex
	db       *sql.DB
	dbPath   string
	Settings Settings
}

func NewDBStore(dbPath string) *DBStore {
	return &DBStore{
		dbPath: dbPath,
		Settings: Settings{
			Theme:        "system",
			OpenMethod:   "inner",
			OpenGpMethod: "inner",
			SyncStrategy: "skip",
			SyncPaths:    []string{},
			KeyBindings: KeyBindings{
				ScrollDown:      "j",
				ScrollUp:        "k",
				Metronome:       "m",
				PlayPause:       "p",
				Stop:            "o",
				BpmPlus:         "l",
				BpmMinus:        "h",
				ToggleLoop:      "r",
				ClearSelection:  "escape",
				JumpToBar:       "t",
				JumpToStart:     "i",
				AutoScroll:      "n",
				ScrollSpeedUp:   ",",
				ScrollSpeedDown: ".",
			},
		},
	}
}

// Initialize creates the database and tables
func (s *DBStore) Initialize() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Ensure directory exists
	dir := filepath.Dir(s.dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	db, err := sql.Open("sqlite", s.dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	s.db = db

	// Enable WAL mode for better read/write concurrency
	// This allows reading while writing, preventing UI freezes during sync
	if _, err := s.db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Optimize SQLite settings for better performance
	if _, err := s.db.Exec("PRAGMA synchronous=NORMAL"); err != nil {
		return fmt.Errorf("failed to set synchronous mode: %w", err)
	}
	if _, err := s.db.Exec("PRAGMA cache_size=-64000"); err != nil { // 64MB cache
		return fmt.Errorf("failed to set cache size: %w", err)
	}
	if _, err := s.db.Exec("PRAGMA temp_store=MEMORY"); err != nil {
		return fmt.Errorf("failed to set temp store: %w", err)
	}
	if _, err := s.db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Create tables
	if err := s.createTables(); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	// Run migrations for schema updates
	if err := s.runMigrations(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Load settings into memory
	if err := s.loadSettings(); err != nil {
		return fmt.Errorf("failed to load settings: %w", err)
	}

	return nil
}

func (s *DBStore) createTables() error {
	schema := `
	CREATE TABLE IF NOT EXISTS tabs (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		artist TEXT DEFAULT '',
		album TEXT DEFAULT '',
		file_path TEXT NOT NULL,
		type TEXT NOT NULL,
		is_managed INTEGER DEFAULT 0,
		cover_path TEXT DEFAULT '',
		category_id TEXT DEFAULT '',
		country TEXT DEFAULT '',
		language TEXT DEFAULT '',
		tag TEXT DEFAULT '',
		added_at INTEGER DEFAULT 0,
		last_opened INTEGER DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS categories (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		parent_id TEXT DEFAULT '',
		cover_path TEXT DEFAULT ''
	);

	CREATE TABLE IF NOT EXISTS tab_categories (
		tab_id TEXT,
		category_id TEXT,
		added_at INTEGER DEFAULT 0,
		PRIMARY KEY (tab_id, category_id),
		FOREIGN KEY(tab_id) REFERENCES tabs(id) ON DELETE CASCADE,
		FOREIGN KEY(category_id) REFERENCES categories(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_tabs_category ON tabs(category_id);
	CREATE INDEX IF NOT EXISTS idx_categories_parent ON categories(parent_id);
	CREATE INDEX IF NOT EXISTS idx_tab_categories_tab ON tab_categories(tab_id);
	CREATE INDEX IF NOT EXISTS idx_tab_categories_cat ON tab_categories(category_id);
	`

	if _, err := s.db.Exec(schema); err != nil {
		return err
	}

	// Create FTS5 virtual table for full-text search
	// Using content= option for external content table (keeps data in sync with tabs table)
	ftsSchema := `
	CREATE VIRTUAL TABLE IF NOT EXISTS tabs_fts USING fts5(
		title, artist, album, tag,
		content='tabs',
		content_rowid='rowid'
	);

	-- Triggers to keep FTS index in sync with main table
	CREATE TRIGGER IF NOT EXISTS tabs_ai AFTER INSERT ON tabs BEGIN
		INSERT INTO tabs_fts(rowid, title, artist, album, tag) 
		VALUES (NEW.rowid, NEW.title, NEW.artist, NEW.album, NEW.tag);
	END;

	CREATE TRIGGER IF NOT EXISTS tabs_ad AFTER DELETE ON tabs BEGIN
		INSERT INTO tabs_fts(tabs_fts, rowid, title, artist, album, tag) 
		VALUES ('delete', OLD.rowid, OLD.title, OLD.artist, OLD.album, OLD.tag);
	END;

	CREATE TRIGGER IF NOT EXISTS tabs_au AFTER UPDATE ON tabs BEGIN
		INSERT INTO tabs_fts(tabs_fts, rowid, title, artist, album, tag) 
		VALUES ('delete', OLD.rowid, OLD.title, OLD.artist, OLD.album, OLD.tag);
		INSERT INTO tabs_fts(rowid, title, artist, album, tag) 
		VALUES (NEW.rowid, NEW.title, NEW.artist, NEW.album, NEW.tag);
	END;
	`

	_, err := s.db.Exec(ftsSchema)
	return err
}

// runMigrations handles schema updates for existing databases
func (s *DBStore) runMigrations() error {
	// Add tag column if it doesn't exist (for databases created before this version)
	_, err := s.db.Exec("ALTER TABLE tabs ADD COLUMN tag TEXT DEFAULT ''")
	if err != nil {
		// Ignore error if column already exists
		if !strings.Contains(err.Error(), "duplicate column name") {
			// It's okay, column might already exist
		}
	}

	// Add added_at column
	_, err = s.db.Exec("ALTER TABLE tabs ADD COLUMN added_at INTEGER DEFAULT 0")
	if err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			// It's okay
		}
	}

	// Add last_opened column
	_, err = s.db.Exec("ALTER TABLE tabs ADD COLUMN last_opened INTEGER DEFAULT 0")
	if err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			// It's okay
		}
	}

	// Add cover_path column to categories
	_, err = s.db.Exec("ALTER TABLE categories ADD COLUMN cover_path TEXT DEFAULT ''")
	if err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			// It's okay
		}
	}

	// Rebuild FTS index if needed (for existing databases upgrading to FTS5)
	// This populates the FTS table with any existing tab data
	if _, err := s.db.Exec("INSERT INTO tabs_fts(tabs_fts) VALUES('rebuild')"); err != nil {
		// Ignore errors - table might not exist or already be populated
	}

	// Create tab_categories if not exists (handled in createTables, but good for safety if adding later)
	// Migrate existing category_id to tab_categories
	_, err = s.db.Exec(`
		INSERT INTO tab_categories (tab_id, category_id, added_at)
		SELECT id, category_id, added_at FROM tabs
		WHERE category_id != '' AND category_id IS NOT NULL
		AND NOT EXISTS (
			SELECT 1 FROM tab_categories tc WHERE tc.tab_id = tabs.id AND tc.category_id = tabs.category_id
		)
	`)
	if err != nil {
		// Log error or handle gracefully
		fmt.Printf("Migration warning: failed to migrate categories: %v\n", err)
	}

	return nil
}

func (s *DBStore) loadSettings() error {
	rows, err := s.db.Query("SELECT key, value FROM settings")
	if err != nil {
		return err
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return err
		}
		settings[key] = value
	}

	if v, ok := settings["theme"]; ok {
		s.Settings.Theme = v
	}
	if v, ok := settings["background"]; ok {
		s.Settings.Background = v
	}
	if v, ok := settings["bgType"]; ok {
		s.Settings.BgType = v
	}
	if v, ok := settings["openMethod"]; ok {
		s.Settings.OpenMethod = v
	}
	if v, ok := settings["openGpMethod"]; ok {
		s.Settings.OpenGpMethod = v
	}
	if v, ok := settings["audioDevice"]; ok {
		s.Settings.AudioDevice = v
	}
	if v, ok := settings["autoSyncEnabled"]; ok {
		s.Settings.AutoSyncEnabled = (v == "true")
	}
	if v, ok := settings["autoSyncFrequency"]; ok {
		s.Settings.AutoSyncFrequency = v
	}
	if v, ok := settings["lastSyncTime"]; ok {
		var t int64
		fmt.Sscanf(v, "%d", &t)
		s.Settings.LastSyncTime = t
	}
	if v, ok := settings["syncStrategy"]; ok {
		s.Settings.SyncStrategy = v
	}
	if v, ok := settings["syncPaths"]; ok && v != "" {
		s.Settings.SyncPaths = strings.Split(v, "|")
	}

	// Load key bindings
	if v, ok := settings["keyBindings.scrollDown"]; ok && v != "" {
		s.Settings.KeyBindings.ScrollDown = v
	}
	if v, ok := settings["keyBindings.scrollUp"]; ok && v != "" {
		s.Settings.KeyBindings.ScrollUp = v
	}
	if v, ok := settings["keyBindings.metronome"]; ok && v != "" {
		s.Settings.KeyBindings.Metronome = v
	}
	if v, ok := settings["keyBindings.playPause"]; ok && v != "" {
		s.Settings.KeyBindings.PlayPause = v
	}
	if v, ok := settings["keyBindings.stop"]; ok && v != "" {
		s.Settings.KeyBindings.Stop = v
	}
	if v, ok := settings["keyBindings.bpmPlus"]; ok && v != "" {
		s.Settings.KeyBindings.BpmPlus = v
	}
	if v, ok := settings["keyBindings.bpmMinus"]; ok && v != "" {
		s.Settings.KeyBindings.BpmMinus = v
	}
	if v, ok := settings["keyBindings.toggleLoop"]; ok && v != "" {
		s.Settings.KeyBindings.ToggleLoop = v
	}
	if v, ok := settings["keyBindings.clearSelection"]; ok && v != "" {
		s.Settings.KeyBindings.ClearSelection = v
	}
	if v, ok := settings["keyBindings.jumpToBar"]; ok && v != "" {
		s.Settings.KeyBindings.JumpToBar = v
	}
	if v, ok := settings["keyBindings.jumpToStart"]; ok && v != "" {
		s.Settings.KeyBindings.JumpToStart = v
	}
	if v, ok := settings["keyBindings.autoScroll"]; ok && v != "" {
		s.Settings.KeyBindings.AutoScroll = v
	}
	if v, ok := settings["keyBindings.scrollSpeedUp"]; ok && v != "" {
		s.Settings.KeyBindings.ScrollSpeedUp = v
	}
	if v, ok := settings["keyBindings.scrollSpeedDown"]; ok && v != "" {
		s.Settings.KeyBindings.ScrollSpeedDown = v
	}

	return nil
}

// Close closes the database connection
func (s *DBStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// === Tab Operations ===

func (s *DBStore) GetTabs() ([]Tab, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.db.Query(`
		SELECT id, title, artist, album, file_path, type, is_managed, cover_path, category_id, country, language, COALESCE(tag, ''), added_at, last_opened 
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
		var isManaged int
		var legacyCatID sql.NullString // Handle legacy or null category_id
		if err := rows.Scan(&t.ID, &t.Title, &t.Artist, &t.Album, &t.FilePath, &t.Type, &isManaged, &t.CoverPath, &legacyCatID, &t.Country, &t.Language, &t.Tag, &t.AddedAt, &t.LastOpened); err != nil {
			return nil, err
		}
		t.IsManaged = isManaged == 1
		t.CategoryIDs = []string{} // Initialize
		tabs = append(tabs, t)
		tabMap[t.ID] = &tabs[len(tabs)-1]
	}

	// Fetch all categories
	catRows, err := s.db.Query("SELECT tab_id, category_id FROM tab_categories")
	if err != nil {
		// Just return tabs without categories if this fails, or log?
		// For now return error
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
	s.mu.Lock()
	defer s.mu.Unlock()

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
		SELECT tabs.id, tabs.title, tabs.artist, tabs.album, tabs.file_path, tabs.type, tabs.is_managed, tabs.cover_path, tabs.category_id, tabs.country, tabs.language, COALESCE(tabs.tag, ''), tabs.added_at, tabs.last_opened 
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
		var isManaged int
		var legacyCatID sql.NullString
		if err := rows.Scan(&t.ID, &t.Title, &t.Artist, &t.Album, &t.FilePath, &t.Type, &isManaged, &t.CoverPath, &legacyCatID, &t.Country, &t.Language, &t.Tag, &t.AddedAt, &t.LastOpened); err != nil {
			return nil, 0, err
		}
		t.IsManaged = isManaged == 1
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
		SELECT tabs.id, tabs.title, tabs.artist, tabs.album, tabs.file_path, tabs.type, 
			   tabs.is_managed, tabs.cover_path, tabs.category_id, tabs.country, tabs.language, 
			   COALESCE(tabs.tag, ''), tabs.added_at, tabs.last_opened 
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
		var isManaged int
		var legacyCatID sql.NullString
		if err := rows.Scan(&t.ID, &t.Title, &t.Artist, &t.Album, &t.FilePath, &t.Type, &isManaged, &t.CoverPath, &legacyCatID, &t.Country, &t.Language, &t.Tag, &t.AddedAt, &t.LastOpened); err != nil {
			return nil, 0, err
		}
		t.IsManaged = isManaged == 1
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
		SELECT tabs.id, tabs.title, tabs.artist, tabs.album, tabs.file_path, tabs.type, tabs.is_managed, tabs.cover_path, tabs.category_id, tabs.country, tabs.language, COALESCE(tabs.tag, ''), tabs.added_at, tabs.last_opened 
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
		var isManaged int
		var legacyCatID sql.NullString
		if err := rows.Scan(&t.ID, &t.Title, &t.Artist, &t.Album, &t.FilePath, &t.Type, &isManaged, &t.CoverPath, &legacyCatID, &t.Country, &t.Language, &t.Tag, &t.AddedAt, &t.LastOpened); err != nil {
			return nil, 0, err
		}
		t.IsManaged = isManaged == 1
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
	s.mu.Lock()
	defer s.mu.Unlock()

	var t Tab
	var isManaged int
	var legacyCatID sql.NullString
	err := s.db.QueryRow(`
		SELECT id, title, artist, album, file_path, type, is_managed, cover_path, category_id, country, language, COALESCE(tag, ''), added_at, last_opened 
		FROM tabs WHERE id = ?
	`, id).Scan(&t.ID, &t.Title, &t.Artist, &t.Album, &t.FilePath, &t.Type, &isManaged, &t.CoverPath, &legacyCatID, &t.Country, &t.Language, &t.Tag, &t.AddedAt, &t.LastOpened)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	t.IsManaged = isManaged == 1
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
	s.mu.Lock()
	defer s.mu.Unlock()

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

	// For backward compatibility or if we decide to keep a "primary" category, we could use the first one.
	// For now, let's just use empty string for category_id in tabs table
	primaryCatID := ""
	if len(tab.CategoryIDs) > 0 {
		primaryCatID = tab.CategoryIDs[0]
	}

	_, err = tx.Exec(`
		INSERT OR REPLACE INTO tabs (id, title, artist, album, file_path, type, is_managed, cover_path, category_id, country, language, tag, added_at, last_opened)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, tab.ID, tab.Title, tab.Artist, tab.Album, tab.FilePath, tab.Type, isManaged, tab.CoverPath, primaryCatID, tab.Country, tab.Language, tab.Tag, tab.AddedAt, tab.LastOpened)
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

func (s *DBStore) UpdateTab(tab Tab) error {
	return s.AddTab(tab) // INSERT OR REPLACE handles update
}

func (s *DBStore) DeleteTab(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec("DELETE FROM tabs WHERE id = ?", id)
	return err
}

func (s *DBStore) SetTabCategories(id string, categoryIDs []string, addedAt int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

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
	s.mu.Lock()
	defer s.mu.Unlock()

	var t Tab
	var isManaged int
	var legacyCatID sql.NullString
	err := s.db.QueryRow(`
		SELECT id, title, artist, album, file_path, type, is_managed, cover_path, category_id, country, language, COALESCE(tag, ''), added_at, last_opened 
		FROM tabs WHERE file_path = ?
	`, filePath).Scan(&t.ID, &t.Title, &t.Artist, &t.Album, &t.FilePath, &t.Type, &isManaged, &t.CoverPath, &legacyCatID, &t.Country, &t.Language, &t.Tag, &t.AddedAt, &t.LastOpened)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	t.IsManaged = isManaged == 1
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
	s.mu.Lock()
	defer s.mu.Unlock()

	var t Tab
	var isManaged int
	var legacyCatID sql.NullString
	err := s.db.QueryRow(`
		SELECT id, title, artist, album, file_path, type, is_managed, cover_path, category_id, country, language, COALESCE(tag, ''), added_at, last_opened 
		FROM tabs WHERE title = ?
	`, title).Scan(&t.ID, &t.Title, &t.Artist, &t.Album, &t.FilePath, &t.Type, &isManaged, &t.CoverPath, &legacyCatID, &t.Country, &t.Language, &t.Tag, &t.AddedAt, &t.LastOpened)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	t.IsManaged = isManaged == 1
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

// === Category Operations ===

func (s *DBStore) GetCategories() ([]Category, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.db.Query(`
		SELECT c.id, c.name, c.parent_id, c.cover_path,
		COALESCE(NULLIF(c.cover_path, ''), (SELECT cover_path FROM tabs WHERE category_id = c.id ORDER BY added_at ASC LIMIT 1), '') as effective_cover_path
		FROM categories c
	`)
	if err != nil {
		return []Category{}, err
	}
	defer rows.Close()

	categories := []Category{}
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.ParentID, &c.CoverPath, &c.EffectiveCoverPath); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, nil
}

func (s *DBStore) GetRecentCategories(limit int) ([]Category, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if limit <= 0 {
		limit = 10
	}

	rows, err := s.db.Query(`
		SELECT c.id, c.name, c.parent_id, c.cover_path,
		COALESCE(NULLIF(c.cover_path, ''), (SELECT cover_path FROM tabs WHERE category_id = c.id ORDER BY added_at ASC LIMIT 1), '') as effective_cover_path,
		MAX(t.last_opened) as max_opened
		FROM categories c
		JOIN tabs t ON c.id = t.category_id
		GROUP BY c.id
		ORDER BY max_opened DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return []Category{}, err
	}
	defer rows.Close()

	categories := []Category{}
	for rows.Next() {
		var c Category
		var maxOpened int64
		if err := rows.Scan(&c.ID, &c.Name, &c.ParentID, &c.CoverPath, &c.EffectiveCoverPath, &maxOpened); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, nil
}

func (s *DBStore) GetRecentTabs(limit int) ([]Tab, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if limit <= 0 {
		limit = 20
	}

	rows, err := s.db.Query(`
		SELECT id, title, artist, album, file_path, type, is_managed, cover_path, category_id, country, language, COALESCE(tag, ''), added_at, last_opened 
		FROM tabs 
		WHERE last_opened > 0
		ORDER BY last_opened DESC 
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
		var isManaged int
		var legacyCatID sql.NullString
		if err := rows.Scan(&t.ID, &t.Title, &t.Artist, &t.Album, &t.FilePath, &t.Type, &isManaged, &t.CoverPath, &legacyCatID, &t.Country, &t.Language, &t.Tag, &t.AddedAt, &t.LastOpened); err != nil {
			return nil, err
		}
		t.IsManaged = isManaged == 1
		t.CategoryIDs = []string{}
		tabs = append(tabs, t)
		tabMap[t.ID] = &tabs[len(tabs)-1]
	}

	if len(tabs) > 0 {
		// Fetch categories for these tabs
		// Using a simpler approach: fetch all for these IDs
		// Since it's recent tabs (small limit), we can iterate or use IN clause
		// IN clause is better
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

func (s *DBStore) AddCategory(cat Category) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO categories (id, name, parent_id, cover_path)
		VALUES (?, ?, ?, ?)
	`, cat.ID, cat.Name, cat.ParentID, cat.CoverPath)
	return err
}

func (s *DBStore) DeleteCategory(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Start a transaction
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Move sub-categories to root (or delete them recursively? Current behavior is move to root)
	if _, err := tx.Exec("UPDATE categories SET parent_id = '' WHERE parent_id = ?", id); err != nil {
		return err
	}

	// Delete the category.
	// Since we enabled foreign keys and set ON DELETE CASCADE on tab_categories(category_id),
	// this will automatically remove associations in tab_categories.
	if _, err := tx.Exec("DELETE FROM categories WHERE id = ?", id); err != nil {
		return err
	}

	// Optional: Clear legacy category_id in tabs if it matches (for consistency)
	if _, err := tx.Exec("UPDATE tabs SET category_id = '' WHERE category_id = ?", id); err != nil {
		// Ignore error, non-critical
	}

	return tx.Commit()
}

func (s *DBStore) MoveCategory(id, newParentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec("UPDATE categories SET parent_id = ? WHERE id = ?", newParentID, id)
	return err
}

// === Settings Operations ===

func (s *DBStore) GetSettings() Settings {
	return s.Settings
}

func (s *DBStore) UpdateSettings(settings Settings) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Settings = settings

	// Save each setting
	settingsMap := map[string]string{
		"theme":                       settings.Theme,
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
	}

	for key, value := range settingsMap {
		if _, err := s.db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", key, value); err != nil {
			return err
		}
	}

	return nil
}

// HasData checks if the database has any data
func (s *DBStore) HasData() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM tabs").Scan(&count)
	if err != nil {
		return false
	}
	return count > 0
}
