// Package commands provides CLI command implementations using dependency injection.
package commands

import (
	"context"
	"fmt"

	"github.com/antimoji/antimoji/internal/observability/logging"
	"github.com/antimoji/antimoji/internal/ui"
	"github.com/spf13/cobra"
)

// GenerateOptions holds the options for the generate command.
type GenerateOptions struct {
	Output       string
	Type         string
	IncludeTests bool
	IncludeDocs  bool
	IncludeCI    bool
	Recursive    bool
	MinUsage     int
	Format       string
	Profile      string
}

// GenerateHandler handles the generate command with dependency injection.
type GenerateHandler struct {
	logger logging.Logger
	ui     ui.UserOutput
}

// NewGenerateHandler creates a new generate command handler.
func NewGenerateHandler(logger logging.Logger, ui ui.UserOutput) *GenerateHandler {
	return &GenerateHandler{
		logger: logger,
		ui:     ui,
	}
}

// CreateCommand creates the generate cobra command.
func (h *GenerateHandler) CreateCommand() *cobra.Command {
	opts := &GenerateOptions{}

	cmd := &cobra.Command{
		Use:   "generate [flags] [path...]",
		Short: "Generate allowlist configuration based on current project emoji usage",
		Long: `Analyze the current project's emoji usage and generate an appropriate allowlist configuration.

This command scans your project to find all emojis currently in use and generates
a configuration file that allows those emojis while maintaining strict linting
for new emoji additions.

Generation Types:
  ci-lint    - Strict allowlist for CI/CD linting (default)
  dev        - Permissive allowlist for development
  test-only  - Only allow emojis found in test files
  docs-only  - Only allow emojis found in documentation
  minimal    - Only allow most frequently used emojis
  full       - Allow all found emojis with categorization

Examples:
  antimoji generate .                           # Generate CI lint config
  antimoji generate --type=dev .                # Generate dev-friendly config
  antimoji generate --type=test-only .          # Allow only test emojis
  antimoji generate --output=.antimoji.yaml .   # Save to specific file
  antimoji generate --format=yaml --type=full . # Full analysis with YAML output
  antimoji generate --min-usage=3 .             # Only emojis used 3+ times`,
		Args:          cobra.MinimumNArgs(0),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return h.Execute(cmd.Context(), cmd, args, opts)
		},
	}

	// Add generate-specific flags
	cmd.Flags().StringVarP(&opts.Output, "output", "o", "", "output file path (default: stdout)")
	cmd.Flags().StringVar(&opts.Type, "type", "ci-lint", "generation type (ci-lint, dev, test-only, docs-only, minimal, full)")
	cmd.Flags().BoolVar(&opts.IncludeTests, "include-tests", true, "include emojis from test files")
	cmd.Flags().BoolVar(&opts.IncludeDocs, "include-docs", true, "include emojis from documentation files")
	cmd.Flags().BoolVar(&opts.IncludeCI, "include-ci", true, "include emojis from CI/CD files")
	cmd.Flags().BoolVarP(&opts.Recursive, "recursive", "r", true, "scan directories recursively")
	cmd.Flags().IntVar(&opts.MinUsage, "min-usage", 1, "minimum usage count to include emoji in allowlist")
	cmd.Flags().StringVar(&opts.Format, "format", "yaml", "output format (yaml, json)")
	cmd.Flags().StringVar(&opts.Profile, "profile-name", "", "name for the generated profile (default: based on type)")

	return cmd
}

// Execute runs the generate command logic with dependency injection.
func (h *GenerateHandler) Execute(parentCtx context.Context, cmd *cobra.Command, args []string, opts *GenerateOptions) error {
	// Derive from parent for cancellation/values
	ctx := parentCtx
	if ctx == nil {
		ctx = context.Background()
	}

	// If no paths provided, use current directory
	if len(args) == 0 {
		args = []string{"."}
		h.logger.Debug(ctx, "No paths provided, using current directory")
	}

	h.logger.Info(ctx, "Starting emoji analysis for allowlist generation",
		"operation", "generate",
		"type", opts.Type,
		"paths", args)

	// TODO: For now, return a placeholder message until full implementation
	// This maintains the same error pattern as before but with dependency injection
	h.ui.Info(ctx, "Generate command - dependency injection working!")
	h.ui.Info(ctx, "Analysis would be performed for type: %s", opts.Type)
	h.ui.Info(ctx, "Output format: %s", opts.Format)

	h.logger.Info(ctx, "Generate command executed with DI",
		"args", args,
		"type", opts.Type,
		"format", opts.Format)

	return fmt.Errorf("generate command not yet fully refactored for dependency injection - complex implementation pending")
}
