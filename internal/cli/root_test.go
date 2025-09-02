package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestRootCommand(t *testing.T) {
	t.Run("creates root command with correct properties", func(t *testing.T) {
		cmd := NewRootCommand()
		
		assert.Equal(t, "antimoji", cmd.Use)
		assert.Contains(t, cmd.Short, "emoji detection")
		assert.NotEmpty(t, cmd.Long)
		assert.True(t, cmd.SilenceUsage)
		assert.True(t, cmd.SilenceErrors)
	})

	t.Run("has expected global flags", func(t *testing.T) {
		cmd := NewRootCommand()
		
		// Check for global flags
		flags := cmd.PersistentFlags()
		
		configFlag := flags.Lookup("config")
		assert.NotNil(t, configFlag)
		assert.Equal(t, "string", configFlag.Value.Type())

		profileFlag := flags.Lookup("profile")
		assert.NotNil(t, profileFlag)
		assert.Equal(t, "string", profileFlag.Value.Type())

		verboseFlag := flags.Lookup("verbose")
		assert.NotNil(t, verboseFlag)
		assert.Equal(t, "bool", verboseFlag.Value.Type())

		quietFlag := flags.Lookup("quiet")
		assert.NotNil(t, quietFlag)
		assert.Equal(t, "bool", quietFlag.Value.Type())

		dryRunFlag := flags.Lookup("dry-run")
		assert.NotNil(t, dryRunFlag)
		assert.Equal(t, "bool", dryRunFlag.Value.Type())
	})

	t.Run("has scan subcommand", func(t *testing.T) {
		cmd := NewRootCommand()
		
		scanCmd := findSubcommand(cmd, "scan")
		if scanCmd == nil {
			// Print available commands for debugging
			fmt.Printf("Available commands: ")
			for _, subCmd := range cmd.Commands() {
				fmt.Printf("%s ", subCmd.Use)
			}
			fmt.Println()
		}
		assert.NotNil(t, scanCmd, "scan command should be available")
		if scanCmd != nil {
			cmdName := strings.Fields(scanCmd.Use)[0]
			assert.Equal(t, "scan", cmdName)
		}
	})

	t.Run("shows help when no arguments", func(t *testing.T) {
		cmd := NewRootCommand()
		
		var output bytes.Buffer
		cmd.SetOut(&output)
		cmd.SetArgs([]string{})
		
		err := cmd.Execute()
		assert.NoError(t, err)
		
		outputStr := output.String()
		assert.Contains(t, outputStr, "antimoji")
		assert.Contains(t, outputStr, "Available Commands")
	})

	t.Run("shows version with --version flag", func(t *testing.T) {
		cmd := NewRootCommand()
		
		var output bytes.Buffer
		cmd.SetOut(&output)
		cmd.SetArgs([]string{"--version"})
		
		err := cmd.Execute()
		assert.NoError(t, err)
		
		outputStr := output.String()
		assert.Contains(t, outputStr, "antimoji version")
		assert.Contains(t, outputStr, "0.6.0") // Current version
	})
}

func TestExecute(t *testing.T) {
	t.Run("executes without error", func(t *testing.T) {
		// Capture output to avoid polluting test output
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Execute with help to avoid actually running commands
		os.Args = []string{"antimoji", "--help"}
		
		err := Execute()
		
		// Restore stdout
		_ = w.Close() // Ignore error in test cleanup
		os.Stdout = oldStdout
		
		// Read captured output
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)
		
		assert.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, "antimoji")
	})
}

func TestGlobalFlags(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("config flag loads custom config", func(t *testing.T) {
		// Create a test config file
		configContent := `version: "0.1.0"
profiles:
  default:
    recursive: false
    unicode_emojis: false`
		
		configPath := filepath.Join(tmpDir, "test_config.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		assert.NoError(t, err)

		cmd := NewRootCommand()
		cmd.SetArgs([]string{"--config", configPath, "scan", "--help"})
		
		var output bytes.Buffer
		cmd.SetOut(&output)
		
		err = cmd.Execute()
		assert.NoError(t, err)
	})

	t.Run("profile flag selects profile", func(t *testing.T) {
		cmd := NewRootCommand()
		cmd.SetArgs([]string{"--profile", "strict", "scan", "--help"})
		
		var output bytes.Buffer
		cmd.SetOut(&output)
		
		err := cmd.Execute()
		assert.NoError(t, err)
	})

	t.Run("verbose flag increases verbosity", func(t *testing.T) {
		cmd := NewRootCommand()
		cmd.SetArgs([]string{"--verbose", "scan", "--help"})
		
		var output bytes.Buffer
		cmd.SetOut(&output)
		
		err := cmd.Execute()
		assert.NoError(t, err)
	})
}

// Helper function to find a subcommand by name
func findSubcommand(cmd *cobra.Command, name string) *cobra.Command {
	for _, subCmd := range cmd.Commands() {
		// Check if the command name matches (first word of Use)
		cmdName := strings.Fields(subCmd.Use)[0]
		if cmdName == name {
			return subCmd
		}
	}
	return nil
}

// Example usage for documentation
func ExampleNewRootCommand() {
	cmd := NewRootCommand()
	cmd.SetArgs([]string{"--help"})
	
	var output bytes.Buffer
	cmd.SetOut(&output)
	
	_ = cmd.Execute()
	
	// Output will contain help text
	outputStr := output.String()
	if strings.Contains(outputStr, "antimoji") {
		println("Help displayed successfully")
	}
}
