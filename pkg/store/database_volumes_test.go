package store

import (
	"testing"
	"time"
)

func TestDBStore_AddVolume(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	volume := CloudVolume{
		ID:              "vol-1",
		Name:            "Test Volume",
		MountPath:       "/mnt/test",
		FingerprintPath: ".volume-id",
		CreatedAt:       time.Now().Unix(),
		LastSeenAt:      time.Now().Unix(),
		IsAvailable:     true,
	}

	err := store.AddVolume(volume)
	if err != nil {
		t.Fatalf("AddVolume() error = %v", err)
	}

	// Verify volume was added
	retrieved, err := store.GetVolume("vol-1")
	if err != nil {
		t.Fatalf("GetVolume() error = %v", err)
	}
	if retrieved == nil {
		t.Fatal("GetVolume() returned nil")
	}
	if retrieved.Name != "Test Volume" {
		t.Errorf("Name = %q, want %q", retrieved.Name, "Test Volume")
	}
}

func TestDBStore_GetVolume(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	volume := CloudVolume{
		ID:              "vol-1",
		Name:            "Test Volume",
		MountPath:       "/mnt/test",
		FingerprintPath: ".volume-id",
		CreatedAt:       1000,
		LastSeenAt:      2000,
		IsAvailable:     true,
	}
	store.AddVolume(volume)

	tests := []struct {
		name    string
		id      string
		wantNil bool
	}{
		{
			name:    "existing volume",
			id:      "vol-1",
			wantNil: false,
		},
		{
			name:    "non-existent volume",
			id:      "vol-999",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := store.GetVolume(tt.id)
			if err != nil {
				t.Fatalf("GetVolume() error = %v", err)
			}
			if (got == nil) != tt.wantNil {
				t.Errorf("GetVolume() = %v, wantNil = %v", got, tt.wantNil)
			}
		})
	}
}

func TestDBStore_GetVolumeByMountPath(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	volume := CloudVolume{
		ID:              "vol-1",
		Name:            "Test Volume",
		MountPath:       "/mnt/cloud",
		FingerprintPath: ".volume-id",
		CreatedAt:       1000,
		LastSeenAt:      2000,
		IsAvailable:     true,
	}
	store.AddVolume(volume)

	tests := []struct {
		name      string
		mountPath string
		wantNil   bool
		wantID    string
	}{
		{
			name:      "existing mount path",
			mountPath: "/mnt/cloud",
			wantNil:   false,
			wantID:    "vol-1",
		},
		{
			name:      "non-existent mount path",
			mountPath: "/mnt/nonexistent",
			wantNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := store.GetVolumeByMountPath(tt.mountPath)
			if err != nil {
				t.Fatalf("GetVolumeByMountPath() error = %v", err)
			}
			if (got == nil) != tt.wantNil {
				t.Errorf("GetVolumeByMountPath() = %v, wantNil = %v", got, tt.wantNil)
			}
			if got != nil && got.ID != tt.wantID {
				t.Errorf("GetVolumeByMountPath() ID = %v, want %v", got.ID, tt.wantID)
			}
		})
	}
}

func TestDBStore_GetAllVolumes(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	// Initially empty
	volumes, err := store.GetAllVolumes()
	if err != nil {
		t.Fatalf("GetAllVolumes() error = %v", err)
	}
	if len(volumes) != 0 {
		t.Errorf("Expected 0 volumes initially, got %d", len(volumes))
	}

	// Add volumes
	store.AddVolume(CloudVolume{ID: "vol-1", Name: "Volume A", MountPath: "/a"})
	store.AddVolume(CloudVolume{ID: "vol-2", Name: "Volume B", MountPath: "/b"})

	volumes, err = store.GetAllVolumes()
	if err != nil {
		t.Fatalf("GetAllVolumes() error = %v", err)
	}
	if len(volumes) != 2 {
		t.Errorf("Expected 2 volumes, got %d", len(volumes))
	}
}

func TestDBStore_GetActiveVolumes(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	// Add available and unavailable volumes
	store.AddVolume(CloudVolume{ID: "vol-1", Name: "Available", MountPath: "/a", IsAvailable: true, LastSeenAt: 1000})
	store.AddVolume(CloudVolume{ID: "vol-2", Name: "Unavailable", MountPath: "/b", IsAvailable: false})
	store.AddVolume(CloudVolume{ID: "vol-3", Name: "Also Available", MountPath: "/c", IsAvailable: true, LastSeenAt: 2000})

	volumes, err := store.GetActiveVolumes()
	if err != nil {
		t.Fatalf("GetActiveVolumes() error = %v", err)
	}
	if len(volumes) != 2 {
		t.Errorf("Expected 2 active volumes, got %d", len(volumes))
	}
}

func TestDBStore_GetActiveVolumes_DeduplicatesByMountPath(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	// Add two volumes with same mount path, different timestamps
	store.AddVolume(CloudVolume{ID: "vol-old", Name: "Old", MountPath: "/same", IsAvailable: true, LastSeenAt: 1000})
	store.AddVolume(CloudVolume{ID: "vol-new", Name: "New", MountPath: "/same", IsAvailable: true, LastSeenAt: 2000})

	volumes, err := store.GetActiveVolumes()
	if err != nil {
		t.Fatalf("GetActiveVolumes() error = %v", err)
	}
	if len(volumes) != 1 {
		t.Errorf("Expected 1 volume (deduplicated), got %d", len(volumes))
	}
	// Should return the most recent one
	if len(volumes) > 0 && volumes[0].ID != "vol-new" {
		t.Errorf("Expected most recent volume vol-new, got %s", volumes[0].ID)
	}
}

func TestDBStore_MarkAllVolumesUnavailable(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	store.AddVolume(CloudVolume{ID: "vol-1", Name: "A", MountPath: "/a", IsAvailable: true})
	store.AddVolume(CloudVolume{ID: "vol-2", Name: "B", MountPath: "/b", IsAvailable: true})

	err := store.MarkAllVolumesUnavailable()
	if err != nil {
		t.Fatalf("MarkAllVolumesUnavailable() error = %v", err)
	}

	// All volumes should be unavailable now
	volumes, _ := store.GetActiveVolumes()
	if len(volumes) != 0 {
		t.Errorf("Expected 0 active volumes after marking unavailable, got %d", len(volumes))
	}
}

func TestDBStore_UpdateVolume(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	store.AddVolume(CloudVolume{ID: "vol-1", Name: "Original", MountPath: "/orig"})

	updatedVolume := CloudVolume{
		ID:              "vol-1",
		Name:            "Updated",
		MountPath:       "/updated",
		FingerprintPath: ".new-fingerprint",
		LastSeenAt:      time.Now().Unix(),
		IsAvailable:     true,
	}

	err := store.UpdateVolume(updatedVolume)
	if err != nil {
		t.Fatalf("UpdateVolume() error = %v", err)
	}

	retrieved, _ := store.GetVolume("vol-1")
	if retrieved.Name != "Updated" {
		t.Errorf("Name = %q, want %q", retrieved.Name, "Updated")
	}
	if retrieved.MountPath != "/updated" {
		t.Errorf("MountPath = %q, want %q", retrieved.MountPath, "/updated")
	}
}

func TestDBStore_UpdateVolumeMountPath(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	store.AddVolume(CloudVolume{ID: "vol-1", Name: "Test", MountPath: "/old"})

	err := store.UpdateVolumeMountPath("vol-1", "/new")
	if err != nil {
		t.Fatalf("UpdateVolumeMountPath() error = %v", err)
	}

	retrieved, _ := store.GetVolume("vol-1")
	if retrieved.MountPath != "/new" {
		t.Errorf("MountPath = %q, want %q", retrieved.MountPath, "/new")
	}
}

func TestDBStore_MarkVolumeAvailable(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	store.AddVolume(CloudVolume{ID: "vol-1", Name: "Test", MountPath: "/test", IsAvailable: false})

	// Mark available
	err := store.MarkVolumeAvailable("vol-1", true)
	if err != nil {
		t.Fatalf("MarkVolumeAvailable() error = %v", err)
	}

	retrieved, _ := store.GetVolume("vol-1")
	if !retrieved.IsAvailable {
		t.Error("Expected volume to be available")
	}

	// Mark unavailable
	err = store.MarkVolumeAvailable("vol-1", false)
	if err != nil {
		t.Fatalf("MarkVolumeAvailable() error = %v", err)
	}

	retrieved, _ = store.GetVolume("vol-1")
	if retrieved.IsAvailable {
		t.Error("Expected volume to be unavailable")
	}
}

func TestDBStore_DeleteVolume(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	// Add volume with associated tabs
	store.AddVolume(CloudVolume{ID: "vol-1", Name: "Test", MountPath: "/test"})
	store.AddTab(Tab{ID: "tab1", Title: "Tab 1", FilePath: "/test/a.gp5", Type: "gp5", VolumeID: "vol-1"})
	store.AddTab(Tab{ID: "tab2", Title: "Tab 2", FilePath: "/test/b.gp5", Type: "gp5", VolumeID: "vol-1"})

	err := store.DeleteVolume("vol-1")
	if err != nil {
		t.Fatalf("DeleteVolume() error = %v", err)
	}

	// Volume should be deleted
	retrieved, _ := store.GetVolume("vol-1")
	if retrieved != nil {
		t.Error("Expected volume to be deleted")
	}

	// Associated tabs should be deleted
	tabs, _ := store.GetTabsByVolume("vol-1")
	if len(tabs) != 0 {
		t.Errorf("Expected 0 tabs after volume deletion, got %d", len(tabs))
	}
}

func TestDBStore_GetTabsByVolume(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	store.AddVolume(CloudVolume{ID: "vol-1", Name: "Test", MountPath: "/test"})
	store.AddTab(Tab{ID: "tab1", Title: "Song A", FilePath: "/test/a.gp5", Type: "gp5", VolumeID: "vol-1"})
	store.AddTab(Tab{ID: "tab2", Title: "Song B", FilePath: "/test/b.gp5", Type: "gp5", VolumeID: "vol-1"})
	store.AddTab(Tab{ID: "tab3", Title: "Other", FilePath: "/other/c.gp5", Type: "gp5", VolumeID: "vol-2"})

	tabs, err := store.GetTabsByVolume("vol-1")
	if err != nil {
		t.Fatalf("GetTabsByVolume() error = %v", err)
	}
	if len(tabs) != 2 {
		t.Errorf("Expected 2 tabs for vol-1, got %d", len(tabs))
	}
}

func TestDBStore_GetTabByVolumeAndPath(t *testing.T) {
	store, tmpDir := setupTestDB(t)
	defer cleanupTestDB(store, tmpDir)

	store.AddVolume(CloudVolume{ID: "vol-1", Name: "Test", MountPath: "/test"})
	store.AddTab(Tab{ID: "tab1", Title: "Song A", FilePath: "a.gp5", Type: "gp5", VolumeID: "vol-1"})

	tests := []struct {
		name       string
		volumeID   string
		path       string
		wantNil    bool
		wantTabID  string
	}{
		{
			name:      "existing tab",
			volumeID:  "vol-1",
			path:      "a.gp5",
			wantNil:   false,
			wantTabID: "tab1",
		},
		{
			name:     "non-existent path",
			volumeID: "vol-1",
			path:     "nonexistent.gp5",
			wantNil:  true,
		},
		{
			name:     "non-existent volume",
			volumeID: "vol-999",
			path:     "a.gp5",
			wantNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := store.GetTabByVolumeAndPath(tt.volumeID, tt.path)
			if err != nil {
				t.Fatalf("GetTabByVolumeAndPath() error = %v", err)
			}
			if (got == nil) != tt.wantNil {
				t.Errorf("GetTabByVolumeAndPath() = %v, wantNil = %v", got, tt.wantNil)
			}
			if got != nil && got.ID != tt.wantTabID {
				t.Errorf("GetTabByVolumeAndPath() ID = %v, want %v", got.ID, tt.wantTabID)
			}
		})
	}
}

func TestBoolToInt(t *testing.T) {
	tests := []struct {
		input    bool
		expected int
	}{
		{true, 1},
		{false, 0},
	}

	for _, tt := range tests {
		got := boolToInt(tt.input)
		if got != tt.expected {
			t.Errorf("boolToInt(%v) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestIntToBool(t *testing.T) {
	tests := []struct {
		input    int
		expected bool
	}{
		{0, false},
		{1, true},
		{2, true},
		{-1, true},
	}

	for _, tt := range tests {
		got := intToBool(tt.input)
		if got != tt.expected {
			t.Errorf("intToBool(%d) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}
