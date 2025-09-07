// Package cli provides the generate command implementation for allowlist generation.
package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/antimoji/antimoji/internal/config"
	"github.com/antimoji/antimoji/internal/core/detector"
	"github.com/antimoji/antimoji/internal/core/processor"
	"github.com/antimoji/antimoji/internal/observability/logging"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
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

// EmojiUsageAnalysis represents the analysis of emoji usage in the project.
type EmojiUsageAnalysis struct {
	EmojisByCategory map[string][]EmojiUsage `json:"emojis_by_category" yaml:"emojis_by_category"`
	EmojisByFile     map[string][]EmojiUsage `json:"emojis_by_file" yaml:"emojis_by_file"`
	FilesByType      map[string][]string     `json:"files_by_type" yaml:"files_by_type"`
	Statistics       UsageStatistics         `json:"statistics" yaml:"statistics"`
}

// EmojiUsage represents usage information for a specific emoji.
type EmojiUsage struct {
	Emoji     string   `json:"emoji" yaml:"emoji"`
	Count     int      `json:"count" yaml:"count"`
	Files     []string `json:"files" yaml:"files"`
	Category  string   `json:"category" yaml:"category"`
	FileTypes []string `json:"file_types" yaml:"file_types"`
}

// UsageStatistics provides overall statistics about emoji usage.
type UsageStatistics struct {
	TotalEmojis       int `json:"total_emojis" yaml:"total_emojis"`
	UniqueEmojis      int `json:"unique_emojis" yaml:"unique_emojis"`
	FilesWithEmojis   int `json:"files_with_emojis" yaml:"files_with_emojis"`
	TotalFilesScanned int `json:"total_files_scanned" yaml:"total_files_scanned"`
}

// AllowlistConfig represents the generated allowlist configuration.
type AllowlistConfig struct {
	Version  string             `yaml:"version"`
	Profiles map[string]Profile `yaml:"profiles"`
}

// Profile represents a generated profile configuration.
type Profile struct {
	EmojiAllowlist      []string `yaml:"emoji_allowlist"`
	FileIgnoreList      []string `yaml:"file_ignore_list,omitempty"`
	DirectoryIgnoreList []string `yaml:"directory_ignore_list,omitempty"`
	Description         string   `yaml:"# description,omitempty"`
}

// NewGenerateCommand creates the generate command.
func NewGenerateCommand() *cobra.Command {
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
		Args: cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenerate(cmd, args, opts)
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

// runGenerate executes the generate command logic.
func runGenerate(_ *cobra.Command, args []string, opts *GenerateOptions) error {
	startTime := time.Now()

	// If no paths provided, use current directory
	if len(args) == 0 {
		args = []string{"."}
	}

	ctx := context.Background()
	logging.Info(ctx, "Starting emoji analysis for allowlist generation",
		"operation", "generate",
		"type", opts.Type,
		"paths", args)

	// Analyze emoji usage in the project
	analysis, err := analyzeEmojiUsage(args, opts)
	if err != nil {
		return fmt.Errorf("failed to analyze emoji usage: %w", err)
	}

	logging.Info(ctx, "Emoji analysis completed",
		"operation", "generate",
		"unique_emojis", analysis.Statistics.UniqueEmojis,
		"files_with_emojis", analysis.Statistics.FilesWithEmojis,
		"total_files_scanned", analysis.Statistics.TotalFilesScanned)

	// Generate allowlist configuration based on type
	allowlistConfig, err := generateAllowlistConfig(analysis, opts)
	if err != nil {
		return fmt.Errorf("failed to generate allowlist config: %w", err)
	}

	// Output the configuration
	if err := outputConfiguration(ctx, allowlistConfig, opts, time.Since(startTime)); err != nil {
		return fmt.Errorf("failed to output configuration: %w", err)
	}

	return nil
}

// analyzeEmojiUsage analyzes emoji usage across the project.
func analyzeEmojiUsage(paths []string, opts *GenerateOptions) (*EmojiUsageAnalysis, error) {
	// Create a temporary config for scanning everything
	scanProfile := config.Profile{
		Recursive:           opts.Recursive,
		UnicodeEmojis:       true,
		TextEmoticons:       true,
		CustomPatterns:      []string{"", "", "", "", ":warning:", ":check:", ":cross:"},
		IncludePatterns:     []string{"*"}, // Include all file types for analysis
		ExcludePatterns:     []string{},    // Don't exclude anything initially
		DirectoryIgnoreList: []string{".git", "vendor", "node_modules", "dist", "bin"},
		MaxWorkers:          0,
		BufferSize:          64 * 1024,
		MaxFileSize:         100 * 1024 * 1024,
	}

	// Discover all files
	filePaths, err := discoverAllFiles(paths, scanProfile)
	if err != nil {
		return nil, err
	}

	// Create emoji patterns
	patterns := detector.DefaultEmojiPatterns()
	processingConfig := config.ToProcessingConfig(scanProfile)

	// Process files to find all emojis
	results := processor.ProcessFiles(filePaths, patterns, processingConfig)

	// Analyze the results
	analysis := &EmojiUsageAnalysis{
		EmojisByCategory: make(map[string][]EmojiUsage),
		EmojisByFile:     make(map[string][]EmojiUsage),
		FilesByType:      make(map[string][]string),
		Statistics:       UsageStatistics{},
	}

	emojiCounts := make(map[string]*EmojiUsage)
	fileTypeMap := make(map[string][]string)

	for _, result := range results {
		if result.Error != nil {
			continue
		}

		analysis.Statistics.TotalFilesScanned++

		if result.DetectionResult.TotalCount > 0 {
			analysis.Statistics.FilesWithEmojis++
			analysis.Statistics.TotalEmojis += result.DetectionResult.TotalCount

			// Categorize file type
			fileType := categorizeFile(result.FilePath)
			fileTypeMap[fileType] = append(fileTypeMap[fileType], result.FilePath)

			// Track emojis by file
			var fileEmojis []EmojiUsage
			for _, emoji := range result.DetectionResult.Emojis {
				emojiStr := emoji.Emoji

				// Update global emoji count
				if usage, exists := emojiCounts[emojiStr]; exists {
					usage.Count++
					usage.Files = appendUnique(usage.Files, result.FilePath)
					usage.FileTypes = appendUnique(usage.FileTypes, fileType)
				} else {
					emojiCounts[emojiStr] = &EmojiUsage{
						Emoji:     emojiStr,
						Count:     1,
						Files:     []string{result.FilePath},
						Category:  string(emoji.Category),
						FileTypes: []string{fileType},
					}
				}

				fileEmojis = append(fileEmojis, EmojiUsage{
					Emoji:    emojiStr,
					Count:    1,
					Files:    []string{result.FilePath},
					Category: string(emoji.Category),
				})
			}

			if len(fileEmojis) > 0 {
				analysis.EmojisByFile[result.FilePath] = fileEmojis
			}
		}
	}

	// Convert maps to analysis structure
	analysis.Statistics.UniqueEmojis = len(emojiCounts)
	analysis.FilesByType = fileTypeMap

	// Group emojis by category
	for _, usage := range emojiCounts {
		categoryKey := usage.Category
		if categoryKey == "" {
			categoryKey = "unknown"
		}
		analysis.EmojisByCategory[categoryKey] = append(analysis.EmojisByCategory[categoryKey], *usage)
	}

	// Sort emojis by usage count within each category
	for category := range analysis.EmojisByCategory {
		sort.Slice(analysis.EmojisByCategory[category], func(i, j int) bool {
			return analysis.EmojisByCategory[category][i].Count > analysis.EmojisByCategory[category][j].Count
		})
	}

	return analysis, nil
}

// generateAllowlistConfig generates the allowlist configuration based on analysis and type.
func generateAllowlistConfig(analysis *EmojiUsageAnalysis, opts *GenerateOptions) (*AllowlistConfig, error) {
	profileName := opts.Profile
	if profileName == "" {
		profileName = opts.Type
	}

	var allowedEmojis []string
	var fileIgnoreList []string
	var directoryIgnoreList []string
	description := ""

	switch opts.Type {
	case "ci-lint":
		allowedEmojis, fileIgnoreList, directoryIgnoreList = generateCILintAllowlist(analysis, opts)
		description = "CI/CD linting profile - strict but allows necessary emojis for tests and documentation"

	case "dev":
		allowedEmojis, fileIgnoreList, directoryIgnoreList = generateDevAllowlist(analysis, opts)
		description = "Development profile - permissive allowlist for local development"

	case "test-only":
		allowedEmojis, fileIgnoreList, directoryIgnoreList = generateTestOnlyAllowlist(analysis, opts)
		description = "Test-only profile - allows emojis found in test files only"

	case "docs-only":
		allowedEmojis, fileIgnoreList, directoryIgnoreList = generateDocsOnlyAllowlist(analysis, opts)
		description = "Documentation-only profile - allows emojis found in documentation files only"

	case "minimal":
		allowedEmojis, fileIgnoreList, directoryIgnoreList = generateMinimalAllowlist(analysis, opts)
		description = "Minimal profile - allows only frequently used emojis"

	case "full":
		allowedEmojis, fileIgnoreList, directoryIgnoreList = generateFullAllowlist(analysis, opts)
		description = "Full profile - allows all found emojis with comprehensive categorization"

	default:
		return nil, fmt.Errorf("unsupported generation type: %s", opts.Type)
	}

	// Sort the allowlist for consistency
	sort.Strings(allowedEmojis)

	profile := Profile{
		EmojiAllowlist:      allowedEmojis,
		FileIgnoreList:      fileIgnoreList,
		DirectoryIgnoreList: directoryIgnoreList,
		Description:         description,
	}

	config := &AllowlistConfig{
		Version: "0.5.0",
		Profiles: map[string]Profile{
			profileName: profile,
		},
	}

	return config, nil
}

// generateCILintAllowlist generates a strict allowlist suitable for CI/CD linting.
func generateCILintAllowlist(analysis *EmojiUsageAnalysis, opts *GenerateOptions) ([]string, []string, []string) {
	var allowedEmojis []string

	// Include emojis from test files if requested
	if opts.IncludeTests {
		allowedEmojis = append(allowedEmojis, getEmojisFromFileType(analysis, "test")...)
	}

	// Include emojis from documentation if requested
	if opts.IncludeDocs {
		allowedEmojis = append(allowedEmojis, getEmojisFromFileType(analysis, "documentation")...)
		allowedEmojis = append(allowedEmojis, getEmojisFromFileType(analysis, "markdown")...)
	}

	// Include emojis from CI files if requested
	if opts.IncludeCI {
		allowedEmojis = append(allowedEmojis, getEmojisFromFileType(analysis, "ci")...)
		allowedEmojis = append(allowedEmojis, getEmojisFromFileType(analysis, "script")...)
	}

	// Add commonly acceptable emojis for status/documentation
	commonEmojis := []string{"", "", "", "", "", "⭐", "", "", "", "", "", ""}
	allowedEmojis = append(allowedEmojis, commonEmojis...)

	// Filter by minimum usage
	allowedEmojis = filterByMinUsage(allowedEmojis, analysis, opts.MinUsage)

	fileIgnoreList := []string{
		"**/*_test.go", "**/test/**/*", "**/testdata/**/*", "**/fixtures/**/*",
		"README.md", "CHANGELOG.md", ".github/**/*", "scripts/**/*",
		"vendor/**/*", "dist/**/*", "bin/**/*",
	}

	directoryIgnoreList := []string{
		".git", "vendor", "dist", "bin", "test", "tests", "testdata", "fixtures", ".github",
	}

	return removeDuplicates(allowedEmojis), fileIgnoreList, directoryIgnoreList
}

// generateDevAllowlist generates a permissive allowlist for development.
func generateDevAllowlist(analysis *EmojiUsageAnalysis, opts *GenerateOptions) ([]string, []string, []string) {
	var allowedEmojis []string

	// Include all emojis found in the project
	for _, categoryEmojis := range analysis.EmojisByCategory {
		for _, usage := range categoryEmojis {
			if usage.Count >= opts.MinUsage {
				allowedEmojis = append(allowedEmojis, usage.Emoji)
			}
		}
	}

	fileIgnoreList := []string{
		"vendor/**/*", "dist/**/*", "bin/**/*", ".git/**/*",
	}

	directoryIgnoreList := []string{
		".git", "vendor", "dist", "bin",
	}

	return removeDuplicates(allowedEmojis), fileIgnoreList, directoryIgnoreList
}

// generateTestOnlyAllowlist generates allowlist with only test file emojis.
func generateTestOnlyAllowlist(analysis *EmojiUsageAnalysis, opts *GenerateOptions) ([]string, []string, []string) {
	allowedEmojis := getEmojisFromFileType(analysis, "test")
	allowedEmojis = filterByMinUsage(allowedEmojis, analysis, opts.MinUsage)

	fileIgnoreList := []string{
		"**/*_test.go", "**/test/**/*", "**/testdata/**/*", "**/fixtures/**/*",
		"vendor/**/*", "dist/**/*", "bin/**/*",
	}

	directoryIgnoreList := []string{
		".git", "vendor", "dist", "bin", "test", "tests", "testdata", "fixtures",
	}

	return removeDuplicates(allowedEmojis), fileIgnoreList, directoryIgnoreList
}

// generateDocsOnlyAllowlist generates allowlist with only documentation emojis.
func generateDocsOnlyAllowlist(analysis *EmojiUsageAnalysis, opts *GenerateOptions) ([]string, []string, []string) {
	var allowedEmojis []string
	allowedEmojis = append(allowedEmojis, getEmojisFromFileType(analysis, "documentation")...)
	allowedEmojis = append(allowedEmojis, getEmojisFromFileType(analysis, "markdown")...)
	allowedEmojis = filterByMinUsage(allowedEmojis, analysis, opts.MinUsage)

	fileIgnoreList := []string{
		"README.md", "CHANGELOG.md", "**/*.md",
		"vendor/**/*", "dist/**/*", "bin/**/*",
	}

	directoryIgnoreList := []string{
		".git", "vendor", "dist", "bin",
	}

	return removeDuplicates(allowedEmojis), fileIgnoreList, directoryIgnoreList
}

// generateMinimalAllowlist generates a minimal allowlist with most used emojis.
func generateMinimalAllowlist(analysis *EmojiUsageAnalysis, opts *GenerateOptions) ([]string, []string, []string) {
	var allowedEmojis []string

	// Get top used emojis across all categories
	allUsages := make([]EmojiUsage, 0)
	for _, categoryEmojis := range analysis.EmojisByCategory {
		allUsages = append(allUsages, categoryEmojis...)
	}

	// Sort by usage count
	sort.Slice(allUsages, func(i, j int) bool {
		return allUsages[i].Count > allUsages[j].Count
	})

	// Take top emojis or those above minimum usage
	minUsage := opts.MinUsage
	if minUsage < 2 {
		minUsage = 2 // For minimal, require at least 2 uses
	}

	for _, usage := range allUsages {
		if usage.Count >= minUsage {
			allowedEmojis = append(allowedEmojis, usage.Emoji)
		}
	}

	// Limit to top 20 for minimal
	if len(allowedEmojis) > 20 {
		allowedEmojis = allowedEmojis[:20]
	}

	fileIgnoreList := []string{
		"vendor/**/*", "dist/**/*", "bin/**/*",
	}

	directoryIgnoreList := []string{
		".git", "vendor", "dist", "bin",
	}

	return removeDuplicates(allowedEmojis), fileIgnoreList, directoryIgnoreList
}

// generateFullAllowlist generates a comprehensive allowlist with all found emojis.
func generateFullAllowlist(analysis *EmojiUsageAnalysis, opts *GenerateOptions) ([]string, []string, []string) {
	var allowedEmojis []string

	// Include all emojis found in the project
	for _, categoryEmojis := range analysis.EmojisByCategory {
		for _, usage := range categoryEmojis {
			if usage.Count >= opts.MinUsage {
				allowedEmojis = append(allowedEmojis, usage.Emoji)
			}
		}
	}

	fileIgnoreList := []string{
		"vendor/**/*", "dist/**/*", "bin/**/*",
	}

	directoryIgnoreList := []string{
		".git", "vendor", "dist", "bin",
	}

	return removeDuplicates(allowedEmojis), fileIgnoreList, directoryIgnoreList
}

// discoverAllFiles discovers all files for comprehensive analysis.
func discoverAllFiles(paths []string, profile config.Profile) ([]string, error) {
	var filePaths []string

	for _, path := range paths {
		stat, err := os.Stat(path)
		if err != nil {
			return nil, err
		}

		if stat.IsDir() {
			err := filepath.WalkDir(path, func(filePath string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}

				if d.IsDir() {
					// Check if directory should be ignored
					dirName := filepath.Base(filePath)
					for _, ignorePattern := range profile.DirectoryIgnoreList {
						if matched, _ := filepath.Match(ignorePattern, dirName); matched {
							return filepath.SkipDir
						}
					}
					return nil
				}

				// Include all text-based files for analysis
				if isTextFile(filePath) {
					filePaths = append(filePaths, filePath)
				}

				return nil
			})
			if err != nil {
				return nil, err
			}
		} else {
			if isTextFile(path) {
				filePaths = append(filePaths, path)
			}
		}
	}

	return filePaths, nil
}

// categorizeFile categorizes a file based on its path and extension.
func categorizeFile(filePath string) string {
	fileName := filepath.Base(filePath)
	ext := filepath.Ext(fileName)
	dir := filepath.Dir(filePath)

	// Normalize path separators for cross-platform compatibility
	normalizedPath := filepath.ToSlash(filePath)
	normalizedDir := filepath.ToSlash(dir)

	// Check for test files first (most specific)
	if strings.Contains(fileName, "_test.") || strings.Contains(fileName, "test_") ||
		strings.Contains(normalizedDir, "/test/") || strings.Contains(normalizedDir, "/tests/") ||
		strings.Contains(normalizedDir, "/testdata/") || strings.Contains(normalizedDir, "/fixtures/") ||
		strings.HasSuffix(normalizedDir, "/test") || strings.HasSuffix(normalizedDir, "/tests") ||
		strings.HasSuffix(normalizedDir, "/testdata") || strings.HasSuffix(normalizedDir, "/fixtures") ||
		strings.Contains(normalizedPath, "/test/") || strings.Contains(normalizedPath, "/tests/") {
		return "test"
	}

	// Check for CI files
	if strings.Contains(normalizedDir, ".github") || strings.Contains(normalizedDir, "scripts") ||
		(ext == ".yml" || ext == ".yaml") && (strings.Contains(normalizedDir, ".github") || strings.Contains(fileName, "ci")) {
		return "ci"
	}

	// Check for documentation
	if ext == ".md" {
		if strings.Contains(fileName, "README") || strings.Contains(fileName, "CHANGELOG") {
			return "documentation"
		}
		if strings.Contains(dir, "docs") || strings.Contains(dir, "/doc/") {
			return "markdown"
		}
		return "documentation" // Default for .md files
	}

	// Check for config files
	if ext == ".yaml" || ext == ".yml" || ext == ".json" || ext == ".toml" {
		return "config"
	}

	// Check for source code
	sourceExts := map[string]bool{
		".go": true, ".js": true, ".ts": true, ".py": true, ".java": true,
		".c": true, ".cpp": true, ".h": true, ".hpp": true, ".rs": true,
	}
	if sourceExts[ext] {
		return "source"
	}

	return "other"
}

// getEmojisFromFileType extracts emojis used in specific file types.
func getEmojisFromFileType(analysis *EmojiUsageAnalysis, fileType string) []string {
	var emojis []string

	files, exists := analysis.FilesByType[fileType]
	if !exists {
		return emojis
	}

	emojiSet := make(map[string]bool)
	for _, filePath := range files {
		if fileEmojis, exists := analysis.EmojisByFile[filePath]; exists {
			for _, usage := range fileEmojis {
				emojiSet[usage.Emoji] = true
			}
		}
	}

	for emoji := range emojiSet {
		emojis = append(emojis, emoji)
	}

	return emojis
}

// filterByMinUsage filters emojis by minimum usage count.
func filterByMinUsage(emojis []string, analysis *EmojiUsageAnalysis, minUsage int) []string {
	var filtered []string

	for _, emoji := range emojis {
		totalUsage := 0
		for _, categoryEmojis := range analysis.EmojisByCategory {
			for _, usage := range categoryEmojis {
				if usage.Emoji == emoji {
					totalUsage += usage.Count
					break
				}
			}
		}

		if totalUsage >= minUsage {
			filtered = append(filtered, emoji)
		}
	}

	return filtered
}

// outputConfiguration outputs the generated configuration.
func outputConfiguration(ctx context.Context, config *AllowlistConfig, opts *GenerateOptions, duration time.Duration) error {
	var output []byte
	var err error

	switch opts.Format {
	case "yaml":
		output, err = yaml.Marshal(config)
		if err != nil {
			return fmt.Errorf("failed to marshal YAML: %w", err)
		}
	case "json":
		// For JSON, we'd use encoding/json, but for simplicity using YAML for now
		output, err = yaml.Marshal(config)
		if err != nil {
			return fmt.Errorf("failed to marshal configuration: %w", err)
		}
	default:
		return fmt.Errorf("unsupported output format: %s", opts.Format)
	}

	// Add header comment
	header := fmt.Sprintf("# Generated by antimoji generate --type=%s\n# Generated at: %s\n# Analysis duration: %v\n\n",
		opts.Type, time.Now().Format(time.RFC3339), duration)

	output = append([]byte(header), output...)

	// Output to file or stdout
	if opts.Output != "" {
		logging.Info(ctx, "Writing configuration to file",
			"operation", "generate",
			"output_file", opts.Output,
			"format", opts.Format)
		return os.WriteFile(opts.Output, output, 0600)
	}
	fmt.Print(string(output))
	return nil
}

// isTextFile determines if a file is likely a text file based on extension.
func isTextFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	textExtensions := map[string]bool{
		".go": true, ".js": true, ".ts": true, ".jsx": true, ".tsx": true,
		".py": true, ".rb": true, ".java": true, ".c": true, ".cpp": true,
		".h": true, ".hpp": true, ".rs": true, ".php": true, ".swift": true,
		".kt": true, ".scala": true, ".md": true, ".txt": true, ".yaml": true,
		".yml": true, ".json": true, ".toml": true, ".ini": true, ".conf": true,
		".sh": true, ".bash": true, ".zsh": true, ".fish": true, ".ps1": true,
		".html": true, ".htm": true, ".xml": true, ".css": true, ".scss": true,
		".sass": true, ".less": true, ".sql": true, ".dockerfile": true, ".gitignore": true,
	}

	// Also check for files without extension that are likely text
	if ext == "" {
		fileName := strings.ToLower(filepath.Base(filePath))
		textFileNames := map[string]bool{
			"dockerfile": true, "makefile": true, "readme": true, "changelog": true,
			"license": true, "authors": true, "contributors": true, "todo": true,
			".gitignore": true, ".dockerignore": true, ".editorconfig": true,
		}
		return textFileNames[fileName]
	}

	return textExtensions[ext]
}

// appendUnique appends a string to a slice if it's not already present.
func appendUnique(slice []string, item string) []string {
	for _, existing := range slice {
		if existing == item {
			return slice
		}
	}
	return append(slice, item)
}

// removeDuplicates removes duplicate strings from a slice.
func removeDuplicates(slice []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}
