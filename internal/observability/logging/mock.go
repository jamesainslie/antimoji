// Package logging provides mock implementations for testing.
package logging

import (
	"context"
	"sync"
)

// mockCore holds the shared state for all derived MockLogger instances.
type mockCore struct {
	mu      sync.RWMutex
	logs    []LogEntry
	enabled bool
}

// MockLogger is a mock implementation of Logger for testing.
type MockLogger struct {
	core    *mockCore
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
		core: &mockCore{
			logs:    make([]LogEntry, 0),
			enabled: true,
		},
		ctx: context.Background(),
	}
}

// Debug logs a debug message to the mock.
func (m *MockLogger) Debug(ctx context.Context, msg string, keysAndValues ...any) {
	// Prefer bound context when available, fall back to parameter or Background
	effectiveCtx := m.ctx
	if effectiveCtx == nil {
		if ctx != nil {
			effectiveCtx = ctx
		} else {
			effectiveCtx = context.Background()
		}
	}

	m.core.mu.Lock()
	defer m.core.mu.Unlock()

	if m.core.enabled {
		allKV := append(m.kvPairs, keysAndValues...)
		m.core.logs = append(m.core.logs, LogEntry{
			Level:         LevelDebug,
			Message:       msg,
			KeysAndValues: allKV,
			Context:       effectiveCtx,
		})
	}
}

// Info logs an info message to the mock.
func (m *MockLogger) Info(ctx context.Context, msg string, keysAndValues ...any) {
	// Prefer bound context when available, fall back to parameter or Background
	effectiveCtx := m.ctx
	if effectiveCtx == nil {
		if ctx != nil {
			effectiveCtx = ctx
		} else {
			effectiveCtx = context.Background()
		}
	}

	m.core.mu.Lock()
	defer m.core.mu.Unlock()

	if m.core.enabled {
		allKV := append(m.kvPairs, keysAndValues...)
		m.core.logs = append(m.core.logs, LogEntry{
			Level:         LevelInfo,
			Message:       msg,
			KeysAndValues: allKV,
			Context:       effectiveCtx,
		})
	}
}

// Warn logs a warning message to the mock.
func (m *MockLogger) Warn(ctx context.Context, msg string, keysAndValues ...any) {
	// Prefer bound context when available, fall back to parameter or Background
	effectiveCtx := m.ctx
	if effectiveCtx == nil {
		if ctx != nil {
			effectiveCtx = ctx
		} else {
			effectiveCtx = context.Background()
		}
	}

	m.core.mu.Lock()
	defer m.core.mu.Unlock()

	if m.core.enabled {
		allKV := append(m.kvPairs, keysAndValues...)
		m.core.logs = append(m.core.logs, LogEntry{
			Level:         LevelWarn,
			Message:       msg,
			KeysAndValues: allKV,
			Context:       effectiveCtx,
		})
	}
}

// Error logs an error message to the mock.
func (m *MockLogger) Error(ctx context.Context, msg string, keysAndValues ...any) {
	// Prefer bound context when available, fall back to parameter or Background
	effectiveCtx := m.ctx
	if effectiveCtx == nil {
		if ctx != nil {
			effectiveCtx = ctx
		} else {
			effectiveCtx = context.Background()
		}
	}

	m.core.mu.Lock()
	defer m.core.mu.Unlock()

	if m.core.enabled {
		allKV := append(m.kvPairs, keysAndValues...)
		m.core.logs = append(m.core.logs, LogEntry{
			Level:         LevelError,
			Message:       msg,
			KeysAndValues: allKV,
			Context:       effectiveCtx,
		})
	}
}

// With returns a logger with the given key-value pairs added.
func (m *MockLogger) With(keysAndValues ...any) Logger {
	// No need to lock here since we're only reading immutable fields and creating a new logger
	newKV := make([]any, len(m.kvPairs)+len(keysAndValues))
	copy(newKV, m.kvPairs)
	copy(newKV[len(m.kvPairs):], keysAndValues)

	return &MockLogger{
		core:    m.core, // Share the same core (with shared mutex, logs, and enabled state)
		kvPairs: newKV,
		ctx:     m.ctx,
	}
}

// WithContext returns a logger that uses the given context.
func (m *MockLogger) WithContext(ctx context.Context) Logger {
	// Copy kvPairs to avoid shared mutation between logger instances
	kvCopy := make([]any, len(m.kvPairs))
	copy(kvCopy, m.kvPairs)

	return &MockLogger{
		core:    m.core, // Share the same core (with shared mutex, logs, and enabled state)
		kvPairs: kvCopy,
		ctx:     ctx,
	}
}

// IsEnabled returns true if the logger is enabled.
func (m *MockLogger) IsEnabled(level LogLevel) bool {
	m.core.mu.RLock()
	defer m.core.mu.RUnlock()
	return m.core.enabled
}

// GetLogs returns all logged entries for testing verification.
func (m *MockLogger) GetLogs() []LogEntry {
	m.core.mu.RLock()
	defer m.core.mu.RUnlock()

	// Return a copy to prevent race conditions
	logs := make([]LogEntry, len(m.core.logs))
	copy(logs, m.core.logs)
	return logs
}

// GetLogsByLevel returns all logged entries for a specific level.
func (m *MockLogger) GetLogsByLevel(level LogLevel) []LogEntry {
	m.core.mu.RLock()
	defer m.core.mu.RUnlock()

	var filtered []LogEntry
	for _, log := range m.core.logs {
		if log.Level == level {
			filtered = append(filtered, log)
		}
	}
	return filtered
}

// Reset clears all logged entries.
func (m *MockLogger) Reset() {
	m.core.mu.Lock()
	defer m.core.mu.Unlock()
	m.core.logs = m.core.logs[:0]
}

// SetEnabled controls whether the logger is enabled.
func (m *MockLogger) SetEnabled(enabled bool) {
	m.core.mu.Lock()
	defer m.core.mu.Unlock()
	m.core.enabled = enabled
}

// HasLogWithMessage checks if any log entry contains the given message.
func (m *MockLogger) HasLogWithMessage(message string) bool {
	m.core.mu.RLock()
	defer m.core.mu.RUnlock()

	for _, log := range m.core.logs {
		if log.Message == message {
			return true
		}
	}
	return false
}

// HasLogWithLevel checks if any log entry has the given level.
func (m *MockLogger) HasLogWithLevel(level LogLevel) bool {
	m.core.mu.RLock()
	defer m.core.mu.RUnlock()

	for _, log := range m.core.logs {
		if log.Level == level {
			return true
		}
	}
	return false
}
