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
	mu       sync.RWMutex // Protects Settings only; DB operations rely on database/sql pool + WAL
	db       *sql.DB
	dbPath   string
	Settings Settings
}

func NewDBStore(dbPath string) *DBStore {
	return &DBStore{
		dbPath: dbPath,
		Settings: Settings{
			Theme:        "system",
			Language:     DetectSystemLocale(),
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

	// Add is_cloud column for online tabs feature
	_, err = s.db.Exec("ALTER TABLE tabs ADD COLUMN is_cloud INTEGER DEFAULT 0")
	if err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			// It's okay
		}
	}

	// Add origin_country column for artist's origin country from MusicBrainz
	_, err = s.db.Exec("ALTER TABLE tabs ADD COLUMN origin_country TEXT DEFAULT ''")
	if err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			// It's okay
		}
	}

	// Add initial_az column for A-Z quick jump (EN/ZH UI)
	_, err = s.db.Exec("ALTER TABLE tabs ADD COLUMN initial_az TEXT DEFAULT '#'")
	if err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			// It's okay
		}
	}

	// Add initial_kana column for Kana quick jump (JA UI)
	_, err = s.db.Exec("ALTER TABLE tabs ADD COLUMN initial_kana TEXT DEFAULT '#'")
	if err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			// It's okay
		}
	}

	return nil
}

func (s *DBStore) loadSettings() error {
	// Note: This method is called from Initialize() which already holds the lock
	// Do NOT acquire lock here to avoid deadlock (Go mutexes are not reentrant)
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
	if v, ok := settings["language"]; ok {
		s.Settings.Language = v
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
	if v, ok := settings["storagePath"]; ok {
		s.Settings.StoragePath = v
	}
	if v, ok := settings["coversPath"]; ok {
		s.Settings.CoversPath = v
	}

	// WebDAV Settings
	if v, ok := settings["webdavEnabled"]; ok {
		s.Settings.WebDAVEnabled = (v == "true")
	}
	if v, ok := settings["webdavUrl"]; ok {
		s.Settings.WebDAVURL = v
	}
	if v, ok := settings["webdavUser"]; ok {
		s.Settings.WebDAVUser = v
	}
	if v, ok := settings["webdavPassword"]; ok {
		// Try to decrypt, fallback to raw value if failed (migration)
		if decrypted, err := Decrypt(v); err == nil {
			s.Settings.WebDAVPassword = decrypted
		} else {
			s.Settings.WebDAVPassword = v
		}
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
	// sql.DB.Close() is thread-safe, no mutex needed
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// HasData checks if the database has any data
func (s *DBStore) HasData() bool {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM tabs").Scan(&count)
	if err != nil {
		return false
	}
	return count > 0
}
