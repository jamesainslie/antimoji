package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/antimoji/antimoji/internal/observability/logging"
	"github.com/antimoji/antimoji/internal/types"
	"github.com/antimoji/antimoji/internal/ui"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewScanHandler(t *testing.T) {
	t.Run("creates handler with dependencies", func(t *testing.T) {
		logger := logging.NewMockLogger()
		ui := ui.NewUserOutput(ui.DefaultConfig())

		handler := NewScanHandler(logger, ui)
		assert.NotNil(t, handler)
		assert.Equal(t, logger, handler.logger)
		assert.Equal(t, ui, handler.ui)
	})
}

func TestScanHandler_CreateCommand(t *testing.T) {
	t.Run("creates command with correct properties", func(t *testing.T) {
		handler := NewScanHandler(logging.NewMockLogger(), ui.NewUserOutput(ui.DefaultConfig()))
		cmd := handler.CreateCommand()

		assert.Equal(t, "scan", cmd.Use[:4])
		assert.Contains(t, cmd.Short, "Scan files for emojis")
		assert.NotEmpty(t, cmd.Long)
		assert.NotNil(t, cmd.RunE)
	})

	t.Run("has expected flags", func(t *testing.T) {
		handler := NewScanHandler(logging.NewMockLogger(), ui.NewUserOutput(ui.DefaultConfig()))
		cmd := handler.CreateCommand()

		flags := cmd.Flags()
		assert.NotNil(t, flags.Lookup("recursive"))
		assert.NotNil(t, flags.Lookup("include"))
		assert.NotNil(t, flags.Lookup("exclude"))
		assert.NotNil(t, flags.Lookup("format"))
		assert.NotNil(t, flags.Lookup("count-only"))
		assert.NotNil(t, flags.Lookup("threshold"))
		assert.NotNil(t, flags.Lookup("ignore-allowlist"))
		assert.NotNil(t, flags.Lookup("stats"))
		assert.NotNil(t, flags.Lookup("workers"))
	})
}

func TestScanHandler_Execute(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("scans directory successfully", func(t *testing.T) {
		// Create test files
		testFile := filepath.Join(tempDir, "test.txt")
		err := os.WriteFile(testFile, []byte("Hello world! No emojis here."), 0644)
		require.NoError(t, err)

		mockLogger := logging.NewMockLogger()
		handler := NewScanHandler(mockLogger, ui.NewUserOutput(ui.DefaultConfig()))

		// Create a root command with persistent flags for testing
		rootCmd := &cobra.Command{Use: "antimoji"}
		rootCmd.PersistentFlags().String("config", "", "config file path")
		rootCmd.PersistentFlags().String("profile", "default", "configuration profile")

		// Create scan command and add to root
		scanCmd := handler.CreateCommand()
		rootCmd.AddCommand(scanCmd)

		opts := &ScanOptions{
			Recursive: true,
			Format:    "table",
		}

		err = handler.Execute(context.Background(), scanCmd, []string{tempDir}, opts)
		assert.NoError(t, err)

		// Verify logging occurred
		logs := mockLogger.GetLogs()
		assert.True(t, len(logs) > 0, "Expected some log entries")

		// Check that scan operation was logged
		found := false
		for _, log := range logs {
			if log.Message == "Starting scan operation" {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected 'Starting scan operation' log entry")
	})

	t.Run("handles empty directory", func(t *testing.T) {
		emptyDir := filepath.Join(tempDir, "empty")
		err := os.MkdirAll(emptyDir, 0755)
		require.NoError(t, err)

		mockLogger := logging.NewMockLogger()
		handler := NewScanHandler(mockLogger, ui.NewUserOutput(ui.DefaultConfig()))

		// Create a root command with persistent flags
		rootCmd := &cobra.Command{Use: "antimoji"}
		rootCmd.PersistentFlags().String("config", "", "config file path")
		rootCmd.PersistentFlags().String("profile", "default", "configuration profile")

		scanCmd := handler.CreateCommand()
		rootCmd.AddCommand(scanCmd)

		opts := &ScanOptions{
			Recursive: true,
			Format:    "table",
		}

		err = handler.Execute(context.Background(), scanCmd, []string{emptyDir}, opts)
		assert.NoError(t, err)
	})

	t.Run("uses current directory when no args provided", func(t *testing.T) {
		mockLogger := logging.NewMockLogger()
		handler := NewScanHandler(mockLogger, ui.NewUserOutput(ui.DefaultConfig()))

		// Create a root command with persistent flags
		rootCmd := &cobra.Command{Use: "antimoji"}
		rootCmd.PersistentFlags().String("config", "", "config file path")
		rootCmd.PersistentFlags().String("profile", "default", "configuration profile")

		scanCmd := handler.CreateCommand()
		rootCmd.AddCommand(scanCmd)

		opts := &ScanOptions{
			Recursive: true,
			Format:    "table",
		}

		err := handler.Execute(context.Background(), scanCmd, []string{}, opts)
		assert.NoError(t, err)

		// Verify that default directory was used
		logs := mockLogger.GetLogs()
		found := false
		for _, log := range logs {
			if log.Message == "No paths provided, using current directory" {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected default directory log entry")
	})
}

func TestScanHandler_CountTotalEmojis(t *testing.T) {
	t.Run("counts emojis correctly", func(t *testing.T) {
		handler := NewScanHandler(logging.NewMockLogger(), ui.NewUserOutput(ui.DefaultConfig()))

		// Create mock results
		results := []types.ProcessResult{
			{
				FilePath: "file1.txt",
				DetectionResult: types.DetectionResult{
					TotalCount: 5,
				},
			},
			{
				FilePath: "file2.txt",
				DetectionResult: types.DetectionResult{
					TotalCount: 3,
				},
			},
			{
				FilePath: "file3.txt",
				Error:    fmt.Errorf("some error"),
			},
		}

		total := handler.countTotalEmojis(results)
		assert.Equal(t, 8, total) // 5 + 3, error file ignored
	})
}
