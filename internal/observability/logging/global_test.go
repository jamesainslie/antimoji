package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitGlobalLogger(t *testing.T) {
	// Reset global logger
	SetGlobalLogger(nil)

	var buf bytes.Buffer
	config := &Config{
		Level:          LevelInfo,
		Format:         FormatJSON,
		Output:         &buf,
		ServiceName:    "test-global",
		ServiceVersion: "1.0.0",
	}

	err := InitGlobalLogger(config)
	require.NoError(t, err)

	ctx := context.Background()
	Info(ctx, "global test message", "component", "test")

	output := buf.String()
	assert.NotEmpty(t, output)

	var logEntry map[string]interface{}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	require.Greater(t, len(lines), 0)

	err = json.Unmarshal([]byte(lines[0]), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "global test message", logEntry["msg"])
	assert.Equal(t, "test", logEntry["component"])
	assert.Equal(t, "test-global", logEntry["service.name"])
	assert.Equal(t, "1.0.0", logEntry["service.version"])
}

func TestGetGlobalLoggerWithoutInit(t *testing.T) {
	// Reset global logger
	SetGlobalLogger(nil)

	logger := GetGlobalLogger()
	assert.NotNil(t, logger)

	// Should be a no-op logger
	ctx := context.Background()
	assert.False(t, logger.IsEnabled(LevelDebug))
	assert.False(t, logger.IsEnabled(LevelInfo))
	assert.False(t, logger.IsEnabled(LevelWarn))
	assert.False(t, logger.IsEnabled(LevelError))

	// Should not panic
	logger.Debug(ctx, "debug")
	logger.Info(ctx, "info")
	logger.Warn(ctx, "warn")
	logger.Error(ctx, "error")
}

func TestGlobalLoggerFunctions(t *testing.T) {
	// Reset and setup global logger
	SetGlobalLogger(nil)

	var buf bytes.Buffer
	config := &Config{
		Level:          LevelDebug,
		Format:         FormatJSON,
		Output:         &buf,
		ServiceName:    "test-global",
		ServiceVersion: "1.0.0",
	}

	err := InitGlobalLogger(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Test all global logging functions
	Debug(ctx, "debug message", "level", "debug")
	Info(ctx, "info message", "level", "info")
	Warn(ctx, "warn message", "level", "warn")
	Error(ctx, "error message", "level", "error")

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Filter out empty lines
	var nonEmptyLines []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			nonEmptyLines = append(nonEmptyLines, line)
		}
	}

	assert.Len(t, nonEmptyLines, 4, "should have 4 log entries")

	// Verify each log entry
	messages := []string{"debug message", "info message", "warn message", "error message"}
	levels := []string{"debug", "info", "warn", "error"}

	for i, line := range nonEmptyLines {
		var logEntry map[string]interface{}
		err := json.Unmarshal([]byte(line), &logEntry)
		require.NoError(t, err, "log entry %d should be valid JSON", i)

		assert.Equal(t, messages[i], logEntry["msg"], "message should match for entry %d", i)
		assert.Equal(t, levels[i], logEntry["level"], "level should match for entry %d", i)
	}
}

func TestGlobalLoggerWith(t *testing.T) {
	// Reset and setup global logger
	SetGlobalLogger(nil)

	var buf bytes.Buffer
	config := &Config{
		Level:          LevelInfo,
		Format:         FormatJSON,
		Output:         &buf,
		ServiceName:    "test-global",
		ServiceVersion: "1.0.0",
	}

	err := InitGlobalLogger(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Test With function
	enrichedLogger := With("component", "global-test", "requestID", "456")
	enrichedLogger.Info(ctx, "enriched message")

	output := buf.String()
	assert.NotEmpty(t, output)

	var logEntry map[string]interface{}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	require.Greater(t, len(lines), 0)

	err = json.Unmarshal([]byte(lines[0]), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "enriched message", logEntry["msg"])
	assert.Equal(t, "global-test", logEntry["component"])
	assert.Equal(t, "456", logEntry["requestID"])
}

func TestGlobalLoggerWithContext(t *testing.T) {
	// Reset and setup global logger
	SetGlobalLogger(nil)

	var buf bytes.Buffer
	config := &Config{
		Level:          LevelInfo,
		Format:         FormatJSON,
		Output:         &buf,
		ServiceName:    "test-global",
		ServiceVersion: "1.0.0",
	}

	err := InitGlobalLogger(config)
	require.NoError(t, err)

	ctx := context.WithValue(context.Background(), "traceID", "xyz789")

	// Test WithContext function
	contextLogger := WithContext(ctx)
	contextLogger.Info(ctx, "context message")

	output := buf.String()
	assert.NotEmpty(t, output)

	var logEntry map[string]interface{}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	require.Greater(t, len(lines), 0)

	err = json.Unmarshal([]byte(lines[0]), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "context message", logEntry["msg"])
}

func TestGlobalIsEnabled(t *testing.T) {
	// Reset and setup global logger
	SetGlobalLogger(nil)

	var buf bytes.Buffer
	config := &Config{
		Level:          LevelWarn,
		Format:         FormatJSON,
		Output:         &buf,
		ServiceName:    "test-global",
		ServiceVersion: "1.0.0",
	}

	err := InitGlobalLogger(config)
	require.NoError(t, err)

	// Test IsEnabled function
	assert.False(t, IsEnabled(LevelDebug))
	assert.False(t, IsEnabled(LevelInfo))
	assert.True(t, IsEnabled(LevelWarn))
	assert.True(t, IsEnabled(LevelError))
}

func TestSetGlobalLogger(t *testing.T) {
	// Create a custom logger
	var buf bytes.Buffer
	config := &Config{
		Level:          LevelInfo,
		Format:         FormatJSON,
		Output:         &buf,
		ServiceName:    "custom-service",
		ServiceVersion: "2.0.0",
	}

	customLogger, err := NewLogger(config)
	require.NoError(t, err)

	// Set it as global
	SetGlobalLogger(customLogger)

	ctx := context.Background()
	Info(ctx, "custom logger message")

	output := buf.String()
	assert.NotEmpty(t, output)

	var logEntry map[string]interface{}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	require.Greater(t, len(lines), 0)

	err = json.Unmarshal([]byte(lines[0]), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "custom logger message", logEntry["msg"])
	assert.Equal(t, "custom-service", logEntry["service.name"])
	assert.Equal(t, "2.0.0", logEntry["service.version"])
}

func TestConcurrentGlobalLoggerAccess(t *testing.T) {
	// Reset and setup global logger
	SetGlobalLogger(nil)

	var buf bytes.Buffer
	config := &Config{
		Level:          LevelInfo,
		Format:         FormatJSON,
		Output:         &buf,
		ServiceName:    "concurrent-test",
		ServiceVersion: "1.0.0",
	}

	err := InitGlobalLogger(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Run concurrent logging operations
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			logger := GetGlobalLogger()
			logger.Info(ctx, "concurrent message", "goroutine", id)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	output := buf.String()
	assert.NotEmpty(t, output)

	// Should have multiple log entries
	lines := strings.Split(strings.TrimSpace(output), "\n")
	nonEmptyLines := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			nonEmptyLines++
		}
	}
	assert.Greater(t, nonEmptyLines, 0, "should have log entries from concurrent operations")
}
