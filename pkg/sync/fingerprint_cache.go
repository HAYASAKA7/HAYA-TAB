package sync

import (
	"fmt"
	"sync"
	"time"
)

// FingerprintCache provides an in-memory LRU cache for fingerprint operations
// with delayed batch writes to reduce WebDAV I/O overhead
// OPTIMIZED: Cache persists after flush, only dirty buckets are written
type FingerprintCache struct {
	client        *WebDAVClient
	cache         map[string]map[int]*cachedBucket // volumePath -> bucketNum -> cachedBucket
	dirtyBuckets  map[string]map[int]bool          // volumePath -> bucketNum -> isDirty
	mu            sync.RWMutex
	flushInterval time.Duration
	stopChan      chan struct{}
	wg            sync.WaitGroup
	maxSize       int // Maximum number of buckets to cache (0 = unlimited)
	hitCount      int64
	missCount     int64
}

type cachedBucket struct {
	data       *BucketData
	lastAccess time.Time
}

// NewFingerprintCache creates a new fingerprint cache with automatic flushing
// maxSize: maximum number of buckets to cache (0 = unlimited, recommended: 100)
func NewFingerprintCache(client *WebDAVClient, flushInterval time.Duration, maxSize int) *FingerprintCache {
	if flushInterval == 0 {
		flushInterval = 5 * time.Second // Default: flush every 5 seconds
	}
	if maxSize == 0 {
		maxSize = 100 // Default: cache up to 100 buckets
	}

	cache := &FingerprintCache{
		client:        client,
		cache:         make(map[string]map[int]*cachedBucket),
		dirtyBuckets:  make(map[string]map[int]bool),
		flushInterval: flushInterval,
		stopChan:      make(chan struct{}),
		maxSize:       maxSize,
	}

	// Start background flusher
	cache.wg.Add(1)
	go cache.backgroundFlusher()

	return cache
}

// getBucket retrieves a bucket from cache or loads it from WebDAV
func (c *FingerprintCache) getBucket(volumePath string, bucketNum int) (*BucketData, error) {
	// Check cache first
	if c.cache[volumePath] != nil && c.cache[volumePath][bucketNum] != nil {
		cached := c.cache[volumePath][bucketNum]
		cached.lastAccess = time.Now()
		c.hitCount++
		return cached.data, nil
	}

	// Cache miss - load from WebDAV
	c.missCount++
	bucket, err := c.client.ReadBucket(volumePath, bucketNum)
	if err != nil {
		return nil, fmt.Errorf("failed to read bucket %d: %w", bucketNum, err)
	}

	// Add to cache
	c.putBucket(volumePath, bucketNum, bucket, false)

	return bucket, nil
}

// putBucket stores a bucket in cache
func (c *FingerprintCache) putBucket(volumePath string, bucketNum int, bucket *BucketData, isDirty bool) {
	// Initialize maps if needed
	if c.cache[volumePath] == nil {
		c.cache[volumePath] = make(map[int]*cachedBucket)
	}
	if c.dirtyBuckets[volumePath] == nil {
		c.dirtyBuckets[volumePath] = make(map[int]bool)
	}

	// Store in cache
	c.cache[volumePath][bucketNum] = &cachedBucket{
		data:       bucket,
		lastAccess: time.Now(),
	}

	// Mark as dirty if modified
	if isDirty {
		c.dirtyBuckets[volumePath][bucketNum] = true
	}

	// Evict old entries if cache is full
	c.evictIfNeeded()
}

// evictIfNeeded removes least recently used buckets if cache exceeds maxSize
func (c *FingerprintCache) evictIfNeeded() {
	totalBuckets := 0
	for _, buckets := range c.cache {
		totalBuckets += len(buckets)
	}

	if totalBuckets <= c.maxSize {
		return
	}

	// Find oldest non-dirty bucket to evict
	var oldestVolume string
	var oldestBucket int
	var oldestTime time.Time

	for volumePath, buckets := range c.cache {
		for bucketNum, cached := range buckets {
			// Don't evict dirty buckets
			if c.dirtyBuckets[volumePath] != nil && c.dirtyBuckets[volumePath][bucketNum] {
				continue
			}

			if oldestTime.IsZero() || cached.lastAccess.Before(oldestTime) {
				oldestTime = cached.lastAccess
				oldestVolume = volumePath
				oldestBucket = bucketNum
			}
		}
	}

	// Evict oldest bucket
	if !oldestTime.IsZero() {
		delete(c.cache[oldestVolume], oldestBucket)
		if len(c.cache[oldestVolume]) == 0 {
			delete(c.cache, oldestVolume)
		}
	}
}

// AddFile adds a file to the cache (non-blocking)
func (c *FingerprintCache) AddFile(volumePath string, file FingerprintFile) error {
	bucketNum := CalculateBucketNumber(file.RelativePath)

	c.mu.Lock()
	defer c.mu.Unlock()

	// Get bucket from cache or load it
	bucket, err := c.getBucket(volumePath, bucketNum)
	if err != nil {
		return err
	}

	// Check if file already exists
	fileExists := false
	for i, f := range bucket.Files {
		if f.RelativePath == file.RelativePath {
			bucket.Files[i] = file
			fileExists = true
			break
		}
	}

	if !fileExists {
		bucket.Files = append(bucket.Files, file)
	}

	// Mark as dirty
	c.putBucket(volumePath, bucketNum, bucket, true)

	return nil
}

// RemoveFile removes a file from the cache (non-blocking)
func (c *FingerprintCache) RemoveFile(volumePath, relativePath string) error {
	bucketNum := CalculateBucketNumber(relativePath)

	c.mu.Lock()
	defer c.mu.Unlock()

	// Get bucket from cache or load it
	bucket, err := c.getBucket(volumePath, bucketNum)
	if err != nil {
		return err
	}

	// Remove file from bucket
	newFiles := make([]FingerprintFile, 0, len(bucket.Files))
	for _, f := range bucket.Files {
		if f.RelativePath != relativePath {
			newFiles = append(newFiles, f)
		}
	}
	bucket.Files = newFiles

	// Mark as dirty
	c.putBucket(volumePath, bucketNum, bucket, true)

	return nil
}

// BatchAddFiles adds multiple files to the cache (non-blocking)
func (c *FingerprintCache) BatchAddFiles(volumePath string, files []FingerprintFile) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Group files by bucket
	bucketFiles := make(map[int][]FingerprintFile)
	for _, file := range files {
		bucketNum := CalculateBucketNumber(file.RelativePath)
		bucketFiles[bucketNum] = append(bucketFiles[bucketNum], file)
	}

	// Process each bucket
	for bucketNum, newFiles := range bucketFiles {
		// Get bucket from cache or load it
		bucket, err := c.getBucket(volumePath, bucketNum)
		if err != nil {
			return fmt.Errorf("failed to get bucket %d: %w", bucketNum, err)
		}

		// Create map of existing files for fast lookup
		existingFiles := make(map[string]bool)
		for _, file := range bucket.Files {
			existingFiles[file.RelativePath] = true
		}

		// Add new files
		for _, file := range newFiles {
			if !existingFiles[file.RelativePath] {
				bucket.Files = append(bucket.Files, file)
			}
		}

		// Mark as dirty
		c.putBucket(volumePath, bucketNum, bucket, true)
	}

	return nil
}

// Flush writes all dirty buckets to WebDAV (blocking)
// OPTIMIZED: Only writes dirty buckets, keeps cache intact
func (c *FingerprintCache) Flush() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.dirtyBuckets) == 0 {
		return nil
	}

	var lastErr error
	flushedCount := 0

	for volumePath, buckets := range c.dirtyBuckets {
		for bucketNum := range buckets {
			// Get bucket from cache
			if c.cache[volumePath] == nil || c.cache[volumePath][bucketNum] == nil {
				continue
			}

			bucket := c.cache[volumePath][bucketNum].data

			// Write to WebDAV
			if err := c.client.WriteBucket(volumePath, bucketNum, bucket); err != nil {
				lastErr = fmt.Errorf("failed to write bucket %d for volume %s: %w", bucketNum, volumePath, err)
				fmt.Printf("[Warning] %v\n", lastErr)
			} else {
				flushedCount++
			}
		}
	}

	// Clear dirty flags (but keep cache)
	c.dirtyBuckets = make(map[string]map[int]bool)

	if flushedCount > 0 {
		fmt.Printf("[Info] Flushed %d dirty buckets to WebDAV\n", flushedCount)
	}

	return lastErr
}

// GetStats returns cache statistics for monitoring
func (c *FingerprintCache) GetStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	totalBuckets := 0
	dirtyCount := 0

	for _, buckets := range c.cache {
		totalBuckets += len(buckets)
	}

	for _, buckets := range c.dirtyBuckets {
		dirtyCount += len(buckets)
	}

	hitRate := float64(0)
	totalAccess := c.hitCount + c.missCount
	if totalAccess > 0 {
		hitRate = float64(c.hitCount) / float64(totalAccess) * 100
	}

	return map[string]interface{}{
		"cached_buckets": totalBuckets,
		"dirty_buckets":  dirtyCount,
		"max_size":       c.maxSize,
		"hit_count":      c.hitCount,
		"miss_count":     c.missCount,
		"hit_rate":       fmt.Sprintf("%.2f%%", hitRate),
	}
}

// backgroundFlusher periodically flushes dirty buckets
func (c *FingerprintCache) backgroundFlusher() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := c.Flush(); err != nil {
				fmt.Printf("[Warning] Background flush failed: %v\n", err)
			}
		case <-c.stopChan:
			// Final flush before stopping
			if err := c.Flush(); err != nil {
				fmt.Printf("[Warning] Final flush failed: %v\n", err)
			}
			return
		}
	}
}

// Close stops the background flusher and performs a final flush
func (c *FingerprintCache) Close() error {
	close(c.stopChan)
	c.wg.Wait()
	return nil
}
