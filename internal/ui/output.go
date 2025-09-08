// Package ui provides user interface and output utilities for the Antimoji CLI.
package ui

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/antimoji/antimoji/internal/observability/logging"
)

// OutputLevel determines what type of output should be shown to users.
type OutputLevel int

const (
	// OutputSilent shows no user output (only errors)
	OutputSilent OutputLevel = iota
	// OutputNormal shows standard operation results
	OutputNormal
	// OutputVerbose shows detailed operation information
	OutputVerbose
	// OutputDebug shows all available information
	OutputDebug
)

// UserOutput handles all user-facing output, separate from diagnostic logging.
type UserOutput interface {
	// Info displays informational messages to the user
	Info(ctx context.Context, msg string, args ...interface{})
	// Success displays success messages to the user
	Success(ctx context.Context, msg string, args ...interface{})
	// Warning displays warning messages to the user
	Warning(ctx context.Context, msg string, args ...interface{})
	// Error displays error messages to the user
	Error(ctx context.Context, msg string, args ...interface{})
	// Result displays operation results to the user
	Result(ctx context.Context, msg string, args ...interface{})
	// Progress displays progress information to the user
	Progress(ctx context.Context, msg string, args ...interface{})
	// SetLevel sets the output level for filtering messages
	SetLevel(level OutputLevel)
	// IsLevelEnabled checks if a given level would produce output
	IsLevelEnabled(level OutputLevel) bool
}

// Config holds the user output configuration.
type Config struct {
	// Level determines what output is shown to users
	Level OutputLevel
	// Writer is where user output goes (typically os.Stdout)
	Writer io.Writer
	// ErrorWriter is where error output goes (typically os.Stderr)
	ErrorWriter io.Writer
	// EnableColors enables colored output
	EnableColors bool
}

// DefaultConfig returns a default user output configuration.
func DefaultConfig() *Config {
	return &Config{
		Level:        OutputNormal,
		Writer:       os.Stdout,
		ErrorWriter:  os.Stderr,
		EnableColors: true,
	}
}

// userOutput implements the UserOutput interface.
type userOutput struct {
	config *Config
}

// NewUserOutput creates a new user output handler.
func NewUserOutput(config *Config) UserOutput {
	if config == nil {
		config = DefaultConfig()
	}
	return &userOutput{config: config}
}

// Info displays informational messages to the user.
func (u *userOutput) Info(ctx context.Context, msg string, args ...interface{}) {
	if !u.IsLevelEnabled(OutputNormal) {
		return
	}

	// Log diagnostically while showing user output
	logging.Debug(ctx, "User info message displayed", "message", fmt.Sprintf(msg, args...))

	formatted := fmt.Sprintf(msg, args...)
	if u.config.EnableColors {
		_, _ = fmt.Fprintf(u.config.Writer, "\033[36mINFO:\033[0m %s\n", formatted)
	} else {
		_, _ = fmt.Fprintf(u.config.Writer, "INFO: %s\n", formatted)
	}
}

// Success displays success messages to the user.
func (u *userOutput) Success(ctx context.Context, msg string, args ...interface{}) {
	if !u.IsLevelEnabled(OutputNormal) {
		return
	}

	// Log diagnostically while showing user output
	logging.Info(ctx, "User success message displayed", "message", fmt.Sprintf(msg, args...))

	formatted := fmt.Sprintf(msg, args...)
	if u.config.EnableColors {
		_, _ = fmt.Fprintf(u.config.Writer, "\033[32m\033[0m %s\n", formatted)
	} else {
		_, _ = fmt.Fprintf(u.config.Writer, "SUCCESS: %s\n", formatted)
	}
}

// Warning displays warning messages to the user.
func (u *userOutput) Warning(ctx context.Context, msg string, args ...interface{}) {
	if !u.IsLevelEnabled(OutputNormal) {
		return
	}

	// Log diagnostically while showing user output
	logging.Warn(ctx, "User warning message displayed", "message", fmt.Sprintf(msg, args...))

	formatted := fmt.Sprintf(msg, args...)
	if u.config.EnableColors {
		_, _ = fmt.Fprintf(u.config.ErrorWriter, "\033[33m\033[0m %s\n", formatted)
	} else {
		_, _ = fmt.Fprintf(u.config.ErrorWriter, "WARNING: %s\n", formatted)
	}
}

// Error displays error messages to the user.
func (u *userOutput) Error(ctx context.Context, msg string, args ...interface{}) {
	// Always show errors regardless of level

	// Log diagnostically while showing user output
	logging.Error(ctx, "User error message displayed", "message", fmt.Sprintf(msg, args...))

	formatted := fmt.Sprintf(msg, args...)
	if u.config.EnableColors {
		_, _ = fmt.Fprintf(u.config.ErrorWriter, "\033[31m\033[0m %s\n", formatted)
	} else {
		_, _ = fmt.Fprintf(u.config.ErrorWriter, "ERROR: %s\n", formatted)
	}
}

// Result displays operation results to the user.
func (u *userOutput) Result(ctx context.Context, msg string, args ...interface{}) {
	if !u.IsLevelEnabled(OutputNormal) {
		return
	}

	// Log diagnostically while showing user output
	logging.Info(ctx, "User result displayed", "message", fmt.Sprintf(msg, args...))

	_, _ = fmt.Fprintf(u.config.Writer, msg, args...)
	_, _ = fmt.Fprintln(u.config.Writer)
}

// Progress displays progress information to the user.
func (u *userOutput) Progress(ctx context.Context, msg string, args ...interface{}) {
	if !u.IsLevelEnabled(OutputVerbose) {
		return
	}

	// Log diagnostically while showing user output
	logging.Debug(ctx, "User progress message displayed", "message", fmt.Sprintf(msg, args...))

	formatted := fmt.Sprintf(msg, args...)
	if u.config.EnableColors {
		_, _ = fmt.Fprintf(u.config.Writer, "\033[90m%s\033[0m\n", formatted)
	} else {
		_, _ = fmt.Fprintf(u.config.Writer, "%s\n", formatted)
	}
}

// SetLevel sets the output level for filtering messages.
func (u *userOutput) SetLevel(level OutputLevel) {
	u.config.Level = level
}

// IsLevelEnabled checks if a given level would produce output.
func (u *userOutput) IsLevelEnabled(level OutputLevel) bool {
	return level <= u.config.Level
}

// Global user output instance
var globalUserOutput UserOutput
var globalOutputMutex sync.RWMutex

// InitGlobalUserOutput initializes the global user output handler.
func InitGlobalUserOutput(config *Config) {
	globalOutputMutex.Lock()
	defer globalOutputMutex.Unlock()
	globalUserOutput = NewUserOutput(config)
}

// GetGlobalUserOutput returns the global user output handler.
func GetGlobalUserOutput() UserOutput {
	globalOutputMutex.RLock()
	defer globalOutputMutex.RUnlock()

	if globalUserOutput == nil {
		// Return a default user output if none has been initialized
		return NewUserOutput(DefaultConfig())
	}

	return globalUserOutput
}

// Global convenience functions for user output
func Info(ctx context.Context, msg string, args ...interface{}) {
	GetGlobalUserOutput().Info(ctx, msg, args...)
}

func Success(ctx context.Context, msg string, args ...interface{}) {
	GetGlobalUserOutput().Success(ctx, msg, args...)
}

func Warning(ctx context.Context, msg string, args ...interface{}) {
	GetGlobalUserOutput().Warning(ctx, msg, args...)
}

func Error(ctx context.Context, msg string, args ...interface{}) {
	GetGlobalUserOutput().Error(ctx, msg, args...)
}

func Result(ctx context.Context, msg string, args ...interface{}) {
	GetGlobalUserOutput().Result(ctx, msg, args...)
}

func Progress(ctx context.Context, msg string, args ...interface{}) {
	GetGlobalUserOutput().Progress(ctx, msg, args...)
}
