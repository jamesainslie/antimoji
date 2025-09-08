// Package cli provides command implementations for the Antimoji CLI.
package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/antimoji/antimoji/internal/config"
	"github.com/antimoji/antimoji/internal/core/allowlist"
	"github.com/antimoji/antimoji/internal/core/detector"
	"github.com/antimoji/antimoji/internal/core/processor"
	"github.com/antimoji/antimoji/internal/infra/filtering"
	"github.com/antimoji/antimoji/internal/observability/logging"
	"github.com/antimoji/antimoji/internal/types"
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

// NewScanCommand creates the scan command.
func NewScanCommand() *cobra.Command {
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
  antimoji scan --format json .     # Output results as JSON
  antimoji scan --count-only .       # Show only emoji counts
  antimoji scan --stats .            # Include performance statistics`,
		Args: cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScan(cmd, args, opts)
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

// runScan executes the scan command logic.
func runScan(cmd *cobra.Command, args []string, opts *ScanOptions) error {
	startTime := time.Now()

	// If no paths provided, use current directory
	if len(args) == 0 {
		args = []string{"."}
	}

	// Load configuration
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

	// Convert to processing config
	processingConfig := config.ToProcessingConfig(profile)

	// Override config with command-line flags
	// TODO: Implement command-line flag overrides for configuration
	// For now, the flags are handled directly in the scan options

	// Discover files to process using unified filtering engine
	discoveryOpts := filtering.DiscoveryOptions{
		Recursive:      opts.Recursive,
		IncludePattern: opts.IncludePattern,
		ExcludePattern: opts.ExcludePattern,
	}
	filePaths, err := filtering.DiscoverFiles(args, discoveryOpts, profile)
	if err != nil {
		return fmt.Errorf("failed to discover files: %w", err)
	}

	ctx := context.Background()
	logging.Info(ctx, "File discovery completed",
		"files_found", len(filePaths),
		"operation", "scan")

	// Create emoji patterns
	patterns := detector.DefaultEmojiPatterns()

	// Create allowlist using unified processing logic
	allowlistOpts := allowlist.ProcessingOptions{
		IgnoreAllowlist:  opts.IgnoreAllowlist,
		RespectAllowlist: true, // scan always respects allowlist by default unless ignored
		Operation:        "scan",
	}
	emojiAllowlist, err := allowlist.CreateAllowlistForProcessing(ctx, profile, allowlistOpts)
	if err != nil {
		return fmt.Errorf("failed to create allowlist: %w", err)
	}

	// Process files (use concurrent processing if multiple files and workers configured)
	var results []types.ProcessResult
	if opts.Workers > 0 && len(filePaths) > 1 {
		results = processor.ProcessFilesConcurrently(filePaths, patterns, processingConfig, opts.Workers)
	} else {
		results = processor.ProcessFiles(filePaths, patterns, processingConfig)
	}

	// Apply allowlist filtering to results
	if emojiAllowlist != nil {
		for i, result := range results {
			if result.Error == nil && result.DetectionResult.Success {
				filteredResult := allowlist.ApplyAllowlist(result.DetectionResult, emojiAllowlist)
				if filteredResult.IsOk() {
					results[i].DetectionResult = filteredResult.Unwrap()
				}
			}
		}
	}

	// Display results (unless in quiet mode)
	if !quiet {
		if err := displayResults(results, opts, time.Since(startTime)); err != nil {
			return fmt.Errorf("failed to display results: %w", err)
		}
	}

	// Check threshold for linting mode (including zero threshold for strict linting)
	if cmd.Flags().Changed("threshold") {
		totalEmojis := 0
		for _, result := range results {
			totalEmojis += result.DetectionResult.TotalCount
		}
		if totalEmojis > opts.Threshold {
			logging.Error(ctx, "Emoji threshold exceeded",
				"emojis_found", totalEmojis,
				"threshold", opts.Threshold,
				"operation", "scan")
			return fmt.Errorf("emoji threshold exceeded: found %d emojis (limit: %d)", totalEmojis, opts.Threshold)
		}
	}

	return nil
}

// discoverFiles discovers files to process based on arguments and options.
// DEPRECATED: Use discoverFilesWithEngine instead. Kept for backward compatibility.
func discoverFiles(args []string, opts *ScanOptions, profile config.Profile) ([]string, error) {
	var filePaths []string

	for _, arg := range args {
		stat, err := os.Stat(arg)
		if err != nil {
			// For non-existent files, we'll let the processor handle the error
			// but still include them in the list so they show up in results
			filePaths = append(filePaths, arg)
			continue
		}

		if stat.IsDir() {
			if opts.Recursive {
				err := filepath.WalkDir(arg, func(path string, d os.DirEntry, err error) error {
					if err != nil {
						return err
					}

					if d.IsDir() {
						// Check if directory should be ignored
						if shouldIgnoreDirectory(path, profile.DirectoryIgnoreList) {
							return filepath.SkipDir
						}
						return nil
					}

					// Check if file should be included
					if shouldIncludeFile(path, opts, profile) {
						filePaths = append(filePaths, path)
					}

					return nil
				})
				if err != nil {
					return nil, err
				}
			} else {
				return nil, fmt.Errorf("directory %s requires --recursive flag", arg)
			}
		} else {
			// Single file
			if shouldIncludeFile(arg, opts, profile) {
				filePaths = append(filePaths, arg)
			}
		}
	}

	return filePaths, nil
}

// shouldIgnoreDirectory checks if a directory should be ignored.
func shouldIgnoreDirectory(dirPath string, ignoreList []string) bool {
	dirName := filepath.Base(dirPath)

	for _, pattern := range ignoreList {
		if matched, _ := filepath.Match(pattern, dirName); matched {
			return true
		}
		if matched, _ := filepath.Match(pattern, dirPath); matched {
			return true
		}
	}

	return false
}

// shouldIncludeFile checks if a file should be included in processing.
func shouldIncludeFile(filePath string, opts *ScanOptions, profile config.Profile) bool {
	fileName := filepath.Base(filePath)

	// Check exclude patterns first
	for _, pattern := range profile.ExcludePatterns {
		if matched, _ := filepath.Match(pattern, fileName); matched {
			return false
		}
	}

	// Check file ignore list
	for _, pattern := range profile.FileIgnoreList {
		if matched, _ := filepath.Match(pattern, fileName); matched {
			return false
		}
	}

	// Check include patterns - only apply filtering if patterns are explicitly defined
	// If no include patterns are specified, include all files (unless excluded above)
	if len(profile.IncludePatterns) > 0 {
		included := false
		for _, pattern := range profile.IncludePatterns {
			if matched, _ := filepath.Match(pattern, fileName); matched {
				included = true
				break
			}
		}
		if !included {
			return false
		}
	}
	// If IncludePatterns is empty, default to including all files

	// Additional command-line filters
	if opts.IncludePattern != "" {
		if matched, _ := filepath.Match(opts.IncludePattern, fileName); !matched {
			return false
		}
	}

	if opts.ExcludePattern != "" {
		if matched, _ := filepath.Match(opts.ExcludePattern, fileName); matched {
			return false
		}
	}

	return true
}

// displayResults displays the scan results in the requested format.
func displayResults(results []types.ProcessResult, opts *ScanOptions, duration time.Duration) error {
	switch opts.Format {
	case "table":
		return displayTableFormat(results, opts, duration)
	case "json":
		return displayJSONFormat(results, opts, duration)
	case "csv":
		return displayCSVFormat(results, opts, duration)
	default:
		return fmt.Errorf("unsupported output format: %s", opts.Format)
	}
}

// displayTableFormat displays results in table format.
func displayTableFormat(results []types.ProcessResult, opts *ScanOptions, duration time.Duration) error {
	if opts.CountOnly {
		totalEmojis := 0
		totalFiles := 0
		for _, result := range results {
			if result.Error == nil {
				totalFiles++
				totalEmojis += result.DetectionResult.TotalCount
			}
		}
		fmt.Printf("Total: %d emojis in %d files\n", totalEmojis, totalFiles)
		return nil
	}

	// Display detailed results
	fmt.Printf("%-50s %-8s %-8s %-10s\n", "File", "Emojis", "Unique", "Status")
	fmt.Printf("%s\n", strings.Repeat("-", 80))

	totalEmojis := 0
	processedFiles := 0
	errorFiles := 0

	for _, result := range results {
		if result.Error != nil {
			fmt.Printf("%-50s %-8s %-8s %-10s\n",
				truncateString(result.FilePath, 50),
				"-", "-", "ERROR")
			errorFiles++
			continue
		}

		status := "OK"
		if !result.DetectionResult.Success {
			status = "SKIPPED"
		}

		fmt.Printf("%-50s %-8d %-8d %-10s\n",
			truncateString(result.FilePath, 50),
			result.DetectionResult.TotalCount,
			result.DetectionResult.UniqueCount,
			status)

		if result.DetectionResult.Success {
			totalEmojis += result.DetectionResult.TotalCount
			processedFiles++
		}
	}

	fmt.Printf("%s\n", strings.Repeat("-", 80))
	fmt.Printf("Summary: %d emojis in %d files (%d errors)\n", totalEmojis, processedFiles, errorFiles)

	if opts.Stats {
		fmt.Printf("Processing time: %v\n", duration)
		fmt.Printf("Average: %.2f files/second\n", float64(len(results))/duration.Seconds())
	}

	return nil
}

// displayJSONFormat displays results in JSON format.
func displayJSONFormat(results []types.ProcessResult, _ *ScanOptions, duration time.Duration) error {
	// For now, just print a simple JSON structure
	// In a real implementation, we'd use encoding/json
	fmt.Printf(`{"results": %d, "duration": "%v"}`, len(results), duration)
	fmt.Println()
	return nil
}

// displayCSVFormat displays results in CSV format.
func displayCSVFormat(results []types.ProcessResult, _ *ScanOptions, _ time.Duration) error {
	fmt.Println("file_path,emoji_count,unique_count,status")

	for _, result := range results {
		status := "ok"
		if result.Error != nil {
			status = "error"
		} else if !result.DetectionResult.Success {
			status = "skipped"
		}

		fmt.Printf("%s,%d,%d,%s\n",
			result.FilePath,
			result.DetectionResult.TotalCount,
			result.DetectionResult.UniqueCount,
			status)
	}

	return nil
}

// truncateString truncates a string to the specified length.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return "..."
	}
	return s[:maxLen-3] + "..."
}
