package sync

import (
	"fmt"
	"sync"
	"time"
)

// FingerprintCache provides an in-memory cache for fingerprint operations
// with delayed batch writes to reduce WebDAV I/O overhead
type FingerprintCache struct {
	client         *WebDAVClient
	pendingUpdates map[string]map[int]*BucketData // volumePath -> bucketNum -> BucketData
	mu             sync.RWMutex
	flushInterval  time.Duration
	stopChan       chan struct{}
	wg             sync.WaitGroup
}

// NewFingerprintCache creates a new fingerprint cache with automatic flushing
func NewFingerprintCache(client *WebDAVClient, flushInterval time.Duration) *FingerprintCache {
	if flushInterval == 0 {
		flushInterval = 5 * time.Second // Default: flush every 5 seconds
	}

	cache := &FingerprintCache{
		client:         client,
		pendingUpdates: make(map[string]map[int]*BucketData),
		flushInterval:  flushInterval,
		stopChan:       make(chan struct{}),
	}

	// Start background flusher
	cache.wg.Add(1)
	go cache.backgroundFlusher()

	return cache
}

// AddFile adds a file to the cache (non-blocking)
func (c *FingerprintCache) AddFile(volumePath string, file FingerprintFile) error {
	bucketNum := CalculateBucketNumber(file.RelativePath)

	c.mu.Lock()
	defer c.mu.Unlock()

	// Initialize volume map if needed
	if c.pendingUpdates[volumePath] == nil {
		c.pendingUpdates[volumePath] = make(map[int]*BucketData)
	}

	// Load bucket if not in cache
	if c.pendingUpdates[volumePath][bucketNum] == nil {
		bucket, err := c.client.ReadBucket(volumePath, bucketNum)
		if err != nil {
			return fmt.Errorf("failed to read bucket %d: %w", bucketNum, err)
		}
		c.pendingUpdates[volumePath][bucketNum] = bucket
	}

	bucket := c.pendingUpdates[volumePath][bucketNum]

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

	return nil
}

// RemoveFile removes a file from the cache (non-blocking)
func (c *FingerprintCache) RemoveFile(volumePath, relativePath string) error {
	bucketNum := CalculateBucketNumber(relativePath)

	c.mu.Lock()
	defer c.mu.Unlock()

	// Initialize volume map if needed
	if c.pendingUpdates[volumePath] == nil {
		c.pendingUpdates[volumePath] = make(map[int]*BucketData)
	}

	// Load bucket if not in cache
	if c.pendingUpdates[volumePath][bucketNum] == nil {
		bucket, err := c.client.ReadBucket(volumePath, bucketNum)
		if err != nil {
			return fmt.Errorf("failed to read bucket %d: %w", bucketNum, err)
		}
		c.pendingUpdates[volumePath][bucketNum] = bucket
	}

	bucket := c.pendingUpdates[volumePath][bucketNum]

	// Remove file from bucket
	newFiles := make([]FingerprintFile, 0, len(bucket.Files))
	for _, f := range bucket.Files {
		if f.RelativePath != relativePath {
			newFiles = append(newFiles, f)
		}
	}
	bucket.Files = newFiles

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

	// Initialize volume map if needed
	if c.pendingUpdates[volumePath] == nil {
		c.pendingUpdates[volumePath] = make(map[int]*BucketData)
	}

	// Process each bucket
	for bucketNum, newFiles := range bucketFiles {
		// Load bucket if not in cache
		if c.pendingUpdates[volumePath][bucketNum] == nil {
			bucket, err := c.client.ReadBucket(volumePath, bucketNum)
			if err != nil {
				return fmt.Errorf("failed to read bucket %d: %w", bucketNum, err)
			}
			c.pendingUpdates[volumePath][bucketNum] = bucket
		}

		bucket := c.pendingUpdates[volumePath][bucketNum]

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
	}

	return nil
}

// Flush writes all pending updates to WebDAV (blocking)
func (c *FingerprintCache) Flush() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.pendingUpdates) == 0 {
		return nil
	}

	var lastErr error
	for volumePath, buckets := range c.pendingUpdates {
		for bucketNum, bucket := range buckets {
			if err := c.client.WriteBucket(volumePath, bucketNum, bucket); err != nil {
				lastErr = fmt.Errorf("failed to write bucket %d for volume %s: %w", bucketNum, volumePath, err)
				fmt.Printf("[Warning] %v\n", lastErr)
			}
		}
	}

	// Clear pending updates after flush
	c.pendingUpdates = make(map[string]map[int]*BucketData)

	return lastErr
}

// backgroundFlusher periodically flushes pending updates
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
