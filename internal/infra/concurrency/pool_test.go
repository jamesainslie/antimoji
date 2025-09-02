package concurrency

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/antimoji/antimoji/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestNewWorkerPool(t *testing.T) {
	t.Run("creates worker pool with correct size", func(t *testing.T) {
		pool := NewWorkerPool(4)

		assert.Equal(t, 4, pool.Size())
		assert.NotNil(t, pool.JobQueue())
		assert.NotNil(t, pool.ResultQueue())
	})

	t.Run("auto-detects worker count when size is 0", func(t *testing.T) {
		pool := NewWorkerPool(0)

		// Should use runtime.NumCPU()
		assert.True(t, pool.Size() > 0)
		assert.True(t, pool.Size() <= 64) // Reasonable upper bound
	})

	t.Run("handles invalid worker count", func(t *testing.T) {
		pool := NewWorkerPool(-1)

		// Should default to 1 worker for invalid input
		assert.Equal(t, 1, pool.Size())
	})
}

func TestWorkerPool_Lifecycle(t *testing.T) {
	t.Run("starts and stops workers correctly", func(t *testing.T) {
		pool := NewWorkerPool(2)
		ctx, cancel := context.WithCancel(context.Background())

		// Start the pool
		err := pool.Start(ctx)
		assert.NoError(t, err)

		// Verify workers are running
		assert.True(t, pool.IsRunning())
		assert.Equal(t, 2, pool.ActiveWorkers())

		// Stop the pool
		cancel()

		// Give workers time to stop
		time.Sleep(10 * time.Millisecond)

		// Verify workers stopped
		assert.False(t, pool.IsRunning())
		assert.Equal(t, 0, pool.ActiveWorkers())
	})

	t.Run("handles context cancellation gracefully", func(t *testing.T) {
		pool := NewWorkerPool(1)
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		err := pool.Start(ctx)
		assert.NoError(t, err)

		// Wait for context timeout
		<-ctx.Done()

		// Pool should stop gracefully
		time.Sleep(10 * time.Millisecond)
		assert.False(t, pool.IsRunning())
	})
}

func TestWorkerPool_Processing(t *testing.T) {
	t.Run("processes jobs correctly", func(t *testing.T) {
		pool := NewWorkerPool(2)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err := pool.Start(ctx)
		assert.NoError(t, err)

		// Create test jobs
		jobs := []Job{
			{ID: "job1", Data: "test data 1"},
			{ID: "job2", Data: "test data 2"},
			{ID: "job3", Data: "test data 3"},
		}

		// Submit jobs
		go func() {
			defer pool.CloseJobs()
			for _, job := range jobs {
				pool.JobQueue() <- job
			}
		}()

		// Collect results
		var results []Result
		for result := range pool.ResultQueue() {
			results = append(results, result)
		}

		// Verify all jobs were processed
		assert.Len(t, results, 3)

		// Check that all jobs completed successfully
		for _, result := range results {
			assert.NoError(t, result.Error)
			assert.True(t, result.Success)
		}
	})

	t.Run("handles job processing errors", func(t *testing.T) {
		pool := NewWorkerPool(1)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err := pool.Start(ctx)
		assert.NoError(t, err)

		// Create job that will cause an error
		errorJob := Job{
			ID:   "error_job",
			Data: "error", // Special data that causes processing error
		}

		// Submit error job
		go func() {
			defer pool.CloseJobs()
			pool.JobQueue() <- errorJob
		}()

		// Collect result
		result := <-pool.ResultQueue()

		// Verify error was handled
		assert.Error(t, result.Error)
		assert.False(t, result.Success)
		assert.Equal(t, "error_job", result.JobID)
	})

	t.Run("processes jobs concurrently", func(t *testing.T) {
		pool := NewWorkerPool(3)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err := pool.Start(ctx)
		assert.NoError(t, err)

		// Create jobs with artificial delay to test concurrency
		numJobs := 6
		startTime := time.Now()

		// Submit jobs
		go func() {
			defer pool.CloseJobs()
			for i := 0; i < numJobs; i++ {
				job := Job{
					ID:   fmt.Sprintf("job%d", i),
					Data: "delay", // Special data that causes delay
				}
				pool.JobQueue() <- job
			}
		}()

		// Collect all results
		var results []Result
		for result := range pool.ResultQueue() {
			results = append(results, result)
		}

		processingTime := time.Since(startTime)

		// With 3 workers processing 6 jobs with 10ms delay each,
		// should complete in roughly 20ms (2 rounds) rather than 60ms (sequential)
		assert.Len(t, results, numJobs)
		assert.Less(t, processingTime, 40*time.Millisecond, "should process concurrently")
	})
}

func TestWorkerPool_Metrics(t *testing.T) {
	t.Run("tracks worker metrics correctly", func(t *testing.T) {
		pool := NewWorkerPool(3)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err := pool.Start(ctx)
		assert.NoError(t, err)

		// Check initial metrics
		assert.Equal(t, 3, pool.Size())
		assert.Equal(t, 3, pool.ActiveWorkers())
		assert.Equal(t, 0, pool.QueueDepth())
		assert.Equal(t, 0, pool.ProcessedJobs())

		// Submit and process a job
		go func() {
			defer pool.CloseJobs()
			pool.JobQueue() <- Job{ID: "test", Data: "test"}
		}()

		// Wait for processing
		<-pool.ResultQueue()

		// Check updated metrics
		assert.Equal(t, 1, pool.ProcessedJobs())
	})
}

func TestProcessFiles_Integration(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	testFiles := map[string]string{
		"file1.txt": "Hello world!",
		"file2.txt": "Test content",
		"file3.txt": "More test data",
	}

	var filePaths []string
	for name, content := range testFiles {
		filePath := filepath.Join(tmpDir, name)
		err := os.WriteFile(filePath, []byte(content), 0644)
		assert.NoError(t, err)
		filePaths = append(filePaths, filePath)
	}

	t.Run("processes files with mock processor", func(t *testing.T) {
		processor := func(filePath string) types.Result[types.ProcessResult] {
			return types.Ok(types.ProcessResult{
				FilePath: filePath,
				DetectionResult: types.DetectionResult{
					TotalCount: 1,
					Success:    true,
				},
			})
		}

		results := ProcessFiles(filePaths, 2, processor)
		assert.Len(t, results, 3)

		for _, result := range results {
			assert.NoError(t, result.Error)
			assert.Equal(t, 1, result.DetectionResult.TotalCount)
		}
	})

	t.Run("handles processor errors", func(t *testing.T) {
		processor := func(filePath string) types.Result[types.ProcessResult] {
			if filepath.Base(filePath) == "file2.txt" {
				return types.Err[types.ProcessResult](fmt.Errorf("processing error"))
			}
			return types.Ok(types.ProcessResult{
				FilePath:        filePath,
				DetectionResult: types.DetectionResult{Success: true},
			})
		}

		results := ProcessFiles(filePaths, 2, processor)
		assert.Len(t, results, 3)

		errorCount := 0
		successCount := 0
		for _, result := range results {
			if result.Error != nil {
				errorCount++
			} else {
				successCount++
			}
		}

		assert.Equal(t, 1, errorCount)
		assert.Equal(t, 2, successCount)
	})
}

func TestWorkerPool_EdgeCases(t *testing.T) {
	t.Run("handles double start error", func(t *testing.T) {
		pool := NewWorkerPool(1)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err1 := pool.Start(ctx)
		assert.NoError(t, err1)

		err2 := pool.Start(ctx)
		assert.Error(t, err2, "should error on double start")
		assert.Contains(t, err2.Error(), "already running")
	})

	t.Run("handles stop when not running", func(t *testing.T) {
		pool := NewWorkerPool(1)

		// Should not panic or error
		pool.Stop()
		assert.False(t, pool.IsRunning())
	})

	t.Run("handles empty job submission", func(t *testing.T) {
		pool := NewWorkerPool(1)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err := pool.Start(ctx)
		assert.NoError(t, err)

		// Close jobs immediately without submitting any
		pool.CloseJobs()

		// Should handle gracefully
		results := make([]Result, 0)
		for result := range pool.ResultQueue() {
			results = append(results, result)
		}

		assert.Empty(t, results)
		assert.False(t, pool.IsRunning())
	})

	t.Run("handles large worker pool", func(t *testing.T) {
		pool := NewWorkerPool(100) // Should be capped at 64
		assert.Equal(t, 64, pool.Size())
	})
}

func TestWorker_ProcessJob(t *testing.T) {
	testCases := []struct {
		name        string
		jobData     interface{}
		expectError bool
	}{
		{"normal_job", "test_data", false},
		{"error_job", "error", true},
		{"delay_job", "delay", false},
		{"nil_data", nil, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pool := NewWorkerPool(1)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			err := pool.Start(ctx)
			assert.NoError(t, err)

			job := Job{
				ID:   tc.name,
				Data: tc.jobData,
			}

			go func() {
				pool.JobQueue() <- job
				pool.CloseJobs()
			}()

			result := <-pool.ResultQueue()
			assert.Equal(t, tc.name, result.JobID)

			if tc.expectError {
				assert.Error(t, result.Error)
				assert.False(t, result.Success)
			} else {
				assert.NoError(t, result.Error)
				assert.True(t, result.Success)
			}
		})
	}
}

// Benchmark tests for performance
func BenchmarkWorkerPool_Processing(b *testing.B) {
	pool := NewWorkerPool(4)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := pool.Start(ctx)
	assert.NoError(b, err)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Submit job
		job := Job{
			ID:   fmt.Sprintf("bench_job_%d", i),
			Data: "benchmark",
		}

		go func() {
			pool.JobQueue() <- job
		}()

		// Wait for result
		<-pool.ResultQueue()
	}
}

// Example usage for documentation
func ExampleNewWorkerPool() {
	pool := NewWorkerPool(1) // Use 1 worker for deterministic output
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_ = pool.Start(ctx)

	// Submit jobs
	go func() {
		defer pool.CloseJobs()
		for i := 0; i < 3; i++ {
			job := Job{
				ID:   fmt.Sprintf("job%d", i),
				Data: fmt.Sprintf("data%d", i),
			}
			pool.JobQueue() <- job
		}
	}()

	// Process results
	for result := range pool.ResultQueue() {
		fmt.Printf("Job %s completed: %t\n", result.JobID, result.Success)
	}
	// Output:
	// Job job0 completed: true
	// Job job1 completed: true
	// Job job2 completed: true
}
