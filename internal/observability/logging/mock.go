// Package logging provides mock implementations for testing.
package logging

import (
	"context"
	"sync"
)

// MockLogger is a mock implementation of Logger for testing.
type MockLogger struct {
	mu      sync.RWMutex
	logs    []LogEntry
	enabled bool
	kvPairs []any
	ctx     context.Context
}

// LogEntry represents a logged entry for testing verification.
type LogEntry struct {
	Level         LogLevel
	Message       string
	KeysAndValues []any
	Context       context.Context
}

// NewMockLogger creates a new mock logger.
func NewMockLogger() *MockLogger {
	return &MockLogger{
		logs:    make([]LogEntry, 0),
		enabled: true,
		ctx:     context.Background(),
	}
}

// Debug logs a debug message to the mock.
func (m *MockLogger) Debug(ctx context.Context, msg string, keysAndValues ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.enabled {
		allKV := append(m.kvPairs, keysAndValues...)
		m.logs = append(m.logs, LogEntry{
			Level:         LevelDebug,
			Message:       msg,
			KeysAndValues: allKV,
			Context:       ctx,
		})
	}
}

// Info logs an info message to the mock.
func (m *MockLogger) Info(ctx context.Context, msg string, keysAndValues ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.enabled {
		allKV := append(m.kvPairs, keysAndValues...)
		m.logs = append(m.logs, LogEntry{
			Level:         LevelInfo,
			Message:       msg,
			KeysAndValues: allKV,
			Context:       ctx,
		})
	}
}

// Warn logs a warning message to the mock.
func (m *MockLogger) Warn(ctx context.Context, msg string, keysAndValues ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.enabled {
		allKV := append(m.kvPairs, keysAndValues...)
		m.logs = append(m.logs, LogEntry{
			Level:         LevelWarn,
			Message:       msg,
			KeysAndValues: allKV,
			Context:       ctx,
		})
	}
}

// Error logs an error message to the mock.
func (m *MockLogger) Error(ctx context.Context, msg string, keysAndValues ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.enabled {
		allKV := append(m.kvPairs, keysAndValues...)
		m.logs = append(m.logs, LogEntry{
			Level:         LevelError,
			Message:       msg,
			KeysAndValues: allKV,
			Context:       ctx,
		})
	}
}

// With returns a logger with the given key-value pairs added.
func (m *MockLogger) With(keysAndValues ...any) Logger {
	m.mu.RLock()
	defer m.mu.RUnlock()

	newKV := make([]any, len(m.kvPairs)+len(keysAndValues))
	copy(newKV, m.kvPairs)
	copy(newKV[len(m.kvPairs):], keysAndValues)

	return &MockLogger{
		logs:    m.logs, // Share the same log slice
		enabled: m.enabled,
		kvPairs: newKV,
		ctx:     m.ctx,
	}
}

// WithContext returns a logger that uses the given context.
func (m *MockLogger) WithContext(ctx context.Context) Logger {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return &MockLogger{
		logs:    m.logs, // Share the same log slice
		enabled: m.enabled,
		kvPairs: m.kvPairs,
		ctx:     ctx,
	}
}

// IsEnabled returns true if the logger is enabled.
func (m *MockLogger) IsEnabled(level LogLevel) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.enabled
}

// GetLogs returns all logged entries for testing verification.
func (m *MockLogger) GetLogs() []LogEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to prevent race conditions
	logs := make([]LogEntry, len(m.logs))
	copy(logs, m.logs)
	return logs
}

// GetLogsByLevel returns all logged entries for a specific level.
func (m *MockLogger) GetLogsByLevel(level LogLevel) []LogEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var filtered []LogEntry
	for _, log := range m.logs {
		if log.Level == level {
			filtered = append(filtered, log)
		}
	}
	return filtered
}

// Reset clears all logged entries.
func (m *MockLogger) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = m.logs[:0]
}

// SetEnabled controls whether the logger is enabled.
func (m *MockLogger) SetEnabled(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = enabled
}

// HasLogWithMessage checks if any log entry contains the given message.
func (m *MockLogger) HasLogWithMessage(message string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, log := range m.logs {
		if log.Message == message {
			return true
		}
	}
	return false
}

// HasLogWithLevel checks if any log entry has the given level.
func (m *MockLogger) HasLogWithLevel(level LogLevel) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, log := range m.logs {
		if log.Level == level {
			return true
		}
	}
	return false
}
