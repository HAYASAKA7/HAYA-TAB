package watcher

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Mock logger for testing
type mockLogger struct {
	mu     sync.Mutex
	infos  []string
	errors []string
}

func (m *mockLogger) Info(format string, args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.infos = append(m.infos, format)
}

func (m *mockLogger) Error(format string, args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errors = append(m.errors, format)
}

func TestNewFileWatcher(t *testing.T) {
	onChange := func() {
		// Test callback
	}

	watcher := NewFileWatcher(onChange)

	if watcher == nil {
		t.Fatal("NewFileWatcher returned nil")
	}
	if watcher.onChange == nil {
		t.Error("onChange callback is nil")
	}
	if watcher.debounceMs != 500 {
		t.Errorf("debounceMs = %v, want 500", watcher.debounceMs)
	}
}

func TestFileWatcher_StartStop(t *testing.T) {
	watcher := NewFileWatcher(func() {})
	logger := &mockLogger{}
	watcher.SetLogger(logger)

	err := watcher.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if !watcher.IsRunning() {
		t.Error("Watcher should be running after Start()")
	}

	watcher.Stop()

	if watcher.IsRunning() {
		t.Error("Watcher should not be running after Stop()")
	}
}

func TestFileWatcher_MultipleStartStop(t *testing.T) {
	watcher := NewFileWatcher(func() {})

	// Multiple starts should be safe
	err := watcher.Start()
	if err != nil {
		t.Fatalf("First Start() error = %v", err)
	}

	err = watcher.Start()
	if err != nil {
		t.Errorf("Second Start() error = %v", err)
	}

	if !watcher.IsRunning() {
		t.Error("Watcher should be running")
	}

	// Multiple stops should be safe
	watcher.Stop()
	watcher.Stop()

	if watcher.IsRunning() {
		t.Error("Watcher should not be running")
	}
}

func TestFileWatcher_AddRemovePath(t *testing.T) {
	watcher := NewFileWatcher(func() {})
	logger := &mockLogger{}
	watcher.SetLogger(logger)

	err := watcher.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer watcher.Stop()

	// Create a temporary directory
	tmpDir := t.TempDir()

	// Add path
	err = watcher.AddPath(tmpDir)
	if err != nil {
		t.Errorf("AddPath() error = %v", err)
	}

	paths := watcher.GetPaths()
	if len(paths) != 1 {
		t.Errorf("GetPaths() length = %v, want 1", len(paths))
	}
	if paths[0] != tmpDir {
		t.Errorf("GetPaths()[0] = %v, want %v", paths[0], tmpDir)
	}

	// Remove path
	err = watcher.RemovePath(tmpDir)
	if err != nil {
		t.Errorf("RemovePath() error = %v", err)
	}

	paths = watcher.GetPaths()
	if len(paths) != 0 {
		t.Errorf("GetPaths() length = %v, want 0", len(paths))
	}
}

func TestFileWatcher_SetPaths(t *testing.T) {
	watcher := NewFileWatcher(func() {})
	logger := &mockLogger{}
	watcher.SetLogger(logger)

	err := watcher.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer watcher.Stop()

	// Create temporary directories
	tmpDir1 := t.TempDir()

	tmpDir2 := t.TempDir()

	// Set paths
	err = watcher.SetPaths([]string{tmpDir1, tmpDir2})
	if err != nil {
		t.Errorf("SetPaths() error = %v", err)
	}

	paths := watcher.GetPaths()
	if len(paths) != 2 {
		t.Errorf("GetPaths() length = %v, want 2", len(paths))
	}
}

func TestFileWatcher_FileChange(t *testing.T) {
	var changeCount int32
	onChange := func() {
		atomic.AddInt32(&changeCount, 1)
	}

	watcher := NewFileWatcher(onChange)
	logger := &mockLogger{}
	watcher.SetLogger(logger)

	err := watcher.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer watcher.Stop()

	// Create a temporary directory
	tmpDir := t.TempDir()

	// Add path
	err = watcher.AddPath(tmpDir)
	if err != nil {
		t.Fatalf("AddPath() error = %v", err)
	}

	// Create a PDF file (relevant file type)
	testFile := filepath.Join(tmpDir, "test.pdf")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Wait for debounce + processing
	time.Sleep(1 * time.Second)

	// Verify onChange was called
	count := atomic.LoadInt32(&changeCount)
	if count < 1 {
		t.Errorf("onChange called %d times, want >= 1", count)
	}
}

func TestFileWatcher_IgnoreNonRelevantFiles(t *testing.T) {
	var changeCount int32
	onChange := func() {
		atomic.AddInt32(&changeCount, 1)
	}

	watcher := NewFileWatcher(onChange)
	logger := &mockLogger{}
	watcher.SetLogger(logger)

	err := watcher.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer watcher.Stop()

	// Create a temporary directory
	tmpDir := t.TempDir()

	// Add path
	err = watcher.AddPath(tmpDir)
	if err != nil {
		t.Fatalf("AddPath() error = %v", err)
	}

	// Create a non-relevant file (e.g., .txt)
	testFile := filepath.Join(tmpDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Wait for debounce + processing
	time.Sleep(1 * time.Second)

	// Verify onChange was NOT called (non-relevant file)
	count := atomic.LoadInt32(&changeCount)
	if count != 0 {
		t.Errorf("onChange called %d times for non-relevant file, want 0", count)
	}
}

func TestFileWatcher_Debounce(t *testing.T) {
	var changeCount int32
	onChange := func() {
		atomic.AddInt32(&changeCount, 1)
	}

	watcher := NewFileWatcher(onChange)
	logger := &mockLogger{}
	watcher.SetLogger(logger)

	err := watcher.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer watcher.Stop()

	// Create a temporary directory
	tmpDir := t.TempDir()

	// Add path
	err = watcher.AddPath(tmpDir)
	if err != nil {
		t.Fatalf("AddPath() error = %v", err)
	}

	// Create multiple files quickly
	for i := 0; i < 5; i++ {
		testFile := filepath.Join(tmpDir, fmt.Sprintf("test%d.pdf", i))
		err = os.WriteFile(testFile, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Wait for debounce + processing
	time.Sleep(1 * time.Second)

	// Verify onChange was called only once due to debouncing
	count := atomic.LoadInt32(&changeCount)
	if count != 1 {
		t.Errorf("onChange called %d times, want 1 (debounced)", count)
	}
}

func TestIsRelevantFile(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{"PDF file", "/path/to/file.pdf", true},
		{"GP file", "/path/to/file.gp", true},
		{"GP3 file", "/path/to/file.gp3", true},
		{"GP4 file", "/path/to/file.gp4", true},
		{"GP5 file", "/path/to/file.gp5", true},
		{"GPX file", "/path/to/file.gpx", true},
		{"XML file", "/path/to/file.xml", true},
		{"MusicXML file", "/path/to/file.musicxml", true},
		{"MXL file", "/path/to/file.mxl", true},
		{"TXT file", "/path/to/file.txt", false},
		{"JPG file", "/path/to/file.jpg", false},
		{"No extension", "/path/to/file", false},
		{"Uppercase PDF", "/path/to/file.PDF", true},
		{"Mixed case GPX", "/path/to/file.GpX", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRelevantFile(tt.path)
			if got != tt.want {
				t.Errorf("isRelevantFile(%v) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestFileWatcher_AddPathBeforeStart(t *testing.T) {
	watcher := NewFileWatcher(func() {})

	// Try to add path before starting
	err := watcher.AddPath("/some/path")
	if err == nil {
		t.Error("AddPath() should return error when watcher not started")
	}
}

func TestFileWatcher_SetPathsBeforeStart(t *testing.T) {
	watcher := NewFileWatcher(func() {})

	// Try to set paths before starting
	err := watcher.SetPaths([]string{"/path1", "/path2"})
	if err == nil {
		t.Error("SetPaths() should return error when watcher not started")
	}
}
