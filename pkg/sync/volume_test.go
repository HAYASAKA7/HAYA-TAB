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
