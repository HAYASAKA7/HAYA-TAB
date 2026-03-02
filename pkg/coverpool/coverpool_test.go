package coverpool

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewCoverPool(t *testing.T) {
	tests := []struct {
		name    string
		workers int
		want    int
	}{
		{
			name:    "Valid worker count",
			workers: 5,
			want:    5,
		},
		{
			name:    "Zero workers defaults to 3",
			workers: 0,
			want:    3,
		},
		{
			name:    "Negative workers defaults to 3",
			workers: -1,
			want:    3,
		},
	}

	mockDownload := func(artist, album, title, country, lang, dstPath string) error {
		return nil
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := NewCoverPool(tt.workers, mockDownload)
			if pool.workers != tt.want {
				t.Errorf("NewCoverPool() workers = %v, want %v", pool.workers, tt.want)
			}
			if pool.downloadFn == nil {
				t.Error("NewCoverPool() downloadFn is nil")
			}
		})
	}
}

func TestCoverPool_StartStop(t *testing.T) {
	mockDownload := func(artist, album, title, country, lang, dstPath string) error {
		return nil
	}

	pool := NewCoverPool(2, mockDownload)
	pool.Start()

	// Give workers time to start
	time.Sleep(10 * time.Millisecond)

	pool.Stop()

	// Verify pool is stopped by checking if context is cancelled
	select {
	case <-pool.ctx.Done():
		// Expected: context should be cancelled
	default:
		t.Error("Pool context not cancelled after Stop()")
	}
}

func TestCoverPool_Submit(t *testing.T) {
	var processedCount int32
	var mu sync.Mutex
	var processedJobs []string

	mockDownload := func(artist, album, title, country, lang, dstPath string) error {
		atomic.AddInt32(&processedCount, 1)
		mu.Lock()
		processedJobs = append(processedJobs, artist)
		mu.Unlock()
		return nil
	}

	pool := NewCoverPool(2, mockDownload)
	pool.Start()
	defer pool.Stop()

	// Submit multiple jobs
	jobs := []CoverJob{
		{TabID: "1", Artist: "Artist1", Album: "Album1", Title: "Title1", CoverPath: "/path/1.jpg"},
		{TabID: "2", Artist: "Artist2", Album: "Album2", Title: "Title2", CoverPath: "/path/2.jpg"},
		{TabID: "3", Artist: "Artist3", Album: "Album3", Title: "Title3", CoverPath: "/path/3.jpg"},
	}

	for _, job := range jobs {
		pool.Submit(job)
	}

	// Wait for jobs to complete
	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt32(&processedCount) != 3 {
		t.Errorf("Expected 3 jobs processed, got %d", processedCount)
	}

	mu.Lock()
	if len(processedJobs) != 3 {
		t.Errorf("Expected 3 jobs in processedJobs, got %d", len(processedJobs))
	}
	mu.Unlock()
}

func TestCoverPool_OnComplete(t *testing.T) {
	var completedTabID string
	var completedPath string
	var completedErr error
	var wg sync.WaitGroup

	mockDownload := func(artist, album, title, country, lang, dstPath string) error {
		return nil
	}

	pool := NewCoverPool(1, mockDownload)
	pool.Start()
	defer pool.Stop()

	wg.Add(1)
	job := CoverJob{
		TabID:     "test-tab",
		Artist:    "Test Artist",
		CoverPath: "/test/path.jpg",
		OnComplete: func(tabID, coverPath string, err error) {
			completedTabID = tabID
			completedPath = coverPath
			completedErr = err
			wg.Done()
		},
	}

	pool.Submit(job)
	wg.Wait()

	if completedTabID != "test-tab" {
		t.Errorf("OnComplete tabID = %v, want test-tab", completedTabID)
	}
	if completedPath != "/test/path.jpg" {
		t.Errorf("OnComplete coverPath = %v, want /test/path.jpg", completedPath)
	}
	if completedErr != nil {
		t.Errorf("OnComplete err = %v, want nil", completedErr)
	}
}

func TestCoverPool_ErrorHandling(t *testing.T) {
	var receivedErr error
	var wg sync.WaitGroup

	mockDownload := func(artist, album, title, country, lang, dstPath string) error {
		return errors.New("download failed")
	}

	pool := NewCoverPool(1, mockDownload)
	pool.Start()
	defer pool.Stop()

	wg.Add(1)
	job := CoverJob{
		TabID:     "error-tab",
		Artist:    "Error Artist",
		CoverPath: "/error/path.jpg",
		OnComplete: func(tabID, coverPath string, err error) {
			receivedErr = err
			wg.Done()
		},
	}

	pool.Submit(job)
	wg.Wait()

	if receivedErr == nil {
		t.Error("Expected error in OnComplete, got nil")
	}
	if receivedErr.Error() != "download failed" {
		t.Errorf("Expected 'download failed', got %v", receivedErr.Error())
	}
}

func TestCoverPool_SubmitAsync(t *testing.T) {
	var processedCount int32
	mockDownload := func(artist, album, title, country, lang, dstPath string) error {
		atomic.AddInt32(&processedCount, 1)
		return nil
	}

	pool := NewCoverPool(2, mockDownload)
	pool.Start()
	defer pool.Stop()

	// Test successful async submit
	job := CoverJob{
		TabID:     "async-tab",
		Artist:    "Async Artist",
		CoverPath: "/async.jpg",
	}

	result := pool.SubmitAsync(job)
	if !result {
		t.Error("SubmitAsync should return true when queue has space")
	}

	// Wait for job to be processed
	time.Sleep(50 * time.Millisecond)

	if atomic.LoadInt32(&processedCount) != 1 {
		t.Errorf("Expected 1 job processed, got %d", processedCount)
	}
}

func TestCoverPool_QueueSize(t *testing.T) {
	mockDownload := func(artist, album, title, country, lang, dstPath string) error {
		time.Sleep(50 * time.Millisecond)
		return nil
	}

	pool := NewCoverPool(1, mockDownload)
	pool.Start()
	defer pool.Stop()

	// Submit jobs
	for i := 0; i < 5; i++ {
		job := CoverJob{
			TabID:     "tab-" + string(rune(i)),
			Artist:    "Artist",
			CoverPath: "/path.jpg",
		}
		pool.Submit(job)
	}

	// Check queue size
	size := pool.QueueSize()
	if size < 1 || size > 5 {
		t.Errorf("QueueSize() = %v, expected between 1 and 5", size)
	}
}

func TestCoverPool_ConcurrentWorkers(t *testing.T) {
	var activeWorkers int32
	var maxConcurrent int32
	var mu sync.Mutex

	mockDownload := func(artist, album, title, country, lang, dstPath string) error {
		current := atomic.AddInt32(&activeWorkers, 1)

		mu.Lock()
		if current > maxConcurrent {
			maxConcurrent = current
		}
		mu.Unlock()

		time.Sleep(50 * time.Millisecond)
		atomic.AddInt32(&activeWorkers, -1)
		return nil
	}

	workerCount := 3
	pool := NewCoverPool(workerCount, mockDownload)
	pool.Start()
	defer pool.Stop()

	// Submit more jobs than workers
	for i := 0; i < 10; i++ {
		job := CoverJob{
			TabID:     "tab-" + string(rune(i)),
			Artist:    "Artist",
			CoverPath: "/path.jpg",
		}
		pool.Submit(job)
	}

	// Wait for all jobs to complete
	time.Sleep(500 * time.Millisecond)

	mu.Lock()
	max := maxConcurrent
	mu.Unlock()

	if max > int32(workerCount) {
		t.Errorf("Max concurrent workers = %v, want <= %v", max, workerCount)
	}
	if max < 1 {
		t.Error("No workers were active")
	}
}
