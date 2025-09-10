// Package app provides the main application structure and lifecycle management.
package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/antimoji/antimoji/internal/app/commands"
	"github.com/spf13/cobra"
)

// Application represents the main application with its dependencies.
type Application struct {
	deps    *Dependencies
	rootCmd *cobra.Command
	ctx     context.Context
	cancel  context.CancelFunc
}

// New creates a new Application instance with the given dependencies.
func New(deps *Dependencies) (*Application, error) {
	if deps == nil {
		return nil, fmt.Errorf("dependencies cannot be nil")
	}

	if err := deps.Validate(); err != nil {
		return nil, fmt.Errorf("invalid dependencies: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	app := &Application{
		deps:   deps,
		ctx:    ctx,
		cancel: cancel,
	}

	// Create root command with dependency injection
	app.rootCmd = app.createRootCommand()

	return app, nil
}

// Run starts the application and handles the command execution.
func (a *Application) Run(args []string) error {
	// Set up graceful shutdown
	go a.handleSignals()

	// Set command arguments
	a.rootCmd.SetArgs(args)

	// Execute the command
	if err := a.rootCmd.ExecuteContext(a.ctx); err != nil {
		return fmt.Errorf("command execution failed: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the application.
func (a *Application) Shutdown() error {
	// Create a fresh timeout-bound context for cleanup
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	// Run cleanup with the timeout context
	err := a.deps.Close(shutdownCtx)

	// Cancel background work after cleanup completes
	a.cancel()

	return err
}

// GetDependencies returns the application dependencies (useful for testing).
func (a *Application) GetDependencies() *Dependencies {
	return a.deps
}

// GetRootCommand returns the root cobra command (useful for testing).
func (a *Application) GetRootCommand() *cobra.Command {
	return a.rootCmd
}

// handleSignals sets up signal handling for graceful shutdown.
func (a *Application) handleSignals() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)
	defer close(sigChan)

	select {
	case sig := <-sigChan:
		a.deps.Logger.Info(a.ctx, "Received shutdown signal", "signal", sig.String())

		// Create a timeout-bound context for shutdown operations
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		// Perform graceful shutdown with timeout
		if err := a.deps.Close(shutdownCtx); err != nil {
			a.deps.Logger.Error(shutdownCtx, "Error during shutdown", "error", err)
		}

		// Cancel background work after shutdown completes
		a.cancel()
	case <-a.ctx.Done():
		// Context was cancelled elsewhere
		return
	}
}

// createRootCommand creates the root cobra command with dependency injection.
func (a *Application) createRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "antimoji",
		Short: "High-performance emoji detection and removal CLI tool",
		Long: `Antimoji is a blazing-fast CLI tool for detecting and removing emojis
from code files, markdown documents, and other text-based artifacts.

Built with Go using functional programming principles, Antimoji provides:
- Unicode emoji detection across all major ranges
- Text emoticon detection (e.g., , , )
- Custom emoji pattern detection (e.g., :party:, :thumbsup:)
- Configurable allowlists and ignore patterns
- High-performance concurrent processing
- Git integration and CI/CD pipeline support`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       a.getBuildVersion(),
	}

	// Add global persistent flags
	cmd.PersistentFlags().String("config", "", "config file path")
	cmd.PersistentFlags().String("profile", "default", "configuration profile")
	cmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output (deprecated, use --log-level=info)")
	cmd.PersistentFlags().BoolP("quiet", "q", false, "quiet mode (deprecated, use --log-level=silent)")
	cmd.PersistentFlags().Bool("dry-run", false, "show what would be changed without modifying files")
	cmd.PersistentFlags().String("log-level", "silent", "log level (silent, debug, info, warn, error)")
	cmd.PersistentFlags().String("log-format", "json", "log format (json, text)")

	// Add subcommands with dependency injection
	cmd.AddCommand(a.createScanCommand())
	cmd.AddCommand(a.createCleanCommand())
	cmd.AddCommand(a.createGenerateCommand())
	cmd.AddCommand(a.createSetupLintCommand())
	cmd.AddCommand(a.createVersionCommand())

	return cmd
}

// getBuildVersion returns the current build version.
// This will be set by build-time variables later.
func (a *Application) getBuildVersion() string {
	return "0.9.16-refactor"
}

// Placeholder methods for command creation - these will be implemented
// as we refactor each command to use dependency injection.

func (a *Application) createScanCommand() *cobra.Command {
	handler := commands.NewScanHandler(a.deps.Logger, a.deps.UI)
	return handler.CreateCommand()
}

func (a *Application) createCleanCommand() *cobra.Command {
	handler := commands.NewCleanHandler(a.deps.Logger, a.deps.UI)
	return handler.CreateCommand()
}

func (a *Application) createGenerateCommand() *cobra.Command {
	handler := commands.NewGenerateHandler(a.deps.Logger, a.deps.UI)
	return handler.CreateCommand()
}

func (a *Application) createSetupLintCommand() *cobra.Command {
	handler := commands.NewSetupLintHandler(a.deps.Logger, a.deps.UI)
	return handler.CreateCommand()
}

func (a *Application) createVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			version := a.getBuildVersion()
			a.deps.UI.Info(a.ctx, "Antimoji version %s", version)
			a.deps.Logger.Info(a.ctx, "Version command executed", "version", version)
			return nil
		},
	}
}
