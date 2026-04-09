package sync

import (
	"sync"
	"time"

	"haya-tab/pkg/store"
)

const (
	// DefaultVolumeCacheTTL is the default time-to-live for cached volume entries.
	DefaultVolumeCacheTTL = 5 * time.Minute
)

// VolumeCache provides in-memory caching of volume metadata with TTL
// This eliminates redundant volume scanning operations
type VolumeCache struct {
	volumes    map[string]*cachedVolume // mountPath -> cachedVolume
	mu         sync.RWMutex
	ttl        time.Duration
	lastUpdate time.Time
}

type cachedVolume struct {
	volume      store.CloudVolume
	fingerprint *VolumeFingerprint
	cachedAt    time.Time
}

// NewVolumeCache creates a new volume cache with specified TTL
func NewVolumeCache(ttl time.Duration) *VolumeCache {
	if ttl == 0 {
		ttl = DefaultVolumeCacheTTL
	}

	return &VolumeCache{
		volumes: make(map[string]*cachedVolume),
		ttl:     ttl,
	}
}

// Get retrieves a volume from cache by mount path
// Returns nil if not found or expired
func (c *VolumeCache) Get(mountPath string) (*store.CloudVolume, *VolumeFingerprint) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cached, exists := c.volumes[mountPath]
	if !exists {
		return nil, nil
	}

	// Check if expired
	if time.Since(cached.cachedAt) > c.ttl {
		return nil, nil
	}

	return &cached.volume, cached.fingerprint
}

// Set stores a volume in cache
func (c *VolumeCache) Set(mountPath string, volume store.CloudVolume, fingerprint *VolumeFingerprint) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.volumes[mountPath] = &cachedVolume{
		volume:      volume,
		fingerprint: fingerprint,
		cachedAt:    time.Now(),
	}
}

// GetAll retrieves all non-expired volumes from cache
// Returns nil if cache is stale (older than TTL)
func (c *VolumeCache) GetAll() []store.CloudVolume {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Check if entire cache is stale
	if time.Since(c.lastUpdate) > c.ttl {
		return nil
	}

	volumes := make([]store.CloudVolume, 0, len(c.volumes))
	now := time.Now()

	for _, cached := range c.volumes {
		// Skip expired entries
		if now.Sub(cached.cachedAt) <= c.ttl {
			volumes = append(volumes, cached.volume)
		}
	}

	return volumes
}

// SetAll replaces entire cache with new volumes
func (c *VolumeCache) SetAll(volumes []store.CloudVolume, fingerprints map[string]*VolumeFingerprint) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.volumes = make(map[string]*cachedVolume)
	now := time.Now()

	for _, vol := range volumes {
		fingerprint := fingerprints[vol.MountPath]
		c.volumes[vol.MountPath] = &cachedVolume{
			volume:      vol,
			fingerprint: fingerprint,
			cachedAt:    now,
		}
	}

	c.lastUpdate = now
}

// Invalidate removes a specific volume from cache
func (c *VolumeCache) Invalidate(mountPath string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.volumes, mountPath)
}

// Clear removes all volumes from cache
func (c *VolumeCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.volumes = make(map[string]*cachedVolume)
	c.lastUpdate = time.Time{}
}

// IsStale checks if the cache needs refresh
func (c *VolumeCache) IsStale() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return time.Since(c.lastUpdate) > c.ttl
}

// GetStats returns cache statistics for monitoring
func (c *VolumeCache) GetStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"size":        len(c.volumes),
		"ttl_seconds": c.ttl.Seconds(),
		"age_seconds": time.Since(c.lastUpdate).Seconds(),
		"is_stale":    time.Since(c.lastUpdate) > c.ttl,
	}
}
