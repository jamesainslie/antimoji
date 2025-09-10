// Package commands provides CLI command implementations using dependency injection.
package commands

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/antimoji/antimoji/internal/config"
	"github.com/antimoji/antimoji/internal/core/allowlist"
	"github.com/antimoji/antimoji/internal/core/detector"
	"github.com/antimoji/antimoji/internal/core/processor"
	"github.com/antimoji/antimoji/internal/infra/filtering"
	ctxutil "github.com/antimoji/antimoji/internal/observability/context"
	"github.com/antimoji/antimoji/internal/observability/logging"
	"github.com/antimoji/antimoji/internal/types"
	"github.com/antimoji/antimoji/internal/ui"
	"github.com/spf13/cobra"
)

// ScanOptions holds the options for the scan command.
type ScanOptions struct {
	Recursive       bool
	IncludePattern  string
	ExcludePattern  string
	Format          string
	CountOnly       bool
	Threshold       int
	IgnoreAllowlist bool
	Stats           bool
	Benchmark       bool
	Workers         int
}

// ErrEmojiThresholdExceeded indicates the total emoji count exceeded the provided threshold.
var ErrEmojiThresholdExceeded = errors.New("emoji threshold exceeded")

// ScanHandler handles the scan command with dependency injection.
type ScanHandler struct {
	logger logging.Logger
	ui     ui.UserOutput
}

// NewScanHandler creates a new scan command handler.
func NewScanHandler(logger logging.Logger, ui ui.UserOutput) *ScanHandler {
	return &ScanHandler{
		logger: logger,
		ui:     ui,
	}
}

// CreateCommand creates the scan cobra command.
func (h *ScanHandler) CreateCommand() *cobra.Command {
	opts := &ScanOptions{}

	cmd := &cobra.Command{
		Use:   "scan [flags] [path...]",
		Short: "Scan files for emojis without modifying them",
		Long: `Scan files and directories for emoji usage without making any modifications.

This command analyzes files to detect Unicode emojis, text emoticons, and custom
emoji patterns. It respects configuration settings including allowlists and
ignore patterns.

Examples:
  antimoji scan .                    # Scan current directory
  antimoji scan file.go              # Scan specific file
  antimoji scan --recursive src/     # Scan directory recursively
  antimoji scan --format table .    # Output results as a table
  antimoji scan --count-only .       # Show only emoji counts
  antimoji scan --stats .            # Include performance statistics`,
		Args:          cobra.MinimumNArgs(0),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return h.Execute(cmd.Context(), cmd, args, opts)
		},
	}

	// Add scan-specific flags
	cmd.Flags().BoolVarP(&opts.Recursive, "recursive", "r", true, "scan directories recursively")
	cmd.Flags().StringVar(&opts.IncludePattern, "include", "", "include file patterns (glob)")
	cmd.Flags().StringVar(&opts.ExcludePattern, "exclude", "", "exclude file patterns (glob)")
	cmd.Flags().StringVar(&opts.Format, "format", "table", "output format (table, json, csv)")
	cmd.Flags().BoolVar(&opts.CountOnly, "count-only", false, "show only emoji counts")
	cmd.Flags().IntVar(&opts.Threshold, "threshold", 0, "maximum allowed emoji count (for linting)")
	cmd.Flags().BoolVar(&opts.IgnoreAllowlist, "ignore-allowlist", false, "ignore configured emoji allowlist")
	cmd.Flags().BoolVar(&opts.Stats, "stats", false, "show performance statistics")
	cmd.Flags().BoolVar(&opts.Benchmark, "benchmark", false, "run in benchmark mode with detailed metrics")
	cmd.Flags().IntVar(&opts.Workers, "workers", 0, "number of concurrent workers (0 = auto-detect)")

	return cmd
}

// Execute runs the scan command logic with dependency injection.
func (h *ScanHandler) Execute(parentCtx context.Context, cmd *cobra.Command, args []string, opts *ScanOptions) error {
	startTime := time.Now()

	// Validate output format (table-only for now)
	switch strings.ToLower(opts.Format) {
	case "table":
		// ok
	default:
		return fmt.Errorf("unsupported format %q; supported: table", opts.Format)
	}

	// Derive from parent for cancellation/values, enhance with component context
	ctx := parentCtx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = ctxutil.WithOperation(ctx, "scan")
	ctx = ctxutil.WithComponent(ctx, "cli")

	// If no paths provided, use current directory
	if len(args) == 0 {
		args = []string{"."}
		h.logger.Debug(ctx, "No paths provided, using current directory")
	}

	h.logger.Info(ctx, "Starting scan operation", "paths", args, "options", opts)

	// Get config and profile from persistent flags
	configFile, _ := cmd.Root().PersistentFlags().GetString("config")
	profileName, _ := cmd.Root().PersistentFlags().GetString("profile")

	// Load configuration
	cfg := config.DefaultConfig()
	if configFile != "" {
		h.logger.Debug(ctx, "Loading configuration file", "config_file", configFile)
		configResult := config.LoadConfig(configFile)
		if configResult.IsErr() {
			h.logger.Error(ctx, "Failed to load configuration", "config_file", configFile, "error", configResult.Error())
			return fmt.Errorf("failed to load config: %w", configResult.Error())
		}
		cfg = configResult.Unwrap()
		h.logger.Debug(ctx, "Configuration loaded successfully")
	}

	// Get the specified profile
	profileResult := config.GetProfile(cfg, profileName)
	if profileResult.IsErr() {
		return fmt.Errorf("failed to get profile '%s': %w", profileName, profileResult.Error())
	}
	profile := profileResult.Unwrap()

	h.logger.Debug(ctx, "Profile loaded successfully", "profile_name", profileName)

	// Create allowlist for processing
	allowlistOpts := allowlist.ProcessingOptions{
		IgnoreAllowlist:  opts.IgnoreAllowlist,
		RespectAllowlist: !opts.IgnoreAllowlist,
		Operation:        "scan",
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

	// Discover files
	discoveryOptions := filtering.DiscoveryOptions{
		Recursive:      opts.Recursive,
		IncludePattern: opts.IncludePattern,
		ExcludePattern: opts.ExcludePattern,
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

	// Create processing configuration
	processingConfig := config.ToProcessingConfig(profile)
	h.logger.Debug(ctx, "Processing configuration created", "config", processingConfig)

	// Create emoji patterns
	patterns := detector.DefaultEmojiPatterns()
	h.logger.Debug(ctx, "Emoji patterns created", "unicode_ranges", len(patterns.UnicodeRanges))

	// Process files
	h.logger.Info(ctx, "Starting file processing", "total_files", len(filePaths))
	results := processor.ProcessFiles(filePaths, patterns, processingConfig)
	h.logger.Info(ctx, "File processing completed", "total_results", len(results))

	// Filter results through allowlist if configured
	if shouldUseAllowlist {
		h.logger.Debug(ctx, "Applying allowlist filtering to results")
		results = h.filterResultsThroughAllowlist(ctx, results, emojiAllowlist)
		h.logger.Debug(ctx, "Allowlist filtering completed")
	}

	// Display results
	if err := h.displayResults(ctx, results, opts, time.Since(startTime)); err != nil {
		h.logger.Error(ctx, "Failed to display results", "error", err)
		return fmt.Errorf("failed to display results: %w", err)
	}

	// Check threshold for linting
	if opts.Threshold > 0 {
		totalEmojis := h.countTotalEmojis(results)
		if totalEmojis > opts.Threshold {
			h.logger.Error(ctx, "Emoji threshold exceeded",
				"threshold", opts.Threshold,
				"found", totalEmojis)
			h.ui.Error(ctx, "Emoji threshold exceeded: found %d emojis, threshold is %d", totalEmojis, opts.Threshold)
			return fmt.Errorf("%w: found %d emojis (threshold %d)", ErrEmojiThresholdExceeded, totalEmojis, opts.Threshold)
		}
	}

	h.logger.Info(ctx, "Scan operation completed successfully")
	return nil
}

// filterResultsThroughAllowlist filters detection results through the allowlist.
func (h *ScanHandler) filterResultsThroughAllowlist(ctx context.Context, results []types.ProcessResult, allowlist *allowlist.Allowlist) []types.ProcessResult {
	filtered := make([]types.ProcessResult, 0, len(results))

	for _, result := range results {
		if result.Error != nil {
			// Keep error results as-is
			filtered = append(filtered, result)
			continue
		}

		// Filter detections through allowlist
		filteredEmojis := make([]types.EmojiMatch, 0)
		for _, emoji := range result.DetectionResult.Emojis {
			if !allowlist.IsAllowed(emoji.Emoji) {
				filteredEmojis = append(filteredEmojis, emoji)
			}
		}

		// Update result with filtered detections
		newResult := result
		newResult.DetectionResult.Emojis = filteredEmojis
		newResult.DetectionResult.TotalCount = len(filteredEmojis)

		// Recompute unique count
		uniq := make(map[string]struct{}, len(filteredEmojis))
		for _, e := range filteredEmojis {
			uniq[e.Emoji] = struct{}{}
		}
		newResult.DetectionResult.UniqueCount = len(uniq)

		filtered = append(filtered, newResult)
	}

	return filtered
}

// displayResults displays the scan results based on the output options.
func (h *ScanHandler) displayResults(ctx context.Context, results []types.ProcessResult, opts *ScanOptions, duration time.Duration) error {
	h.logger.Debug(ctx, "Displaying scan results", "total_results", len(results), "format", opts.Format)

	// Count totals
	totalFiles := len(results)
	totalEmojis := h.countTotalEmojis(results)
	filesWithEmojis := 0
	errorCount := 0

	for _, result := range results {
		if result.Error != nil {
			errorCount++
			h.logger.Error(ctx, "File processing error", "file", result.FilePath, "error", result.Error)
		} else if result.DetectionResult.TotalCount > 0 {
			filesWithEmojis++
			h.logger.Info(ctx, "File contains emojis",
				"file", result.FilePath,
				"emoji_count", result.DetectionResult.TotalCount)
		}
	}

	// Display summary
	if opts.CountOnly {
		h.ui.Result(ctx, "Total emojis found: %d", totalEmojis)
	} else {
		h.ui.Result(ctx, "Scanned %d files, found %d emojis in %d files (%d errors)",
			totalFiles, totalEmojis, filesWithEmojis, errorCount)

		// Show detailed results if not count-only
		for _, result := range results {
			if result.Error != nil {
				h.ui.Error(ctx, "Error processing %s: %v", result.FilePath, result.Error)
			} else if result.DetectionResult.TotalCount > 0 {
				h.ui.Info(ctx, "%s: %d emojis found", result.FilePath, result.DetectionResult.TotalCount)
			}
		}
	}

	// Show stats if requested
	if opts.Stats {
		h.ui.Info(ctx, "Processing time: %v", duration)
		secs := duration.Seconds()
		fps := 0.0
		if secs > 0 {
			fps = float64(totalFiles) / secs
		}
		h.ui.Info(ctx, "Files per second: %.2f", fps)
	}

	return nil
}

// countTotalEmojis counts the total number of emojis across all results.
func (h *ScanHandler) countTotalEmojis(results []types.ProcessResult) int {
	total := 0
	for _, result := range results {
		if result.Error == nil {
			total += result.DetectionResult.TotalCount
		}
	}
	return total
}
