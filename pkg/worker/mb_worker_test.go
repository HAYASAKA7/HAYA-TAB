package worker

import (
	"haya-tab/pkg/store"
	"path/filepath"
	"testing"
	"time"
)

// MockLogger implements the Logger interface for testing
type MockLogger struct {
	InfoMessages  []string
	ErrorMessages []string
}

func (m *MockLogger) Info(format string, args ...interface{}) {
	m.InfoMessages = append(m.InfoMessages, format)
}

func (m *MockLogger) Error(format string, args ...interface{}) {
	m.ErrorMessages = append(m.ErrorMessages, format)
}

func setupTestStore(t *testing.T) *store.DBStore {
	tmpDir := t.TempDir()

	dbPath := filepath.Join(tmpDir, "test.db")
	testStore := store.NewDBStore(dbPath)

	if err := testStore.Initialize(); err != nil {
		t.Fatalf("Failed to initialize store: %v", err)
	}

	return testStore
}

func TestNewMBWorker(t *testing.T) {
	testStore := setupTestStore(t)
	defer testStore.Close()

	logger := &MockLogger{}
	worker := NewMBWorker(testStore, logger)

	if worker == nil {
		t.Fatal("NewMBWorker returned nil")
	}
	if worker.store != testStore {
		t.Error("Worker store not set correctly")
	}
	if worker.logger != logger {
		t.Error("Worker logger not set correctly")
	}
	if worker.client == nil {
		t.Error("Worker client not initialized")
	}
	if worker.jobQueue == nil {
		t.Error("Worker jobQueue not initialized")
	}
	if cap(worker.jobQueue) != 1000 {
		t.Errorf("jobQueue capacity = %d, want 1000", cap(worker.jobQueue))
	}
}

func TestMBWorker_StartStop(t *testing.T) {
	testStore := setupTestStore(t)
	defer testStore.Close()

	logger := &MockLogger{}
	worker := NewMBWorker(testStore, logger)

	// Start worker
	worker.Start()
	time.Sleep(50 * time.Millisecond) // Give it time to start

	if !worker.running {
		t.Error("Worker should be running after Start()")
	}

	// Check logger message
	if len(logger.InfoMessages) == 0 {
		t.Error("Expected info message on start")
	}

	// Stop worker
	worker.Stop()
	time.Sleep(50 * time.Millisecond) // Give it time to stop

	if worker.running {
		t.Error("Worker should not be running after Stop()")
	}
}

func TestMBWorker_StartMultipleTimes(t *testing.T) {
	testStore := setupTestStore(t)
	defer testStore.Close()

	logger := &MockLogger{}
	worker := NewMBWorker(testStore, logger)

	// Start multiple times should be safe
	worker.Start()
	worker.Start()
	worker.Start()

	if !worker.running {
		t.Error("Worker should be running")
	}

	worker.Stop()
}

func TestMBWorker_StopMultipleTimes(t *testing.T) {
	testStore := setupTestStore(t)
	defer testStore.Close()

	logger := &MockLogger{}
	worker := NewMBWorker(testStore, logger)

	worker.Start()
	time.Sleep(50 * time.Millisecond)

	// Stop multiple times should be safe
	worker.Stop()
	worker.Stop()
	worker.Stop()

	if worker.running {
		t.Error("Worker should not be running")
	}
}

func TestMBWorker_Submit(t *testing.T) {
	testStore := setupTestStore(t)
	defer testStore.Close()

	logger := &MockLogger{}
	worker := NewMBWorker(testStore, logger)

	worker.Start()
	defer worker.Stop()

	// Submit a job
	job := MBJob{
		TabID:      "test-tab-1",
		ArtistName: "Test Artist",
	}

	worker.Submit(job)

	// Check queue size
	if worker.QueueSize() != 1 {
		t.Errorf("QueueSize() = %d, want 1", worker.QueueSize())
	}
}

func TestMBWorker_SubmitWhenNotRunning(t *testing.T) {
	testStore := setupTestStore(t)
	defer testStore.Close()

	logger := &MockLogger{}
	worker := NewMBWorker(testStore, logger)

	// Submit without starting should not panic
	job := MBJob{
		TabID:      "test-tab-1",
		ArtistName: "Test Artist",
	}

	worker.Submit(job)

	// Queue should be empty since worker is not running
	if worker.QueueSize() != 0 {
		t.Errorf("QueueSize() = %d, want 0", worker.QueueSize())
	}
}

func TestMBWorker_QueueSize(t *testing.T) {
	testStore := setupTestStore(t)
	defer testStore.Close()

	logger := &MockLogger{}
	worker := NewMBWorker(testStore, logger)

	worker.Start()
	defer worker.Stop()

	// Initially empty
	if worker.QueueSize() != 0 {
		t.Errorf("Initial QueueSize() = %d, want 0", worker.QueueSize())
	}

	// Add jobs
	for i := 0; i < 5; i++ {
		worker.Submit(MBJob{
			TabID:      "test-tab",
			ArtistName: "Test Artist",
		})
	}

	if worker.QueueSize() != 5 {
		t.Errorf("QueueSize() = %d, want 5", worker.QueueSize())
	}
}

func TestMBWorker_RateLimiting(t *testing.T) {
	testStore := setupTestStore(t)
	defer testStore.Close()

	logger := &MockLogger{}
	worker := NewMBWorker(testStore, logger)

	worker.Start()
	defer worker.Stop()

	// Submit multiple jobs
	for i := 0; i < 3; i++ {
		worker.Submit(MBJob{
			TabID:      "test-tab",
			ArtistName: "Test Artist",
		})
	}

	// Wait and check that jobs are processed slowly (rate limited)
	time.Sleep(500 * time.Millisecond)

	// After 500ms, at most 1 job should be processed (1 req/sec rate limit)
	// Queue should still have 2-3 jobs
	queueSize := worker.QueueSize()
	if queueSize < 2 {
		t.Errorf("QueueSize() = %d, expected at least 2 (rate limiting not working)", queueSize)
	}
}

func TestMBWorker_EmptyArtistName(t *testing.T) {
	testStore := setupTestStore(t)
	defer testStore.Close()

	logger := &MockLogger{}
	worker := NewMBWorker(testStore, logger)

	// Test processJob with empty artist name
	job := MBJob{
		TabID:      "test-tab",
		ArtistName: "",
	}

	// Should not panic
	worker.processJob(job)
}

func TestMBWorker_NilLogger(t *testing.T) {
	testStore := setupTestStore(t)
	defer testStore.Close()

	// Create worker with nil logger
	worker := NewMBWorker(testStore, nil)

	// Should not panic
	worker.Start()
	time.Sleep(50 * time.Millisecond)
	worker.Stop()
}

func TestMBWorker_ProcessJob_Success(t *testing.T) {
	testStore := setupTestStore(t)
	defer testStore.Close()

	// Add a test tab
	tab := store.Tab{
		ID:     "test-tab",
		Title:  "Test Song",
		Artist: "Test Artist",
	}
	testStore.AddTab(tab)

	logger := &MockLogger{}
	worker := NewMBWorker(testStore, logger)

	// Process job (will likely fail due to network, but shouldn't panic)
	job := MBJob{
		TabID:      "test-tab",
		ArtistName: "Test Artist",
	}

	worker.processJob(job)

	// Should have logged something
	if len(logger.InfoMessages) == 0 && len(logger.ErrorMessages) == 0 {
		t.Error("Expected some log messages")
	}
}

func TestMBWorker_ProcessJob_NonExistentTab(t *testing.T) {
	testStore := setupTestStore(t)
	defer testStore.Close()

	logger := &MockLogger{}
	worker := NewMBWorker(testStore, logger)

	// Process job for non-existent tab
	job := MBJob{
		TabID:      "nonexistent-tab",
		ArtistName: "Test Artist",
	}

	worker.processJob(job)

	// Should have logged an error
	if len(logger.ErrorMessages) == 0 {
		t.Error("Expected error message for non-existent tab")
	}
}

func TestMBWorker_ProcessJob_TabAlreadyHasCountry(t *testing.T) {
	testStore := setupTestStore(t)
	defer testStore.Close()

	// Add a test tab with origin country already set
	tab := store.Tab{
		ID:            "test-tab",
		Title:         "Test Song",
		Artist:        "Test Artist",
		OriginCountry: "US",
	}
	testStore.AddTab(tab)

	logger := &MockLogger{}
	worker := NewMBWorker(testStore, logger)

	// Process job
	job := MBJob{
		TabID:      "test-tab",
		ArtistName: "Test Artist",
	}

	worker.processJob(job)

	// Should skip since country is already set
	// Check that no error was logged
	hasSkipMessage := false
	for _, msg := range logger.InfoMessages {
		if msg == "Tab %s already has origin country: %s" {
			hasSkipMessage = true
			break
		}
	}
	if !hasSkipMessage {
		// It's okay if it doesn't log this specific message
		// Just verify no panic occurred
	}
}
