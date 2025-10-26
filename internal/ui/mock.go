// Package ui provides mock implementations for testing.
package ui

import (
	"context"
	"sync"
)

// mockCore holds the shared state for all derived MockUserOutput instances.
type mockCore struct {
	mu       sync.RWMutex
	messages []OutputMessage
	level    OutputLevel
}

// MockUserOutput is a mock implementation of UserOutput for testing.
type MockUserOutput struct {
	core *mockCore
}

// OutputMessage represents a message logged for testing verification.
type OutputMessage struct {
	Level   string
	Message string
	Args    []interface{}
	Context context.Context
}

// NewMockUserOutput creates a new mock user output.
func NewMockUserOutput() *MockUserOutput {
	return &MockUserOutput{
		core: &mockCore{
			messages: make([]OutputMessage, 0),
			level:    OutputNormal,
		},
	}
}

// Info displays informational messages to the user.
func (m *MockUserOutput) Info(ctx context.Context, msg string, args ...interface{}) {
	m.core.mu.Lock()
	defer m.core.mu.Unlock()

	m.core.messages = append(m.core.messages, OutputMessage{
		Level:   "INFO",
		Message: msg,
		Args:    args,
		Context: ctx,
	})
}

// Success displays success messages to the user.
func (m *MockUserOutput) Success(ctx context.Context, msg string, args ...interface{}) {
	m.core.mu.Lock()
	defer m.core.mu.Unlock()

	m.core.messages = append(m.core.messages, OutputMessage{
		Level:   "SUCCESS",
		Message: msg,
		Args:    args,
		Context: ctx,
	})
}

// Warning displays warning messages to the user.
func (m *MockUserOutput) Warning(ctx context.Context, msg string, args ...interface{}) {
	m.core.mu.Lock()
	defer m.core.mu.Unlock()

	m.core.messages = append(m.core.messages, OutputMessage{
		Level:   "WARNING",
		Message: msg,
		Args:    args,
		Context: ctx,
	})
}

// Error displays error messages to the user.
func (m *MockUserOutput) Error(ctx context.Context, msg string, args ...interface{}) {
	m.core.mu.Lock()
	defer m.core.mu.Unlock()

	m.core.messages = append(m.core.messages, OutputMessage{
		Level:   "ERROR",
		Message: msg,
		Args:    args,
		Context: ctx,
	})
}

// Result displays operation results to the user.
func (m *MockUserOutput) Result(ctx context.Context, msg string, args ...interface{}) {
	m.core.mu.Lock()
	defer m.core.mu.Unlock()

	m.core.messages = append(m.core.messages, OutputMessage{
		Level:   "RESULT",
		Message: msg,
		Args:    args,
		Context: ctx,
	})
}

// Progress displays progress information to the user.
func (m *MockUserOutput) Progress(ctx context.Context, msg string, args ...interface{}) {
	m.core.mu.Lock()
	defer m.core.mu.Unlock()

	m.core.messages = append(m.core.messages, OutputMessage{
		Level:   "PROGRESS",
		Message: msg,
		Args:    args,
		Context: ctx,
	})
}

// SetLevel sets the output level for filtering messages.
func (m *MockUserOutput) SetLevel(level OutputLevel) {
	m.core.mu.Lock()
	defer m.core.mu.Unlock()

	m.core.level = level
}

// IsLevelEnabled checks if a given level would produce output.
func (m *MockUserOutput) IsLevelEnabled(level OutputLevel) bool {
	m.core.mu.RLock()
	defer m.core.mu.RUnlock()

	return level >= m.core.level
}

// GetMessages returns all logged messages for testing verification.
func (m *MockUserOutput) GetMessages() []OutputMessage {
	m.core.mu.RLock()
	defer m.core.mu.RUnlock()

	messages := make([]OutputMessage, len(m.core.messages))
	copy(messages, m.core.messages)
	return messages
}

// GetMessagesOfLevel returns all logged messages of a specific level.
func (m *MockUserOutput) GetMessagesOfLevel(level string) []OutputMessage {
	m.core.mu.RLock()
	defer m.core.mu.RUnlock()

	var filtered []OutputMessage
	for _, msg := range m.core.messages {
		if msg.Level == level {
			filtered = append(filtered, msg)
		}
	}
	return filtered
}

// Clear clears all logged messages.
func (m *MockUserOutput) Clear() {
	m.core.mu.Lock()
	defer m.core.mu.Unlock()

	m.core.messages = make([]OutputMessage, 0)
}

// Count returns the number of logged messages.
func (m *MockUserOutput) Count() int {
	m.core.mu.RLock()
	defer m.core.mu.RUnlock()

	return len(m.core.messages)
}

// CountLevel returns the number of logged messages of a specific level.
func (m *MockUserOutput) CountLevel(level string) int {
	m.core.mu.RLock()
	defer m.core.mu.RUnlock()

	count := 0
	for _, msg := range m.core.messages {
		if msg.Level == level {
			count++
		}
	}
	return count
}
