package worker

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestPoolBasic(t *testing.T) {
	processed := make(map[string]bool)
	var mu sync.Mutex

	processFunc := func(job Job) Result {
		mu.Lock()
		processed[job.FilePath] = true
		mu.Unlock()
		return Result{
			FilePath: job.FilePath,
			Summary:  "processed",
		}
	}

	pool := NewPool(2, processFunc)
	pool.Start()

	// Submit jobs
	files := []string{"file1.md", "file2.md", "file3.md"}
	go func() {
		for _, f := range files {
			pool.Submit(Job{FilePath: f})
		}
		close(pool.jobs)
	}()

	// Collect results
	var results []Result
	go func() {
		pool.wg.Wait()
		close(pool.results)
	}()

	for r := range pool.results {
		results = append(results, r)
	}

	// Verify all files were processed
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}

	for _, f := range files {
		if !processed[f] {
			t.Errorf("file %s was not processed", f)
		}
	}
}

func TestProcessAll(t *testing.T) {
	jobs := []Job{
		{FilePath: "file1.md", Data: 100},
		{FilePath: "file2.md", Data: 100},
		{FilePath: "file3.md", Data: 100},
	}

	processFunc := func(job Job) Result {
		return Result{
			FilePath: job.FilePath,
			Summary:  "summary for " + job.FilePath,
		}
	}

	results := ProcessAll(jobs, 2, processFunc)

	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}

	for _, job := range jobs {
		result, ok := results[job.FilePath]
		if !ok {
			t.Errorf("missing result for %s", job.FilePath)
			continue
		}
		if result.Summary != "summary for "+job.FilePath {
			t.Errorf("unexpected summary for %s: %s", job.FilePath, result.Summary)
		}
	}
}

func TestProcessSequential(t *testing.T) {
	var order []string
	var mu sync.Mutex

	jobs := []Job{
		{FilePath: "file1.md"},
		{FilePath: "file2.md"},
		{FilePath: "file3.md"},
	}

	processFunc := func(job Job) Result {
		mu.Lock()
		order = append(order, job.FilePath)
		mu.Unlock()
		return Result{
			FilePath: job.FilePath,
			Summary:  "done",
		}
	}

	results := ProcessSequential(jobs, processFunc)

	// All should be processed
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}

	// Order should be preserved in sequential mode
	for i, job := range jobs {
		if order[i] != job.FilePath {
			t.Errorf("position %d: expected %s, got %s", i, job.FilePath, order[i])
		}
	}
}

func TestPoolConcurrency(t *testing.T) {
	var concurrent int
	var maxConcurrent int
	var mu sync.Mutex

	processFunc := func(job Job) Result {
		mu.Lock()
		concurrent++
		if concurrent > maxConcurrent {
			maxConcurrent = concurrent
		}
		mu.Unlock()

		time.Sleep(10 * time.Millisecond) // Simulate work

		mu.Lock()
		concurrent--
		mu.Unlock()

		return Result{FilePath: job.FilePath}
	}

	jobs := make([]Job, 10)
	for i := range jobs {
		jobs[i] = Job{FilePath: "file.md"}
	}

	ProcessAll(jobs, 4, processFunc)

	// With 4 workers, max concurrent should be around 4
	if maxConcurrent < 2 || maxConcurrent > 4 {
		t.Errorf("expected max concurrent ~4, got %d", maxConcurrent)
	}
}

func TestPoolEmptyJobs(t *testing.T) {
	processFunc := func(job Job) Result {
		return Result{FilePath: job.FilePath}
	}

	results := ProcessAll([]Job{}, 4, processFunc)

	if len(results) != 0 {
		t.Errorf("expected 0 results for empty jobs, got %d", len(results))
	}
}

func TestPoolWithErrors(t *testing.T) {
	jobs := []Job{
		{FilePath: "good.md"},
		{FilePath: "bad.md"},
	}

	processFunc := func(job Job) Result {
		if job.FilePath == "bad.md" {
			return Result{
				FilePath: job.FilePath,
				Error:    os.ErrNotExist,
			}
		}
		return Result{
			FilePath: job.FilePath,
			Summary:  "success",
		}
	}

	results := ProcessAll(jobs, 2, processFunc)

	if results["good.md"].Error != nil {
		t.Error("good.md should not have error")
	}
	if results["bad.md"].Error == nil {
		t.Error("bad.md should have error")
	}
}

// For TestPoolWithErrors
var os = struct {
	ErrNotExist error
}{
	ErrNotExist: &testError{"file not found"},
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestPoolDoubleClose(t *testing.T) {
	processFunc := func(job Job) Result {
		return Result{FilePath: job.FilePath, Summary: "done"}
	}

	pool := NewPool(2, processFunc)
	pool.Start()

	// Submit job and close properly via Close()
	go func() {
		pool.Submit(Job{FilePath: "test.md"})
	}()

	// Give time for job to be submitted
	time.Sleep(10 * time.Millisecond)

	// This should not panic due to double close
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Close() panicked on double close: %v", r)
		}
	}()

	pool.Close()
	pool.Close() // Second close should be safe
}

func TestProcessAllWithContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	var processed int
	var mu sync.Mutex

	jobs := make([]Job, 100)
	for i := range jobs {
		jobs[i] = Job{FilePath: "file.md"}
	}

	processFunc := func(job Job) Result {
		mu.Lock()
		processed++
		count := processed
		mu.Unlock()

		// Cancel after processing 5 jobs
		if count == 5 {
			cancel()
		}

		time.Sleep(10 * time.Millisecond) // Simulate work
		return Result{FilePath: job.FilePath, Summary: "done"}
	}

	results := ProcessAllWithContext(ctx, jobs, 2, processFunc)

	// Should have partial results due to cancellation
	if len(results) >= len(jobs) {
		t.Errorf("expected partial results due to cancellation, got all %d", len(results))
	}
}

func TestProcessSequentialWithContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	jobs := make([]Job, 10)
	for i := range jobs {
		jobs[i] = Job{FilePath: "file.md", Data: i}
	}

	processCount := 0
	processFunc := func(job Job) Result {
		processCount++
		if processCount == 3 {
			cancel()
		}
		return Result{FilePath: job.FilePath, Summary: "done"}
	}

	results := ProcessSequentialWithContext(ctx, jobs, processFunc)

	// Should have stopped after 3 jobs (the one that called cancel still completes)
	if len(results) > 4 {
		t.Errorf("expected at most 4 results due to cancellation, got %d", len(results))
	}
}

func TestProcessAllWithContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	jobs := make([]Job, 100)
	for i := range jobs {
		jobs[i] = Job{FilePath: "file.md"}
	}

	processFunc := func(job Job) Result {
		time.Sleep(20 * time.Millisecond) // Each job takes 20ms
		return Result{FilePath: job.FilePath, Summary: "done"}
	}

	results := ProcessAllWithContext(ctx, jobs, 2, processFunc)

	// With 50ms timeout and 20ms per job with 2 workers, should process ~4-5 jobs
	if len(results) >= len(jobs) {
		t.Errorf("expected partial results due to timeout, got all %d", len(results))
	}
}
