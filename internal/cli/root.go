// Package cli provides the command-line interface for Antimoji.
package cli

import (
	"fmt"
	"os"
	"path/filepath"

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

	// Build information (will be set by main package)
	buildVersion   = "0.9.5"
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
- Text emoticon detection (,  etc.)
- Custom emoji pattern detection (, )
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
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	cmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "quiet mode")
	cmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "show what would be changed without modifying files")

	// Add subcommands
	cmd.AddCommand(NewScanCommand())
	cmd.AddCommand(NewCleanCommand())
	cmd.AddCommand(NewGenerateCommand())
	cmd.AddCommand(NewSetupLintCommand())
	cmd.AddCommand(NewVersionCommand())

	// Set up configuration
	cobra.OnInitialize(initConfig)

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
