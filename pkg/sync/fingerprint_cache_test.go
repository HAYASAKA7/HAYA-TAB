package sync

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFingerprintCache_AddFile(t *testing.T) {
	// Mock bucket 0 (with metadata)
	bucket0 := struct {
		Metadata FingerprintMetadata `json:"metadata"`
		Files    []FingerprintFile   `json:"files"`
	}{
		Metadata: FingerprintMetadata{VolumeID: "vol-1", BucketCount: 16},
		Files:    []FingerprintFile{},
	}

	// Mock other buckets
	buckets := make(map[int]*BucketData)
	for i := 0; i < 16; i++ {
		buckets[i] = &BucketData{BucketNumber: i, Files: []FingerprintFile{}}
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract bucket number from path
		var bucketNum int
		fmt.Sscanf(r.URL.Path, "/vol1/haya-metadata/bucket-%02d.json", &bucketNum)

		switch r.Method {
		case "GET":
			w.WriteHeader(http.StatusOK)
			var data []byte
			if bucketNum == 0 {
				data, _ = json.Marshal(bucket0)
			} else {
				data, _ = json.Marshal(buckets[bucketNum])
			}
			w.Write(data)
		case "PUT":
			w.WriteHeader(http.StatusCreated)
		}
	}))
	defer server.Close()

	client := NewWebDAVClient(server.URL, "user", "pass")
	cache := NewFingerprintCache(client, 100*time.Millisecond, 10)
	defer cache.Close()

	file := FingerprintFile{
		RelativePath: "test.pdf",
		Title:        "Test",
	}

	err := cache.AddFile("/vol1", file)
	if err != nil {
		t.Fatalf("AddFile failed: %v", err)
	}

	stats := cache.GetStats()
	if stats["cached_buckets"].(int) != 1 {
		t.Errorf("expected 1 cached bucket, got %v", stats["cached_buckets"])
	}
	if stats["dirty_buckets"].(int) != 1 {
		t.Errorf("expected 1 dirty bucket, got %v", stats["dirty_buckets"])
	}

	// Wait for background flush
	time.Sleep(200 * time.Millisecond)

	stats = cache.GetStats()
	if stats["dirty_buckets"].(int) != 0 {
		t.Errorf("expected 0 dirty buckets after flush, got %v", stats["dirty_buckets"])
	}
}

func TestFingerprintCache_RemoveFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			bucket := &BucketData{
				BucketNumber: 0,
				Files: []FingerprintFile{
					{RelativePath: "remove-me.pdf", Title: "Remove Me"},
				},
			}
			data, _ := json.Marshal(bucket)
			w.Write(data)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	client := NewWebDAVClient(server.URL, "user", "pass")
	cache := NewFingerprintCache(client, 1*time.Hour, 10) // No auto flush
	defer cache.Close()

	// Initial add to cache (will load from mock server)
	err := cache.RemoveFile("/vol1", "remove-me.pdf")
	if err != nil {
		t.Fatalf("RemoveFile failed: %v", err)
	}

	stats := cache.GetStats()
	if stats["dirty_buckets"].(int) != 1 {
		t.Errorf("expected 1 dirty bucket, got %v", stats["dirty_buckets"])
	}
}

func TestFingerprintCache_BatchAddFiles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			bucket := &BucketData{Files: []FingerprintFile{}}
			data, _ := json.Marshal(bucket)
			w.Write(data)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	client := NewWebDAVClient(server.URL, "user", "pass")
	cache := NewFingerprintCache(client, 1*time.Hour, 10)
	defer cache.Close()

	files := []FingerprintFile{
		{RelativePath: "file1.pdf", Title: "File 1"},
		{RelativePath: "file2.pdf", Title: "File 2"},
	}

	err := cache.BatchAddFiles("/vol1", files)
	if err != nil {
		t.Fatalf("BatchAddFiles failed: %v", err)
	}

	stats := cache.GetStats()
	// Depending on hashes, these might be in 1 or 2 buckets
	if stats["dirty_buckets"].(int) == 0 {
		t.Error("expected dirty buckets after batch add")
	}
}

func pathsForBucket(bucketNum, count int) []string {
	paths := make([]string, 0, count)
	for i := 0; len(paths) < count; i++ {
		path := fmt.Sprintf("file-%d.pdf", i)
		if CalculateBucketNumber(path) == bucketNum {
			paths = append(paths, path)
		}
	}
	return paths
}

func TestFingerprintCache_BatchRemoveFiles(t *testing.T) {
	targetBucket := 3
	paths := pathsForBucket(targetBucket, 3)
	keepPath := paths[2]
	removePaths := paths[:2]

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var bucketNum int
		fmt.Sscanf(r.URL.Path, "/vol1/haya-metadata/bucket-%02d.json", &bucketNum)

		if r.Method == "GET" {
			bucket := &BucketData{BucketNumber: bucketNum, Files: []FingerprintFile{}}
			if bucketNum == targetBucket {
				bucket.Files = []FingerprintFile{
					{RelativePath: keepPath, Title: "Keep"},
					{RelativePath: removePaths[0], Title: "Remove 1"},
					{RelativePath: removePaths[1], Title: "Remove 2"},
				}
			}
			data, _ := json.Marshal(bucket)
			w.Write(data)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewWebDAVClient(server.URL, "user", "pass")
	cache := NewFingerprintCache(client, 1*time.Hour, 10)
	defer cache.Close()

	removed, err := cache.BatchRemoveFiles("/vol1", removePaths)
	if err != nil {
		t.Fatalf("BatchRemoveFiles failed: %v", err)
	}
	if removed != len(removePaths) {
		t.Fatalf("removed = %d, want %d", removed, len(removePaths))
	}

	bucket, err := cache.getBucket("/vol1", targetBucket)
	if err != nil {
		t.Fatalf("getBucket failed: %v", err)
	}

	if len(bucket.Files) != 1 {
		t.Fatalf("expected 1 file remaining, got %d", len(bucket.Files))
	}
	if bucket.Files[0].RelativePath != keepPath {
		t.Fatalf("remaining file = %q, want %q", bucket.Files[0].RelativePath, keepPath)
	}

	stats := cache.GetStats()
	if stats["dirty_buckets"].(int) != 1 {
		t.Fatalf("expected 1 dirty bucket, got %v", stats["dirty_buckets"])
	}
}

func TestFingerprintCache_Eviction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bucket := &BucketData{Files: []FingerprintFile{}}
		data, _ := json.Marshal(bucket)
		w.Write(data)
	}))
	defer server.Close()

	client := NewWebDAVClient(server.URL, "user", "pass")
	// Small cache size to trigger eviction
	cache := NewFingerprintCache(client, 1*time.Hour, 2)
	defer cache.Close()

	// Add files that land in different buckets
	// We'll just use different paths and hope they land in different buckets,
	// or we can manually put buckets if we want to be precise.

	// Since we want to test eviction of non-dirty buckets,
	// we need to load them first (making them non-dirty), then load more.

	// Force load 3 different buckets (0, 1, 2)
	_, _ = cache.getBucket("/vol1", 0)
	_, _ = cache.getBucket("/vol1", 1)
	_, _ = cache.getBucket("/vol1", 2)

	stats := cache.GetStats()
	if stats["cached_buckets"].(int) > 2 {
		t.Errorf("expected eviction to limit cache to 2, got %v", stats["cached_buckets"])
	}
}

func TestFingerprintCache_FlushError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			bucket := &BucketData{Files: []FingerprintFile{}}
			data, _ := json.Marshal(bucket)
			w.Write(data)
		}
	}))
	defer server.Close()

	client := NewWebDAVClient(server.URL, "user", "pass")
	cache := NewFingerprintCache(client, 1*time.Hour, 10)
	defer cache.Close()

	_ = cache.AddFile("/vol1", FingerprintFile{RelativePath: "test.pdf"})

	err := cache.Flush()
	if err == nil {
		t.Error("expected error on flush failure")
	}
}

func TestFingerprintCache_FlushRetainsDirtyOnFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		bucket := &BucketData{Files: []FingerprintFile{}}
		data, _ := json.Marshal(bucket)
		w.Write(data)
	}))
	defer server.Close()

	client := NewWebDAVClient(server.URL, "user", "pass")
	cache := NewFingerprintCache(client, 1*time.Hour, 10)
	defer cache.Close()

	if err := cache.AddFile("/vol1", FingerprintFile{RelativePath: "test.pdf", Title: "Test"}); err != nil {
		t.Fatalf("AddFile failed: %v", err)
	}

	err := cache.Flush()
	if err == nil {
		t.Fatal("expected error on flush failure")
	}

	stats := cache.GetStats()
	if stats["dirty_buckets"].(int) != 1 {
		t.Fatalf("expected dirty bucket to remain after flush failure, got %v", stats["dirty_buckets"])
	}
}

func TestFingerprintCache_FlushRetainsTombstonesOnPartialFailure(t *testing.T) {
	failedBucket := 4
	okBucket := 11
	failedPath := pathsForBucket(failedBucket, 1)[0]
	okPath := pathsForBucket(okBucket, 1)[0]

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var bucketNum int
		fmt.Sscanf(r.URL.Path, "/vol1/haya-metadata/bucket-%02d.json", &bucketNum)

		if r.Method == "PUT" {
			if bucketNum == failedBucket {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusCreated)
			return
		}

		bucket := &BucketData{BucketNumber: bucketNum, Files: []FingerprintFile{}}
		if bucketNum == failedBucket {
			bucket.Files = []FingerprintFile{{RelativePath: failedPath, Title: "Fail"}}
		}
		if bucketNum == okBucket {
			bucket.Files = []FingerprintFile{{RelativePath: okPath, Title: "OK"}}
		}
		data, _ := json.Marshal(bucket)
		w.Write(data)
	}))
	defer server.Close()

	client := NewWebDAVClient(server.URL, "user", "pass")
	cache := NewFingerprintCache(client, 1*time.Hour, 10)
	defer cache.Close()

	removed, err := cache.BatchRemoveFiles("/vol1", []string{failedPath, okPath})
	if err != nil {
		t.Fatalf("BatchRemoveFiles failed: %v", err)
	}
	if removed != 2 {
		t.Fatalf("removed = %d, want 2", removed)
	}

	err = cache.Flush()
	if err == nil {
		t.Fatal("expected flush to fail for one bucket")
	}

	if !cache.deletedFiles["/vol1"][failedPath] {
		t.Fatalf("expected tombstone for failed delete to remain")
	}
	if cache.deletedFiles["/vol1"][okPath] {
		t.Fatalf("expected tombstone for successful delete to be cleared")
	}
}

func TestReadBucket_CapturesHeadETag(t *testing.T) {
	etag := `"etag-123"`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			bucket := &BucketData{
				BucketNumber: 5,
				Files:        []FingerprintFile{{RelativePath: "songs/shared.pdf", Title: "Shared"}},
			}
			data, _ := json.Marshal(bucket)
			w.Write(data)
		case "HEAD":
			w.Header().Set("ETag", etag)
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	client := NewWebDAVClient(server.URL, "user", "pass")
	bucket, err := client.ReadBucket("/vol1", 5)
	if err != nil {
		t.Fatalf("ReadBucket failed: %v", err)
	}
	if bucket.ETag != etag {
		t.Fatalf("ETag = %q, want %q", bucket.ETag, etag)
	}
}

func TestFingerprintCache_FlushRetriesAndPreservesNewerLocalFile(t *testing.T) {
	volumePath := "/vol1"
	relativePath := "songs/shared.pdf"
	bucketNum := CalculateBucketNumber(relativePath)

	localBucket := &BucketData{
		BucketNumber: bucketNum,
		Files: []FingerprintFile{
			{
				RelativePath: relativePath,
				Title:        "Local New",
				UploadedAt:   "2024-06-01T00:00:00Z",
				UploadedBy:   "device-local",
			},
		},
	}
	remoteBucket := &BucketData{
		BucketNumber: bucketNum,
		Files: []FingerprintFile{
			{
				RelativePath: relativePath,
				Title:        "Remote Old",
				UploadedAt:   "2024-01-01T00:00:00Z",
				UploadedBy:   "device-remote",
			},
		},
	}

	etag := `"v1"`
	putCount := 0
	var storedBucket *BucketData

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var gotBucketNum int
		fmt.Sscanf(r.URL.Path, "/vol1/haya-metadata/bucket-%02d.json", &gotBucketNum)
		if gotBucketNum != bucketNum {
			t.Fatalf("unexpected bucket %d, want %d", gotBucketNum, bucketNum)
		}

		switch r.Method {
		case "GET":
			data, _ := json.Marshal(remoteBucket)
			w.Header().Set("Content-Type", "application/json")
			w.Write(data)
		case "HEAD":
			w.Header().Set("ETag", etag)
			w.WriteHeader(http.StatusOK)
		case "PUT":
			putCount++
			if putCount == 1 {
				remoteBucket = &BucketData{
					BucketNumber: bucketNum,
					Files: []FingerprintFile{
						{
							RelativePath: relativePath,
							Title:        "Remote Conflict",
							UploadedAt:   "2024-02-01T00:00:00Z",
							UploadedBy:   "device-remote",
						},
					},
				}
				etag = `"v2"`
				w.WriteHeader(http.StatusPreconditionFailed)
				return
			}

			if got := r.Header.Get("If-Match"); got != etag {
				t.Fatalf("If-Match = %q, want %q", got, etag)
			}

			if err := json.NewDecoder(r.Body).Decode(&storedBucket); err != nil {
				t.Fatalf("failed to decode bucket body: %v", err)
			}
			etag = `"v3"`
			w.Header().Set("ETag", etag)
			w.WriteHeader(http.StatusCreated)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	client := NewWebDAVClient(server.URL, "user", "pass")
	cache := NewFingerprintCache(client, 1*time.Hour, 10)
	defer cache.Close()

	cache.putBucket(volumePath, bucketNum, localBucket.Clone(), true)

	if err := cache.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	if putCount != 2 {
		t.Fatalf("putCount = %d, want 2", putCount)
	}

	stats := cache.GetStats()
	if stats["dirty_buckets"].(int) != 0 {
		t.Fatalf("expected 0 dirty buckets after successful retry, got %v", stats["dirty_buckets"])
	}

	bucket, err := cache.getBucket(volumePath, bucketNum)
	if err != nil {
		t.Fatalf("getBucket failed: %v", err)
	}
	if len(bucket.Files) != 1 {
		t.Fatalf("bucket files = %d, want 1", len(bucket.Files))
	}
	if bucket.Files[0].Title != "Local New" {
		t.Fatalf("bucket file title = %q, want %q", bucket.Files[0].Title, "Local New")
	}
	if storedBucket == nil || len(storedBucket.Files) != 1 || storedBucket.Files[0].Title != "Local New" {
		t.Fatalf("stored bucket was not updated with newer local file")
	}
}
