// Package main provides the entry point for the Antimoji CLI tool.
package main

import (
	"fmt"
	"os"

	"github.com/antimoji/antimoji/internal/app"
	"github.com/antimoji/antimoji/internal/observability/logging"
	"github.com/antimoji/antimoji/internal/ui"
)

// Build information (set by ldflags during build)
var (
	version   = "dev"
	buildTime = "unknown"
	gitCommit = "unknown"
)

func main() {
	// Create application configuration
	config := &app.Config{
		// Default logging configuration (silent mode)
		LogLevel:  logging.LevelSilent,
		LogFormat: logging.FormatJSON,
		LogOutput: os.Stderr,

		// Default UI configuration
		UILevel:        ui.OutputNormal,
		UIWriter:       os.Stdout,
		UIErrorWriter:  os.Stderr,
		UIEnableColors: true,

		// Application metadata
		ServiceName:    "antimoji",
		ServiceVersion: version,
	}

	// Create dependencies
	deps, err := app.NewDependencies(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize application dependencies: %v\n", err)
		os.Exit(1)
	}

	// Create application
	application, err := app.New(deps)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create application: %v\n", err)
		os.Exit(1)
	}

	// Run application
	if err := application.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
