// Package cli provides the clean command implementation for emoji removal.
package cli

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
}

// NewCleanCommand creates the clean command.
func NewCleanCommand() *cobra.Command {
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
			return runClean(cmd, args, opts)
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

// runClean executes the clean command logic.
func runClean(_ *cobra.Command, args []string, opts *CleanOptions) error {
	startTime := time.Now()

	// Create operation context with proper tracing
	ctx := ctxutil.NewComponentContext("clean", "cli")

	// Log operation start
	logging.Info(ctx, "Starting clean operation",
		"dry_run", dryRun,
		"recursive", opts.Recursive,
		"backup", opts.Backup,
		"args", args)

	// Validate options
	if err := validateCleanOptions(opts); err != nil {
		logging.Error(ctx, "Clean options validation failed", "error", err)
		ui.Error(ctx, "Invalid options: %v", err)
		return err
	}

	// If no paths provided, use current directory
	if len(args) == 0 {
		args = []string{"."}
		logging.Debug(ctx, "No paths provided, using current directory")
	}

	// Load configuration (same as scan command)
	cfg := config.DefaultConfig()
	if cfgFile != "" {
		logging.Debug(ctx, "Loading configuration file", "config_file", cfgFile)
		configResult := config.LoadConfig(cfgFile)
		if configResult.IsErr() {
			logging.Error(ctx, "Failed to load configuration", "config_file", cfgFile, "error", configResult.Error())
			ui.Error(ctx, "Failed to load config: %v", configResult.Error())
			return fmt.Errorf("failed to load config: %w", configResult.Error())
		}
		cfg = configResult.Unwrap()
		logging.Debug(ctx, "Configuration loaded successfully")
	}

	// Get the specified profile
	logging.Debug(ctx, "Loading profile", "profile_name", profileName)
	profileResult := config.GetProfile(cfg, profileName)
	if profileResult.IsErr() {
		logging.Error(ctx, "Failed to get profile", "profile_name", profileName, "error", profileResult.Error())
		ui.Error(ctx, "Failed to get profile '%s': %v", profileName, profileResult.Error())
		return fmt.Errorf("failed to get profile '%s': %w", profileName, profileResult.Error())
	}
	profile := profileResult.Unwrap()
	logging.Debug(ctx, "Profile loaded successfully", "profile_name", profileName)

	// Create allowlist using unified processing logic
	logging.Debug(ctx, "Creating allowlist for processing")
	allowlistOpts := allowlist.ProcessingOptions{
		IgnoreAllowlist:  opts.IgnoreAllowlist,
		RespectAllowlist: opts.RespectAllowlist,
		Operation:        "clean",
	}
	emojiAllowlist, err := allowlist.CreateAllowlistForProcessing(ctx, profile, allowlistOpts)
	if err != nil {
		logging.Error(ctx, "Failed to create allowlist", "error", err)
		ui.Error(ctx, "Failed to create allowlist: %v", err)
		return fmt.Errorf("failed to create allowlist: %w", err)
	}
	shouldUseAllowlist := allowlist.ShouldUseAllowlist(allowlistOpts, profile)
	logging.Debug(ctx, "Allowlist created", "should_use_allowlist", shouldUseAllowlist)

	// Discover files to process using unified filtering engine
	logging.Debug(ctx, "Starting file discovery", "paths", args, "recursive", opts.Recursive)
	discoveryOpts := filtering.DiscoveryOptions{
		Recursive: opts.Recursive,
		// Clean command doesn't have include/exclude pattern flags, so leave empty
	}
	filePaths, err := filtering.DiscoverFiles(args, discoveryOpts, profile)
	if err != nil {
		logging.Error(ctx, "File discovery failed", "error", err, "paths", args)
		ui.Error(ctx, "Failed to discover files: %v", err)
		return fmt.Errorf("failed to discover files: %w", err)
	}

	// Use proper user output instead of direct stderr
	ui.Info(ctx, "File discovery completed for cleaning - files found: %d", len(filePaths))
	logging.Info(ctx, "File discovery completed", "files_found", len(filePaths), "paths", args)

	// Create modification configuration
	logging.Debug(ctx, "Creating modification configuration")
	// Use the resolved allowlist behavior (ignore-allowlist takes precedence)
	modifyConfig := processor.ModifyConfig{
		Replacement:         opts.Replace,
		CreateBackup:        opts.Backup,
		RespectAllowlist:    shouldUseAllowlist,
		PreservePermissions: true,
		DryRun:              dryRun,
	}
	logging.Debug(ctx, "Modification configuration created",
		"dry_run", modifyConfig.DryRun,
		"backup", modifyConfig.CreateBackup,
		"respect_allowlist", modifyConfig.RespectAllowlist)

	// Create emoji patterns
	logging.Debug(ctx, "Creating emoji patterns")
	patterns := detector.DefaultEmojiPatterns()
	logging.Debug(ctx, "Emoji patterns created", "unicode_ranges", len(patterns.UnicodeRanges))

	// Modify files
	ui.Progress(ctx, "Starting file modification process - total files: %d", len(filePaths))
	logging.Info(ctx, "Starting file modification process", "total_files", len(filePaths))
	results := processor.ModifyFiles(filePaths, patterns, modifyConfig, emojiAllowlist)
	logging.Info(ctx, "File modification process completed",
		"total_results", len(results))

	// Display results using proper user output system
	if err := displayCleanResults(ctx, results, opts, time.Since(startTime)); err != nil {
		logging.Error(ctx, "Failed to display results", "error", err)
		return fmt.Errorf("failed to display results: %w", err)
	}

	return nil
}

// validateCleanOptions validates the clean command options.
func validateCleanOptions(opts *CleanOptions) error {
	if !opts.InPlace && !dryRun {
		return fmt.Errorf("must specify --in-place or --dry-run to modify files")
	}
	return nil
}

// displayCleanResults displays the clean operation results using proper user output.
func displayCleanResults(ctx context.Context, results []processor.ModifyResult, opts *CleanOptions, duration time.Duration) error {
	logging.Debug(ctx, "Displaying clean results", "total_results", len(results), "stats", opts.Stats)

	if opts.Stats {
		ui.Result(ctx, "Clean Results Summary")
		ui.Result(ctx, "=====================")
	}

	totalFiles := 0
	modifiedFiles := 0
	totalEmojisRemoved := 0
	errorFiles := 0
	backupFiles := 0

	for _, result := range results {
		if result.Error != nil {
			errorFiles++
			logging.Error(ctx, "File processing error",
				"file_path", result.FilePath,
				"error", result.Error)
			if verbose {
				ui.Error(ctx, "%s - %v", result.FilePath, result.Error)
			}
			continue
		}

		totalFiles++
		if result.Modified {
			modifiedFiles++
			totalEmojisRemoved += result.EmojisRemoved

			logging.Info(ctx, "File modified",
				"file_path", result.FilePath,
				"emojis_removed", result.EmojisRemoved,
				"dry_run", dryRun)

			if verbose {
				action := "would remove"
				if !dryRun {
					action = "removed"
				}
				ui.Success(ctx, "%s - %s %d emojis", result.FilePath, action, result.EmojisRemoved)
			}
		}

		// Track backup files
		if result.BackupPath != "" {
			backupFiles++
			logging.Debug(ctx, "Backup created",
				"original_path", result.FilePath,
				"backup_path", result.BackupPath)
			if verbose {
				ui.Info(ctx, "BACKUP: %s -> %s", result.FilePath, result.BackupPath)
			}
		}
	}

	// Display summary
	action := "would remove"
	if !dryRun {
		action = "removed"
	}

	ui.Success(ctx, "Summary: %s %d emojis from %d files (%d modified, %d errors)",
		action, totalEmojisRemoved, totalFiles, modifiedFiles, errorFiles)

	logging.Info(ctx, "Clean operation completed",
		"total_files", totalFiles,
		"modified_files", modifiedFiles,
		"total_emojis_removed", totalEmojisRemoved,
		"error_files", errorFiles,
		"backup_files", backupFiles,
		"duration", duration)

	if backupFiles > 0 {
		ui.Info(ctx, "Created %d backup files", backupFiles)
	}

	if opts.Benchmark {
		ui.Info(ctx, "Processing time: %v", duration)
		if totalFiles > 0 {
			ui.Info(ctx, "Average: %.2f files/second", float64(totalFiles)/duration.Seconds())
		}
	}

	return nil
}
