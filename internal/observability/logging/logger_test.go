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

func TestLogLevels(t *testing.T) {
	tests := []struct {
		name     string
		level    LogLevel
		expected []LogLevel
	}{
		{
			name:     "silent mode disables all logging",
			level:    LevelSilent,
			expected: []LogLevel{},
		},
		{
			name:     "debug enables all levels",
			level:    LevelDebug,
			expected: []LogLevel{LevelDebug, LevelInfo, LevelWarn, LevelError},
		},
		{
			name:     "info enables info and above",
			level:    LevelInfo,
			expected: []LogLevel{LevelInfo, LevelWarn, LevelError},
		},
		{
			name:     "warn enables warn and error",
			level:    LevelWarn,
			expected: []LogLevel{LevelWarn, LevelError},
		},
		{
			name:     "error enables only error",
			level:    LevelError,
			expected: []LogLevel{LevelError},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			config := &Config{
				Level:          tt.level,
				Format:         FormatJSON,
				Output:         &buf,
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
			}

			logger, err := NewLogger(config)
			require.NoError(t, err)

			// Test all log levels
			ctx := context.Background()
			logger.Debug(ctx, "debug message")
			logger.Info(ctx, "info message")
			logger.Warn(ctx, "warn message")
			logger.Error(ctx, "error message")

			output := buf.String()
			lines := strings.Split(strings.TrimSpace(output), "\n")

			if tt.level == LevelSilent {
				assert.Empty(t, strings.TrimSpace(output), "silent mode should produce no output")
				return
			}

			// Filter out empty lines
			var nonEmptyLines []string
			for _, line := range lines {
				if strings.TrimSpace(line) != "" {
					nonEmptyLines = append(nonEmptyLines, line)
				}
			}

			assert.Len(t, nonEmptyLines, len(tt.expected), "unexpected number of log entries")

			// Verify each expected level appears in output
			for i := range tt.expected {
				if i < len(nonEmptyLines) {
					var logEntry map[string]interface{}
					err := json.Unmarshal([]byte(nonEmptyLines[i]), &logEntry)
					require.NoError(t, err, "failed to parse log entry as JSON")

					assert.Contains(t, logEntry, "level", "log entry should contain level field")
					assert.Contains(t, logEntry, "msg", "log entry should contain msg field")
					assert.Contains(t, logEntry, "service.name", "log entry should contain service name")
					assert.Equal(t, "test-service", logEntry["service.name"])
				}
			}
		})
	}
}

func TestLogFormats(t *testing.T) {
	tests := []struct {
		name   string
		format LogFormat
	}{
		{
			name:   "JSON format",
			format: FormatJSON,
		},
		{
			name:   "text format",
			format: FormatText,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			config := &Config{
				Level:          LevelInfo,
				Format:         tt.format,
				Output:         &buf,
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
			}

			logger, err := NewLogger(config)
			require.NoError(t, err)

			ctx := context.Background()
			logger.Info(ctx, "test message", "key", "value", "count", 42)

			output := buf.String()
			assert.NotEmpty(t, output, "should produce log output")

			if tt.format == FormatJSON {
				// Verify it's valid JSON
				var logEntry map[string]interface{}
				lines := strings.Split(strings.TrimSpace(output), "\n")
				require.Greater(t, len(lines), 0, "should have at least one log line")

				err := json.Unmarshal([]byte(lines[0]), &logEntry)
				require.NoError(t, err, "should be valid JSON")

				assert.Equal(t, "test message", logEntry["msg"])
				assert.Equal(t, "value", logEntry["key"])
				assert.Equal(t, float64(42), logEntry["count"]) // JSON numbers are float64
			} else {
				// Text format should be human readable
				assert.Contains(t, output, "test message")
				assert.Contains(t, output, "key=value")
				assert.Contains(t, output, "count=42")
			}
		})
	}
}

func TestLoggerWith(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Level:          LevelInfo,
		Format:         FormatJSON,
		Output:         &buf,
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	}

	logger, err := NewLogger(config)
	require.NoError(t, err)

	// Create logger with additional fields
	enrichedLogger := logger.With("component", "test", "requestID", "123")

	ctx := context.Background()
	enrichedLogger.Info(ctx, "test message")

	output := buf.String()
	assert.NotEmpty(t, output)

	var logEntry map[string]interface{}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	require.Greater(t, len(lines), 0)

	err = json.Unmarshal([]byte(lines[0]), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "test message", logEntry["msg"])
	assert.Equal(t, "test", logEntry["component"])
	assert.Equal(t, "123", logEntry["requestID"])
}

func TestLoggerIsEnabled(t *testing.T) {
	tests := []struct {
		name         string
		configLevel  LogLevel
		checkLevel   LogLevel
		shouldEnable bool
	}{
		{"silent disables all", LevelSilent, LevelError, false},
		{"debug enables debug", LevelDebug, LevelDebug, true},
		{"debug enables info", LevelDebug, LevelInfo, true},
		{"info disables debug", LevelInfo, LevelDebug, false},
		{"info enables info", LevelInfo, LevelInfo, true},
		{"warn disables info", LevelWarn, LevelInfo, false},
		{"warn enables warn", LevelWarn, LevelWarn, true},
		{"error disables warn", LevelError, LevelWarn, false},
		{"error enables error", LevelError, LevelError, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Level:          tt.configLevel,
				Format:         FormatJSON,
				Output:         &bytes.Buffer{},
				ServiceName:    "test",
				ServiceVersion: "1.0.0",
			}

			logger, err := NewLogger(config)
			require.NoError(t, err)

			assert.Equal(t, tt.shouldEnable, logger.IsEnabled(tt.checkLevel))
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, LevelSilent, config.Level)
	assert.Equal(t, FormatJSON, config.Format)
	assert.Equal(t, "antimoji", config.ServiceName)
	assert.Equal(t, "unknown", config.ServiceVersion)
	assert.NotNil(t, config.Output)
}

func TestNoOpLogger(t *testing.T) {
	logger := newNoOpLogger()
	ctx := context.Background()

	// Should not panic and should do nothing
	logger.Debug(ctx, "debug")
	logger.Info(ctx, "info")
	logger.Warn(ctx, "warn")
	logger.Error(ctx, "error")

	enriched := logger.With("key", "value")
	assert.NotNil(t, enriched)

	contextLogger := logger.WithContext(ctx)
	assert.NotNil(t, contextLogger)

	assert.False(t, logger.IsEnabled(LevelDebug))
	assert.False(t, logger.IsEnabled(LevelInfo))
	assert.False(t, logger.IsEnabled(LevelWarn))
	assert.False(t, logger.IsEnabled(LevelError))
}

func TestLoggerWithContext(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Level:          LevelInfo,
		Format:         FormatJSON,
		Output:         &buf,
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	}

	logger, err := NewLogger(config)
	require.NoError(t, err)

	type contextKey string
	ctx := context.WithValue(context.Background(), contextKey("traceID"), "abc123")
	contextLogger := logger.WithContext(ctx)

	contextLogger.Info(ctx, "test message")

	output := buf.String()
	assert.NotEmpty(t, output)

	// The context is passed but the specific context values aren't automatically
	// included in logs unless explicitly added via With()
	var logEntry map[string]interface{}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	require.Greater(t, len(lines), 0)

	err = json.Unmarshal([]byte(lines[0]), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "test message", logEntry["msg"])
}
