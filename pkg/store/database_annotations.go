package store

import (
	"database/sql"
	"time"
)

// SaveTabAnnotation saves annotation JSON for a given tab/page.
func (s *DBStore) SaveTabAnnotation(tabID string, pageNumber int, annotationData string) error {
	_, err := s.db.Exec(`
		INSERT INTO tab_annotations (tab_id, page_number, annotation_data, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(tab_id, page_number) DO UPDATE SET
			annotation_data = excluded.annotation_data,
			updated_at = excluded.updated_at
	`, tabID, pageNumber, annotationData, time.Now().Unix())
	return err
}

// GetTabAnnotation returns annotation JSON for a given tab/page.
// Returns "[]" when no annotation exists.
func (s *DBStore) GetTabAnnotation(tabID string, pageNumber int) (string, error) {
	var annotationData string
	err := s.db.QueryRow(
		"SELECT annotation_data FROM tab_annotations WHERE tab_id = ? AND page_number = ?",
		tabID,
		pageNumber,
	).Scan(&annotationData)
	if err != nil {
		if err == sql.ErrNoRows {
			return "[]", nil
		}
		return "", err
	}
	if annotationData == "" {
		return "[]", nil
	}
	return annotationData, nil
}

// DeleteTabAnnotations removes all annotations for a tab.
func (s *DBStore) DeleteTabAnnotations(tabID string) error {
	_, err := s.db.Exec("DELETE FROM tab_annotations WHERE tab_id = ?", tabID)
	return err
}
