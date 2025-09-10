package logging

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewNoOpLogger(t *testing.T) {
	t.Run("creates no-op logger", func(t *testing.T) {
		logger := newNoOpLogger()
		assert.NotNil(t, logger)
	})

	t.Run("no-op logger methods don't panic", func(t *testing.T) {
		logger := newNoOpLogger()
		ctx := context.Background()

		assert.NotPanics(t, func() {
			logger.Debug(ctx, "debug message", "key", "value")
		})

		assert.NotPanics(t, func() {
			logger.Info(ctx, "info message", "key", "value")
		})

		assert.NotPanics(t, func() {
			logger.Warn(ctx, "warn message", "key", "value")
		})

		assert.NotPanics(t, func() {
			logger.Error(ctx, "error message", "key", "value")
		})
	})

	t.Run("no-op logger With method", func(t *testing.T) {
		logger := newNoOpLogger()

		assert.NotPanics(t, func() {
			withLogger := logger.With("service", "test")
			assert.NotNil(t, withLogger)
		})
	})

	t.Run("no-op logger WithContext method", func(t *testing.T) {
		logger := newNoOpLogger()
		ctx := context.Background()

		assert.NotPanics(t, func() {
			withLogger := logger.WithContext(ctx)
			assert.NotNil(t, withLogger)
		})
	})

	t.Run("no-op logger IsEnabled always returns false", func(t *testing.T) {
		logger := newNoOpLogger()

		assert.False(t, logger.IsEnabled(LevelDebug))
		assert.False(t, logger.IsEnabled(LevelInfo))
		assert.False(t, logger.IsEnabled(LevelWarn))
		assert.False(t, logger.IsEnabled(LevelError))
	})

	t.Run("no-op logger handles TODO context", func(t *testing.T) {
		logger := newNoOpLogger()
		ctx := context.TODO()

		assert.NotPanics(t, func() {
			logger.Info(ctx, "message with TODO context")
		})

		assert.NotPanics(t, func() {
			logger.Error(ctx, "error with TODO context", "error", "test")
		})
	})
}

func TestGlobalLoggerInitialization(t *testing.T) {
	t.Run("global logger exists by default", func(t *testing.T) {
		// The global logger should be initialized
		logger := GetGlobalLogger()
		assert.NotNil(t, logger)
	})

	t.Run("can set and get global logger", func(t *testing.T) {
		// Store original
		original := GetGlobalLogger()
		defer func() { SetGlobalLogger(original) }()

		// Create a new logger
		testLogger := newNoOpLogger()
		SetGlobalLogger(testLogger)

		// Should get the same logger back
		retrieved := GetGlobalLogger()
		assert.Equal(t, testLogger, retrieved)
	})
}
