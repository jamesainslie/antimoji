// Package main provides the entry point for the Antimoji CLI tool.
package main

import (
	"fmt"
	"os"

	"github.com/antimoji/antimoji/internal/cli"
)

// Build information (set by ldflags during build)
var (
	version   = "dev"
	buildTime = "unknown"
	gitCommit = "unknown"
)

func main() {
	// Set build information for CLI
	cli.SetBuildInfo(version, buildTime, gitCommit)
	
	if err := cli.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
