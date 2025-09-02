// Package cli provides the clean command implementation for emoji removal.
package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/antimoji/antimoji/internal/config"
	"github.com/antimoji/antimoji/internal/core/allowlist"
	"github.com/antimoji/antimoji/internal/core/detector"
	"github.com/antimoji/antimoji/internal/core/processor"
	"github.com/spf13/cobra"
)

// CleanOptions holds the options for the clean command.
type CleanOptions struct {
	Recursive        bool
	Backup           bool
	Replace          string
	InPlace          bool
	RespectAllowlist bool
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
	cmd.Flags().BoolVar(&opts.RespectAllowlist, "respect-allowlist", true, "respect configured emoji allowlist during cleaning")
	cmd.Flags().BoolVar(&opts.Stats, "stats", false, "show performance statistics")
	cmd.Flags().BoolVar(&opts.Benchmark, "benchmark", false, "run in benchmark mode with detailed metrics")

	return cmd
}

// runClean executes the clean command logic.
func runClean(cmd *cobra.Command, args []string, opts *CleanOptions) error {
	startTime := time.Now()

	// Validate options
	if err := validateCleanOptions(opts); err != nil {
		return err
	}

	// If no paths provided, use current directory
	if len(args) == 0 {
		args = []string{"."}
	}

	// Load configuration (same as scan command)
	cfg := config.DefaultConfig()
	if cfgFile != "" {
		configResult := config.LoadConfig(cfgFile)
		if configResult.IsErr() {
			return fmt.Errorf("failed to load config: %w", configResult.Error())
		}
		cfg = configResult.Unwrap()
	}

	// Get the specified profile
	profileResult := config.GetProfile(cfg, profileName)
	if profileResult.IsErr() {
		return fmt.Errorf("failed to get profile '%s': %w", profileName, profileResult.Error())
	}
	profile := profileResult.Unwrap()

	// Create allowlist if configured
	var emojiAllowlist *allowlist.Allowlist
	if opts.RespectAllowlist && len(profile.EmojiAllowlist) > 0 {
		allowlistResult := allowlist.NewAllowlist(profile.EmojiAllowlist)
		if allowlistResult.IsErr() {
			return fmt.Errorf("failed to create allowlist: %w", allowlistResult.Error())
		}
		emojiAllowlist = allowlistResult.Unwrap()

		if verbose {
			fmt.Fprintf(os.Stderr, "Using allowlist with %d patterns\n", emojiAllowlist.Size())
		}
	}

	// Discover files to process (reuse scan logic)
	scanOpts := &ScanOptions{
		Recursive: opts.Recursive,
	}
	filePaths, err := discoverFiles(args, scanOpts, profile)
	if err != nil {
		return fmt.Errorf("failed to discover files: %w", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Found %d files to clean\n", len(filePaths))
	}

	// Create modification configuration
	modifyConfig := processor.ModifyConfig{
		Replacement:         opts.Replace,
		CreateBackup:        opts.Backup,
		RespectAllowlist:    opts.RespectAllowlist,
		PreservePermissions: true,
		DryRun:              dryRun,
	}

	// Create emoji patterns
	patterns := detector.DefaultEmojiPatterns()

	// Modify files
	results := processor.ModifyFiles(filePaths, patterns, modifyConfig, emojiAllowlist)

	// Display results
	if err := displayCleanResults(results, opts, time.Since(startTime)); err != nil {
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

// displayCleanResults displays the clean operation results.
func displayCleanResults(results []processor.ModifyResult, opts *CleanOptions, duration time.Duration) error {
	if opts.Stats {
		fmt.Printf("Clean Results Summary\n")
		fmt.Printf("=====================\n")
	}

	totalFiles := 0
	modifiedFiles := 0
	totalEmojisRemoved := 0
	errorFiles := 0
	backupFiles := 0

	for _, result := range results {
		if result.Error != nil {
			errorFiles++
			if verbose {
				fmt.Printf("ERROR: %s - %v\n", result.FilePath, result.Error)
			}
			continue
		}

		totalFiles++
		if result.Modified {
			modifiedFiles++
			totalEmojisRemoved += result.EmojisRemoved

			if verbose {
				action := "would remove"
				if !dryRun {
					action = "removed"
				}
				fmt.Printf("MODIFIED: %s - %s %d emojis\n", result.FilePath, action, result.EmojisRemoved)
			}
		}

		if result.BackupPath != "" {
			backupFiles++
			if verbose {
				fmt.Printf("BACKUP: %s -> %s\n", result.FilePath, result.BackupPath)
			}
		}
	}

	// Display summary
	action := "would remove"
	if !dryRun {
		action = "removed"
	}

	fmt.Printf("Summary: %s %d emojis from %d files (%d modified, %d errors)\n",
		action, totalEmojisRemoved, totalFiles, modifiedFiles, errorFiles)

	if backupFiles > 0 {
		fmt.Printf("Created %d backup files\n", backupFiles)
	}

	if opts.Stats {
		fmt.Printf("Processing time: %v\n", duration)
		if totalFiles > 0 {
			fmt.Printf("Average: %.2f files/second\n", float64(totalFiles)/duration.Seconds())
		}
	}

	return nil
}
