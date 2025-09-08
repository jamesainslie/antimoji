// Package context provides utilities for context propagation and management.
package context

import (
	"context"
	"time"
)

// ContextKey is a type for context keys to avoid collisions.
type ContextKey string

const (
	// OperationKey holds the current operation name
	OperationKey ContextKey = "operation"
	// ComponentKey holds the current component name
	ComponentKey ContextKey = "component"
	// FilePathKey holds the current file path being processed
	FilePathKey ContextKey = "file_path"
	// RequestIDKey holds a unique request identifier
	RequestIDKey ContextKey = "request_id"
	// UserIDKey holds user identification
	UserIDKey ContextKey = "user_id"
	// SessionIDKey holds session identification
	SessionIDKey ContextKey = "session_id"
)

// WithOperation adds an operation name to the context.
func WithOperation(ctx context.Context, operation string) context.Context {
	return context.WithValue(ctx, OperationKey, operation)
}

// WithComponent adds a component name to the context.
func WithComponent(ctx context.Context, component string) context.Context {
	return context.WithValue(ctx, ComponentKey, component)
}

// WithFilePath adds a file path to the context.
func WithFilePath(ctx context.Context, filePath string) context.Context {
	return context.WithValue(ctx, FilePathKey, filePath)
}

// WithRequestID adds a request ID to the context.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// WithTimeout creates a context with timeout and proper cleanup.
func WithTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, timeout)
}

// WithCancel creates a context with cancellation and proper cleanup.
func WithCancel(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithCancel(parent)
}

// GetOperation retrieves the operation name from context.
func GetOperation(ctx context.Context) string {
	if op, ok := ctx.Value(OperationKey).(string); ok {
		return op
	}
	return "unknown"
}

// GetComponent retrieves the component name from context.
func GetComponent(ctx context.Context) string {
	if comp, ok := ctx.Value(ComponentKey).(string); ok {
		return comp
	}
	return "unknown"
}

// GetFilePath retrieves the file path from context.
func GetFilePath(ctx context.Context) string {
	if path, ok := ctx.Value(FilePathKey).(string); ok {
		return path
	}
	return ""
}

// GetRequestID retrieves the request ID from context.
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}

// ExtractContextFields extracts common context fields for structured logging.
func ExtractContextFields(ctx context.Context) []interface{} {
	var fields []interface{}

	if op := GetOperation(ctx); op != "unknown" {
		fields = append(fields, "operation", op)
	}

	if comp := GetComponent(ctx); comp != "unknown" {
		fields = append(fields, "component", comp)
	}

	if path := GetFilePath(ctx); path != "" {
		fields = append(fields, "file_path", path)
	}

	if reqID := GetRequestID(ctx); reqID != "" {
		fields = append(fields, "request_id", reqID)
	}

	return fields
}

// NewRootContext creates a new root context for operations.
func NewRootContext() context.Context {
	return context.Background()
}

// NewOperationContext creates a context for a specific operation.
func NewOperationContext(operation string) context.Context {
	return WithOperation(NewRootContext(), operation)
}

// NewComponentContext creates a context for a specific component operation.
func NewComponentContext(operation, component string) context.Context {
	ctx := NewOperationContext(operation)
	return WithComponent(ctx, component)
}
