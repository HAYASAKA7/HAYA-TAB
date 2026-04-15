package sync

import (
	"testing"

	"haya-tab/pkg/store"
)

func TestCalculateBucketNumber(t *testing.T) {
	tests := []struct {
		path string
		want int
	}{
		{"file1.gp5", -1}, // We don't know exact value, just check range
		{"file2.pdf", -1},
		{"dir/file.gp", -1},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := CalculateBucketNumber(tt.path)
			if got < 0 || got >= BucketCount {
				t.Errorf("CalculateBucketNumber(%q) = %d, want 0-%d", tt.path, got, BucketCount-1)
			}
		})
	}
}

func TestCalculateBucketNumber_Consistency(t *testing.T) {
	path := "test/file.gp5"
	bucket1 := CalculateBucketNumber(path)
	bucket2 := CalculateBucketNumber(path)
	
	if bucket1 != bucket2 {
		t.Errorf("CalculateBucketNumber not consistent: %d != %d", bucket1, bucket2)
	}
}

func TestGetMetadataPath(t *testing.T) {
	tests := []struct {
		volumePath string
		want       string
	}{
		{"/mnt/volume", "/mnt/volume/haya-metadata"},
		{"/root", "/root/haya-metadata"},
		{"volume", "volume/haya-metadata"},
	}

	for _, tt := range tests {
		t.Run(tt.volumePath, func(t *testing.T) {
			got := getMetadataPath(tt.volumePath)
			if got != tt.want {
				t.Errorf("getMetadataPath(%q) = %q, want %q", tt.volumePath, got, tt.want)
			}
		})
	}
}

func TestGetBucketPath(t *testing.T) {
	tests := []struct {
		volumePath string
		bucketNum  int
		want       string
	}{
		{"/mnt/volume", 0, "/mnt/volume/haya-metadata/bucket-00.json"},
		{"/mnt/volume", 15, "/mnt/volume/haya-metadata/bucket-15.json"},
		{"/root", 5, "/root/haya-metadata/bucket-05.json"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := getBucketPath(tt.volumePath, tt.bucketNum)
			if got != tt.want {
				t.Errorf("getBucketPath(%q, %d) = %q, want %q", tt.volumePath, tt.bucketNum, got, tt.want)
			}
		})
	}
}

func TestGetLegacyFingerprintPath(t *testing.T) {
	tests := []struct {
		volumePath string
		want       string
	}{
		{"/mnt/volume", "/mnt/volume/.haya-volume-fingerprint"},
		{"/root", "/root/.haya-volume-fingerprint"},
	}

	for _, tt := range tests {
		t.Run(tt.volumePath, func(t *testing.T) {
			got := getLegacyFingerprintPath(tt.volumePath)
			if got != tt.want {
				t.Errorf("getLegacyFingerprintPath(%q) = %q, want %q", tt.volumePath, got, tt.want)
			}
		})
	}
}


func TestRegisterOrUpdateVolume_AddsNewVolume(t *testing.T) {
	_, db, _, cleanup := setupTestSyncService(t)
	defer cleanup()

	fingerprint := &VolumeFingerprint{
		VolumeID:   "vol-1",
		VolumeName: "Cloud One",
	}

	volume, err := RegisterOrUpdateVolume(db, "/remote/cloud-one", fingerprint)
	if err != nil {
		t.Fatalf("RegisterOrUpdateVolume() error = %v", err)
	}

	if volume.ID != "vol-1" {
		t.Errorf("volume.ID = %q, want %q", volume.ID, "vol-1")
	}
	if volume.MountPath != "/remote/cloud-one" {
		t.Errorf("volume.MountPath = %q, want %q", volume.MountPath, "/remote/cloud-one")
	}
	if volume.FingerprintPath != "/remote/cloud-one/haya-metadata" {
		t.Errorf("volume.FingerprintPath = %q, want %q", volume.FingerprintPath, "/remote/cloud-one/haya-metadata")
	}
	if !volume.IsAvailable {
		t.Error("expected new volume to be available")
	}

	stored, err := db.GetVolume("vol-1")
	if err != nil {
		t.Fatalf("GetVolume() error = %v", err)
	}
	if stored == nil {
		t.Fatal("expected volume to be stored in database")
	}
}

func TestRegisterOrUpdateVolume_UpdatesExistingVolume(t *testing.T) {
	_, db, _, cleanup := setupTestSyncService(t)
	defer cleanup()

	// Use a known old timestamp to avoid timing issues
	oldTimestamp := int64(1000)

	existing := store.CloudVolume{
		ID:              "vol-1",
		Name:            "Cloud One",
		MountPath:       "/old/path",
		FingerprintPath: "/old/path/haya-metadata",
		CreatedAt:       oldTimestamp,
		LastSeenAt:      oldTimestamp,
		IsAvailable:     false,
	}
	if err := db.AddVolume(existing); err != nil {
		t.Fatalf("AddVolume() error = %v", err)
	}

	fingerprint := &VolumeFingerprint{
		VolumeID:   "vol-1",
		VolumeName: "Cloud One",
	}

	volume, err := RegisterOrUpdateVolume(db, "/new/path", fingerprint)
	if err != nil {
		t.Fatalf("RegisterOrUpdateVolume() error = %v", err)
	}

	if volume.MountPath != "/new/path" {
		t.Errorf("volume.MountPath = %q, want %q", volume.MountPath, "/new/path")
	}
	if volume.FingerprintPath != "/new/path/haya-metadata" {
		t.Errorf("volume.FingerprintPath = %q, want %q", volume.FingerprintPath, "/new/path/haya-metadata")
	}
	if !volume.IsAvailable {
		t.Error("expected updated volume to be available")
	}
	// New timestamp should be greater than the known old timestamp
	if volume.LastSeenAt <= oldTimestamp {
		t.Errorf("volume.LastSeenAt = %d, want > %d", volume.LastSeenAt, oldTimestamp)
	}
}

// === FingerprintFile ===

func TestFingerprintFile_Clone(t *testing.T) {
	original := &FingerprintFile{
		RelativePath: "songs/test.pdf",
		Title:        "Test Song",
		Artist:       "Test Artist",
		Album:        "Test Album",
		Type:         "pdf",
		Categories:   []string{"rock", "jazz"},
		UploadedAt:   "2024-01-01T00:00:00Z",
		UploadedBy:   "device-1",
	}

	clone := original.Clone()

	// Values should be equal
	if clone.RelativePath != original.RelativePath {
		t.Errorf("RelativePath: got %q, want %q", clone.RelativePath, original.RelativePath)
	}
	if clone.Title != original.Title {
		t.Errorf("Title: got %q, want %q", clone.Title, original.Title)
	}
	if len(clone.Categories) != len(original.Categories) {
		t.Fatalf("Categories length: got %d, want %d", len(clone.Categories), len(original.Categories))
	}
	for i, c := range original.Categories {
		if clone.Categories[i] != c {
			t.Errorf("Categories[%d]: got %q, want %q", i, clone.Categories[i], c)
		}
	}

	// Mutating the clone's slice should not affect the original
	clone.Categories[0] = "mutated"
	if original.Categories[0] == "mutated" {
		t.Error("Clone did not deep-copy Categories slice — original was mutated")
	}
}

func TestFingerprintFile_Clone_NilCategories(t *testing.T) {
	original := &FingerprintFile{
		RelativePath: "test.pdf",
		Categories:   nil,
	}
	clone := original.Clone()
	if clone.Categories == nil {
		t.Error("Clone with nil categories: expected empty slice, got nil")
	}
	if len(clone.Categories) != 0 {
		t.Errorf("Clone with nil categories: len = %d, want 0", len(clone.Categories))
	}
}

func TestFingerprintFile_IsNewerThan(t *testing.T) {
	older := &FingerprintFile{UploadedAt: "2024-01-01T00:00:00Z"}
	newer := &FingerprintFile{UploadedAt: "2024-06-01T00:00:00Z"}
	empty := &FingerprintFile{UploadedAt: ""}
	invalid := &FingerprintFile{UploadedAt: "not-a-date"}

	tests := []struct {
		name  string
		f     *FingerprintFile
		other *FingerprintFile
		want  bool
	}{
		{"newer > older", newer, older, true},
		{"older < newer", older, newer, false},
		{"same timestamp", older, older, false},
		{"empty self returns false", empty, older, false},
		{"empty other returns true (self wins)", newer, empty, true},
		{"both empty returns false", empty, empty, false},
		{"invalid timestamps return false", invalid, older, false},
		{"self invalid, other invalid", invalid, invalid, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.f.IsNewerThan(tt.other)
			if got != tt.want {
				t.Errorf("IsNewerThan() = %v, want %v", got, tt.want)
			}
		})
	}
}

// === BucketData ===

func TestBucketData_Clone(t *testing.T) {
	original := &BucketData{
		BucketNumber: 3,
		ETag:         "abc123",
		Files: []FingerprintFile{
			{RelativePath: "a.pdf", Title: "A"},
			{RelativePath: "b.pdf", Title: "B"},
		},
	}

	clone := original.Clone()

	if clone.BucketNumber != original.BucketNumber {
		t.Errorf("BucketNumber: got %d, want %d", clone.BucketNumber, original.BucketNumber)
	}
	if len(clone.Files) != len(original.Files) {
		t.Fatalf("Files length: got %d, want %d", len(clone.Files), len(original.Files))
	}

	// Mutating the clone's slice should not affect the original
	clone.Files[0].Title = "mutated"
	if original.Files[0].Title == "mutated" {
		t.Error("Clone did not copy Files — original was mutated")
	}
}

func TestBucketData_Merge_NewFilesAdded(t *testing.T) {
	base := &BucketData{
		BucketNumber: 0,
		Files: []FingerprintFile{
			{RelativePath: "a.pdf", Title: "A", UploadedAt: "2024-01-01T00:00:00Z"},
		},
	}
	other := &BucketData{
		BucketNumber: 0,
		Files: []FingerprintFile{
			{RelativePath: "b.pdf", Title: "B", UploadedAt: "2024-01-02T00:00:00Z"},
		},
	}

	merged := base.Merge(other)

	if len(merged.Files) != 2 {
		t.Fatalf("expected 2 files after merge, got %d", len(merged.Files))
	}
}

func TestBucketData_Merge_NewerVersionWins(t *testing.T) {
	base := &BucketData{
		Files: []FingerprintFile{
			{RelativePath: "a.pdf", Title: "OldTitle", UploadedAt: "2024-01-01T00:00:00Z"},
		},
	}
	other := &BucketData{
		Files: []FingerprintFile{
			{RelativePath: "a.pdf", Title: "NewTitle", UploadedAt: "2024-06-01T00:00:00Z"},
		},
	}

	merged := base.Merge(other)

	if len(merged.Files) != 1 {
		t.Fatalf("expected 1 file after conflict merge, got %d", len(merged.Files))
	}
	if merged.Files[0].Title != "NewTitle" {
		t.Errorf("Title = %q, want NewTitle (newer version should win)", merged.Files[0].Title)
	}
}

func TestBucketData_Merge_OlderVersionIgnored(t *testing.T) {
	base := &BucketData{
		Files: []FingerprintFile{
			{RelativePath: "a.pdf", Title: "NewTitle", UploadedAt: "2024-06-01T00:00:00Z"},
		},
	}
	other := &BucketData{
		Files: []FingerprintFile{
			{RelativePath: "a.pdf", Title: "OldTitle", UploadedAt: "2024-01-01T00:00:00Z"},
		},
	}

	merged := base.Merge(other)

	if len(merged.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(merged.Files))
	}
	if merged.Files[0].Title != "NewTitle" {
		t.Errorf("Title = %q, want NewTitle (base is newer)", merged.Files[0].Title)
	}
}

func TestBucketData_Merge_DoesNotMutateBase(t *testing.T) {
	base := &BucketData{
		Files: []FingerprintFile{
			{RelativePath: "a.pdf", Title: "Original"},
		},
	}
	other := &BucketData{
		Files: []FingerprintFile{
			{RelativePath: "b.pdf", Title: "New"},
		},
	}

	_ = base.Merge(other)

	if len(base.Files) != 1 {
		t.Errorf("Merge mutated base: base.Files length = %d, want 1", len(base.Files))
	}
}

// === MergeFingerprintFiles ===

func TestMergeFingerprintFiles_AddsRemoteOnlyFiles(t *testing.T) {
	local := []FingerprintFile{
		{RelativePath: "a.pdf", UploadedAt: "2024-01-01T00:00:00Z"},
	}
	remote := []FingerprintFile{
		{RelativePath: "b.pdf", UploadedAt: "2024-01-02T00:00:00Z"},
	}

	result := MergeFingerprintFiles(local, remote, nil)

	if len(result) != 2 {
		t.Fatalf("expected 2 files, got %d", len(result))
	}
}

func TestMergeFingerprintFiles_RemoteNewerWins(t *testing.T) {
	local := []FingerprintFile{
		{RelativePath: "a.pdf", Title: "Local", UploadedAt: "2024-01-01T00:00:00Z"},
	}
	remote := []FingerprintFile{
		{RelativePath: "a.pdf", Title: "Remote", UploadedAt: "2024-06-01T00:00:00Z"},
	}

	result := MergeFingerprintFiles(local, remote, nil)

	if len(result) != 1 {
		t.Fatalf("expected 1 file, got %d", len(result))
	}
	if result[0].Title != "Remote" {
		t.Errorf("Title = %q, want Remote (remote is newer)", result[0].Title)
	}
}

func TestMergeFingerprintFiles_LocalNewerKept(t *testing.T) {
	local := []FingerprintFile{
		{RelativePath: "a.pdf", Title: "Local", UploadedAt: "2024-06-01T00:00:00Z"},
	}
	remote := []FingerprintFile{
		{RelativePath: "a.pdf", Title: "Remote", UploadedAt: "2024-01-01T00:00:00Z"},
	}

	result := MergeFingerprintFiles(local, remote, nil)

	if result[0].Title != "Local" {
		t.Errorf("Title = %q, want Local (local is newer)", result[0].Title)
	}
}

func TestMergeFingerprintFiles_TombstonesExcludeRemoteFiles(t *testing.T) {
	// Tombstones prevent remote files from being merged in, and prevent
	// local tombstoned files from being used as a conflict-resolution base.
	// However, they do NOT remove already-present entries from the local slice.
	local := []FingerprintFile{
		// b.pdf is NOT tombstoned — it stays
		{RelativePath: "b.pdf", Title: "B"},
	}
	remote := []FingerprintFile{
		// a.pdf is tombstoned — remote copy must NOT be added
		{RelativePath: "a.pdf", Title: "A-remote"},
		// c.pdf is not tombstoned — it gets added
		{RelativePath: "c.pdf", Title: "C"},
	}
	tombstones := map[string]bool{
		"a.pdf": true,
	}

	result := MergeFingerprintFiles(local, remote, tombstones)

	// a.pdf from remote should be excluded; b.pdf (local) + c.pdf (remote) = 2
	for _, f := range result {
		if f.RelativePath == "a.pdf" {
			t.Error("tombstoned remote file a.pdf should not appear in merged result")
		}
	}
	if len(result) != 2 {
		t.Errorf("expected 2 files (b.pdf + c.pdf), got %d: %v", len(result), result)
	}
}

func TestMergeFingerprintFiles_NilTombstones(t *testing.T) {
	local := []FingerprintFile{{RelativePath: "a.pdf"}}
	remote := []FingerprintFile{{RelativePath: "b.pdf"}}

	// nil tombstones should not panic
	result := MergeFingerprintFiles(local, remote, nil)
	if len(result) != 2 {
		t.Errorf("expected 2 files with nil tombstones, got %d", len(result))
	}
}

func TestMergeFingerprintFiles_EmptyInputs(t *testing.T) {
	result := MergeFingerprintFiles(nil, nil, nil)
	if result != nil && len(result) != 0 {
		t.Errorf("expected empty result for nil inputs, got %v", result)
	}

	result2 := MergeFingerprintFiles([]FingerprintFile{}, []FingerprintFile{}, nil)
	if len(result2) != 0 {
		t.Errorf("expected 0 files for empty inputs, got %d", len(result2))
	}
}
