// Package concurrency provides high-performance concurrent processing primitives.
package concurrency

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/antimoji/antimoji/internal/types"
)

// Job represents a unit of work to be processed by workers.
type Job struct {
	ID   string      `json:"id"`
	Data interface{} `json:"data"`
}

// Result represents the result of processing a job.
type Result struct {
	JobID   string      `json:"job_id"`
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   error       `json:"error,omitempty"`
}

// WorkerPool manages a pool of worker goroutines for concurrent processing.
type WorkerPool struct {
	size          int
	workers       []*Worker
	jobQueue      chan Job
	resultQueue   chan Result
	isRunning     int32 // atomic
	activeWorkers int32 // atomic
	processedJobs int64 // atomic
	mu            sync.RWMutex
}

// Worker represents a single worker in the pool.
type Worker struct {
	ID          int
	pool        *WorkerPool
	jobQueue    <-chan Job
	resultQueue chan<- Result
	quit        chan bool
}

// NewWorkerPool creates a new worker pool with the specified size.
func NewWorkerPool(size int) *WorkerPool {
	if size < 0 {
		size = 1 // Default to 1 for invalid input
	} else if size == 0 {
		size = runtime.NumCPU()
	}
	if size > 64 {
		size = 64 // Reasonable upper limit
	}

	return &WorkerPool{
		size:        size,
		workers:     make([]*Worker, size),
		jobQueue:    make(chan Job, size*2), // Buffer for better throughput
		resultQueue: make(chan Result, size*2),
	}
}

// Size returns the number of workers in the pool.
func (wp *WorkerPool) Size() int {
	return wp.size
}

// JobQueue returns the job submission channel.
func (wp *WorkerPool) JobQueue() chan<- Job {
	return wp.jobQueue
}

// ResultQueue returns the result collection channel.
func (wp *WorkerPool) ResultQueue() <-chan Result {
	return wp.resultQueue
}

// Start initializes and starts all workers in the pool.
func (wp *WorkerPool) Start(ctx context.Context) error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if atomic.LoadInt32(&wp.isRunning) == 1 {
		return fmt.Errorf("worker pool is already running")
	}

	// Create and start workers
	for i := 0; i < wp.size; i++ {
		worker := &Worker{
			ID:          i,
			pool:        wp,
			jobQueue:    wp.jobQueue,
			resultQueue: wp.resultQueue,
			quit:        make(chan bool),
		}
		wp.workers[i] = worker
		go worker.start(ctx)
	}

	atomic.StoreInt32(&wp.isRunning, 1)
	atomic.StoreInt32(&wp.activeWorkers, int32(wp.size))

	// Start result collector
	go wp.resultCollector(ctx)

	return nil
}

// Stop gracefully stops all workers in the pool.
func (wp *WorkerPool) Stop() {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if atomic.LoadInt32(&wp.isRunning) == 0 {
		return
	}

	// Signal all workers to quit
	for _, worker := range wp.workers {
		if worker != nil {
			close(worker.quit)
		}
	}

	atomic.StoreInt32(&wp.isRunning, 0)
	atomic.StoreInt32(&wp.activeWorkers, 0)
}

// IsRunning returns true if the worker pool is currently running.
func (wp *WorkerPool) IsRunning() bool {
	return atomic.LoadInt32(&wp.isRunning) == 1
}

// ActiveWorkers returns the number of currently active workers.
func (wp *WorkerPool) ActiveWorkers() int {
	return int(atomic.LoadInt32(&wp.activeWorkers))
}

// QueueDepth returns the current depth of the job queue.
func (wp *WorkerPool) QueueDepth() int {
	return len(wp.jobQueue)
}

// ProcessedJobs returns the total number of jobs processed.
func (wp *WorkerPool) ProcessedJobs() int {
	return int(atomic.LoadInt64(&wp.processedJobs))
}

// CloseJobs closes the job queue to signal no more jobs will be submitted.
func (wp *WorkerPool) CloseJobs() {
	close(wp.jobQueue)
}

// start begins the worker's processing loop.
func (w *Worker) start(ctx context.Context) {
	defer func() {
		atomic.AddInt32(&w.pool.activeWorkers, -1)
	}()

	for {
		select {
		case job, ok := <-w.jobQueue:
			if !ok {
				// Job queue closed, exit worker
				return
			}

			result := w.processJob(job)
			atomic.AddInt64(&w.pool.processedJobs, 1)

			select {
			case w.resultQueue <- result:
				// Result sent successfully
			case <-ctx.Done():
				return
			case <-w.quit:
				return
			}

		case <-ctx.Done():
			return
		case <-w.quit:
			return
		}
	}
}

// processJob processes a single job and returns the result.
func (w *Worker) processJob(job Job) Result {
	// Simulate job processing based on data
	switch job.Data {
	case "error":
		return Result{
			JobID:   job.ID,
			Success: false,
			Error:   fmt.Errorf("simulated processing error"),
		}
	case "delay":
		// Simulate processing delay for testing
		time.Sleep(10 * time.Millisecond)
		return Result{
			JobID:   job.ID,
			Success: true,
			Data:    fmt.Sprintf("processed_%s", job.Data),
		}
	default:
		return Result{
			JobID:   job.ID,
			Success: true,
			Data:    fmt.Sprintf("processed_%s", job.Data),
		}
	}
}

// resultCollector collects results and closes the result queue when done.
func (wp *WorkerPool) resultCollector(ctx context.Context) {
	defer close(wp.resultQueue)

	for {
		select {
		case <-ctx.Done():
			atomic.StoreInt32(&wp.isRunning, 0)
			return
		default:
			// Check if all workers are done
			if atomic.LoadInt32(&wp.activeWorkers) == 0 {
				atomic.StoreInt32(&wp.isRunning, 0)
				return
			}
			time.Sleep(1 * time.Millisecond) // Small delay to prevent busy waiting
		}
	}
}

// ProcessFiles processes files using the worker pool for concurrent processing.
func ProcessFiles(filePaths []string, workerCount int, processor func(string) types.Result[types.ProcessResult]) []types.ProcessResult {
	if workerCount <= 0 {
		workerCount = runtime.NumCPU()
	}

	pool := NewWorkerPool(workerCount)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := pool.Start(ctx)
	if err != nil {
		// Fallback to sequential processing
		return processFilesSequentially(filePaths, processor)
	}
	defer pool.Stop()

	// Submit jobs
	go func() {
		defer pool.CloseJobs()
		for _, filePath := range filePaths {
			job := Job{
				ID:   filePath,
				Data: filePath,
			}
			pool.JobQueue() <- job
		}
	}()

	// Collect results
	results := make([]types.ProcessResult, 0, len(filePaths))
	for result := range pool.ResultQueue() {
		if result.Success {
			// Process the file
			processResult := processor(result.JobID)
			if processResult.IsOk() {
				results = append(results, processResult.Unwrap())
			} else {
				// Handle processing error
				errorResult := types.ProcessResult{
					FilePath: result.JobID,
					Error:    processResult.Error(),
				}
				results = append(results, errorResult)
			}
		} else {
			// Handle job error
			errorResult := types.ProcessResult{
				FilePath: result.JobID,
				Error:    result.Error,
			}
			results = append(results, errorResult)
		}
	}

	return results
}

// processFilesSequentially provides fallback sequential processing.
func processFilesSequentially(filePaths []string, processor func(string) types.Result[types.ProcessResult]) []types.ProcessResult {
	results := make([]types.ProcessResult, 0, len(filePaths))

	for _, filePath := range filePaths {
		processResult := processor(filePath)
		if processResult.IsOk() {
			results = append(results, processResult.Unwrap())
		} else {
			errorResult := types.ProcessResult{
				FilePath: filePath,
				Error:    processResult.Error(),
			}
			results = append(results, errorResult)
		}
	}

	return results
}
