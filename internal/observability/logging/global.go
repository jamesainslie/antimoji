// Package logging provides global logger access and initialization utilities.
package logging

import (
	"context"
	"sync"
)

var (
	// globalLogger holds the application-wide logger instance
	globalLogger Logger
	// globalMutex protects access to the global logger
	globalMutex sync.RWMutex
)

// InitGlobalLogger initializes the global logger with the given configuration.
// This should be called once during application startup.
func InitGlobalLogger(config *Config) error {
	logger, err := NewLogger(config)
	if err != nil {
		return err
	}

	globalMutex.Lock()
	globalLogger = logger
	globalMutex.Unlock()

	return nil
}

// GetGlobalLogger returns the global logger instance.
// If no logger has been initialized, it returns a silent logger.
func GetGlobalLogger() Logger {
	globalMutex.RLock()
	defer globalMutex.RUnlock()

	if globalLogger == nil {
		// Return a no-op logger if none has been initialized
		return newNoOpLogger()
	}

	return globalLogger
}

// SetGlobalLogger sets the global logger instance.
// This is primarily for testing purposes.
func SetGlobalLogger(logger Logger) {
	globalMutex.Lock()
	globalLogger = logger
	globalMutex.Unlock()
}

// Debug logs a debug message using the global logger.
func Debug(ctx context.Context, msg string, keysAndValues ...any) {
	GetGlobalLogger().Debug(ctx, msg, keysAndValues...)
}

// Info logs an info message using the global logger.
func Info(ctx context.Context, msg string, keysAndValues ...any) {
	GetGlobalLogger().Info(ctx, msg, keysAndValues...)
}

// Warn logs a warning message using the global logger.
func Warn(ctx context.Context, msg string, keysAndValues ...any) {
	GetGlobalLogger().Warn(ctx, msg, keysAndValues...)
}

// Error logs an error message using the global logger.
func Error(ctx context.Context, msg string, keysAndValues ...any) {
	GetGlobalLogger().Error(ctx, msg, keysAndValues...)
}

// With returns a logger with the given key-value pairs added to all log entries.
func With(keysAndValues ...any) Logger {
	return GetGlobalLogger().With(keysAndValues...)
}

// WithContext returns a logger that uses the given context.
func WithContext(ctx context.Context) Logger {
	return GetGlobalLogger().WithContext(ctx)
}

// IsEnabled returns true if the global logger would emit a log record at the given level.
func IsEnabled(level LogLevel) bool {
	return GetGlobalLogger().IsEnabled(level)
}
