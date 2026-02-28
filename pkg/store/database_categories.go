package store

import (
	"database/sql"
)

// === Category Operations ===

func (s *DBStore) GetCategories() ([]Category, error) {
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

func (s *DBStore) AddCategory(cat Category) error {
	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO categories (id, name, parent_id, cover_path)
		VALUES (?, ?, ?, ?)
	`, cat.ID, cat.Name, cat.ParentID, cat.CoverPath)
	return err
}

func (s *DBStore) DeleteCategory(id string) error {
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
	_, err := s.db.Exec("UPDATE categories SET parent_id = ? WHERE id = ?", newParentID, id)
	return err
}

// EnsureCloudCategory creates or updates the system cloud category
func (s *DBStore) EnsureCloudCategory() error {
	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO categories (id, name, parent_id, cover_path)
		VALUES (?, ?, '', '')
	`, SystemCloudCategoryID, "Cloud Storage")
	return err
}

// GetCategory retrieves a single category by ID
func (s *DBStore) GetCategory(id string) (*Category, error) {
	var c Category
	err := s.db.QueryRow(`
		SELECT c.id, c.name, c.parent_id, c.cover_path,
		COALESCE(NULLIF(c.cover_path, ''), (SELECT cover_path FROM tabs WHERE category_id = c.id ORDER BY added_at ASC LIMIT 1), '') as effective_cover_path
		FROM categories c WHERE c.id = ?
	`, id).Scan(&c.ID, &c.Name, &c.ParentID, &c.CoverPath, &c.EffectiveCoverPath)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}
