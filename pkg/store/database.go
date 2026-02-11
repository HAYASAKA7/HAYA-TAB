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

	_, err := s.db.Exec(schema)
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

	// 1. Build Base Query & Args
	var whereClauses []string
	var args []interface{}

	// Category Filter
	if !isGlobal {
		// If categoryId is empty, we might want root tabs?
		// Current app logic seems to imply categoryId="" means root.
		// However, if the user selects "All Tabs" (global), we skip this.
		// Let's follow the app.go logic:
		// if (categoryId == "" && tab.CategoryID == "") || tab.CategoryID == categoryId
		if categoryId == "" {
			whereClauses = append(whereClauses, "(category_id = '' OR category_id IS NULL)")
		} else {
			whereClauses = append(whereClauses, "category_id = ?")
			args = append(args, categoryId)
		}
	}

	// Search Filter
	if searchQuery != "" && len(filterBy) > 0 {
		var searchConditions []string
		term := "%" + searchQuery + "%"
		for _, field := range filterBy {
			// Sanitize field name to prevent SQL injection (though these come from code, safe to check)
			switch field {
			case "title", "artist", "album", "tag":
				searchConditions = append(searchConditions, fmt.Sprintf("%s LIKE ?", field))
				args = append(args, term)
			}
		}
		if len(searchConditions) > 0 {
			whereClauses = append(whereClauses, "("+strings.Join(searchConditions, " OR ")+")")
		}
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// 2. Count Total
	countQuery := "SELECT COUNT(*) FROM tabs " + whereSQL
	var total int
	if err := s.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// 3. Get Page Data
	offset := (page - 1) * pageSize
	limit := pageSize

	query := fmt.Sprintf(`
		SELECT id, title, artist, album, file_path, type, is_managed, cover_path, category_id, country, language, COALESCE(tag, '') 
		FROM tabs 
		%s 
		ORDER BY title ASC 
		LIMIT ? OFFSET ?
	`, whereSQL)

	// Append limit/offset args
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
		"theme":             settings.Theme,
		"background":        settings.Background,
		"bgType":            settings.BgType,
		"openMethod":        settings.OpenMethod,
		"openGpMethod":      settings.OpenGpMethod,
		"audioDevice":       settings.AudioDevice,
		"autoSyncEnabled":   fmt.Sprintf("%v", settings.AutoSyncEnabled),
		"autoSyncFrequency": settings.AutoSyncFrequency,
		"lastSyncTime":      fmt.Sprintf("%d", settings.LastSyncTime),
		"syncStrategy":      settings.SyncStrategy,
		"syncPaths":         strings.Join(settings.SyncPaths, "|"),
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
