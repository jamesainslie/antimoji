// Package commands provides CLI command implementations using dependency injection.
package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/antimoji/antimoji/internal/config"
	"github.com/antimoji/antimoji/internal/core/allowlist"
	"github.com/antimoji/antimoji/internal/core/detector"
	"github.com/antimoji/antimoji/internal/core/processor"
	"github.com/antimoji/antimoji/internal/infra/filtering"
	ctxutil "github.com/antimoji/antimoji/internal/observability/context"
	"github.com/antimoji/antimoji/internal/observability/logging"
	"github.com/antimoji/antimoji/internal/ui"
	"github.com/spf13/cobra"
)

// CleanOptions holds the options for the clean command.
type CleanOptions struct {
	Recursive        bool
	Backup           bool
	Replace          string
	InPlace          bool
	RespectAllowlist bool
	IgnoreAllowlist  bool
	Stats            bool
	Benchmark        bool
	DryRun           bool
}

// CleanHandler handles the clean command with dependency injection.
type CleanHandler struct {
	logger logging.Logger
	ui     ui.UserOutput
}

// NewCleanHandler creates a new clean command handler.
func NewCleanHandler(logger logging.Logger, ui ui.UserOutput) *CleanHandler {
	return &CleanHandler{
		logger: logger,
		ui:     ui,
	}
}

// CreateCommand creates the clean cobra command.
func (h *CleanHandler) CreateCommand() *cobra.Command {
	opts := &CleanOptions{}

	cmd := &cobra.Command{
		Use:   "clean [flags] [path...]",
		Short: "Remove emojis from files",
		Long: `Remove emojis from files and directories with optional backup and allowlist support.

This command modifies files to remove detected emojis while preserving code
structure and functionality. It supports atomic file operations, backup creation,
and allowlist filtering to keep certain emojis.

Examples:
  antimoji clean file.go                    # Clean specific file (requires --in-place)
  antimoji clean --in-place .               # Clean current directory in-place
  antimoji clean --backup --in-place src/   # Clean with backup creation
  antimoji clean --replace "[EMOJI]" .      # Replace emojis with text
  antimoji clean --respect-allowlist .      # Keep allowlisted emojis
  antimoji clean --dry-run .                # Preview changes without modifying`,
		Args: cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get dry-run from persistent flag (parent command)
			dryRun, _ := cmd.Root().PersistentFlags().GetBool("dry-run")
			opts.DryRun = dryRun
			return h.Execute(cmd.Context(), args, opts)
		},
	}

	// Add clean-specific flags
	cmd.Flags().BoolVarP(&opts.Recursive, "recursive", "r", true, "clean directories recursively")
	cmd.Flags().BoolVar(&opts.Backup, "backup", false, "create backup files")
	cmd.Flags().StringVar(&opts.Replace, "replace", "", "replacement text for emojis")
	cmd.Flags().BoolVarP(&opts.InPlace, "in-place", "i", false, "modify files in place")
	cmd.Flags().BoolVar(&opts.RespectAllowlist, "respect-allowlist", true, "respect configured emoji allowlist during cleaning (deprecated, use --ignore-allowlist)")
	cmd.Flags().BoolVar(&opts.IgnoreAllowlist, "ignore-allowlist", false, "ignore configured emoji allowlist (overrides --respect-allowlist)")
	cmd.Flags().BoolVar(&opts.Stats, "stats", false, "show performance statistics")
	cmd.Flags().BoolVar(&opts.Benchmark, "benchmark", false, "run in benchmark mode with detailed metrics")

	return cmd
}

// Execute runs the clean command logic with dependency injection.
func (h *CleanHandler) Execute(parentCtx context.Context, args []string, opts *CleanOptions) error {
	startTime := time.Now()

	// Create component context for better tracing
	ctx := ctxutil.NewComponentContext("clean", "cli")

	h.logger.Info(ctx, "Starting clean operation",
		"dry_run", opts.DryRun,
		"recursive", opts.Recursive,
		"backup", opts.Backup,
		"args", args)

	// Validate options
	if err := h.validateCleanOptions(opts); err != nil {
		h.logger.Error(ctx, "Clean options validation failed", "error", err)
		h.ui.Error(ctx, "Invalid options: %v", err)
		return err
	}

	// If no paths provided, use current directory
	if len(args) == 0 {
		args = []string{"."}
		h.logger.Debug(ctx, "No paths provided, using current directory")
	}

	// TODO: For now, use default config - this will be replaced with proper CLI flag parsing
	cfg := config.DefaultConfig()
	profileName := "default"

	h.logger.Debug(ctx, "Loading profile", "profile_name", profileName)
	profileResult := config.GetProfile(cfg, profileName)
	if profileResult.IsErr() {
		h.logger.Error(ctx, "Failed to get profile", "profile_name", profileName, "error", profileResult.Error())
		return fmt.Errorf("failed to get profile '%s': %w", profileName, profileResult.Error())
	}

	profile := profileResult.Unwrap()
	h.logger.Debug(ctx, "Profile loaded successfully", "profile_name", profileName)

	// Create allowlist for processing
	h.logger.Debug(ctx, "Creating allowlist for processing")
	allowlistOpts := allowlist.ProcessingOptions{
		IgnoreAllowlist:  opts.IgnoreAllowlist,
		RespectAllowlist: opts.RespectAllowlist && !opts.IgnoreAllowlist,
		Operation:        "clean",
	}

	emojiAllowlist, err := allowlist.CreateAllowlistForProcessing(ctx, profile, allowlistOpts)
	if err != nil {
		h.logger.Error(ctx, "Failed to create allowlist", "error", err)
		return fmt.Errorf("failed to create allowlist: %w", err)
	}

	shouldUseAllowlist := emojiAllowlist != nil
	h.logger.Debug(ctx, "Allowlist created", "should_use_allowlist", shouldUseAllowlist)

	// Start file discovery
	h.logger.Debug(ctx, "Starting file discovery", "paths", args, "recursive", opts.Recursive)

	discoveryOptions := filtering.DiscoveryOptions{
		Recursive:      opts.Recursive,
		IncludePattern: "", // TODO: Add CLI support for include/exclude patterns
		ExcludePattern: "",
	}

	filePaths, err := filtering.DiscoverFiles(args, discoveryOptions, profile)
	if err != nil {
		h.logger.Error(ctx, "File discovery failed", "error", err, "paths", args)
		return fmt.Errorf("file discovery failed: %w", err)
	}

	if len(filePaths) == 0 {
		h.ui.Warning(ctx, "No files found matching the criteria")
		return nil
	}

	h.logger.Info(ctx, "File discovery completed", "files_found", len(filePaths), "paths", args)

	// Create modification configuration
	h.logger.Debug(ctx, "Creating modification configuration")
	modifyConfig := processor.ModifyConfig{
		DryRun:              opts.DryRun,
		CreateBackup:        opts.Backup,
		RespectAllowlist:    shouldUseAllowlist,
		Replacement:         opts.Replace,
		PreservePermissions: true,
	}

	h.logger.Debug(ctx, "Modification configuration created",
		"dry_run", modifyConfig.DryRun,
		"create_backup", modifyConfig.CreateBackup,
		"preserve_permissions", modifyConfig.PreservePermissions)

	// Create emoji patterns
	patterns := detector.DefaultEmojiPatterns()
	h.logger.Debug(ctx, "Emoji patterns created", "unicode_ranges", len(patterns.UnicodeRanges))

	// Process files for modification
	h.logger.Info(ctx, "Starting file modification process", "total_files", len(filePaths))
	results := processor.ModifyFiles(filePaths, patterns, modifyConfig, emojiAllowlist)
	h.logger.Info(ctx, "File modification process completed", "total_results", len(results))

	// Display results
	if err := h.displayResults(ctx, results, opts, time.Since(startTime)); err != nil {
		h.logger.Error(ctx, "Failed to display results", "error", err)
		return fmt.Errorf("failed to display results: %w", err)
	}

	h.logger.Info(ctx, "Clean operation completed successfully")
	return nil
}

// validateCleanOptions validates the clean command options.
func (h *CleanHandler) validateCleanOptions(opts *CleanOptions) error {
	if !opts.InPlace && !opts.DryRun {
		return fmt.Errorf("must specify --in-place to modify files, or --dry-run to preview changes")
	}
	return nil
}

// displayResults displays the clean operation results.
func (h *CleanHandler) displayResults(ctx context.Context, results []processor.ModifyResult, opts *CleanOptions, duration time.Duration) error {
	h.logger.Debug(ctx, "Displaying clean results", "total_results", len(results), "stats", opts.Stats)

	// Count statistics
	totalFiles := len(results)
	modifiedFiles := 0
	errorCount := 0
	totalEmojisRemoved := 0

	for _, result := range results {
		if result.Error != nil {
			errorCount++
			h.logger.Error(ctx, "File processing error",
				"file_path", result.FilePath,
				"error", result.Error)
			h.ui.Error(ctx, "Error processing %s: %v", result.FilePath, result.Error)
		} else if result.Modified {
			modifiedFiles++
			h.logger.Info(ctx, "File modified",
				"file_path", result.FilePath,
				"emojis_removed", result.EmojisRemoved)

			if !opts.DryRun {
				h.ui.Success(ctx, "Cleaned %s: %d emojis removed", result.FilePath, result.EmojisRemoved)
			} else {
				h.ui.Info(ctx, "Would clean %s: %d emojis to remove", result.FilePath, result.EmojisRemoved)
			}

			totalEmojisRemoved += result.EmojisRemoved

			// Show backup information if created
			if result.BackupPath != "" {
				h.logger.Debug(ctx, "Backup created",
					"original_file", result.FilePath,
					"backup_file", result.BackupPath)
				h.ui.Info(ctx, "Backup created: %s", result.BackupPath)
			}
		}
	}

	// Display summary
	if opts.DryRun {
		h.ui.Result(ctx, "Summary: would remove %d emojis from %d files (%d modified, %d errors)",
			totalEmojisRemoved, totalFiles, modifiedFiles, errorCount)
	} else {
		h.ui.Result(ctx, "Summary: removed %d emojis from %d files (%d modified, %d errors)",
			totalEmojisRemoved, totalFiles, modifiedFiles, errorCount)
	}

	// Show performance statistics if requested
	if opts.Stats {
		h.ui.Info(ctx, "Processing time: %v", duration)
		if totalFiles > 0 {
			h.ui.Info(ctx, "Files per second: %.2f", float64(totalFiles)/duration.Seconds())
		}
	}

	h.logger.Info(ctx, "Clean operation completed",
		"total_files", totalFiles,
		"modified_files", modifiedFiles,
		"errors", errorCount,
		"emojis_removed", totalEmojisRemoved,
		"duration", duration)

	return nil
}
