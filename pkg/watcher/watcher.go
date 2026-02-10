package watcher

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Logger interface for dependency injection
type Logger interface {
	Info(format string, args ...interface{})
	Error(format string, args ...interface{})
}

// FileWatcher watches directories for file changes
type FileWatcher struct {
	watcher    *fsnotify.Watcher
	paths      []string
	onChange   func()
	mu         sync.Mutex
	running    bool
	debounceMs int
	stopChan   chan struct{}
	logger     Logger
}

// NewFileWatcher creates a new file watcher
func NewFileWatcher(onChange func()) *FileWatcher {
	return &FileWatcher{
		onChange:   onChange,
		debounceMs: 1000, // 1 second debounce
		stopChan:   make(chan struct{}),
	}
}

// SetLogger sets the logger
func (w *FileWatcher) SetLogger(l Logger) {
	w.logger = l
}

// Start initializes and starts the file watcher
func (w *FileWatcher) Start() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.running {
		return nil
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	w.watcher = watcher
	w.running = true
	w.stopChan = make(chan struct{})

	go w.watchLoop()

	return nil
}

// Stop stops the file watcher
func (w *FileWatcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.running {
		return
	}

	w.running = false
	close(w.stopChan)

	if w.watcher != nil {
		w.watcher.Close()
		w.watcher = nil
	}
}

// AddPath adds a path to watch
func (w *FileWatcher) AddPath(path string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.watcher == nil {
		return fmt.Errorf("watcher not started")
	}

	// Check if already watching
	for _, p := range w.paths {
		if p == path {
			return nil
		}
	}

	if err := w.watcher.Add(path); err != nil {
		return fmt.Errorf("failed to add path %s: %w", path, err)
	}

	w.paths = append(w.paths, path)
	if w.logger != nil {
		w.logger.Info("Watching path: %s", path)
	}
	return nil
}

// RemovePath removes a path from watching
func (w *FileWatcher) RemovePath(path string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.watcher == nil {
		return nil
	}

	if err := w.watcher.Remove(path); err != nil {
		return err
	}

	// Remove from paths slice
	newPaths := make([]string, 0, len(w.paths))
	for _, p := range w.paths {
		if p != path {
			newPaths = append(newPaths, p)
		}
	}
	w.paths = newPaths

	return nil
}

// SetPaths sets all paths to watch (replaces existing)
func (w *FileWatcher) SetPaths(paths []string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.watcher == nil {
		return fmt.Errorf("watcher not started")
	}

	// Remove old paths
	for _, p := range w.paths {
		w.watcher.Remove(p)
	}
	w.paths = nil

	// Add new paths
	for _, path := range paths {
		if err := w.watcher.Add(path); err != nil {
			if w.logger != nil {
				w.logger.Error("Warning: failed to watch path %s: %v", path, err)
			}
			continue
		}
		w.paths = append(w.paths, path)
		if w.logger != nil {
			w.logger.Info("Watching path: %s", path)
		}
	}

	return nil
}

// GetPaths returns the currently watched paths
func (w *FileWatcher) GetPaths() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]string{}, w.paths...)
}

// IsRunning returns whether the watcher is running
func (w *FileWatcher) IsRunning() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.running
}

// isRelevantFile checks if the file is a tab file we care about
func isRelevantFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".pdf" || ext == ".gp" || ext == ".gp5" || ext == ".gpx"
}

func (w *FileWatcher) watchLoop() {
	var debounceTimer *time.Timer
	var pendingChange bool

	for {
		select {
		case <-w.stopChan:
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			return

		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			// Only care about relevant file types
			if !isRelevantFile(event.Name) {
				continue
			}

			// Only care about create, write, remove, rename
			if event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Remove|fsnotify.Rename) == 0 {
				continue
			}

			if w.logger != nil {
				w.logger.Info("File change detected: %s (%s)", event.Name, event.Op)
			}

			// Debounce: wait for changes to settle
			pendingChange = true
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.AfterFunc(time.Duration(w.debounceMs)*time.Millisecond, func() {
				if pendingChange && w.onChange != nil {
					w.onChange()
					pendingChange = false
				}
			})

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			if w.logger != nil {
				w.logger.Error("Watcher error: %v", err)
			}
		}
	}
}
