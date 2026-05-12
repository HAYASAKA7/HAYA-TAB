package sync

import (
	"testing"
	"time"

	"haya-tab/pkg/store"
)

func TestVolumeCache_GetSet(t *testing.T) {
	cache := NewVolumeCache(100 * time.Millisecond)

	vol := store.CloudVolume{ID: "vol1", Name: "Volume 1"}
	fp := &VolumeFingerprint{VolumeID: "vol1"}

	cache.Set("/mount/path", vol, fp)

	gotVol, gotFp := cache.Get("/mount/path")
	if gotVol == nil || gotVol.ID != "vol1" {
		t.Errorf("expected volume vol1, got %v", gotVol)
	}
	if gotFp == nil || gotFp.VolumeID != "vol1" {
		t.Errorf("expected fingerprint vol1, got %v", gotFp)
	}

	// Test expiration
	time.Sleep(200 * time.Millisecond)
	gotVol, _ = cache.Get("/mount/path")
	if gotVol != nil {
		t.Error("expected nil for expired volume")
	}
}

func TestVolumeCache_GetAllSetAll(t *testing.T) {
	cache := NewVolumeCache(1 * time.Hour)

	vols := []store.CloudVolume{
		{ID: "vol1", MountPath: "/m1"},
		{ID: "vol2", MountPath: "/m2"},
	}
	fps := map[string]*VolumeFingerprint{
		"/m1": {VolumeID: "vol1"},
		"/m2": {VolumeID: "vol2"},
	}

	cache.SetAll(vols, fps)

	all := cache.GetAll()
	if len(all) != 2 {
		t.Errorf("expected 2 volumes, got %d", len(all))
	}

	if cache.IsStale() {
		t.Error("cache should not be stale after SetAll")
	}

	// Test invalidation
	cache.Invalidate("/m1")
	all = cache.GetAll()
	if len(all) != 1 {
		t.Errorf("expected 1 volume after invalidation, got %d", len(all))
	}

	// Test clear
	cache.Clear()
	all = cache.GetAll()
	if len(all) != 0 {
		t.Errorf("expected 0 volumes after clear, got %d", len(all))
	}
}

func TestVolumeCache_Stats(t *testing.T) {
	cache := NewVolumeCache(5 * time.Minute)
	stats := cache.GetStats()

	if stats["size"].(int) != 0 {
		t.Errorf("expected size 0, got %v", stats["size"])
	}
	if stats["ttl_seconds"].(float64) != 300 {
		t.Errorf("expected 300s TTL, got %v", stats["ttl_seconds"])
	}
}
