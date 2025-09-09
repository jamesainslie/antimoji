package logging

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMockLogger_SharedCore(t *testing.T) {
	t.Run("derived loggers share the same log entries", func(t *testing.T) {
		logger1 := NewMockLogger()
		logger2 := logger1.With("component", "test")
		logger3 := logger1.WithContext(context.Background())

		// Log from different derived loggers
		logger1.Info(context.Background(), "message from logger1")
		logger2.Info(context.Background(), "message from logger2")
		logger3.Info(context.Background(), "message from logger3")

		// All loggers should see all messages since they share the core
		logs1 := logger1.GetLogs()
		logs2 := logger2.(*MockLogger).GetLogs()
		logs3 := logger3.(*MockLogger).GetLogs()

		assert.Len(t, logs1, 3, "logger1 should see all 3 messages")
		assert.Len(t, logs2, 3, "logger2 should see all 3 messages")
		assert.Len(t, logs3, 3, "logger3 should see all 3 messages")

		// Verify the messages are the same across all loggers
		for i := 0; i < 3; i++ {
			assert.Equal(t, logs1[i].Message, logs2[i].Message)
			assert.Equal(t, logs1[i].Message, logs3[i].Message)
		}
	})

	t.Run("concurrent access to derived loggers is safe", func(t *testing.T) {
		baseLogger := NewMockLogger()

		// Create multiple derived loggers
		loggers := make([]Logger, 10)
		for i := 0; i < 10; i++ {
			loggers[i] = baseLogger.With("worker", i)
		}

		// Concurrent logging from all derived loggers
		var wg sync.WaitGroup
		for i, l := range loggers {
			wg.Add(1)
			go func(id int, logger Logger) {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					logger.Info(context.Background(), "concurrent message", "worker", id, "iteration", j)
				}
			}(i, l)
		}

		wg.Wait()

		// Should have 10 workers * 100 messages = 1000 total messages
		logs := baseLogger.GetLogs()
		assert.Len(t, logs, 1000, "Should have 1000 log entries from concurrent access")

		// Verify no data corruption - all messages should be valid
		for _, log := range logs {
			assert.Equal(t, LevelInfo, log.Level)
			assert.Equal(t, "concurrent message", log.Message)
			assert.NotEmpty(t, log.KeysAndValues)
		}
	})

	t.Run("Reset affects all derived loggers", func(t *testing.T) {
		logger1 := NewMockLogger()
		logger2 := logger1.With("component", "test")

		// Log from both
		logger1.Info(context.Background(), "message1")
		logger2.Info(context.Background(), "message2")

		assert.Len(t, logger1.GetLogs(), 2)
		assert.Len(t, logger2.(*MockLogger).GetLogs(), 2)

		// Reset from one should affect both
		logger1.Reset()

		assert.Len(t, logger1.GetLogs(), 0)
		assert.Len(t, logger2.(*MockLogger).GetLogs(), 0)
	})
}

func TestMockLogger_BasicFunctionality(t *testing.T) {
	t.Run("logs messages correctly", func(t *testing.T) {
		logger := NewMockLogger()

		logger.Debug(context.Background(), "debug message", "key", "value")
		logger.Info(context.Background(), "info message")
		logger.Warn(context.Background(), "warn message")
		logger.Error(context.Background(), "error message")

		logs := logger.GetLogs()
		assert.Len(t, logs, 4)

		assert.Equal(t, LevelDebug, logs[0].Level)
		assert.Equal(t, "debug message", logs[0].Message)
		assert.Contains(t, logs[0].KeysAndValues, "key")
		assert.Contains(t, logs[0].KeysAndValues, "value")

		assert.Equal(t, LevelInfo, logs[1].Level)
		assert.Equal(t, "info message", logs[1].Message)

		assert.Equal(t, LevelWarn, logs[2].Level)
		assert.Equal(t, "warn message", logs[2].Message)

		assert.Equal(t, LevelError, logs[3].Level)
		assert.Equal(t, "error message", logs[3].Message)
	})

	t.Run("With() preserves key-value pairs", func(t *testing.T) {
		logger := NewMockLogger()
		derivedLogger := logger.With("component", "test", "version", "1.0")

		derivedLogger.Info(context.Background(), "test message")

		logs := derivedLogger.(*MockLogger).GetLogs()
		assert.Len(t, logs, 1)

		kv := logs[0].KeysAndValues
		assert.Contains(t, kv, "component")
		assert.Contains(t, kv, "test")
		assert.Contains(t, kv, "version")
		assert.Contains(t, kv, "1.0")
	})
}
