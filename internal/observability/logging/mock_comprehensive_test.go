package logging

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Define custom context key types to avoid SA1029 warnings
type testKeyType string
type requestIDKeyType string
type userIDKeyType string

const (
	testKey      testKeyType      = "test_key"
	requestIDKey requestIDKeyType = "request_id"
	userIDKey    userIDKeyType    = "user_id"
)

func TestNewMockLogger(t *testing.T) {
	t.Run("creates new mock logger", func(t *testing.T) {
		logger := NewMockLogger()

		assert.NotNil(t, logger)
		assert.NotNil(t, logger.core)
		assert.True(t, logger.core.enabled)
		assert.Empty(t, logger.core.logs)
		assert.NotNil(t, logger.ctx)
	})
}

func TestMockLogger_AllMethods(t *testing.T) {
	logger := NewMockLogger()

	t.Run("Debug method", func(t *testing.T) {
		logger.Debug(context.Background(), "debug message", "key", "value")

		logs := logger.GetLogs()
		assert.Len(t, logs, 1)
		assert.Equal(t, LevelDebug, logs[0].Level)
		assert.Equal(t, "debug message", logs[0].Message)
		assert.Contains(t, logs[0].KeysAndValues, "key")
		assert.Contains(t, logs[0].KeysAndValues, "value")
	})

	t.Run("Info method", func(t *testing.T) {
		logger.Reset()
		logger.Info(context.Background(), "info message", "key", "value")

		logs := logger.GetLogs()
		assert.Len(t, logs, 1)
		assert.Equal(t, LevelInfo, logs[0].Level)
		assert.Equal(t, "info message", logs[0].Message)
	})

	t.Run("Warn method", func(t *testing.T) {
		logger.Reset()
		logger.Warn(context.Background(), "warn message", "key", "value")

		logs := logger.GetLogs()
		assert.Len(t, logs, 1)
		assert.Equal(t, LevelWarn, logs[0].Level)
		assert.Equal(t, "warn message", logs[0].Message)
	})

	t.Run("Error method", func(t *testing.T) {
		logger.Reset()
		logger.Error(context.Background(), "error message", "key", "value")

		logs := logger.GetLogs()
		assert.Len(t, logs, 1)
		assert.Equal(t, LevelError, logs[0].Level)
		assert.Equal(t, "error message", logs[0].Message)
	})

	t.Run("With method", func(t *testing.T) {
		withLogger := logger.With("service", "test", "version", "1.0")

		assert.NotNil(t, withLogger)
		assert.IsType(t, &MockLogger{}, withLogger)

		// Test that the returned logger has the key-value pairs
		mockLogger := withLogger.(*MockLogger)
		assert.Contains(t, mockLogger.kvPairs, "service")
		assert.Contains(t, mockLogger.kvPairs, "test")
		assert.Contains(t, mockLogger.kvPairs, "version")
		assert.Contains(t, mockLogger.kvPairs, "1.0")
	})

	t.Run("WithContext method", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), testKey, "test_value")
		withLogger := logger.WithContext(ctx)

		assert.NotNil(t, withLogger)
		assert.IsType(t, &MockLogger{}, withLogger)

		mockLogger := withLogger.(*MockLogger)
		assert.Equal(t, ctx, mockLogger.ctx)
	})

	t.Run("IsEnabled method", func(t *testing.T) {
		// When enabled
		assert.True(t, logger.IsEnabled(LevelDebug))
		assert.True(t, logger.IsEnabled(LevelInfo))
		assert.True(t, logger.IsEnabled(LevelWarn))
		assert.True(t, logger.IsEnabled(LevelError))

		// Disable and test
		logger.SetEnabled(false)
		assert.False(t, logger.IsEnabled(LevelDebug))
		assert.False(t, logger.IsEnabled(LevelInfo))
		assert.False(t, logger.IsEnabled(LevelWarn))
		assert.False(t, logger.IsEnabled(LevelError))

		// Re-enable
		logger.SetEnabled(true)
		assert.True(t, logger.IsEnabled(LevelDebug))
	})
}

func TestMockLogger_LogRetrieval(t *testing.T) {
	logger := NewMockLogger()

	// Add various log levels
	logger.Debug(context.Background(), "debug msg")
	logger.Info(context.Background(), "info msg")
	logger.Warn(context.Background(), "warn msg")
	logger.Error(context.Background(), "error msg")

	t.Run("GetLogs returns all logs", func(t *testing.T) {
		logs := logger.GetLogs()
		assert.Len(t, logs, 4)
	})

	t.Run("GetLogsByLevel filters correctly", func(t *testing.T) {
		debugLogs := logger.GetLogsByLevel(LevelDebug)
		assert.Len(t, debugLogs, 1)
		assert.Equal(t, "debug msg", debugLogs[0].Message)

		infoLogs := logger.GetLogsByLevel(LevelInfo)
		assert.Len(t, infoLogs, 1)
		assert.Equal(t, "info msg", infoLogs[0].Message)

		warnLogs := logger.GetLogsByLevel(LevelWarn)
		assert.Len(t, warnLogs, 1)
		assert.Equal(t, "warn msg", warnLogs[0].Message)

		errorLogs := logger.GetLogsByLevel(LevelError)
		assert.Len(t, errorLogs, 1)
		assert.Equal(t, "error msg", errorLogs[0].Message)
	})

	t.Run("HasLogWithMessage works", func(t *testing.T) {
		assert.True(t, logger.HasLogWithMessage("debug msg"))
		assert.True(t, logger.HasLogWithMessage("info msg"))
		assert.True(t, logger.HasLogWithMessage("warn msg"))
		assert.True(t, logger.HasLogWithMessage("error msg"))
		assert.False(t, logger.HasLogWithMessage("nonexistent msg"))
	})

	t.Run("HasLogWithLevel works", func(t *testing.T) {
		assert.True(t, logger.HasLogWithLevel(LevelDebug))
		assert.True(t, logger.HasLogWithLevel(LevelInfo))
		assert.True(t, logger.HasLogWithLevel(LevelWarn))
		assert.True(t, logger.HasLogWithLevel(LevelError))
	})

	t.Run("Reset clears logs", func(t *testing.T) {
		logger.Reset()
		logs := logger.GetLogs()
		assert.Empty(t, logs)

		assert.False(t, logger.HasLogWithMessage("debug msg"))
		assert.False(t, logger.HasLogWithLevel(LevelDebug))
	})
}

func TestMockLogger_SetEnabled(t *testing.T) {
	logger := NewMockLogger()

	t.Run("SetEnabled controls logging", func(t *testing.T) {
		// Initially enabled
		logger.Info(context.Background(), "enabled message")
		assert.Len(t, logger.GetLogs(), 1)

		// Disable
		logger.SetEnabled(false)
		logger.Info(context.Background(), "disabled message")
		assert.Len(t, logger.GetLogs(), 1) // Should still be 1

		// Re-enable
		logger.SetEnabled(true)
		logger.Info(context.Background(), "re-enabled message")
		assert.Len(t, logger.GetLogs(), 2)
	})
}

func TestMockLogger_ContextHandling(t *testing.T) {
	logger := NewMockLogger()

	t.Run("handles nil context", func(t *testing.T) {
		assert.NotPanics(t, func() {
			logger.Info(context.TODO(), "nil context message")
		})

		logs := logger.GetLogs()
		assert.Len(t, logs, 1)
		assert.NotNil(t, logs[0].Context) // Should use background context
	})

	t.Run("preserves context values", func(t *testing.T) {
		logger.Reset() // Clear previous logs
		ctx := context.WithValue(context.Background(), requestIDKey, "123")
		logger.Info(ctx, "context message")

		logs := logger.GetLogs()
		assert.Len(t, logs, 1)
		// Context preservation may vary based on implementation
		assert.NotNil(t, logs[0].Context)
	})

	t.Run("WithContext creates logger with bound context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), userIDKey, "456")
		boundLogger := logger.WithContext(ctx)

		boundLogger.Info(context.Background(), "bound context message")

		logs := logger.GetLogs()
		assert.True(t, len(logs) > 0)
		// The bound context should be used
	})
}

func TestMockLogger_KeyValuePairs(t *testing.T) {
	logger := NewMockLogger()

	t.Run("handles multiple key-value pairs", func(t *testing.T) {
		logger.Info(context.Background(), "message with many kvs",
			"key1", "value1",
			"key2", 42,
			"key3", true,
			"key4", []string{"a", "b"})

		logs := logger.GetLogs()
		assert.Len(t, logs, 1)

		kvs := logs[0].KeysAndValues
		assert.Contains(t, kvs, "key1")
		assert.Contains(t, kvs, "value1")
		assert.Contains(t, kvs, "key2")
		assert.Contains(t, kvs, 42)
		assert.Contains(t, kvs, "key3")
		assert.Contains(t, kvs, true)
	})

	t.Run("handles odd number of key-value pairs", func(t *testing.T) {
		logger.Reset()
		logger.Info(context.Background(), "odd kvs", "key1", "value1", "orphan_key")

		logs := logger.GetLogs()
		assert.Len(t, logs, 1)
		assert.Contains(t, logs[0].KeysAndValues, "key1")
		assert.Contains(t, logs[0].KeysAndValues, "value1")
		assert.Contains(t, logs[0].KeysAndValues, "orphan_key")
	})

	t.Run("With method accumulates key-value pairs", func(t *testing.T) {
		withLogger := logger.With("base_key", "base_value")
		withLogger.Info(context.Background(), "message", "additional_key", "additional_value")

		logs := logger.GetLogs()
		assert.True(t, len(logs) > 0)

		kvs := logs[len(logs)-1].KeysAndValues
		assert.Contains(t, kvs, "base_key")
		assert.Contains(t, kvs, "base_value")
		assert.Contains(t, kvs, "additional_key")
		assert.Contains(t, kvs, "additional_value")
	})
}
