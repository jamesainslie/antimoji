// Package cli provides command implementations for the Antimoji CLI.
package cli

import (
	"github.com/antimoji/antimoji/internal/app/commands"
	"github.com/antimoji/antimoji/internal/observability/logging"
	"github.com/antimoji/antimoji/internal/ui"
	"github.com/spf13/cobra"
)

// NewUpgradeCommand creates the upgrade command
func NewUpgradeCommand() *cobra.Command {
	// Get global logger and UI
	logger := logging.GetGlobalLogger()
	userOutput := ui.GetGlobalUserOutput()

	// Create handler
	handler := commands.NewUpgradeHandler(logger, userOutput)

	// Set version from build info (buildVersion defined in root.go)
	handler.SetVersion(buildVersion)

	// Create and return command
	return handler.CreateCommand()
}
