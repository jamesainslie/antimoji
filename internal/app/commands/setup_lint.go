// Package commands provides CLI command implementations using dependency injection.
package commands

import (
	"context"
	"fmt"

	"github.com/antimoji/antimoji/internal/observability/logging"
	"github.com/antimoji/antimoji/internal/ui"
	"github.com/spf13/cobra"
)

// SetupLintOptions holds the options for the setup-lint command.
type SetupLintOptions struct {
	Mode              string
	OutputDir         string
	PreCommitConfig   bool
	AllowedEmojis     []string
	Force             bool
	SkipPreCommitHook bool
	Repair            bool
	Review            bool
	Validate          bool
}

// SetupLintHandler handles the setup-lint command with dependency injection.
type SetupLintHandler struct {
	logger logging.Logger
	ui     ui.UserOutput
}

// NewSetupLintHandler creates a new setup-lint command handler.
func NewSetupLintHandler(logger logging.Logger, ui ui.UserOutput) *SetupLintHandler {
	return &SetupLintHandler{
		logger: logger,
		ui:     ui,
	}
}

// CreateCommand creates the setup-lint cobra command.
func (h *SetupLintHandler) CreateCommand() *cobra.Command {
	opts := &SetupLintOptions{}

	cmd := &cobra.Command{
		Use:   "setup-lint [flags] [path]",
		Short: "Automatically setup linting configuration for emoji detection",
		Long: `Setup automated linting configuration with pre-commit hooks for emoji detection.

This command configures antimoji for automated emoji linting in your development workflow.
It supports three different modes:

Linting Modes:
  zero-tolerance - Disallows ALL emojis in source code (strictest)
  allow-list     - Allows only specific emojis (1-2 common ones by default)
  permissive     - Allows emojis but warns about excessive usage

The command will:
- Generate appropriate .antimoji.yaml configuration
- Append antimoji hooks to existing .pre-commit-config.yaml (or create new)
- Setup pre-commit hooks for automated emoji cleaning

Examples:
  antimoji setup-lint --mode=zero-tolerance    # Strict: no emojis allowed
  antimoji setup-lint --mode=allow-list        # Allow specific emojis only
  antimoji setup-lint --mode=permissive        # Lenient with warnings
  antimoji setup-lint --force                  # Overwrite existing configs
  antimoji setup-lint --repair                 # Repair missing configs
  antimoji setup-lint --review                 # Review existing configuration`,
		Args:          cobra.MaximumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return h.Execute(cmd.Context(), cmd, args, opts)
		},
	}

	// Add setup-lint specific flags
	cmd.Flags().StringVar(&opts.Mode, "mode", "zero-tolerance", "linting mode (zero-tolerance, allow-list, permissive)")
	cmd.Flags().StringVar(&opts.OutputDir, "output-dir", ".", "output directory for configuration files")
	cmd.Flags().BoolVar(&opts.PreCommitConfig, "precommit", true, "generate/update .pre-commit-config.yaml")
	cmd.Flags().StringSliceVar(&opts.AllowedEmojis, "allowed-emojis", []string{"", ""}, "emojis to allow in allow-list mode")
	cmd.Flags().BoolVar(&opts.Force, "force", false, "overwrite existing configuration files")
	cmd.Flags().BoolVar(&opts.SkipPreCommitHook, "skip-precommit", false, "skip pre-commit hook installation")
	cmd.Flags().BoolVar(&opts.Repair, "repair", false, "repair missing configs")
	cmd.Flags().BoolVar(&opts.Review, "review", false, "review existing configuration")
	cmd.Flags().BoolVar(&opts.Validate, "validate", false, "validate existing configuration")

	return cmd
}

// Execute runs the setup-lint command logic with dependency injection.
func (h *SetupLintHandler) Execute(parentCtx context.Context, cmd *cobra.Command, args []string, opts *SetupLintOptions) error {
	// Derive from parent for cancellation/values
	ctx := parentCtx
	if ctx == nil {
		ctx = context.Background()
	}

	h.logger.Info(ctx, "Starting setup-lint operation",
		"mode", opts.Mode,
		"output_dir", opts.OutputDir,
		"args", args)

	// TODO: For now, return a placeholder message until full implementation
	// This maintains the same error pattern as before but with dependency injection
	h.ui.Info(ctx, "Setup-lint command - dependency injection working!")
	h.ui.Info(ctx, "Would setup linting with mode: %s", opts.Mode)
	h.ui.Info(ctx, "Output directory: %s", opts.OutputDir)

	h.logger.Info(ctx, "Setup-lint command executed with DI",
		"args", args,
		"mode", opts.Mode,
		"output_dir", opts.OutputDir)

	return fmt.Errorf("setup-lint command not yet fully refactored for dependency injection - complex implementation pending")
}
