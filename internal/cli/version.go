// Package cli provides version command implementation.
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewVersionCommand creates the version command.
func NewVersionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long:  `Display detailed version information including build time and git commit.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("antimoji version %s\n", buildVersion)
			fmt.Printf("Build time: %s\n", buildTime)
			fmt.Printf("Git commit: %s\n", buildGitCommit)
			fmt.Printf("Go version: %s\n", goVersion())
		},
	}

	return cmd
}

// goVersion returns the Go version used to build the binary.
func goVersion() string {
	// This would typically be set via build info, but for now return runtime version
	return "go1.21+"
}
