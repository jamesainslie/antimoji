// Package main provides the entry point for the Antimoji CLI tool.
package main

import (
	"fmt"
	"os"

	"github.com/antimoji/antimoji/internal/cli"
)

// Build information (set by goreleaser or build scripts)
// These will be used in future versions for version reporting
var (
	_ = "dev"     // version - reserved for future use
	_ = "unknown" // buildTime - reserved for future use  
	_ = "unknown" // gitCommit - reserved for future use
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
