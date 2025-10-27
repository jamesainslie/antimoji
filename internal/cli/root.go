// Package cli provides the command-line interface for Antimoji.
package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/antimoji/antimoji/internal/observability/logging"
	"github.com/antimoji/antimoji/internal/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Global flags
	cfgFile     string
	profileName string
	verbose     bool
	quiet       bool
	dryRun      bool
	logLevel    string
	logFormat   string

	// Build information - should be overridden at build time via ldflags:
	//   -X github.com/antimoji/antimoji/internal/cli.buildVersion=v1.0.0
	//   -X github.com/antimoji/antimoji/internal/cli.buildTime=2024-01-01T00:00:00Z
	//   -X github.com/antimoji/antimoji/internal/cli.buildGitCommit=abc123
	// Default values are used for development builds.
	buildVersion   = "0.0.0-dev"
	buildTime      = "unknown"
	buildGitCommit = "unknown"
)

// NewRootCommand creates the root command for the CLI.
func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "antimoji",
		Short: "High-performance emoji detection and removal CLI tool",
		Long: `Antimoji is a blazing-fast CLI tool for detecting and removing emojis
from code files, markdown documents, and other text-based artifacts.

Built with Go using functional programming principles, Antimoji provides:
- Unicode emoji detection across all major ranges
- Text emoticon detection (, , , etc.)
- Custom emoji pattern detection (:party:, :thumbsup:)
- Configurable allowlists and ignore patterns
- High-performance concurrent processing
- Git integration and CI/CD pipeline support`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       getBuildVersion(),
	}

	// Add global persistent flags
	cmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path")
	cmd.PersistentFlags().StringVar(&profileName, "profile", "default", "configuration profile")
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output (deprecated, use --log-level=info)")
	cmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "quiet mode (deprecated, use --log-level=silent)")
	cmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "show what would be changed without modifying files")
	cmd.PersistentFlags().StringVar(&logLevel, "log-level", "silent", "log level (silent, debug, info, warn, error)")
	cmd.PersistentFlags().StringVar(&logFormat, "log-format", "json", "log format (json, text)")

	// Add subcommands
	cmd.AddCommand(NewScanCommand())
	cmd.AddCommand(NewCleanCommand())
	cmd.AddCommand(NewGenerateCommand())
	cmd.AddCommand(NewSetupLintCommand())
	cmd.AddCommand(NewVersionCommand())
	cmd.AddCommand(NewUpgradeCommand())

	// Set up configuration and logging
	// Initialize logging and user output before any command execution
	cobra.OnInitialize(initConfig, initLogging, initUserOutput)

	return cmd
}

// getBuildVersion returns the current build version.
func getBuildVersion() string {
	return buildVersion
}

// SetBuildInfo sets the build information for version reporting.
func SetBuildInfo(version, bTime, gCommit string) {
	buildVersion = version
	buildTime = bTime
	buildGitCommit = gCommit
}

// Execute runs the root command.
func Execute() error {
	return NewRootCommand().Execute()
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error finding home directory: %v\n", err)
			return
		}

		// Search config in XDG config directory
		viper.AddConfigPath(filepath.Join(home, ".config", "antimoji"))
		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	// Environment variable support
	viper.SetEnvPrefix("ANTIMOJI")
	viper.AutomaticEnv()

	// Read config file
	if err := viper.ReadInConfig(); err == nil && verbose {
		fmt.Fprintf(os.Stderr, "Using config file: %s\n", viper.ConfigFileUsed())
	}
}

// initLogging initializes the global logger based on CLI flags and configuration.
func initLogging() {
	// Handle legacy flags for backward compatibility
	effectiveLogLevel := logLevel
	effectiveLogFormat := logFormat

	// Override with legacy flags if they are set
	if quiet {
		effectiveLogLevel = "silent"
	} else if verbose && effectiveLogLevel == "silent" {
		effectiveLogLevel = "info"
	}

	// Convert string values to logging types
	var level logging.LogLevel
	switch effectiveLogLevel {
	case "debug":
		level = logging.LevelDebug
	case "info":
		level = logging.LevelInfo
	case "warn":
		level = logging.LevelWarn
	case "error":
		level = logging.LevelError
	case "silent":
		level = logging.LevelSilent
	default:
		level = logging.LevelSilent
	}

	var format logging.LogFormat
	switch effectiveLogFormat {
	case "json":
		format = logging.FormatJSON
	case "text":
		format = logging.FormatText
	default:
		format = logging.FormatJSON
	}

	// Create logging configuration
	config := &logging.Config{
		Level:          level,
		Format:         format,
		Output:         os.Stderr,
		ServiceName:    "antimoji",
		ServiceVersion: buildVersion,
	}

	// Initialize global logger
	if err := logging.InitGlobalLogger(config); err != nil {
		// Fallback to stderr if logger initialization fails
		fmt.Fprintf(os.Stderr, "Warning: Failed to initialize logger: %v\n", err)
	}
}

// initUserOutput initializes the global user output system.
func initUserOutput() {
	// Determine user output level based on flags
	var outputLevel ui.OutputLevel
	if quiet {
		outputLevel = ui.OutputSilent
	} else if verbose {
		outputLevel = ui.OutputVerbose
	} else {
		outputLevel = ui.OutputNormal
	}

	// Create user output configuration
	config := &ui.Config{
		Level:        outputLevel,
		Writer:       os.Stdout,
		ErrorWriter:  os.Stderr,
		EnableColors: true, // TODO: Add flag to control this
	}

	// Initialize global user output
	ui.InitGlobalUserOutput(config)
}
