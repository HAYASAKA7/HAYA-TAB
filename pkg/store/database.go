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
				ScrollSpeedUp:   ">",
				ScrollSpeedDown: "<",
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
		tag TEXT DEFAULT ''
	);

	CREATE TABLE IF NOT EXISTS categories (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		parent_id TEXT DEFAULT ''
	);

	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_tabs_category ON tabs(category_id);
	CREATE INDEX IF NOT EXISTS idx_categories_parent ON categories(parent_id);
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

	// Rebuild FTS index if needed (for existing databases upgrading to FTS5)
	// This populates the FTS table with any existing tab data
	if _, err := s.db.Exec("INSERT INTO tabs_fts(tabs_fts) VALUES('rebuild')"); err != nil {
		// Ignore errors - table might not exist or already be populated
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
		SELECT id, title, artist, album, file_path, type, is_managed, cover_path, category_id, country, language, COALESCE(tag, '') 
		FROM tabs
	`)
	if err != nil {
		return []Tab{}, err
	}
	defer rows.Close()

	tabs := []Tab{}
	for rows.Next() {
		var t Tab
		var isManaged int
		if err := rows.Scan(&t.ID, &t.Title, &t.Artist, &t.Album, &t.FilePath, &t.Type, &isManaged, &t.CoverPath, &t.CategoryID, &t.Country, &t.Language, &t.Tag); err != nil {
			return nil, err
		}
		t.IsManaged = isManaged == 1
		tabs = append(tabs, t)
	}
	return tabs, nil
}

func (s *DBStore) GetTabsPaginated(categoryId string, page, pageSize int, searchQuery string, filterBy []string, isGlobal bool) ([]Tab, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Use FTS5 for search if query is provided
	if searchQuery != "" && len(filterBy) > 0 {
		return s.getTabsPaginatedFTS(categoryId, page, pageSize, searchQuery, filterBy, isGlobal)
	}

	// Standard query without search
	var whereClauses []string
	var args []interface{}

	// Category Filter
	if !isGlobal {
		if categoryId == "" {
			whereClauses = append(whereClauses, "(category_id = '' OR category_id IS NULL)")
		} else {
			whereClauses = append(whereClauses, "category_id = ?")
			args = append(args, categoryId)
		}
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Count Total
	countQuery := "SELECT COUNT(*) FROM tabs " + whereSQL
	var total int
	if err := s.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Get Page Data
	offset := (page - 1) * pageSize
	limit := pageSize

	query := fmt.Sprintf(`
		SELECT id, title, artist, album, file_path, type, is_managed, cover_path, category_id, country, language, COALESCE(tag, '') 
		FROM tabs 
		%s 
		ORDER BY title ASC 
		LIMIT ? OFFSET ?
	`, whereSQL)

	queryArgs := append(args, limit, offset)

	rows, err := s.db.Query(query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	tabs := []Tab{}
	for rows.Next() {
		var t Tab
		var isManaged int
		if err := rows.Scan(&t.ID, &t.Title, &t.Artist, &t.Album, &t.FilePath, &t.Type, &isManaged, &t.CoverPath, &t.CategoryID, &t.Country, &t.Language, &t.Tag); err != nil {
			return nil, 0, err
		}
		t.IsManaged = isManaged == 1
		tabs = append(tabs, t)
	}

	return tabs, total, nil
}

// getTabsPaginatedFTS uses FTS5 for fast full-text search
func (s *DBStore) getTabsPaginatedFTS(categoryId string, page, pageSize int, searchQuery string, filterBy []string, isGlobal bool) ([]Tab, int, error) {
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
	var catArgs []interface{}
	if !isGlobal {
		if categoryId == "" {
			catWhere = " AND (tabs.category_id = '' OR tabs.category_id IS NULL)"
		} else {
			catWhere = " AND tabs.category_id = ?"
			catArgs = append(catArgs, categoryId)
		}
	}

	// Count total with FTS5 join
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM tabs 
		INNER JOIN tabs_fts ON tabs.rowid = tabs_fts.rowid
		WHERE tabs_fts MATCH ?%s
	`, catWhere)

	countArgs := append([]interface{}{ftsQuery}, catArgs...)
	var total int
	if err := s.db.QueryRow(countQuery, countArgs...).Scan(&total); err != nil {
		// Fallback to LIKE query if FTS fails (e.g., special characters)
		return s.getTabsPaginatedLike(categoryId, page, pageSize, searchQuery, filterBy, isGlobal)
	}

	// Get paginated results
	offset := (page - 1) * pageSize
	limit := pageSize

	query := fmt.Sprintf(`
		SELECT tabs.id, tabs.title, tabs.artist, tabs.album, tabs.file_path, tabs.type, 
			   tabs.is_managed, tabs.cover_path, tabs.category_id, tabs.country, tabs.language, 
			   COALESCE(tabs.tag, '') 
		FROM tabs 
		INNER JOIN tabs_fts ON tabs.rowid = tabs_fts.rowid
		WHERE tabs_fts MATCH ?%s
		ORDER BY bm25(tabs_fts), tabs.title ASC 
		LIMIT ? OFFSET ?
	`, catWhere)

	queryArgs := append([]interface{}{ftsQuery}, catArgs...)
	queryArgs = append(queryArgs, limit, offset)

	rows, err := s.db.Query(query, queryArgs...)
	if err != nil {
		// Fallback to LIKE query if FTS fails
		return s.getTabsPaginatedLike(categoryId, page, pageSize, searchQuery, filterBy, isGlobal)
	}
	defer rows.Close()

	tabs := []Tab{}
	for rows.Next() {
		var t Tab
		var isManaged int
		if err := rows.Scan(&t.ID, &t.Title, &t.Artist, &t.Album, &t.FilePath, &t.Type, &isManaged, &t.CoverPath, &t.CategoryID, &t.Country, &t.Language, &t.Tag); err != nil {
			return nil, 0, err
		}
		t.IsManaged = isManaged == 1
		tabs = append(tabs, t)
	}

	return tabs, total, nil
}

// getTabsPaginatedLike is the fallback using LIKE (for special cases or when FTS fails)
func (s *DBStore) getTabsPaginatedLike(categoryId string, page, pageSize int, searchQuery string, filterBy []string, isGlobal bool) ([]Tab, int, error) {
	var whereClauses []string
	var args []interface{}

	// Category Filter
	if !isGlobal {
		if categoryId == "" {
			whereClauses = append(whereClauses, "(category_id = '' OR category_id IS NULL)")
		} else {
			whereClauses = append(whereClauses, "category_id = ?")
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

	// Count Total
	countQuery := "SELECT COUNT(*) FROM tabs " + whereSQL
	var total int
	if err := s.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Get Page Data
	offset := (page - 1) * pageSize
	limit := pageSize

	query := fmt.Sprintf(`
		SELECT id, title, artist, album, file_path, type, is_managed, cover_path, category_id, country, language, COALESCE(tag, '') 
		FROM tabs 
		%s 
		ORDER BY title ASC 
		LIMIT ? OFFSET ?
	`, whereSQL)

	queryArgs := append(args, limit, offset)

	rows, err := s.db.Query(query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	tabs := []Tab{}
	for rows.Next() {
		var t Tab
		var isManaged int
		if err := rows.Scan(&t.ID, &t.Title, &t.Artist, &t.Album, &t.FilePath, &t.Type, &isManaged, &t.CoverPath, &t.CategoryID, &t.Country, &t.Language, &t.Tag); err != nil {
			return nil, 0, err
		}
		t.IsManaged = isManaged == 1
		tabs = append(tabs, t)
	}

	return tabs, total, nil
}

func (s *DBStore) GetTab(id string) (*Tab, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var t Tab
	var isManaged int
	err := s.db.QueryRow(`
		SELECT id, title, artist, album, file_path, type, is_managed, cover_path, category_id, country, language, COALESCE(tag, '') 
		FROM tabs WHERE id = ?
	`, id).Scan(&t.ID, &t.Title, &t.Artist, &t.Album, &t.FilePath, &t.Type, &isManaged, &t.CoverPath, &t.CategoryID, &t.Country, &t.Language, &t.Tag)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	t.IsManaged = isManaged == 1
	return &t, nil
}

func (s *DBStore) AddTab(tab Tab) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	isManaged := 0
	if tab.IsManaged {
		isManaged = 1
	}

	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO tabs (id, title, artist, album, file_path, type, is_managed, cover_path, category_id, country, language, tag)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, tab.ID, tab.Title, tab.Artist, tab.Album, tab.FilePath, tab.Type, isManaged, tab.CoverPath, tab.CategoryID, tab.Country, tab.Language, tab.Tag)
	return err
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

func (s *DBStore) MoveTab(id, categoryID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec("UPDATE tabs SET category_id = ? WHERE id = ?", categoryID, id)
	return err
}

func (s *DBStore) GetTabByPath(filePath string) (*Tab, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var t Tab
	var isManaged int
	err := s.db.QueryRow(`
		SELECT id, title, artist, album, file_path, type, is_managed, cover_path, category_id, country, language, COALESCE(tag, '') 
		FROM tabs WHERE file_path = ?
	`, filePath).Scan(&t.ID, &t.Title, &t.Artist, &t.Album, &t.FilePath, &t.Type, &isManaged, &t.CoverPath, &t.CategoryID, &t.Country, &t.Language, &t.Tag)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	t.IsManaged = isManaged == 1
	return &t, nil
}

func (s *DBStore) GetTabByTitle(title string) (*Tab, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var t Tab
	var isManaged int
	err := s.db.QueryRow(`
		SELECT id, title, artist, album, file_path, type, is_managed, cover_path, category_id, country, language, COALESCE(tag, '') 
		FROM tabs WHERE title = ?
	`, title).Scan(&t.ID, &t.Title, &t.Artist, &t.Album, &t.FilePath, &t.Type, &isManaged, &t.CoverPath, &t.CategoryID, &t.Country, &t.Language, &t.Tag)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	t.IsManaged = isManaged == 1
	return &t, nil
}

// === Category Operations ===

func (s *DBStore) GetCategories() ([]Category, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.db.Query("SELECT id, name, parent_id FROM categories")
	if err != nil {
		return []Category{}, err
	}
	defer rows.Close()

	categories := []Category{}
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.ParentID); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, nil
}

func (s *DBStore) AddCategory(cat Category) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO categories (id, name, parent_id)
		VALUES (?, ?, ?)
	`, cat.ID, cat.Name, cat.ParentID)
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

	// Move tabs in this category to root
	if _, err := tx.Exec("UPDATE tabs SET category_id = '' WHERE category_id = ?", id); err != nil {
		return err
	}

	// Move sub-categories to root
	if _, err := tx.Exec("UPDATE categories SET parent_id = '' WHERE parent_id = ?", id); err != nil {
		return err
	}

	// Delete the category
	if _, err := tx.Exec("DELETE FROM categories WHERE id = ?", id); err != nil {
		return err
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
		"theme":                        settings.Theme,
		"background":                   settings.Background,
		"bgType":                       settings.BgType,
		"openMethod":                   settings.OpenMethod,
		"openGpMethod":                 settings.OpenGpMethod,
		"audioDevice":                  settings.AudioDevice,
		"autoSyncEnabled":              fmt.Sprintf("%v", settings.AutoSyncEnabled),
		"autoSyncFrequency":            settings.AutoSyncFrequency,
		"lastSyncTime":                 fmt.Sprintf("%d", settings.LastSyncTime),
		"syncStrategy":                 settings.SyncStrategy,
		"syncPaths":                    strings.Join(settings.SyncPaths, "|"),
		"keyBindings.scrollDown":       settings.KeyBindings.ScrollDown,
		"keyBindings.scrollUp":         settings.KeyBindings.ScrollUp,
		"keyBindings.metronome":        settings.KeyBindings.Metronome,
		"keyBindings.playPause":        settings.KeyBindings.PlayPause,
		"keyBindings.stop":             settings.KeyBindings.Stop,
		"keyBindings.bpmPlus":          settings.KeyBindings.BpmPlus,
		"keyBindings.bpmMinus":         settings.KeyBindings.BpmMinus,
		"keyBindings.toggleLoop":       settings.KeyBindings.ToggleLoop,
		"keyBindings.clearSelection":   settings.KeyBindings.ClearSelection,
		"keyBindings.jumpToBar":        settings.KeyBindings.JumpToBar,
		"keyBindings.jumpToStart":      settings.KeyBindings.JumpToStart,
		"keyBindings.autoScroll":       settings.KeyBindings.AutoScroll,
		"keyBindings.scrollSpeedUp":    settings.KeyBindings.ScrollSpeedUp,
		"keyBindings.scrollSpeedDown":  settings.KeyBindings.ScrollSpeedDown,
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
