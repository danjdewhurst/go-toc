package worker

import (
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
