package worker

import (
	"sync"
)

// Job represents a unit of work to be processed.
type Job struct {
	FilePath string
	Data     any
}

// Result represents the output of a processed job.
type Result struct {
	FilePath string
	Summary  string
	Error    error
}

// ProcessFunc is the function type for processing jobs.
type ProcessFunc func(job Job) Result

// Pool manages a pool of worker goroutines.
type Pool struct {
	workers    int
	jobs       chan Job
	results    chan Result
	wg         sync.WaitGroup
	processFunc ProcessFunc
}

// NewPool creates a new worker pool with the specified number of workers.
func NewPool(workers int, processFunc ProcessFunc) *Pool {
	if workers <= 0 {
		workers = 1
	}

	return &Pool{
		workers:     workers,
		jobs:        make(chan Job, workers*2),
		results:     make(chan Result, workers*2),
		processFunc: processFunc,
	}
}

// Start begins the worker goroutines.
func (p *Pool) Start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker()
	}
}

// worker is the goroutine that processes jobs.
func (p *Pool) worker() {
	defer p.wg.Done()

	for job := range p.jobs {
		result := p.processFunc(job)
		p.results <- result
	}
}

// Submit adds a job to the pool.
func (p *Pool) Submit(job Job) {
	p.jobs <- job
}

// Close signals that no more jobs will be submitted and waits for completion.
func (p *Pool) Close() {
	close(p.jobs)
	p.wg.Wait()
	close(p.results)
}

// Results returns the results channel for reading.
func (p *Pool) Results() <-chan Result {
	return p.results
}

// ProcessAll processes all jobs and returns results as a map.
// This is a convenience method that handles the full workflow.
func ProcessAll(jobs []Job, workers int, processFunc ProcessFunc) map[string]Result {
	if len(jobs) == 0 {
		return make(map[string]Result)
	}

	pool := NewPool(workers, processFunc)
	pool.Start()

	// Submit all jobs
	go func() {
		for _, job := range jobs {
			pool.Submit(job)
		}
		close(pool.jobs)
	}()

	// Collect results
	results := make(map[string]Result)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		pool.wg.Wait()
		close(pool.results)
	}()

	for result := range pool.results {
		results[result.FilePath] = result
	}

	wg.Wait()
	return results
}

// ProcessSequential processes jobs sequentially (single-threaded).
func ProcessSequential(jobs []Job, processFunc ProcessFunc) map[string]Result {
	results := make(map[string]Result)

	for _, job := range jobs {
		result := processFunc(job)
		results[result.FilePath] = result
	}

	return results
}
