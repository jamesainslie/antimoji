// Package cli provides the setup-lint command implementation for automated linting configuration.
package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/antimoji/antimoji/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// SetupLintOptions holds the options for the setup-lint command.
type SetupLintOptions struct {
	Mode              string // zero-tolerance, allow-list, or permissive
	OutputDir         string
	PreCommitConfig   bool
	GolangCIConfig    bool
	AllowedEmojis     []string
	Force             bool
	SkipPreCommitHook bool
}

// LintMode represents the different linting modes available.
type LintMode string

const (
	ZeroToleranceMode LintMode = "zero-tolerance"
	AllowListMode     LintMode = "allow-list"
	PermissiveMode    LintMode = "permissive"
)

// NewSetupLintCommand creates the setup-lint command.
func NewSetupLintCommand() *cobra.Command {
	opts := &SetupLintOptions{}

	cmd := &cobra.Command{
		Use:   "setup-lint [flags] [path]",
		Short: "Automatically setup linting configuration for emoji detection",
		Long: `Setup automated linting configuration with pre-commit hooks and golangci-lint integration.

This command configures antimoji for automated emoji linting in your development workflow.
It supports three different modes:

Linting Modes:
  zero-tolerance - Disallows ALL emojis in source code (strictest)
  allow-list     - Allows only specific emojis (1-2 common ones by default)
  permissive     - Allows emojis but warns about excessive usage

The command will:
- Generate appropriate .antimoji.yaml configuration
- Update .pre-commit-config.yaml with antimoji hooks
- Configure .golangci.yml for emoji linting integration
- Setup pre-commit hooks for automated emoji cleaning

Examples:
  antimoji setup-lint --mode=zero-tolerance    # Strict: no emojis allowed
  antimoji setup-lint --mode=allow-list        # Allow  and  only
  antimoji setup-lint --mode=allow-list --allowed-emojis=","  # Custom allowlist
  antimoji setup-lint --mode=permissive        # Lenient with warnings
  antimoji setup-lint --force                  # Overwrite existing configs
  antimoji setup-lint --skip-precommit         # Skip pre-commit hook setup`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSetupLint(cmd, args, opts)
		},
	}

	// Add setup-lint specific flags
	cmd.Flags().StringVar(&opts.Mode, "mode", "zero-tolerance", "linting mode (zero-tolerance, allow-list, permissive)")
	cmd.Flags().StringVar(&opts.OutputDir, "output-dir", ".", "output directory for configuration files")
	cmd.Flags().BoolVar(&opts.PreCommitConfig, "precommit", true, "generate/update .pre-commit-config.yaml")
	cmd.Flags().BoolVar(&opts.GolangCIConfig, "golangci", true, "generate/update .golangci.yml")
	cmd.Flags().StringSliceVar(&opts.AllowedEmojis, "allowed-emojis", []string{"", ""}, "emojis to allow in allow-list mode")
	cmd.Flags().BoolVar(&opts.Force, "force", false, "overwrite existing configuration files")
	cmd.Flags().BoolVar(&opts.SkipPreCommitHook, "skip-precommit", false, "skip pre-commit hook installation")

	return cmd
}

// runSetupLint executes the setup-lint command logic.
func runSetupLint(cmd *cobra.Command, args []string, opts *SetupLintOptions) error {
	// Determine target directory
	targetDir := "."
	if len(args) > 0 {
		targetDir = args[0]
	}
	if opts.OutputDir != "." {
		targetDir = opts.OutputDir
	}

	// Validate target directory
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		return fmt.Errorf("target directory does not exist: %s", targetDir)
	}

	// Validate linting mode
	mode := LintMode(opts.Mode)
	if !isValidLintMode(mode) {
		return fmt.Errorf("invalid linting mode: %s (must be: zero-tolerance, allow-list, or permissive)", opts.Mode)
	}

	if !quiet {
		fmt.Printf(" Setting up antimoji linting configuration...\n")
		fmt.Printf(" Target directory: %s\n", targetDir)
		fmt.Printf(" Linting mode: %s\n", opts.Mode)
	}

	// Generate antimoji configuration
	if err := generateAntimojiConfig(targetDir, mode, opts); err != nil {
		return fmt.Errorf("failed to generate antimoji configuration: %w", err)
	}

	// Update pre-commit configuration
	if opts.PreCommitConfig {
		if err := updatePreCommitConfig(targetDir, mode, opts); err != nil {
			return fmt.Errorf("failed to update pre-commit configuration: %w", err)
		}
	}

	// Update golangci-lint configuration
	if opts.GolangCIConfig {
		if err := updateGolangCIConfig(targetDir, mode, opts); err != nil {
			return fmt.Errorf("failed to update golangci-lint configuration: %w", err)
		}
	}

	// Install pre-commit hooks if requested
	if !opts.SkipPreCommitHook {
		if err := installPreCommitHooks(targetDir); err != nil {
			if !quiet {
				fmt.Printf("  Warning: Failed to install pre-commit hooks: %v\n", err)
				fmt.Printf(" You can install them manually with: pre-commit install\n")
			}
		}
	}

	if !quiet {
		printSetupSummary(mode, opts)
	}

	return nil
}

// generateAntimojiConfig creates the .antimoji.yaml configuration file.
func generateAntimojiConfig(targetDir string, mode LintMode, opts *SetupLintOptions) error {
	configPath := filepath.Join(targetDir, ".antimoji.yaml")

	// Check if file exists and force flag
	if _, err := os.Stat(configPath); err == nil && !opts.Force {
		return fmt.Errorf("configuration file already exists: %s (use --force to overwrite)", configPath)
	}

	// Generate configuration based on mode
	cfg := generateConfigForMode(mode, opts)

	// Write configuration to file
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}

	if !quiet {
		fmt.Printf("Generated antimoji configuration: %s\n", configPath)
	}

	return nil
}

// generateConfigForMode creates configuration based on the selected linting mode.
func generateConfigForMode(mode LintMode, opts *SetupLintOptions) config.Config {
	baseConfig := config.DefaultConfig()

	switch mode {
	case ZeroToleranceMode:
		return generateZeroToleranceConfig(baseConfig)
	case AllowListMode:
		return generateAllowListConfig(baseConfig, opts.AllowedEmojis)
	case PermissiveMode:
		return generatePermissiveConfig(baseConfig)
	default:
		return generateZeroToleranceConfig(baseConfig)
	}
}

// generateZeroToleranceConfig creates a strict configuration that disallows all emojis.
func generateZeroToleranceConfig(base config.Config) config.Config {
	// Create ci-lint profile for zero tolerance
	ciLintProfile := config.Profile{
		// File processing
		Recursive:      true,
		FollowSymlinks: false,
		BackupFiles:    false,

		// Emoji detection - detect everything
		UnicodeEmojis:  true,
		TextEmoticons:  true,
		CustomPatterns: []string{"", "", "", "", "", ":warning:", ":check:", ":x:"},

		// Zero tolerance - empty allowlist
		EmojiAllowlist: []string{},
		FileIgnoreList: []string{
			"*.min.js", "*.min.css", "vendor/**/*", "node_modules/**/*",
			".git/**/*", "**/*.generated.*", "**/*.pb.go", "**/wire_gen.go",
			"README.md", "CHANGELOG.md", "*.md", "docs/**/*",
		},
		DirectoryIgnoreList: []string{
			".git", "node_modules", "vendor", "dist", "build", "docs",
		},

		// Strict CI/CD settings
		FailOnFound:       true,
		MaxEmojiThreshold: 0, // Zero tolerance
		ExitCodeOnFound:   1,

		// File filters - focus on source code
		IncludePatterns: []string{
			"*.go", "*.js", "*.ts", "*.jsx", "*.tsx", "*.py", "*.rb",
			"*.java", "*.c", "*.cpp", "*.h", "*.hpp", "*.rs", "*.php",
			"*.swift", "*.kt", "*.scala",
		},
		ExcludePatterns: []string{
			"vendor/*", "node_modules/*", ".git/*", "dist/*", "build/*",
			"*_test.go", "*/test/*", "*/tests/*", "*/testdata/*", "*/fixtures/*",
			"*.md", "docs/*",
		},

		// Performance
		MaxWorkers:  0,
		BufferSize:  64 * 1024,
		MaxFileSize: 100 * 1024 * 1024,

		// Output
		OutputFormat:  "table",
		ShowProgress:  false,
		ColoredOutput: true,
	}

	base.Profiles["ci-lint"] = ciLintProfile
	base.Profiles["zero-tolerance"] = ciLintProfile // Alias

	return base
}

// generateAllowListConfig creates a configuration with a limited emoji allowlist.
func generateAllowListConfig(base config.Config, allowedEmojis []string) config.Config {
	// Create allow-list profile
	allowListProfile := config.Profile{
		// File processing
		Recursive:      true,
		FollowSymlinks: false,
		BackupFiles:    false,

		// Emoji detection
		UnicodeEmojis:  true,
		TextEmoticons:  true,
		CustomPatterns: []string{"", "", "", "", "", ":warning:", ":check:", ":x:"},

		// Limited allowlist
		EmojiAllowlist: allowedEmojis,
		FileIgnoreList: []string{
			"*.min.js", "*.min.css", "vendor/**/*", "node_modules/**/*",
			".git/**/*", "**/*.generated.*", "**/*.pb.go", "**/wire_gen.go",
			"README.md", "CHANGELOG.md", "*.md", "docs/**/*",
		},
		DirectoryIgnoreList: []string{
			".git", "node_modules", "vendor", "dist", "build", "docs",
		},

		// Moderate CI/CD settings
		FailOnFound:       true,
		MaxEmojiThreshold: 5, // Allow some emojis but limit excess
		ExitCodeOnFound:   1,

		// File filters
		IncludePatterns: []string{
			"*.go", "*.js", "*.ts", "*.jsx", "*.tsx", "*.py", "*.rb",
			"*.java", "*.c", "*.cpp", "*.h", "*.hpp", "*.rs", "*.php",
			"*.swift", "*.kt", "*.scala",
		},
		ExcludePatterns: []string{
			"vendor/*", "node_modules/*", ".git/*", "dist/*", "build/*",
			"*_test.go", "*/test/*", "*/tests/*", "*/testdata/*", "*/fixtures/*",
			"*.md", "docs/*",
		},

		// Performance
		MaxWorkers:  0,
		BufferSize:  64 * 1024,
		MaxFileSize: 100 * 1024 * 1024,

		// Output
		OutputFormat:  "table",
		ShowProgress:  false,
		ColoredOutput: true,
	}

	base.Profiles["allow-list"] = allowListProfile
	base.Profiles["ci-lint"] = allowListProfile // Use as default CI profile

	return base
}

// generatePermissiveConfig creates a lenient configuration that warns but doesn't fail.
func generatePermissiveConfig(base config.Config) config.Config {
	// Create permissive profile
	permissiveProfile := config.Profile{
		// File processing
		Recursive:      true,
		FollowSymlinks: false,
		BackupFiles:    false,

		// Emoji detection
		UnicodeEmojis:  true,
		TextEmoticons:  true,
		CustomPatterns: []string{"", "", "", "", "", ":warning:", ":check:", ":x:"},

		// Generous allowlist
		EmojiAllowlist: []string{
			"", "", "", "", "", "", "", "", "", "",
			"", "", "", "", "", "", "", "", "", "",
		},
		FileIgnoreList: []string{
			"*.min.js", "*.min.css", "vendor/**/*", "node_modules/**/*",
			".git/**/*", "**/*.generated.*", "**/*.pb.go", "**/wire_gen.go",
		},
		DirectoryIgnoreList: []string{
			".git", "node_modules", "vendor", "dist", "build",
		},

		// Lenient CI/CD settings
		FailOnFound:       false, // Don't fail, just warn
		MaxEmojiThreshold: 20,    // Allow many emojis
		ExitCodeOnFound:   0,     // Don't exit with error

		// File filters
		IncludePatterns: []string{
			"*.go", "*.js", "*.ts", "*.jsx", "*.tsx", "*.py", "*.rb",
			"*.java", "*.c", "*.cpp", "*.h", "*.hpp", "*.rs", "*.php",
			"*.swift", "*.kt", "*.scala",
		},
		ExcludePatterns: []string{
			"vendor/*", "node_modules/*", ".git/*", "dist/*", "build/*",
		},

		// Performance
		MaxWorkers:  0,
		BufferSize:  64 * 1024,
		MaxFileSize: 100 * 1024 * 1024,

		// Output
		OutputFormat:  "table",
		ShowProgress:  false,
		ColoredOutput: true,
	}

	base.Profiles["permissive"] = permissiveProfile
	base.Profiles["ci-lint"] = permissiveProfile // Use as default CI profile

	return base
}

// updatePreCommitConfig updates or creates .pre-commit-config.yaml.
func updatePreCommitConfig(targetDir string, mode LintMode, opts *SetupLintOptions) error {
	configPath := filepath.Join(targetDir, ".pre-commit-config.yaml")

	// Generate pre-commit configuration based on mode
	preCommitConfig := generatePreCommitConfigForMode(mode, targetDir)

	// Check if file exists
	if _, err := os.Stat(configPath); err == nil && !opts.Force {
		if !quiet {
			fmt.Printf("Pre-commit config already exists: %s (use --force to overwrite)\n", configPath)
		}
		return nil
	}

	// Write configuration
	if err := os.WriteFile(configPath, []byte(preCommitConfig), 0644); err != nil {
		return fmt.Errorf("failed to write pre-commit configuration: %w", err)
	}

	// Validate the generated configuration
	if err := validateConfiguration(configPath); err != nil {
		return fmt.Errorf("generated configuration is invalid: %w", err)
	}

	if !quiet {
		fmt.Printf("Updated pre-commit configuration: %s\n", configPath)
	}

	return nil
}

// generatePreCommitConfigForMode creates pre-commit configuration based on linting mode.
func generatePreCommitConfigForMode(mode LintMode, targetDir string) string {
	// Detect if antimoji is globally installed or needs local build
	antimojiCmd := detectAntimojiCommand()

	hookBehavior := ""
	switch mode {
	case ZeroToleranceMode:
		hookBehavior = fmt.Sprintf(`entry: %s scan --config=.antimoji.yaml --threshold=0 --quiet
        description: Strict emoji linting - zero tolerance for emojis in source code`, antimojiCmd)
	case AllowListMode:
		hookBehavior = fmt.Sprintf(`entry: %s scan --config=.antimoji.yaml --threshold=5 --quiet
        description: Allow-list emoji linting - only specific emojis allowed`, antimojiCmd)
	case PermissiveMode:
		hookBehavior = fmt.Sprintf(`entry: %s scan --config=.antimoji.yaml --threshold=20 --quiet
        description: Permissive emoji linting - warns about excessive emoji usage`, antimojiCmd)
	}

	// Determine if we need a build hook (only for local builds)
	buildHookSection := ""
	if antimojiCmd == "bin/antimoji" {
		buildHookSection = `
      # Build antimoji before running hooks
      - id: build-antimoji
        name: Build Antimoji Binary
        description: Build antimoji binary for linting hooks
        entry: make build
        language: system
        files: \.(go)$
        pass_filenames: false
        require_serial: true
        stages: [pre-commit, pre-push]
`
	}

	// Conditionally add Go hooks if go.mod exists
	goHooksSection := ""
	if hasGoModule(targetDir) {
		goHooksSection = `  # Go-specific hooks
  - repo: https://github.com/dnephin/pre-commit-golang
    rev: v0.5.1
    hooks:
      - id: go-fmt
      - id: go-mod-tidy

`
	}

	return fmt.Sprintf(`# Pre-commit configuration for Antimoji project
# Generated by: antimoji setup-lint --mode=%s

repos:
  # Standard pre-commit hooks
  - repo: https://github.com/pre-commit/pre-commit-hooks
    rev: v6.0.0
    hooks:
      - id: trailing-whitespace
        exclude: \.md$
      - id: end-of-file-fixer
        exclude: \.md$
      - id: check-yaml
        args: [--allow-multiple-documents]
      - id: check-added-large-files
      - id: check-merge-conflict

%s
  # Local antimoji hooks
  - repo: local
    hooks:%s
      # Antimoji emoji linting
      - id: antimoji-lint
        name: Antimoji Emoji Linter (%s)
        %s
        language: system
        files: \.(go|js|ts|jsx|tsx|py|rb|java|c|cpp|h|hpp|rs|php|swift|kt|scala)$
        exclude: |
          (?x)^(
            .*_test\.go|
            .*/test/.*|
            .*/tests/.*|
            .*/testdata/.*|
            .*/fixtures/.*|
            .*/mocks/.*|
            vendor/.*|
            dist/.*|
            bin/.*|
            .*\.md$|
            docs/.*
          )$
        pass_filenames: true
        require_serial: false
`, mode, goHooksSection, buildHookSection, mode, hookBehavior)
}

// updateGolangCIConfig updates or creates .golangci.yml.
func updateGolangCIConfig(targetDir string, mode LintMode, opts *SetupLintOptions) error {
	configPath := filepath.Join(targetDir, ".golangci.yml")

	// Read existing configuration if it exists
	var existingConfig strings.Builder
	if data, err := os.ReadFile(configPath); err == nil && !opts.Force {
		existingConfig.Write(data)
		existingConfig.WriteString("\n")
	}

	// Add antimoji linting configuration
	antimojiConfig := generateGolangCIConfigForMode(mode)

	// If file doesn't exist or force flag is set, write complete config
	if _, err := os.Stat(configPath); os.IsNotExist(err) || opts.Force {
		fullConfig := fmt.Sprintf(`# golangci-lint configuration with Antimoji integration
# Generated by: antimoji setup-lint --mode=%s

version: "2"

run:
  timeout: 5m
  go: '1.21'

linters:
  enable:
    - errcheck      # Check for unchecked errors
    - govet         # Report suspicious constructs
    - ineffassign   # Detect ineffectual assignments
    - staticcheck   # Advanced static analysis
    - unused        # Find unused code
    - misspell      # Finds commonly misspelled English words

issues:
  max-issues-per-linter: 0
  max-same-issues: 0

%s`, mode, antimojiConfig)

		if err := os.WriteFile(configPath, []byte(fullConfig), 0644); err != nil {
			return fmt.Errorf("failed to write golangci-lint configuration: %w", err)
		}
	} else {
		// Append antimoji configuration to existing file
		existingConfig.WriteString(antimojiConfig)
		if err := os.WriteFile(configPath, []byte(existingConfig.String()), 0644); err != nil {
			return fmt.Errorf("failed to update golangci-lint configuration: %w", err)
		}
	}

	if !quiet {
		fmt.Printf(" Updated golangci-lint configuration: %s\n", configPath)
	}

	return nil
}

// generateGolangCIConfigForMode creates golangci-lint configuration for antimoji.
func generateGolangCIConfigForMode(mode LintMode) string {
	antimojiCmd := detectAntimojiCommand()

	// Convert relative path to absolute for golangci-lint
	if antimojiCmd == "bin/antimoji" {
		antimojiCmd = "./bin/antimoji"
	}

	return fmt.Sprintf(`
# Antimoji emoji linting integration
linters-settings:
  custom:
    antimoji:
      path: %s
      description: "Emoji detection and linting"
      original-url: github.com/antimoji/antimoji
      settings:
        config: .antimoji.yaml
        mode: %s

# Enable custom antimoji linter
linters:
  enable:
    - antimoji
`, antimojiCmd, string(mode))
}

// installPreCommitHooks attempts to install pre-commit hooks.
func installPreCommitHooks(targetDir string) error {
	// Check if pre-commit is available
	if _, err := exec.LookPath("pre-commit"); err != nil {
		return fmt.Errorf("pre-commit not found in PATH")
	}

	// Install hooks
	cmd := exec.Command("pre-commit", "install")
	cmd.Dir = targetDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to install pre-commit hooks: %w\nOutput: %s", err, output)
	}

	if !quiet {
		fmt.Printf(" Installed pre-commit hooks\n")
	}

	return nil
}

// printSetupSummary prints a summary of the setup process.
func printSetupSummary(mode LintMode, opts *SetupLintOptions) {
	fmt.Printf("\nAntimoji linting setup complete!\n\n")

	fmt.Printf("Configuration Summary:\n")
	fmt.Printf("  • Linting mode: %s\n", mode)

	switch mode {
	case ZeroToleranceMode:
		fmt.Printf("  • Policy: Zero tolerance - NO emojis allowed in source code\n")
		fmt.Printf("  • Threshold: 0 emojis\n")
		fmt.Printf("  • Behavior: Fails on any emoji detection\n")
	case AllowListMode:
		fmt.Printf("  • Policy: Allow-list - Only specific emojis allowed\n")
		fmt.Printf("  • Allowed emojis: %s\n", strings.Join(opts.AllowedEmojis, ", "))
		fmt.Printf("  • Threshold: 5 emojis maximum\n")
		fmt.Printf("  • Behavior: Fails on non-allowlisted or excessive emojis\n")
	case PermissiveMode:
		fmt.Printf("  • Policy: Permissive - Warns about excessive emoji usage\n")
		fmt.Printf("  • Threshold: 20 emojis maximum\n")
		fmt.Printf("  • Behavior: Warns but doesn't fail builds\n")
	}

	fmt.Printf("\nGenerated Files:\n")
	fmt.Printf("  • .antimoji.yaml - Antimoji configuration\n")
	if opts.PreCommitConfig {
		fmt.Printf("  • .pre-commit-config.yaml - Pre-commit hooks configuration\n")
	}
	if opts.GolangCIConfig {
		fmt.Printf("  • .golangci.yml - GolangCI-Lint integration\n")
	}

	fmt.Printf("\nNext Steps:\n")
	fmt.Printf("  1. Review generated configuration files\n")
	fmt.Printf("  2. Install pre-commit: pip install pre-commit\n")
	fmt.Printf("  3. Install hooks: pre-commit install\n")
	fmt.Printf("  4. Test setup: pre-commit run --all-files\n")
	fmt.Printf("  5. Commit your changes: git add . && git commit -m \"Setup antimoji linting\"\n")

	fmt.Printf("\nUsage Examples:\n")
	fmt.Printf("  • Run manual scan: antimoji scan --config .antimoji.yaml .\n")
	fmt.Printf("  • Run with profile: antimoji scan --profile %s .\n", mode)
	fmt.Printf("  • Clean emojis: antimoji clean --config .antimoji.yaml --in-place .\n")
}

// isValidLintMode checks if the provided mode is valid.
func isValidLintMode(mode LintMode) bool {
	switch mode {
	case ZeroToleranceMode, AllowListMode, PermissiveMode:
		return true
	default:
		return false
	}
}

// detectAntimojiCommand determines the best way to invoke antimoji.
// Returns "antimoji" if globally installed, "bin/antimoji" if local build is preferred.
func detectAntimojiCommand() string {
	// Check if antimoji is globally available
	if _, err := exec.LookPath("antimoji"); err == nil {
		// Global antimoji is available, prefer it
		return "antimoji"
	}

	// Check if we're in the antimoji source directory with build capability
	if _, err := os.Stat("Makefile"); err == nil {
		// We have a Makefile, assume local build is possible
		return "bin/antimoji"
	}

	// Default to global antimoji (user will need to install it)
	return "antimoji"
}

// hasGoModule checks if the target directory has a go.mod file.
func hasGoModule(targetDir string) bool {
	_, err := os.Stat(filepath.Join(targetDir, "go.mod"))
	return err == nil
}

// validateConfiguration validates the generated pre-commit configuration.
func validateConfiguration(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read configuration file: %w", err)
	}

	// Basic YAML validation
	var config interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("invalid YAML syntax: %w", err)
	}

	// Check for common issues in the configuration
	configStr := string(data)
	if strings.Contains(configStr, "--profile=") {
		return fmt.Errorf("configuration contains invalid --profile flag")
	}
	if strings.Contains(configStr, "--fail-on-found") {
		return fmt.Errorf("configuration contains invalid --fail-on-found flag")
	}

	return nil
}
