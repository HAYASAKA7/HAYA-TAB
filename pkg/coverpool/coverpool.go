package coverpool

import (
	"context"
	"sync"
)

// CoverJob represents a cover download task
type CoverJob struct {
	TabID      string
	Artist     string
	Album      string
	Title      string
	Country    string
	Language   string
	CoverPath  string
	OnComplete func(tabID, coverPath string, err error)
}

// CoverPool manages concurrent cover download workers
type CoverPool struct {
	jobs       chan CoverJob
	workers    int
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	downloadFn func(artist, album, title, country, lang, dstPath string) error
}

// NewCoverPool creates a new worker pool with the specified number of workers
func NewCoverPool(workers int, downloadFn func(artist, album, title, country, lang, dstPath string) error) *CoverPool {
	if workers < 1 {
		workers = 3
	}
	ctx, cancel := context.WithCancel(context.Background())
	pool := &CoverPool{
		jobs:       make(chan CoverJob, 100), // Buffer for pending jobs
		workers:    workers,
		ctx:        ctx,
		cancel:     cancel,
		downloadFn: downloadFn,
	}
	return pool
}

// Start launches the worker goroutines
func (p *CoverPool) Start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

// worker processes jobs from the queue
func (p *CoverPool) worker(id int) {
	defer p.wg.Done()
	for {
		select {
		case <-p.ctx.Done():
			return
		case job, ok := <-p.jobs:
			if !ok {
				return
			}
			err := p.downloadFn(job.Artist, job.Album, job.Title, job.Country, job.Language, job.CoverPath)
			if job.OnComplete != nil {
				job.OnComplete(job.TabID, job.CoverPath, err)
			}
		}
	}
}

// Submit adds a new job to the queue
func (p *CoverPool) Submit(job CoverJob) {
	select {
	case p.jobs <- job:
		// Job submitted
	case <-p.ctx.Done():
		// Pool is shutting down
	}
}

// SubmitAsync adds a job without blocking (drops if queue is full)
func (p *CoverPool) SubmitAsync(job CoverJob) bool {
	select {
	case p.jobs <- job:
		return true
	default:
		return false // Queue full, job dropped
	}
}

// Stop gracefully shuts down the worker pool
func (p *CoverPool) Stop() {
	p.cancel()
	close(p.jobs)
	p.wg.Wait()
}

// QueueSize returns the current number of pending jobs
func (p *CoverPool) QueueSize() int {
	return len(p.jobs)
}
