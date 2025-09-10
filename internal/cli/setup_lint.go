// Package cli provides the setup-lint command implementation for automated linting configuration.
package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/antimoji/antimoji/internal/config"
	"github.com/antimoji/antimoji/internal/infra/analysis"
	"github.com/dustin/go-humanize"
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
	Repair            bool
	Review            bool
	Validate          bool
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
- Append antimoji hooks to existing .pre-commit-config.yaml (or create new)
- Configure .golangci.yml for emoji linting integration
- Setup pre-commit hooks for automated emoji cleaning

Behavior with existing configuration files:
- Preserves existing hooks and configuration in .pre-commit-config.yaml
- Detects existing antimoji configuration and prompts for replacement
- Use --force to skip confirmation prompts
- Use --repair to restore missing .antimoji.yaml and .pre-commit-config.yaml antimoji configuration

Examples:
  antimoji setup-lint --mode=zero-tolerance    # Strict: no emojis allowed
  antimoji setup-lint --mode=allow-list        # Allow  and  only
  antimoji setup-lint --mode=allow-list --allowed-emojis=","  # Custom allowlist
  antimoji setup-lint --mode=permissive        # Lenient with warnings
  antimoji setup-lint --force                  # Overwrite existing configs
  antimoji setup-lint --repair                 # Repair missing antimoji configs
  antimoji setup-lint --review                 # Review existing configuration
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
	cmd.Flags().BoolVar(&opts.Repair, "repair", false, "repair missing .antimoji.yaml and .pre-commit-config.yaml antimoji configuration")
	cmd.Flags().BoolVar(&opts.Review, "review", false, "review existing configuration and explain how it will apply")
	cmd.Flags().BoolVar(&opts.Validate, "validate", false, "validate existing configuration and suggest improvements")

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

	// Handle review mode
	if opts.Review {
		return reviewConfiguration(targetDir, opts)
	}

	// Handle validation mode
	if opts.Validate {
		return validateConfigurationFile(targetDir, opts)
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

	// Update golangci-lint configuration (if requested and file doesn't exist)
	if opts.GolangCIConfig {
		if err := ensureBasicGolangCIConfig(targetDir, opts); err != nil {
			return fmt.Errorf("failed to ensure golangci-lint configuration: %w", err)
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
		if opts.Repair {
			printRepairSummary(mode, opts)
		} else {
			printSetupSummary(mode, opts)
		}
	}

	return nil
}

// generateAntimojiConfig creates the .antimoji.yaml configuration file.
func generateAntimojiConfig(targetDir string, mode LintMode, opts *SetupLintOptions) error {
	configPath := filepath.Join(targetDir, ".antimoji.yaml")

	// Check if file exists
	fileExists := true
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fileExists = false
	}

	// Handle different scenarios
	if fileExists {
		if opts.Repair {
			// In repair mode, if file exists, just inform and skip
			if !quiet {
				fmt.Printf(" .antimoji.yaml already exists, skipping\n")
			}
			return nil
		} else if !opts.Force {
			return fmt.Errorf("configuration file already exists: %s (use --force to overwrite)", configPath)
		}
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
		if opts.Repair && !fileExists {
			fmt.Printf(" Repaired missing .antimoji.yaml configuration: %s\n", configPath)
		} else {
			fmt.Printf(" Generated antimoji configuration: %s\n", configPath)
		}
	}

	return nil
}

// generateConfigForMode creates configuration based on the selected linting mode using templates.
func generateConfigForMode(mode LintMode, opts *SetupLintOptions) config.Config {
	baseConfig := config.DefaultConfig()

	// Map setup-lint modes to template names
	templateName := ""
	switch mode {
	case ZeroToleranceMode:
		templateName = "zero-tolerance"
	case AllowListMode:
		templateName = "allow-list"
	case PermissiveMode:
		templateName = "permissive"
	default:
		templateName = "zero-tolerance"
	}

	// Apply template with options
	templateOptions := config.TemplateOptions{
		AllowedEmojis: opts.AllowedEmojis,
		TargetDir:     opts.OutputDir,
		IncludeTests:  false, // setup-lint focuses on source code
		IncludeDocs:   false,
	}

	profile, err := config.GetBuiltInProfile(templateName, templateOptions)
	if err != nil {
		// Fallback to hardcoded generation if template fails
		return generateZeroToleranceConfig(baseConfig)
	}

	// Create config with the template-generated profile
	baseConfig.Profiles[string(mode)] = profile
	baseConfig.Profiles[templateName] = profile // Also add with template name

	return baseConfig
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
		CustomPatterns: []string{"", "", "", "", "", "", "", ":x:"},

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

	base.Profiles["zero-tolerance"] = ciLintProfile
	base.Profiles["zero"] = ciLintProfile // Alias for consistency with examples

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
		CustomPatterns: []string{"", "", "", "", "", "", "", ":x:"},

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
		CustomPatterns: []string{"", "", "", "", "", "", "", ":x:"},

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

	return base
}

// updatePreCommitConfig updates or creates .pre-commit-config.yaml using append-only approach.
func updatePreCommitConfig(targetDir string, mode LintMode, opts *SetupLintOptions) error {
	configPath := filepath.Join(targetDir, ".pre-commit-config.yaml")

	// Check if file exists
	fileExists := true
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fileExists = false
	}

	if !fileExists {
		// Create new file with full configuration
		return createNewPreCommitConfig(configPath, mode, targetDir, opts)
	}

	// File exists - parse and update it
	return updateExistingPreCommitConfig(configPath, mode, targetDir, opts)
}

// PreCommitConfig represents the structure of .pre-commit-config.yaml
type PreCommitConfig struct {
	Repos []PreCommitRepo `yaml:"repos"`
}

// PreCommitRepo represents a repository in pre-commit config
type PreCommitRepo struct {
	Repo  string          `yaml:"repo"`
	Rev   string          `yaml:"rev,omitempty"`
	Hooks []PreCommitHook `yaml:"hooks"`
}

// PreCommitHook represents a hook in pre-commit config
type PreCommitHook struct {
	ID            string   `yaml:"id"`
	Name          string   `yaml:"name,omitempty"`
	Entry         string   `yaml:"entry,omitempty"`
	Args          []string `yaml:"args,omitempty"`
	Description   string   `yaml:"description,omitempty"`
	Language      string   `yaml:"language,omitempty"`
	Files         string   `yaml:"files,omitempty"`
	Exclude       string   `yaml:"exclude,omitempty"`
	PassFilenames bool     `yaml:"pass_filenames,omitempty"`
	RequireSerial bool     `yaml:"require_serial,omitempty"`
	Stages        []string `yaml:"stages,omitempty"`
}

// createNewPreCommitConfig creates a new .pre-commit-config.yaml file
func createNewPreCommitConfig(configPath string, mode LintMode, targetDir string, opts *SetupLintOptions) error {
	// Generate full configuration
	preCommitConfig := generatePreCommitConfigForMode(mode, targetDir)

	// Write configuration
	if err := os.WriteFile(configPath, []byte(preCommitConfig), 0644); err != nil {
		return fmt.Errorf("failed to write pre-commit configuration: %w", err)
	}

	if !quiet {
		fmt.Printf(" Created new pre-commit configuration: %s\n", configPath)
	}

	return nil
}

// updateExistingPreCommitConfig updates an existing .pre-commit-config.yaml file
func updateExistingPreCommitConfig(configPath string, mode LintMode, targetDir string, opts *SetupLintOptions) error {
	// Read existing configuration
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read existing pre-commit configuration: %w", err)
	}

	var config PreCommitConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse existing pre-commit configuration: %w", err)
	}

	// Check for existing antimoji configuration
	hasAntimoji, antimojiRepoIndex := hasAntimojiConfig(&config)

	if hasAntimoji {
		if opts.Repair {
			// In repair mode, if antimoji config exists, just inform and skip
			if !quiet {
				fmt.Printf(" .pre-commit-config.yaml antimoji configuration already exists, skipping\n")
			}
			return nil
		} else if !opts.Force {
			// Prompt user for confirmation
			if !promptForReplacement() {
				if !quiet {
					fmt.Printf("ℹ  Skipped updating antimoji configuration in %s\n", configPath)
				}
				return nil
			}
		}
	} else if opts.Repair {
		// In repair mode, if no antimoji config exists, add it
		if !quiet {
			fmt.Printf(" Adding missing antimoji configuration to .pre-commit-config.yaml\n")
		}
	}

	// Remove existing antimoji configuration if present
	if hasAntimoji {
		config.Repos = append(config.Repos[:antimojiRepoIndex], config.Repos[antimojiRepoIndex+1:]...)
		if !quiet {
			fmt.Printf(" Removed existing antimoji configuration\n")
		}
	}

	// Add new antimoji configuration
	antimojiRepo := generateAntimojiRepo(mode, targetDir)
	config.Repos = append(config.Repos, antimojiRepo)

	// Write updated configuration back
	updatedData, err := yaml.Marshal(&config)
	if err != nil {
		return fmt.Errorf("failed to marshal updated configuration: %w", err)
	}

	if err := os.WriteFile(configPath, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write updated pre-commit configuration: %w", err)
	}

	if !quiet {
		if opts.Repair {
			fmt.Printf(" Repaired missing antimoji configuration in .pre-commit-config.yaml: %s\n", configPath)
		} else {
			fmt.Printf(" Updated pre-commit configuration: %s\n", configPath)
		}
	}

	return nil
}

// hasAntimojiConfig checks if the configuration already contains antimoji hooks
func hasAntimojiConfig(config *PreCommitConfig) (bool, int) {
	antimojiHookIDs := []string{"antimoji-clean", "antimoji-verify", "antimoji-check", "build-antimoji"}

	for i, repo := range config.Repos {
		if repo.Repo == "local" {
			for _, hook := range repo.Hooks {
				for _, antimojiID := range antimojiHookIDs {
					if hook.ID == antimojiID {
						return true, i
					}
				}
			}
		}
	}
	return false, -1
}

// promptForReplacement prompts the user for confirmation to replace existing antimoji config
func promptForReplacement() bool {
	if quiet {
		return false // Don't prompt in quiet mode
	}

	fmt.Print("  Existing antimoji configuration found in .pre-commit-config.yaml\n")
	fmt.Print("   Do you want to replace it with the new configuration? [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

// generateAntimojiRepo creates the antimoji repository configuration for pre-commit
func generateAntimojiRepo(mode LintMode, targetDir string) PreCommitRepo {
	antimojiCmd := detectAntimojiCommand()
	hooks := []PreCommitHook{}

	// Add build hook if needed
	if antimojiCmd == "bin/antimoji" {
		buildHook := PreCommitHook{
			ID:            "build-antimoji",
			Name:          "Build Antimoji Binary",
			Description:   "Build antimoji binary for linting hooks",
			Entry:         "make build",
			Language:      "system",
			Files:         `\.(go)$`,
			PassFilenames: false,
			RequireSerial: true,
			Stages:        []string{"pre-commit", "pre-push"},
		}
		hooks = append(hooks, buildHook)
	}

	// Add mode-specific hooks
	switch mode {
	case ZeroToleranceMode:
		cleanHook := PreCommitHook{
			ID:            "antimoji-clean",
			Name:          "Auto-clean Emojis (zero-tolerance)",
			Entry:         antimojiCmd,
			Args:          []string{"clean", "--config=.antimoji.yaml", "--profile=zero-tolerance", "--in-place", "--quiet"},
			Description:   "Remove all emojis from source code files",
			Language:      "system",
			PassFilenames: true,
			RequireSerial: true,
		}
		verifyHook := PreCommitHook{
			ID:            "antimoji-verify",
			Name:          "Zero-Tolerance Emoji Verification",
			Entry:         antimojiCmd,
			Args:          []string{"scan", "--config=.antimoji.yaml", "--profile=zero-tolerance", "--threshold=0", "--quiet"},
			Description:   "Strict verification - no emojis allowed in source code",
			Language:      "system",
			PassFilenames: true,
			RequireSerial: true,
		}
		hooks = append(hooks, cleanHook, verifyHook)

	case AllowListMode:
		cleanHook := PreCommitHook{
			ID:            "antimoji-clean",
			Name:          "Auto-clean Non-allowed Emojis",
			Entry:         antimojiCmd,
			Args:          []string{"clean", "--config=.antimoji.yaml", "--profile=allow-list", "--in-place", "--quiet"},
			Description:   "Remove emojis not in the allowlist",
			Language:      "system",
			PassFilenames: true,
			RequireSerial: true,
		}
		verifyHook := PreCommitHook{
			ID:            "antimoji-verify",
			Name:          "Allow-list Emoji Verification",
			Entry:         antimojiCmd,
			Args:          []string{"scan", "--config=.antimoji.yaml", "--profile=allow-list", "--threshold=5", "--quiet"},
			Description:   "Allow-list verification - only specific emojis allowed",
			Language:      "system",
			PassFilenames: true,
			RequireSerial: true,
		}
		hooks = append(hooks, cleanHook, verifyHook)

	case PermissiveMode:
		checkHook := PreCommitHook{
			ID:            "antimoji-check",
			Name:          "Permissive Emoji Check",
			Entry:         antimojiCmd,
			Args:          []string{"scan", "--config=.antimoji.yaml", "--profile=permissive", "--threshold=20", "--quiet"},
			Description:   "Permissive emoji check - warns about excessive usage",
			Language:      "system",
			PassFilenames: true,
			RequireSerial: false,
		}
		hooks = append(hooks, checkHook)
	}

	// Add file filtering to all hooks
	filePattern := `\.(go|js|ts|jsx|tsx|py|rb|java|c|cpp|h|hpp|rs|php|swift|kt|scala)$`
	excludePattern := `(?x)^(.*_test\.go|.*/test/.*|.*/tests/.*|.*/testdata/.*|.*/fixtures/.*|.*/mocks/.*|vendor/.*|dist/.*|bin/.*|.*\.md$|docs/.*|\.antimoji\.yaml$)$`

	for i := range hooks {
		if hooks[i].ID != "build-antimoji" {
			hooks[i].Files = filePattern
			hooks[i].Exclude = excludePattern
		}
	}

	return PreCommitRepo{
		Repo:  "local",
		Hooks: hooks,
	}
}

// generatePreCommitConfigForMode creates pre-commit configuration based on linting mode.
func generatePreCommitConfigForMode(mode LintMode, targetDir string) string {
	// Detect if antimoji is globally installed or needs local build
	antimojiCmd := detectAntimojiCommand()

	// Generate improved two-step hook configuration (clean + verify) to avoid the
	// "0 modified but still finds emojis" bug that was reported
	cleanHook := ""
	verifyHook := ""

	switch mode {
	case ZeroToleranceMode:
		// Use consistent zero-tolerance profile for both clean and verify steps
		cleanHook = fmt.Sprintf(`      # Step 1: Auto-clean emojis (zero-tolerance)
      - id: antimoji-clean
        name: "Auto-clean Emojis (zero-tolerance)"
        entry: %s
        args: [clean, --config=.antimoji.yaml, --profile=zero-tolerance, --in-place, --quiet]
        description: Remove all emojis from source code files
        language: system
        pass_filenames: true
        require_serial: true`, antimojiCmd)

		verifyHook = fmt.Sprintf(`      # Step 2: Verify no emojis remain (zero-tolerance)
      - id: antimoji-verify
        name: "Zero-Tolerance Emoji Verification"
        entry: %s
        args: [scan, --config=.antimoji.yaml, --profile=zero-tolerance, --threshold=0, --quiet]
        description: Strict verification - no emojis allowed in source code
        language: system
        pass_filenames: true
        require_serial: true`, antimojiCmd)

	case AllowListMode:
		// Use consistent allow-list profile for both clean and verify steps
		cleanHook = fmt.Sprintf(`      # Step 1: Clean non-allowed emojis
      - id: antimoji-clean
        name: "Auto-clean Non-allowed Emojis"
        entry: %s
        args: [clean, --config=.antimoji.yaml, --profile=allow-list, --in-place, --quiet]
        description: Remove emojis not in the allowlist
        language: system
        pass_filenames: true
        require_serial: true`, antimojiCmd)

		verifyHook = fmt.Sprintf(`      # Step 2: Verify only allowed emojis remain
      - id: antimoji-verify
        name: "Allow-list Emoji Verification"
        entry: %s
        args: [scan, --config=.antimoji.yaml, --profile=allow-list, --threshold=5, --quiet]
        description: Allow-list verification - only specific emojis allowed
        language: system
        pass_filenames: true
        require_serial: true`, antimojiCmd)

	case PermissiveMode:
		// For permissive mode, only use scan (no cleaning needed)
		verifyHook = fmt.Sprintf(`      # Permissive emoji check (warnings only)
      - id: antimoji-check
        name: "Permissive Emoji Check"
        entry: %s
        args: [scan, --config=.antimoji.yaml, --profile=permissive, --threshold=20, --quiet]
        description: Permissive emoji check - warns about excessive usage
        language: system
        pass_filenames: true
        require_serial: false`, antimojiCmd)
	}

	// Combine hooks based on mode
	var hookBehavior string
	if mode == PermissiveMode {
		hookBehavior = verifyHook
	} else {
		hookBehavior = cleanHook + "\n\n" + verifyHook
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
# Uses improved two-step workflow to prevent "0 modified but still finds emojis" issues

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
  # Local antimoji hooks - improved workflow
  - repo: local
    hooks:%s
%s
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
            docs/.*|
            \.antimoji\.yaml$
          )$
`, mode, goHooksSection, buildHookSection, hookBehavior)
}

// updateGolangCIConfig updates or creates .golangci.yml.
func ensureBasicGolangCIConfig(targetDir string, opts *SetupLintOptions) error {
	configPath := filepath.Join(targetDir, ".golangci.yml")

	// If file doesn't exist or force flag is set, write complete config
	if _, err := os.Stat(configPath); os.IsNotExist(err) || opts.Force {
		fullConfig := `# golangci-lint configuration
# Generated by: antimoji setup-lint
# NOTE: Antimoji integration handled via pre-commit hooks

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
    - misspell      # Finds commonly misspelled English words

issues:
  max-issues-per-linter: 0
  max-same-issues: 0

# NOTE: Emoji linting is handled by antimoji via pre-commit hooks
# Run 'antimoji scan .' manually or use pre-commit hooks for emoji detection
`

		if err := os.WriteFile(configPath, []byte(fullConfig), 0644); err != nil {
			return fmt.Errorf("failed to write golangci-lint configuration: %w", err)
		}
	} else {
		// File exists and force not set - leave it alone
		if !quiet {
			fmt.Printf(" golangci-lint configuration already exists: %s\n", configPath)
			fmt.Printf("   (Use --force to overwrite, or configure antimoji via pre-commit hooks)\n")
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

// printRepairSummary prints a summary of the repair process.
func printRepairSummary(mode LintMode, opts *SetupLintOptions) {
	fmt.Printf("\nAntimoji configuration repair complete!\n\n")

	fmt.Printf("Repair Summary:\n")
	fmt.Printf("  • Linting mode: %s\n", mode)
	fmt.Printf("  • Operation: Repaired missing antimoji configuration files\n")

	switch mode {
	case ZeroToleranceMode:
		fmt.Printf("  • Policy: Zero tolerance - NO emojis allowed in source code\n")
	case AllowListMode:
		fmt.Printf("  • Policy: Allow-list - Only specific emojis allowed\n")
		fmt.Printf("  • Allowed emojis: %s\n", strings.Join(opts.AllowedEmojis, ", "))
	case PermissiveMode:
		fmt.Printf("  • Policy: Permissive - Warns about excessive emoji usage\n")
	}

	fmt.Printf("\nRepaired Files:\n")
	fmt.Printf("  • .antimoji.yaml - Antimoji configuration (if missing)\n")
	if opts.PreCommitConfig {
		fmt.Printf("  • .pre-commit-config.yaml - Pre-commit antimoji hooks (if missing)\n")
	}
	if opts.GolangCIConfig {
		fmt.Printf("  • .golangci.yml - GolangCI-Lint integration (if missing)\n")
	}

	fmt.Printf("\nNext Steps:\n")
	fmt.Printf("  1. Review repaired configuration files\n")
	fmt.Printf("  2. Ensure pre-commit is installed: pip install pre-commit\n")
	fmt.Printf("  3. Install/update hooks: pre-commit install\n")
	fmt.Printf("  4. Test repair: pre-commit run --all-files\n")
	fmt.Printf("  5. Commit your changes: git add . && git commit -m \"Repair antimoji configuration\"\n")

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

// ReviewData holds data for the configuration review
type ReviewData struct {
	Mode            string
	Policy          string
	Threshold       string
	AllowedEmojis   []string
	FileCount       int
	CurrentEmojis   int
	PreCommitStatus string
	GolangCIStatus  string
	Impact          string
	Behavior        string
}

// reviewConfiguration analyzes existing configuration and provides natural language explanation
func reviewConfiguration(targetDir string, opts *SetupLintOptions) error {
	if !quiet {
		fmt.Printf("Antimoji Configuration Review\n")
		fmt.Printf("=============================\n\n")
	}

	// Analyze existing configuration files
	review, err := analyzeExistingConfiguration(targetDir)
	if err != nil {
		return fmt.Errorf("failed to analyze configuration: %w", err)
	}

	// Generate and display the review
	return displayConfigurationReview(review)
}

// analyzeExistingConfiguration analyzes all antimoji-related configuration files using the enhanced analyzer.
func analyzeExistingConfiguration(targetDir string) (*ReviewData, error) {
	review := &ReviewData{}

	// Check for .antimoji.yaml
	antimojiPath := filepath.Join(targetDir, ".antimoji.yaml")
	if _, err := os.Stat(antimojiPath); err == nil {
		if err := analyzeAntimojiConfigEnhanced(antimojiPath, targetDir, review); err != nil {
			return nil, err
		}
	} else {
		review.Mode = "not configured"
		review.Policy = "No antimoji configuration found"
	}

	// Check for .pre-commit-config.yaml
	preCommitPath := filepath.Join(targetDir, ".pre-commit-config.yaml")
	if _, err := os.Stat(preCommitPath); err == nil {
		analyzePreCommitHooks(preCommitPath, review)
	} else {
		review.PreCommitStatus = "Not configured"
	}

	// Check for .golangci.yml
	golangCIPath := filepath.Join(targetDir, ".golangci.yml")
	if _, err := os.Stat(golangCIPath); err == nil {
		analyzeGolangCIIntegration(golangCIPath, review)
	} else {
		review.GolangCIStatus = "Not configured"
	}

	return review, nil
}

// analyzeAntimojiConfigEnhanced uses the new analyzer for enhanced configuration analysis.
func analyzeAntimojiConfigEnhanced(configPath, targetDir string, review *ReviewData) error {
	// Load configuration
	configResult := config.LoadConfig(configPath)
	if configResult.IsErr() {
		return configResult.Error()
	}
	cfg := configResult.Unwrap()

	// Find the primary profile to analyze
	var primaryProfile config.Profile
	var primaryName string

	// Priority order for profile selection
	profilePriority := []string{"zero-tolerance", "ci-lint", "allow-list", "permissive", "default"}

	for _, name := range profilePriority {
		if profile, exists := cfg.Profiles[name]; exists {
			primaryProfile = profile
			primaryName = name
			break
		}
	}

	// If none found, take the first available profile
	if primaryName == "" {
		for name, profile := range cfg.Profiles {
			primaryProfile = profile
			primaryName = name
			break
		}
	}

	if primaryName == "" {
		return fmt.Errorf("no profiles found in configuration")
	}

	// Create analyzer and perform deep analysis
	analyzer := analysis.NewConfigAnalyzer(primaryProfile, targetDir)
	analysisResult := analyzer.AnalyzeConfiguration()

	// Map analysis results to review data
	review.Mode = primaryName
	review.Policy = analysisResult.PolicyAnalysis.Description
	review.Threshold = fmt.Sprintf("%d emojis maximum", analysisResult.PolicyAnalysis.Threshold)
	review.AllowedEmojis = analysisResult.PolicyAnalysis.AllowedEmojis
	review.FileCount = analysisResult.ImpactAnalysis.FilesToScan
	review.CurrentEmojis = analysisResult.ImpactAnalysis.CurrentEmojis

	// Add profile count information
	profileCount := len(cfg.Profiles)
	if profileCount > 1 {
		review.Policy += fmt.Sprintf(" (%d total profiles available)", profileCount)
	}

	return nil
}

// analyzeAntimojiConfig parses .antimoji.yaml and extracts key information
func analyzeAntimojiConfig(configPath string, review *ReviewData) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	var cfg config.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return err
	}

	// Analyze all profiles to understand the configuration
	profileCount := len(cfg.Profiles)
	if profileCount == 0 {
		review.Mode = "no profiles"
		review.Policy = "No profiles configured"
		review.Threshold = "No limits set"
		return nil
	}

	// Find the most restrictive or commonly used profile
	var primaryProfile config.Profile
	var primaryName string

	// Priority order: zero-tolerance > ci-lint > allow-list > permissive > default > first available
	profilePriority := []string{"zero-tolerance", "ci-lint", "allow-list", "permissive", "default"}

	for _, name := range profilePriority {
		if profile, exists := cfg.Profiles[name]; exists {
			primaryProfile = profile
			primaryName = name
			break
		}
	}

	// If none of the priority profiles exist, take the first one
	if primaryName == "" {
		for name, profile := range cfg.Profiles {
			primaryProfile = profile
			primaryName = name
			break
		}
	}

	// Analyze the primary profile in detail
	review.Mode = primaryName
	review.AllowedEmojis = primaryProfile.EmojiAllowlist

	// Determine policy based on actual configuration
	if primaryProfile.MaxEmojiThreshold == 0 && len(primaryProfile.EmojiAllowlist) == 0 {
		review.Policy = "Zero tolerance - NO emojis allowed anywhere"
		review.Threshold = "0 emojis maximum"
	} else if len(primaryProfile.EmojiAllowlist) > 0 && primaryProfile.MaxEmojiThreshold <= 10 {
		review.Policy = fmt.Sprintf("Allow-list mode - Only %d specific emojis allowed", len(primaryProfile.EmojiAllowlist))
		review.Threshold = fmt.Sprintf("%d emojis maximum", primaryProfile.MaxEmojiThreshold)
	} else if primaryProfile.MaxEmojiThreshold > 15 {
		review.Policy = "Permissive mode - Allows emojis with high threshold"
		review.Threshold = fmt.Sprintf("%d emojis maximum", primaryProfile.MaxEmojiThreshold)
	} else {
		review.Policy = fmt.Sprintf("Custom policy with %d allowed emojis", len(primaryProfile.EmojiAllowlist))
		review.Threshold = fmt.Sprintf("%d emojis maximum", primaryProfile.MaxEmojiThreshold)
	}

	// Add information about multiple profiles
	if profileCount > 1 {
		review.Policy += fmt.Sprintf(" (%d total profiles available)", profileCount)
	}

	return nil
}

// analyzePreCommitHooks checks for antimoji hooks in .pre-commit-config.yaml
func analyzePreCommitHooks(configPath string, review *ReviewData) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		review.PreCommitStatus = "Error reading file"
		return
	}

	var config PreCommitConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		review.PreCommitStatus = "Error parsing YAML"
		return
	}

	hasAntimoji, _ := hasAntimojiConfig(&config)
	if hasAntimoji {
		review.PreCommitStatus = "Configured with antimoji hooks"
	} else {
		review.PreCommitStatus = "No antimoji hooks found"
	}
}

// analyzeGolangCIIntegration checks for antimoji in .golangci.yml
func analyzeGolangCIIntegration(configPath string, review *ReviewData) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		review.GolangCIStatus = "Error reading file"
		return
	}

	configStr := string(data)
	if strings.Contains(configStr, "antimoji") {
		review.GolangCIStatus = "Antimoji linter enabled"
	} else {
		review.GolangCIStatus = "No antimoji integration"
	}
}

// analyzeCodebaseImpact analyzes the target directory to estimate impact
func analyzeCodebaseImpact(targetDir string, review *ReviewData) {
	fileCount := 0
	emojiCount := 0

	// Walk through the directory and count relevant files
	_ = filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			// Skip common directories we don't want to scan
			name := info.Name()
			if name == ".git" || name == "node_modules" || name == "vendor" || name == "dist" {
				return filepath.SkipDir
			}
			return nil
		}

		// Count files that would be scanned
		if isRelevantFile(path) {
			fileCount++
			// Simple emoji detection (this is a basic implementation)
			if content, err := os.ReadFile(path); err == nil {
				emojiCount += countEmojisInContent(string(content))
			}
		}

		return nil
	})

	review.FileCount = fileCount
	review.CurrentEmojis = emojiCount
}

// isRelevantFile determines if a file would be scanned by antimoji
func isRelevantFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	relevantExts := []string{".go", ".js", ".ts", ".jsx", ".tsx", ".py", ".rb", ".java", ".c", ".cpp", ".h", ".hpp", ".rs", ".php", ".swift", ".kt", ".scala"}

	for _, relevantExt := range relevantExts {
		if ext == relevantExt {
			return true
		}
	}
	return false
}

// countEmojisInContent provides a basic emoji count (simplified implementation)
func countEmojisInContent(content string) int {
	count := 0
	// This is a simplified implementation - in reality, we'd use the full emoji detection logic
	commonEmojis := []string{"", "", "", "", "", "", "", "", "", "", "", "", "", ""}

	for _, emoji := range commonEmojis {
		count += strings.Count(content, emoji)
	}

	return count
}

// displayConfigurationReview shows the configuration review using templates
func displayConfigurationReview(review *ReviewData) error {
	// Template for the review output (no emojis as requested)
	const reviewTemplate = `Configuration Summary:
  Mode: {{.Mode}}{{if eq .Mode "zero-tolerance"}} (strictest){{end}}
  Policy: {{.Policy}}
  {{- if .Threshold}}
  Threshold: {{.Threshold}}
  {{- end}}
  {{- if .AllowedEmojis}}
  Allowed emojis: {{len .AllowedEmojis}} configured
  {{- end}}

Scope Analysis:
  Files to scan: {{humanizeInt .FileCount}} files
  Current emojis found: {{pluralize .CurrentEmojis "emoji" "emojis"}}

Configuration Status:
  .antimoji.yaml: {{if ne .Mode "not configured"}}Present{{else}}Missing{{end}}
  Pre-commit hooks: {{.PreCommitStatus}}
  GolangCI-Lint: {{.GolangCIStatus}}

{{if ne .Mode "not configured"}}Behavior Explanation:
{{explainBehavior .Mode .CurrentEmojis .PreCommitStatus}}{{else}}
Setup Required:
  Run 'antimoji setup-lint --mode=<mode>' to configure antimoji linting.
  Available modes: zero-tolerance, allow-list, permissive
{{end}}

Usage Examples:
  Review configuration: antimoji setup-lint --review
  Manual scan: antimoji scan .
  Clean emojis: antimoji clean --in-place .
`

	// Create template with custom functions
	tmpl := template.Must(template.New("review").Funcs(template.FuncMap{
		"humanizeInt": func(n int) string {
			return humanize.Comma(int64(n))
		},
		"pluralize": func(count int, singular, plural string) string {
			if count == 1 {
				return strconv.Itoa(count) + " " + singular
			}
			return strconv.Itoa(count) + " " + plural
		},
		"explainBehavior": func(mode string, emojiCount int, preCommitStatus string) string {
			hasPreCommit := strings.Contains(preCommitStatus, "Configured with antimoji hooks")
			commitBehavior := "On commit: "
			if !hasPreCommit {
				commitBehavior = "On commit: No pre-commit hooks configured - manual intervention required"
			}
			switch mode {
			case "zero-tolerance":
				if hasPreCommit {
					if emojiCount > 0 {
						return fmt.Sprintf("  %sWill automatically remove all %d emojis, then verify none remain\n  On CI: Will fail build if any emojis are detected\n  Manual usage: Strict scanning and cleaning available", commitBehavior, emojiCount)
					}
					return fmt.Sprintf("  %sWill prevent any emojis from being added\n  On CI: Will fail build if any emojis are detected\n  Manual usage: Strict scanning maintains emoji-free codebase", commitBehavior)
				} else {
					return fmt.Sprintf("  %s\n  On CI: Will fail build if any emojis are detected\n  Manual usage: Run 'antimoji clean --in-place .' to remove %d emojis", commitBehavior, emojiCount)
				}
			case "ci-lint":
				if hasPreCommit {
					if emojiCount > 0 {
						return fmt.Sprintf("  %sWill remove non-allowed emojis from %d total found\n  On CI: Will fail build if non-allowed emojis detected\n  Manual usage: CI-focused linting with allowlist", commitBehavior, emojiCount)
					}
					return fmt.Sprintf("  %sWill enforce allowlist rules\n  On CI: Will fail build if non-allowed emojis detected\n  Manual usage: CI-focused linting with allowlist", commitBehavior)
				} else {
					return fmt.Sprintf("  %s\n  On CI: Will fail build if non-allowed emojis detected\n  Manual usage: Run 'antimoji clean --in-place .' to remove non-allowed emojis from %d found", commitBehavior, emojiCount)
				}
			case "allow-list":
				if hasPreCommit {
					return fmt.Sprintf("  %sWill remove non-allowed emojis and enforce limits\n  On CI: Will fail build if non-allowed or excessive emojis found\n  Manual usage: Selective emoji management", commitBehavior)
				} else {
					return fmt.Sprintf("  %s\n  On CI: Will fail build if non-allowed or excessive emojis found\n  Manual usage: Run 'antimoji clean --in-place .' to manage emojis manually", commitBehavior)
				}
			case "permissive":
				if hasPreCommit {
					return fmt.Sprintf("  %sWill warn about excessive emoji usage\n  On CI: Will warn but not fail builds\n  Manual usage: Lenient emoji monitoring", commitBehavior)
				} else {
					return fmt.Sprintf("  %s\n  On CI: Will warn but not fail builds\n  Manual usage: Run 'antimoji scan .' to monitor emoji usage", commitBehavior)
				}
			default:
				if hasPreCommit {
					if emojiCount > 0 {
						return fmt.Sprintf("  %sCustom behavior based on profile settings\n  On CI: Custom rules apply to %d emojis found\n  Manual usage: Profile-specific emoji management", commitBehavior, emojiCount)
					}
					return fmt.Sprintf("  %sCustom behavior based on profile settings\n  On CI: Custom rules apply\n  Manual usage: Profile-specific emoji management", commitBehavior)
				} else {
					return fmt.Sprintf("  %s\n  On CI: Custom rules apply to %d emojis found\n  Manual usage: Use 'antimoji scan/clean' commands manually", commitBehavior, emojiCount)
				}
			}
		},
	}).Parse(reviewTemplate))

	return tmpl.Execute(os.Stdout, review)
}

// validateConfigurationFile validates existing configuration and provides suggestions.
func validateConfigurationFile(targetDir string, opts *SetupLintOptions) error {
	if !quiet {
		fmt.Printf("Antimoji Configuration Validation\n")
		fmt.Printf("==================================\n\n")
	}

	// Check for .antimoji.yaml
	configPath := filepath.Join(targetDir, ".antimoji.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Printf("No antimoji configuration found at %s\n", configPath)
		fmt.Printf("Run 'antimoji setup-lint --mode=<mode>' to create configuration.\n")
		return nil
	}

	// Validate the configuration file
	validationResult := config.ValidateConfigFile(configPath)

	// Display results
	fmt.Printf("Configuration File: %s\n", configPath)
	fmt.Printf("Status: %s\n\n", validationResult.Summary.String())

	if len(validationResult.Issues) > 0 {
		fmt.Printf("Issues Found:\n")
		for _, issue := range validationResult.Issues {
			fmt.Printf("  %s\n\n", issue.String())
		}
	} else {
		fmt.Printf("No issues found. Configuration is valid!\n")
	}

	// Show suggestions if available
	if validationResult.Summary.Infos > 0 {
		fmt.Printf("Suggestions for improvement:\n")
		for _, issue := range validationResult.Issues {
			if issue.Level == config.ValidationLevelInfo {
				fmt.Printf("  - %s\n", issue.Message)
			}
		}
	}

	return nil
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
	// Note: --profile flag is now supported in both clean and scan commands
	// Only check for flags that are actually invalid
	if strings.Contains(configStr, "--fail-on-found") {
		return fmt.Errorf("configuration contains invalid --fail-on-found flag")
	}
	// Remove the --profile validation since it's now a valid flag

	return nil
}
