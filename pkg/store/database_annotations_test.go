package store

import (
	"testing"
)

func TestDBStore_SaveTabAnnotation(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	// Add a tab first
	tab := Tab{ID: "tab1", Title: "Test", FilePath: "/test.gp5", Type: "gp5"}
	store.AddTab(tab)

	tests := []struct {
		name           string
		tabID          string
		pageNumber     int
		annotationData string
		wantErr        bool
	}{
		{
			name:           "save new annotation",
			tabID:          "tab1",
			pageNumber:     1,
			annotationData: `[{"type":"text","content":"note"}]`,
			wantErr:        false,
		},
		{
			name:           "update existing annotation",
			tabID:          "tab1",
			pageNumber:     1,
			annotationData: `[{"type":"text","content":"updated"}]`,
			wantErr:        false,
		},
		{
			name:           "save annotation for different page",
			tabID:          "tab1",
			pageNumber:     2,
			annotationData: `[{"type":"highlight"}]`,
			wantErr:        false,
		},
		{
			name:           "empty annotation data",
			tabID:          "tab1",
			pageNumber:     3,
			annotationData: "",
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.SaveTabAnnotation(tt.tabID, tt.pageNumber, tt.annotationData)
			if (err != nil) != tt.wantErr {
				t.Errorf("SaveTabAnnotation() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDBStore_GetTabAnnotation(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	tab := Tab{ID: "tab1", Title: "Test", FilePath: "/test.gp5", Type: "gp5"}
	store.AddTab(tab)

	// Save annotation
	annotationData := `[{"type":"text","content":"note"}]`
	store.SaveTabAnnotation("tab1", 1, annotationData)

	tests := []struct {
		name       string
		tabID      string
		pageNumber int
		want       string
		wantErr    bool
	}{
		{
			name:       "get existing annotation",
			tabID:      "tab1",
			pageNumber: 1,
			want:       annotationData,
			wantErr:    false,
		},
		{
			name:       "get non-existent annotation returns empty array",
			tabID:      "tab1",
			pageNumber: 999,
			want:       "[]",
			wantErr:    false,
		},
		{
			name:       "get annotation for non-existent tab",
			tabID:      "nonexistent",
			pageNumber: 1,
			want:       "[]",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := store.GetTabAnnotation(tt.tabID, tt.pageNumber)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTabAnnotation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetTabAnnotation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDBStore_DeleteTabAnnotations(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	tab := Tab{ID: "tab1", Title: "Test", FilePath: "/test.gp5", Type: "gp5"}
	store.AddTab(tab)

	// Save multiple annotations
	store.SaveTabAnnotation("tab1", 1, `[{"type":"text"}]`)
	store.SaveTabAnnotation("tab1", 2, `[{"type":"highlight"}]`)

	// Delete all annotations
	err := store.DeleteTabAnnotations("tab1")
	if err != nil {
		t.Fatalf("DeleteTabAnnotations() error = %v", err)
	}

	// Verify annotations are deleted
	data1, _ := store.GetTabAnnotation("tab1", 1)
	if data1 != "[]" {
		t.Errorf("Expected empty annotation after delete, got %v", data1)
	}

	data2, _ := store.GetTabAnnotation("tab1", 2)
	if data2 != "[]" {
		t.Errorf("Expected empty annotation after delete, got %v", data2)
	}
}

func TestDBStore_DeleteTabAnnotations_NonExistentTab(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	// Should not error when deleting annotations for non-existent tab
	err := store.DeleteTabAnnotations("nonexistent")
	if err != nil {
		t.Errorf("DeleteTabAnnotations() for non-existent tab error = %v", err)
	}
}
