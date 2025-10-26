package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUpgradeCommand(t *testing.T) {
	cmd := NewUpgradeCommand()

	require.NotNil(t, cmd)
	assert.Equal(t, "upgrade [flags]", cmd.Use)
	assert.Contains(t, cmd.Short, "Upgrade")
	assert.Contains(t, cmd.Short, "latest version")
	assert.True(t, cmd.SilenceUsage)
	assert.True(t, cmd.SilenceErrors)

	// Check flags
	checkOnlyFlag := cmd.Flags().Lookup("check-only")
	require.NotNil(t, checkOnlyFlag)
	assert.Equal(t, "false", checkOnlyFlag.DefValue)

	forceFlag := cmd.Flags().Lookup("force")
	require.NotNil(t, forceFlag)
	assert.Equal(t, "false", forceFlag.DefValue)
}

func TestUpgradeCommandHelp(t *testing.T) {
	cmd := NewUpgradeCommand()

	// Verify help text contains key information
	longHelp := cmd.Long
	assert.Contains(t, longHelp, "Homebrew")
	assert.Contains(t, longHelp, "Go install")
	assert.Contains(t, longHelp, "APT/DEB")
	assert.Contains(t, longHelp, "YUM/RPM")
	assert.Contains(t, longHelp, "Pacman")
	assert.Contains(t, longHelp, "source")
}

func TestUpgradeCommandFlags(t *testing.T) {
	cmd := NewUpgradeCommand()

	tests := []struct {
		name         string
		flagName     string
		wantShort    string
		wantDefault  string
		wantRequired bool
	}{
		{
			name:         "check-only flag",
			flagName:     "check-only",
			wantShort:    "",
			wantDefault:  "false",
			wantRequired: false,
		},
		{
			name:         "force flag",
			flagName:     "force",
			wantShort:    "",
			wantDefault:  "false",
			wantRequired: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := cmd.Flags().Lookup(tt.flagName)
			require.NotNil(t, flag, "flag %s should exist", tt.flagName)
			assert.Equal(t, tt.wantDefault, flag.DefValue)

			if tt.wantShort != "" {
				assert.Equal(t, tt.wantShort, flag.Shorthand)
			}

			// Check if required annotation is set
			annotations := flag.Annotations
			if tt.wantRequired {
				assert.NotNil(t, annotations)
			}
		})
	}
}
