// Package app provides application-level dependency management and bootstrap logic.
package app

import (
	"context"
	"fmt"
	"io"

	"github.com/antimoji/antimoji/internal/observability/logging"
	"github.com/antimoji/antimoji/internal/ui"
)

// Dependencies holds all application dependencies.
type Dependencies struct {
	Logger logging.Logger
	UI     ui.UserOutput
}

// Config holds configuration for creating dependencies.
type Config struct {
	// Logging configuration
	LogLevel  logging.LogLevel
	LogFormat logging.LogFormat
	LogOutput io.Writer

	// UI configuration
	UILevel        ui.OutputLevel
	UIWriter       io.Writer
	UIErrorWriter  io.Writer
	UIEnableColors bool

	// Application metadata
	ServiceName    string
	ServiceVersion string
}

// NewDependencies creates a new Dependencies instance with the given configuration.
func NewDependencies(config *Config) (*Dependencies, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Create logger
	loggerConfig := &logging.Config{
		Level:          config.LogLevel,
		Format:         config.LogFormat,
		Output:         config.LogOutput,
		ServiceName:    config.ServiceName,
		ServiceVersion: config.ServiceVersion,
	}

	logger, err := logging.NewLogger(loggerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	// Create UI output
	uiConfig := &ui.Config{
		Level:        config.UILevel,
		Writer:       config.UIWriter,
		ErrorWriter:  config.UIErrorWriter,
		EnableColors: config.UIEnableColors,
	}

	userOutput := ui.NewUserOutput(uiConfig)

	return &Dependencies{
		Logger: logger,
		UI:     userOutput,
	}, nil
}

// NewTestDependencies creates dependencies suitable for testing.
func NewTestDependencies() *Dependencies {
	return &Dependencies{
		Logger: logging.NewMockLogger(),
		UI:     ui.NewUserOutput(ui.DefaultConfig()),
	}
}

// Validate ensures all dependencies are properly initialized.
func (d *Dependencies) Validate() error {
	if d.Logger == nil {
		return fmt.Errorf("logger dependency is nil")
	}
	if d.UI == nil {
		return fmt.Errorf("UI dependency is nil")
	}
	return nil
}

// Close performs cleanup of resources if needed.
func (d *Dependencies) Close(ctx context.Context) error {
	// For now, we don't have resources that need explicit cleanup
	// This method is here for future extensibility
	return nil
}
