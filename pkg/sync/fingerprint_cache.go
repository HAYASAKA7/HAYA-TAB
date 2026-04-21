package sync

import (
	"fmt"
	"reflect"
	"sync"
	"time"
)

const (
	// DefaultCacheFlushInterval is the default interval for flushing dirty buckets to WebDAV.
	DefaultCacheFlushInterval = 5 * time.Second
	// DefaultCacheMaxSize is the default maximum number of buckets to keep in cache.
	DefaultCacheMaxSize = 100
	// MaxFlushRetries is the maximum number of retry attempts when a bucket write conflict occurs.
	MaxFlushRetries = 3
)

// FingerprintCache provides an in-memory LRU cache for fingerprint operations
// with delayed batch writes to reduce WebDAV I/O overhead
// OPTIMIZED: Cache persists after flush, only dirty buckets are written
type FingerprintCache struct {
	client        *WebDAVClient
	cache         map[string]map[int]*cachedBucket // volumePath -> bucketNum -> cachedBucket
	dirtyBuckets  map[string]map[int]bool          // volumePath -> bucketNum -> isDirty
	deletedFiles  map[string]map[string]bool       // volumePath -> relativePath -> isDeleted (tombstone)
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
		flushInterval = DefaultCacheFlushInterval
	}
	if maxSize == 0 {
		maxSize = DefaultCacheMaxSize
	}

	cache := &FingerprintCache{
		client:        client,
		cache:         make(map[string]map[int]*cachedBucket),
		dirtyBuckets:  make(map[string]map[int]bool),
		deletedFiles:  make(map[string]map[string]bool),
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

	if c.deletedFiles[volumePath] != nil {
		delete(c.deletedFiles[volumePath], file.RelativePath)
	}

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
	_, err := c.BatchRemoveFiles(volumePath, []string{relativePath})
	return err
}

// BatchAddFiles adds multiple files to the cache (non-blocking)
func (c *FingerprintCache) BatchAddFiles(volumePath string, files []FingerprintFile) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, file := range files {
		if c.deletedFiles[volumePath] != nil {
			delete(c.deletedFiles[volumePath], file.RelativePath)
		}
	}

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
		existingFiles := make(map[string]int)
		for i, file := range bucket.Files {
			existingFiles[file.RelativePath] = i
		}

		// Add or update files
		for _, file := range newFiles {
			if idx, exists := existingFiles[file.RelativePath]; exists {
				bucket.Files[idx] = file
			} else {
				bucket.Files = append(bucket.Files, file)
				existingFiles[file.RelativePath] = len(bucket.Files) - 1
			}
		}

		// Mark as dirty
		c.putBucket(volumePath, bucketNum, bucket, true)
	}

	return nil
}

// BatchRemoveFiles removes multiple files from the cache and returns how many were removed.
func (c *FingerprintCache) BatchRemoveFiles(volumePath string, relativePaths []string) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(relativePaths) == 0 {
		return 0, nil
	}

	if c.deletedFiles[volumePath] == nil {
		c.deletedFiles[volumePath] = make(map[string]bool)
	}

	pathsToRemoveByBucket := make(map[int]map[string]bool)
	for _, relativePath := range relativePaths {
		bucketNum := CalculateBucketNumber(relativePath)
		if pathsToRemoveByBucket[bucketNum] == nil {
			pathsToRemoveByBucket[bucketNum] = make(map[string]bool)
		}
		pathsToRemoveByBucket[bucketNum][relativePath] = true
	}

	removedCount := 0
	for bucketNum, pathsToRemove := range pathsToRemoveByBucket {
		bucket, err := c.getBucket(volumePath, bucketNum)
		if err != nil {
			return removedCount, fmt.Errorf("failed to get bucket %d: %w", bucketNum, err)
		}

		newFiles := make([]FingerprintFile, 0, len(bucket.Files))
		bucketRemovedCount := 0
		removedPaths := make([]string, 0, len(pathsToRemove))
		for _, file := range bucket.Files {
			if pathsToRemove[file.RelativePath] {
				bucketRemovedCount++
				removedPaths = append(removedPaths, file.RelativePath)
				continue
			}
			newFiles = append(newFiles, file)
		}

		if bucketRemovedCount == 0 {
			continue
		}

		if c.deletedFiles[volumePath] == nil {
			c.deletedFiles[volumePath] = make(map[string]bool)
		}
		for _, removedPath := range removedPaths {
			c.deletedFiles[volumePath][removedPath] = true
		}

		bucket.Files = newFiles
		c.putBucket(volumePath, bucketNum, bucket, true)
		removedCount += bucketRemovedCount
	}

	return removedCount, nil
}

// Flush writes all dirty buckets to WebDAV (blocking)
// OPTIMIZED: Only writes dirty buckets, keeps cache intact and releases lock before I/O
// ENHANCED: Uses ETag-based conditional updates with retry logic for conflict resolution
func (c *FingerprintCache) Flush() error {
	c.mu.Lock()

	if len(c.dirtyBuckets) == 0 {
		c.mu.Unlock()
		return nil
	}

	// Extract dirty buckets and deep copy necessary data to write without lock
	type dirtyItem struct {
		volumePath string
		bucketNum  int
		bucket     *BucketData
		snapshot   *BucketData
	}
	var itemsToWrite []dirtyItem

	// Also make a copy of deletedFiles for this flush to use as tombstones
	deletedFilesCopy := make(map[string]map[string]bool)
	for volPath, filesMap := range c.deletedFiles {
		deletedFilesCopy[volPath] = make(map[string]bool)
		for relPath, isDeleted := range filesMap {
			deletedFilesCopy[volPath][relPath] = isDeleted
		}
	}

	for volumePath, buckets := range c.dirtyBuckets {
		for bucketNum := range buckets {
			// Get bucket from cache
			if c.cache[volumePath] == nil || c.cache[volumePath][bucketNum] == nil {
				continue
			}

			// Clone the bucket for safe concurrent access
			originalBucket := c.cache[volumePath][bucketNum].data
			bucketCopy := originalBucket.Clone()

			itemsToWrite = append(itemsToWrite, dirtyItem{
				volumePath: volumePath,
				bucketNum:  bucketNum,
				bucket:     bucketCopy,
				snapshot:   originalBucket.Clone(),
			})
		}
	}
	c.mu.Unlock()

	var lastErr error
	const maxRetries = MaxFlushRetries
	var successfulItems []dirtyItem

	// Write to WebDAV without lock
	for i := range itemsToWrite {
		item := &itemsToWrite[i]
		var itemErr error
		for retry := 0; retry < maxRetries; retry++ {
			// 1. READ: Fetch the latest bucket state from WebDAV with ETag
			remoteBucket, err := c.client.ReadBucket(item.volumePath, item.bucketNum)
			if err != nil {
				itemErr = fmt.Errorf("failed to read bucket %d for volume %s: %w", item.bucketNum, item.volumePath, err)
				break
			}

			// 2. MODIFY (MERGE): If read succeeds, merge remote files into our local bucket copy
			if remoteBucket != nil {
				// Use timestamp-based merge with tombstone support
				var tombstone map[string]bool
				if deletedFilesCopy[item.volumePath] != nil {
					tombstone = deletedFilesCopy[item.volumePath]
				}

				// Merge remote files into local bucket using timestamp-based conflict resolution
				item.bucket.Files = MergeFingerprintFiles(item.bucket.Files, remoteBucket.Files, tombstone)

				// Update ETag from remote for conditional write
				item.bucket.ETag = remoteBucket.ETag
			}

			// 3. WRITE: Attempt conditional PUT with If-Match header
			err = c.client.WriteBucketWithETag(item.volumePath, item.bucketNum, item.bucket, item.bucket.ETag)

			if err == nil {
				// Success!
				successfulItems = append(successfulItems, *item)
				itemErr = nil
				break // Exit retry loop on success
			}

			// Check if it's a precondition failed error (conflict)
			if err.Error() == "precondition_failed: bucket was modified by another device" {
				// Conflict detected - retry with fresh data
				fmt.Printf("[Debug] Conflict detected for bucket %d, retrying (%d/%d)...\n", item.bucketNum, retry+1, maxRetries)
				if retry == maxRetries-1 {
					itemErr = err
				}
				continue
			}

			// Other errors - log and continue to next bucket
			itemErr = fmt.Errorf("failed to write bucket %d for volume %s: %w", item.bucketNum, item.volumePath, err)
			fmt.Printf("[Warning] %v\n", itemErr)
			break
		}
		if itemErr != nil && lastErr == nil {
			lastErr = itemErr
		}
	}

	c.mu.Lock()
	flushedCount := 0
	for _, item := range successfulItems {
		currentVolumeBuckets := c.cache[item.volumePath]
		if currentVolumeBuckets == nil {
			continue
		}

		currentCached := currentVolumeBuckets[item.bucketNum]
		if currentCached == nil || currentCached.data == nil {
			continue
		}

		// Only clear the dirty flag if nothing changed locally while the flush was in flight.
		if !reflect.DeepEqual(currentCached.data, item.snapshot) {
			continue
		}

		currentCached.data = item.bucket
		if c.dirtyBuckets[item.volumePath] != nil {
			delete(c.dirtyBuckets[item.volumePath], item.bucketNum)
			if len(c.dirtyBuckets[item.volumePath]) == 0 {
				delete(c.dirtyBuckets, item.volumePath)
			}
		}

		if c.deletedFiles[item.volumePath] != nil && deletedFilesCopy[item.volumePath] != nil {
			for relPath := range deletedFilesCopy[item.volumePath] {
				if CalculateBucketNumber(relPath) == item.bucketNum {
					delete(c.deletedFiles[item.volumePath], relPath)
				}
			}
			if len(c.deletedFiles[item.volumePath]) == 0 {
				delete(c.deletedFiles, item.volumePath)
			}
		}

		flushedCount++
	}
	c.mu.Unlock()

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
