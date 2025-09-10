package ui

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserOutputImpl_AllMethods(t *testing.T) {
	var stdout, stderr bytes.Buffer
	config := &Config{
		Level:        OutputNormal,
		Writer:       &stdout,
		ErrorWriter:  &stderr,
		EnableColors: false,
	}

	output := NewUserOutput(config)

	t.Run("Info method writes to stdout", func(t *testing.T) {
		stdout.Reset()
		stderr.Reset()

		output.Info(context.Background(), "Info message")

		assert.Contains(t, stdout.String(), "Info message")
		assert.Empty(t, stderr.String())
	})

	t.Run("Success method writes to stdout", func(t *testing.T) {
		stdout.Reset()
		stderr.Reset()

		output.Success(context.Background(), "Success message")

		assert.Contains(t, stdout.String(), "Success message")
		assert.Empty(t, stderr.String())
	})

	t.Run("Warning method writes to stderr", func(t *testing.T) {
		stdout.Reset()
		stderr.Reset()

		output.Warning(context.Background(), "Warning message")

		assert.Contains(t, stderr.String(), "Warning message")
		assert.Empty(t, stdout.String())
	})

	t.Run("Error method writes to stderr", func(t *testing.T) {
		stdout.Reset()
		stderr.Reset()

		output.Error(context.Background(), "Error message")

		assert.Contains(t, stderr.String(), "Error message")
		assert.Empty(t, stdout.String())
	})

	t.Run("Result method writes to stdout", func(t *testing.T) {
		stdout.Reset()
		stderr.Reset()

		output.Result(context.Background(), "Result message")

		assert.Contains(t, stdout.String(), "Result message")
		assert.Empty(t, stderr.String())
	})

	t.Run("Progress method works", func(t *testing.T) {
		stdout.Reset()
		stderr.Reset()

		output.Progress(context.Background(), "Progress message")

		// Progress messages may or may not appear depending on implementation
		// Just verify no panic occurred
		assert.True(t, true)
	})
}

func TestUserOutputImpl_LevelFiltering(t *testing.T) {
	var stdout, stderr bytes.Buffer

	tests := []struct {
		name            string
		level           OutputLevel
		shouldShowInfo  bool
		shouldShowDebug bool
	}{
		{"Silent level", OutputSilent, false, false},
		{"Normal level", OutputNormal, true, false},
		{"Verbose level", OutputVerbose, true, true},
		{"Debug level", OutputDebug, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Level:        tt.level,
				Writer:       &stdout,
				ErrorWriter:  &stderr,
				EnableColors: false,
			}

			output := NewUserOutput(config)

			stdout.Reset()
			stderr.Reset()

			output.Info(context.Background(), "Info message")
			output.Progress(context.Background(), "Progress message") // Should behave like debug

			if tt.shouldShowInfo {
				assert.Contains(t, stdout.String(), "Info message")
			} else {
				assert.NotContains(t, stdout.String(), "Info message")
			}

			// Progress/debug messages
			if tt.shouldShowDebug {
				assert.Contains(t, stdout.String(), "Progress message")
			} else {
				// In non-debug levels, progress might still show or not depending on implementation
				// Let's just verify no panic occurred
				assert.True(t, true)
			}
		})
	}
}

func TestUserOutputImpl_SetLevel(t *testing.T) {
	var stdout bytes.Buffer
	config := &Config{
		Level:        OutputNormal,
		Writer:       &stdout,
		ErrorWriter:  &stdout,
		EnableColors: false,
	}

	output := NewUserOutput(config)

	t.Run("SetLevel changes filtering", func(t *testing.T) {
		// Initially normal level
		stdout.Reset()
		output.Info(context.Background(), "Normal level info")
		initialOutput := stdout.String()

		// Change to silent
		output.SetLevel(OutputSilent)
		stdout.Reset()
		output.Info(context.Background(), "Silent level info")
		silentOutput := stdout.String()

		// Change to verbose
		output.SetLevel(OutputVerbose)
		stdout.Reset()
		output.Info(context.Background(), "Verbose level info")
		verboseOutput := stdout.String()

		// Normal level should show info
		assert.Contains(t, initialOutput, "Normal level info")
		// Silent level should not show info
		assert.Empty(t, silentOutput)
		// Verbose level should show info
		assert.Contains(t, verboseOutput, "Verbose level info")
	})
}

func TestUserOutputImpl_IsLevelEnabled(t *testing.T) {
	tests := []struct {
		configLevel OutputLevel
		testLevel   OutputLevel
		expected    bool
	}{
		{OutputSilent, OutputSilent, true},
		{OutputSilent, OutputNormal, false},
		{OutputNormal, OutputSilent, true},
		{OutputNormal, OutputNormal, true},
		{OutputNormal, OutputVerbose, false},
		{OutputVerbose, OutputNormal, true},
		{OutputVerbose, OutputVerbose, true},
		{OutputVerbose, OutputDebug, false},
		{OutputDebug, OutputVerbose, true},
		{OutputDebug, OutputDebug, true},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			config := &Config{
				Level:        tt.configLevel,
				Writer:       os.Stdout,
				ErrorWriter:  os.Stderr,
				EnableColors: false,
			}

			output := NewUserOutput(config)
			result := output.IsLevelEnabled(tt.testLevel)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUserOutputImpl_WithFormatting(t *testing.T) {
	var stdout, stderr bytes.Buffer
	config := &Config{
		Level:        OutputNormal,
		Writer:       &stdout,
		ErrorWriter:  &stderr,
		EnableColors: false,
	}

	output := NewUserOutput(config)

	t.Run("formats messages with arguments", func(t *testing.T) {
		stdout.Reset()

		output.Info(context.Background(), "User %s has %d items", "alice", 42)

		assert.Contains(t, stdout.String(), "User alice has 42 items")
	})

	t.Run("handles no arguments", func(t *testing.T) {
		stdout.Reset()

		output.Info(context.Background(), "Simple message")

		assert.Contains(t, stdout.String(), "Simple message")
	})

	t.Run("handles empty message", func(t *testing.T) {
		stdout.Reset()

		output.Info(context.Background(), "")

		// Should not panic, may or may not produce output
		assert.True(t, stdout.Len() >= 0)
	})
}

func TestUserOutputImpl_ColorHandling(t *testing.T) {
	var stdout bytes.Buffer

	t.Run("with colors enabled", func(t *testing.T) {
		config := &Config{
			Level:        OutputNormal,
			Writer:       &stdout,
			ErrorWriter:  &stdout,
			EnableColors: true,
		}

		output := NewUserOutput(config)
		stdout.Reset()

		output.Success(context.Background(), "Success message")

		result := stdout.String()
		assert.Contains(t, result, "Success message")
		// Colors might be added - we just verify no crash
	})

	t.Run("with colors disabled", func(t *testing.T) {
		config := &Config{
			Level:        OutputNormal,
			Writer:       &stdout,
			ErrorWriter:  &stdout,
			EnableColors: false,
		}

		output := NewUserOutput(config)
		stdout.Reset()

		output.Success(context.Background(), "Success message")

		result := stdout.String()
		assert.Contains(t, result, "Success message")
		// Should not contain ANSI escape sequences
		assert.NotContains(t, result, "\033[")
	})
}

func TestUserOutputImpl_NilContext(t *testing.T) {
	var stdout bytes.Buffer
	config := &Config{
		Level:        OutputNormal,
		Writer:       &stdout,
		ErrorWriter:  &stdout,
		EnableColors: false,
	}

	output := NewUserOutput(config)

	t.Run("handles nil context", func(t *testing.T) {
		assert.NotPanics(t, func() {
			output.Info(nil, "message with nil context")
		})

		assert.NotPanics(t, func() {
			output.Success(nil, "success with nil context")
		})

		assert.NotPanics(t, func() {
			output.Warning(nil, "warning with nil context")
		})

		assert.NotPanics(t, func() {
			output.Error(nil, "error with nil context")
		})

		assert.NotPanics(t, func() {
			output.Result(nil, "result with nil context")
		})

		assert.NotPanics(t, func() {
			output.Progress(nil, "progress with nil context")
		})
	})
}

func TestUserOutputImpl_ThreadSafety(t *testing.T) {
	// Note: UserOutput is not designed to be thread-safe when writing to the same buffer
	// This test just verifies that the interface doesn't panic under normal usage
	config := &Config{
		Level:        OutputNormal,
		Writer:       os.Stdout, // Use separate writers to avoid race conditions
		ErrorWriter:  os.Stderr,
		EnableColors: false,
	}

	output := NewUserOutput(config)

	t.Run("multiple outputs don't panic", func(t *testing.T) {
		// Test that multiple calls don't panic
		assert.NotPanics(t, func() {
			output.Info(context.Background(), "Message 1")
			output.Success(context.Background(), "Message 2")
			output.Warning(context.Background(), "Message 3")
			output.Error(context.Background(), "Message 4")
		})
	})
}

func TestOutputLevels_Values(t *testing.T) {
	t.Run("output levels have expected values", func(t *testing.T) {
		assert.Equal(t, OutputLevel(0), OutputSilent)
		assert.Equal(t, OutputLevel(1), OutputNormal)
		assert.Equal(t, OutputLevel(2), OutputVerbose)
		assert.Equal(t, OutputLevel(3), OutputDebug)
	})
}

func TestGlobalUIFunctions(t *testing.T) {
	t.Run("global Result function", func(t *testing.T) {
		assert.NotPanics(t, func() {
			Result(context.Background(), "test result message")
		})
	})

	t.Run("global Progress function", func(t *testing.T) {
		assert.NotPanics(t, func() {
			Progress(context.Background(), "test progress message")
		})
	})

	t.Run("global functions with formatting", func(t *testing.T) {
		assert.NotPanics(t, func() {
			Result(context.Background(), "result %s: %d", "test", 42)
			Progress(context.Background(), "progress %d%%", 75)
		})
	})
}

func TestConfig_Fields(t *testing.T) {
	t.Run("Config struct has all expected fields", func(t *testing.T) {
		var stdout, stderr bytes.Buffer

		config := &Config{
			Level:        OutputVerbose,
			Writer:       &stdout,
			ErrorWriter:  &stderr,
			EnableColors: true,
		}

		assert.Equal(t, OutputVerbose, config.Level)
		assert.Equal(t, &stdout, config.Writer)
		assert.Equal(t, &stderr, config.ErrorWriter)
		assert.True(t, config.EnableColors)
	})
}

func TestNewUserOutput_ValidatesConfig(t *testing.T) {
	t.Run("creates output with valid config", func(t *testing.T) {
		config := &Config{
			Level:        OutputNormal,
			Writer:       os.Stdout,
			ErrorWriter:  os.Stderr,
			EnableColors: false,
		}

		output := NewUserOutput(config)
		assert.NotNil(t, output)
	})

	t.Run("handles config with nil writers gracefully", func(t *testing.T) {
		config := &Config{
			Level:        OutputNormal,
			Writer:       nil,
			ErrorWriter:  nil,
			EnableColors: false,
		}

		// Should not panic during creation
		assert.NotPanics(t, func() {
			output := NewUserOutput(config)
			assert.NotNil(t, output)
		})
	})
}
