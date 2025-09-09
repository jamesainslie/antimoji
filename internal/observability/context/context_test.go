package context

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithOperation(t *testing.T) {
	ctx := WithOperation(context.Background(), "test-operation")

	assert.Equal(t, "test-operation", GetOperation(ctx))
}

func TestWithComponent(t *testing.T) {
	ctx := WithComponent(context.Background(), "test-component")

	assert.Equal(t, "test-component", GetComponent(ctx))
}

func TestWithFilePath(t *testing.T) {
	ctx := WithFilePath(context.Background(), "/test/path")

	assert.Equal(t, "/test/path", GetFilePath(ctx))
}

func TestWithRequestID(t *testing.T) {
	ctx := WithRequestID(context.Background(), "req-123")

	assert.Equal(t, "req-123", GetRequestID(ctx))
}

func TestGetOperation_Unknown(t *testing.T) {
	ctx := context.Background()

	assert.Equal(t, "unknown", GetOperation(ctx))
}

func TestGetComponent_Unknown(t *testing.T) {
	ctx := context.Background()

	assert.Equal(t, "unknown", GetComponent(ctx))
}

func TestGetFilePath_Empty(t *testing.T) {
	ctx := context.Background()

	assert.Equal(t, "", GetFilePath(ctx))
}

func TestGetRequestID_Empty(t *testing.T) {
	ctx := context.Background()

	assert.Equal(t, "", GetRequestID(ctx))
}

func TestExtractContextFields(t *testing.T) {
	ctx := context.Background()
	ctx = WithOperation(ctx, "test-op")
	ctx = WithComponent(ctx, "test-comp")
	ctx = WithFilePath(ctx, "/test/file")
	ctx = WithRequestID(ctx, "req-456")

	fields := ExtractContextFields(ctx)

	// Should have 8 fields (4 keys + 4 values)
	assert.Len(t, fields, 8)
	assert.Contains(t, fields, "operation")
	assert.Contains(t, fields, "test-op")
	assert.Contains(t, fields, "component")
	assert.Contains(t, fields, "test-comp")
	assert.Contains(t, fields, "file_path")
	assert.Contains(t, fields, "/test/file")
	assert.Contains(t, fields, "request_id")
	assert.Contains(t, fields, "req-456")
}

func TestExtractContextFields_EmptyContext(t *testing.T) {
	ctx := context.Background()

	fields := ExtractContextFields(ctx)

	// Should be empty since no context values are set
	assert.Empty(t, fields)
}

func TestNewRootContext(t *testing.T) {
	ctx := NewRootContext()

	assert.NotNil(t, ctx)
	assert.Equal(t, "unknown", GetOperation(ctx))
}

func TestNewOperationContext(t *testing.T) {
	ctx := NewOperationContext("scan")

	assert.Equal(t, "scan", GetOperation(ctx))
	assert.Equal(t, "unknown", GetComponent(ctx))
}

func TestNewComponentContext(t *testing.T) {
	ctx := NewComponentContext("clean", "cli")

	assert.Equal(t, "clean", GetOperation(ctx))
	assert.Equal(t, "cli", GetComponent(ctx))
}

func TestWithTimeout(t *testing.T) {
	ctx, cancel := WithTimeout(context.Background(), 100)
	defer cancel()

	assert.NotNil(t, ctx)
	assert.NotNil(t, cancel)
}

func TestWithCancel(t *testing.T) {
	ctx, cancel := WithCancel(context.Background())
	defer cancel()

	assert.NotNil(t, ctx)
	assert.NotNil(t, cancel)
}
