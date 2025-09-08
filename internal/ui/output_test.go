package ui

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserOutput_Info(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Level:        OutputNormal,
		Writer:       &buf,
		ErrorWriter:  &buf,
		EnableColors: false,
	}
	
	output := NewUserOutput(config)
	ctx := context.Background()
	
	output.Info(ctx, "test message %s", "arg")
	
	assert.Contains(t, buf.String(), "INFO: test message arg")
}

func TestUserOutput_Success(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Level:        OutputNormal,
		Writer:       &buf,
		ErrorWriter:  &buf,
		EnableColors: false,
	}
	
	output := NewUserOutput(config)
	ctx := context.Background()
	
	output.Success(ctx, "success message")
	
	assert.Contains(t, buf.String(), "SUCCESS: success message")
}

func TestUserOutput_Warning(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Level:        OutputNormal,
		Writer:       &buf,
		ErrorWriter:  &buf,
		EnableColors: false,
	}
	
	output := NewUserOutput(config)
	ctx := context.Background()
	
	output.Warning(ctx, "warning message")
	
	assert.Contains(t, buf.String(), "WARNING: warning message")
}

func TestUserOutput_Error(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Level:        OutputNormal,
		Writer:       &buf,
		ErrorWriter:  &buf,
		EnableColors: false,
	}
	
	output := NewUserOutput(config)
	ctx := context.Background()
	
	output.Error(ctx, "error message")
	
	assert.Contains(t, buf.String(), "ERROR: error message")
}

func TestUserOutput_OutputLevels(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Level:        OutputSilent,
		Writer:       &buf,
		ErrorWriter:  &buf,
		EnableColors: false,
	}
	
	output := NewUserOutput(config)
	ctx := context.Background()
	
	// Silent mode should not show info/success
	output.Info(ctx, "should not appear")
	output.Success(ctx, "should not appear")
	
	// But should always show errors
	output.Error(ctx, "should appear")
	
	content := buf.String()
	assert.NotContains(t, content, "should not appear")
	assert.Contains(t, content, "should appear")
}

func TestUserOutput_ColoredOutput(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Level:        OutputNormal,
		Writer:       &buf,
		ErrorWriter:  &buf,
		EnableColors: true,
	}
	
	output := NewUserOutput(config)
	ctx := context.Background()
	
	output.Info(ctx, "colored message")
	
	content := buf.String()
	assert.Contains(t, content, "\033[36m") // ANSI color code
	assert.Contains(t, content, "\033[0m")  // ANSI reset code
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	
	assert.Equal(t, OutputNormal, config.Level)
	assert.True(t, config.EnableColors)
	assert.NotNil(t, config.Writer)
	assert.NotNil(t, config.ErrorWriter)
}

func TestGlobalUserOutput(t *testing.T) {
	// Test global functions
	var buf bytes.Buffer
	config := &Config{
		Level:        OutputNormal,
		Writer:       &buf,
		ErrorWriter:  &buf,
		EnableColors: false,
	}
	
	InitGlobalUserOutput(config)
	ctx := context.Background()
	
	Info(ctx, "global info")
	Success(ctx, "global success")
	Warning(ctx, "global warning")
	Error(ctx, "global error")
	
	content := buf.String()
	assert.Contains(t, content, "INFO: global info")
	assert.Contains(t, content, "SUCCESS: global success")
	assert.Contains(t, content, "WARNING: global warning")
	assert.Contains(t, content, "ERROR: global error")
}
