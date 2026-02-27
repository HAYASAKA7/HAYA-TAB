// Package worker provides background job processing utilities.
package worker

import (
	"haya-tab/pkg/metadata"
	"haya-tab/pkg/store"
	"sync"
	"time"
)

// MBJob represents a job to fetch artist origin country from MusicBrainz
type MBJob struct {
	TabID      string
	ArtistName string
}

// MBWorker is a single-threaded worker that processes MusicBrainz requests
// with strict rate limiting (1 request per second) to comply with API rules
type MBWorker struct {
	store    *store.DBStore
	client   *metadata.MusicBrainzClient
	jobQueue chan MBJob
	quit     chan struct{}
	wg       sync.WaitGroup
	running  bool
	mu       sync.Mutex
	logger   Logger
}

// Logger interface for logging (to avoid circular dependency with logger package)
type Logger interface {
	Info(format string, args ...interface{})
	Error(format string, args ...interface{})
}

// NewMBWorker creates a new MusicBrainz worker
func NewMBWorker(store *store.DBStore, logger Logger) *MBWorker {
	return &MBWorker{
		store:    store,
		client:   metadata.NewMusicBrainzClient(),
		jobQueue: make(chan MBJob, 1000), // Buffer up to 1000 jobs
		quit:     make(chan struct{}),
		logger:   logger,
	}
}

// Start begins the worker goroutine
func (w *MBWorker) Start() {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return
	}
	w.running = true
	w.mu.Unlock()

	w.wg.Add(1)
	go w.run()

	if w.logger != nil {
		w.logger.Info("MusicBrainz worker started (rate limit: 1 req/sec)")
	}
}

// Stop gracefully shuts down the worker
func (w *MBWorker) Stop() {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return
	}
	w.running = false
	w.mu.Unlock()

	close(w.quit)
	w.wg.Wait()

	if w.logger != nil {
		w.logger.Info("MusicBrainz worker stopped")
	}
}

// Submit adds a job to the queue (non-blocking, drops if queue is full)
func (w *MBWorker) Submit(job MBJob) {
	w.mu.Lock()
	running := w.running
	w.mu.Unlock()

	if !running {
		return
	}

	select {
	case w.jobQueue <- job:
		// Job queued successfully
	default:
		// Queue is full, drop the job (will be retried on next startup)
		if w.logger != nil {
			w.logger.Info("MusicBrainz job queue full, dropping job for tab: %s", job.TabID)
		}
	}
}

// QueueSize returns the current number of jobs in the queue
func (w *MBWorker) QueueSize() int {
	return len(w.jobQueue)
}

// run is the main worker loop with strict 1 request per second rate limiting
func (w *MBWorker) run() {
	defer w.wg.Done()

	// CRITICAL: MusicBrainz allows only 1 request per second
	// Using a ticker ensures we never exceed this limit
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-w.quit:
			return
		case <-ticker.C:
			// Only process one job per tick (1 per second)
			select {
			case job := <-w.jobQueue:
				w.processJob(job)
			default:
				// No jobs in queue, continue waiting
			}
		}
	}
}

// processJob fetches the artist's origin country and updates the database
func (w *MBWorker) processJob(job MBJob) {
	if job.ArtistName == "" {
		return
	}

	country, err := w.client.SearchArtistCountry(job.ArtistName)
	if err != nil {
		if w.logger != nil {
			w.logger.Info("MusicBrainz lookup failed for '%s': %v", job.ArtistName, err)
		}
		return
	}

	if country == "" {
		return
	}

	// Update the database
	if err := w.store.UpdateTabOriginCountry(job.TabID, country); err != nil {
		if w.logger != nil {
			w.logger.Error("Failed to update origin_country for tab %s: %v", job.TabID, err)
		}
		return
	}

	if w.logger != nil {
		w.logger.Info("Updated origin_country for '%s': %s", job.ArtistName, country)
	}
}
