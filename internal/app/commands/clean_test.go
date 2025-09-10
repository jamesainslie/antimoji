package commands

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/antimoji/antimoji/internal/observability/logging"
	"github.com/antimoji/antimoji/internal/ui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCleanHandler(t *testing.T) {
	t.Run("creates handler with dependencies", func(t *testing.T) {
		logger := logging.NewMockLogger()
		uiOutput := ui.NewUserOutput(ui.DefaultConfig())

		handler := NewCleanHandler(logger, uiOutput)

		assert.NotNil(t, handler)
		assert.Equal(t, logger, handler.logger)
		assert.Equal(t, uiOutput, handler.ui)
	})
}

func TestCleanHandler_CreateCommand(t *testing.T) {
	logger := logging.NewMockLogger()
	uiOutput := ui.NewUserOutput(ui.DefaultConfig())
	handler := NewCleanHandler(logger, uiOutput)

	t.Run("creates command with correct properties", func(t *testing.T) {
		cmd := handler.CreateCommand()

		assert.Equal(t, "clean", cmd.Use[:5])
		assert.Contains(t, cmd.Short, "Remove")
		assert.NotEmpty(t, cmd.Long)
		assert.NotNil(t, cmd.RunE)
	})

	t.Run("has expected flags", func(t *testing.T) {
		cmd := handler.CreateCommand()
		flags := cmd.Flags()

		// dry-run is a persistent flag, not a local flag
		// Check the flags that actually exist based on the command definition
		assert.NotNil(t, flags.Lookup("recursive"))
		assert.NotNil(t, flags.Lookup("in-place"))
		assert.NotNil(t, flags.Lookup("backup"))
		assert.NotNil(t, flags.Lookup("replace"))
	})

	t.Run("flag defaults are correct", func(t *testing.T) {
		cmd := handler.CreateCommand()
		flags := cmd.Flags()

		dryRun, _ := flags.GetBool("dry-run")
		backup, _ := flags.GetBool("backup")
		recursive, _ := flags.GetBool("recursive")
		replace, _ := flags.GetString("replace")

		assert.False(t, dryRun)
		assert.False(t, backup)
		assert.True(t, recursive)
		assert.Equal(t, "", replace)
	})
}

func TestCleanHandler_Execute(t *testing.T) {
	tempDir := t.TempDir()
	logger := logging.NewMockLogger()
	uiOutput := ui.NewUserOutput(ui.DefaultConfig())
	handler := NewCleanHandler(logger, uiOutput)

	// Create test files with emojis
	testFiles := map[string]string{
		"clean.go":      "package main\n\nfunc main() {\n\tprintln(\"Hello 👋\")\n}",
		"normal.go":     "package main\n\nfunc test() {\n\treturn\n}",
		"emoji_test.go": "// Test file with 🚀 emoji\npackage main",
	}

	for filename, content := range testFiles {
		err := os.WriteFile(filepath.Join(tempDir, filename), []byte(content), 0644)
		require.NoError(t, err)
	}

	t.Run("executes successfully with dry run", func(t *testing.T) {
		opts := &CleanOptions{
			DryRun:    true,
			Backup:    false,
			Replace:   "",
			Recursive: true,
		}

		err := handler.Execute(context.Background(), []string{tempDir}, opts)
		assert.NoError(t, err)

		// Check logging occurred
		logs := logger.GetLogs()
		assert.True(t, len(logs) > 0)

		// Should have logged start and completion
		foundStart := false
		foundComplete := false
		for _, log := range logs {
			if log.Message == "Starting clean operation" {
				foundStart = true
			}
			if log.Message == "File modification process completed" {
				foundComplete = true
			}
		}
		assert.True(t, foundStart)
		assert.True(t, foundComplete)
	})

	t.Run("uses current directory when no args", func(t *testing.T) {
		opts := &CleanOptions{
			DryRun:    true,
			Recursive: true, // Need recursive for directory scanning
		}

		err := handler.Execute(context.Background(), []string{}, opts)
		assert.NoError(t, err)

		// Check default directory logging
		logs := logger.GetLogs()
		found := false
		for _, log := range logs {
			if log.Message == "No paths provided, using current directory" {
				found = true
				break
			}
		}
		assert.True(t, found)
	})

	t.Run("handles nil context", func(t *testing.T) {
		opts := &CleanOptions{
			DryRun:    true,
			Recursive: true,
		}

		err := handler.Execute(nil, []string{tempDir}, opts)
		assert.NoError(t, err)
	})

	t.Run("executes with backup option", func(t *testing.T) {
		opts := &CleanOptions{
			DryRun:    true,
			Backup:    true,
			Replace:   "X",
			Recursive: true,
		}

		err := handler.Execute(context.Background(), []string{tempDir}, opts)
		assert.NoError(t, err)

		// Check that backup option was logged
		logs := logger.GetLogs()
		foundConfig := false
		for _, log := range logs {
			if log.Message == "Modification configuration created" {
				foundConfig = true
				break
			}
		}
		assert.True(t, foundConfig)
	})

	t.Run("handles empty directory", func(t *testing.T) {
		emptyDir := filepath.Join(tempDir, "empty")
		err := os.MkdirAll(emptyDir, 0755)
		require.NoError(t, err)

		opts := &CleanOptions{
			DryRun:    true,
			Recursive: true,
		}

		err = handler.Execute(context.Background(), []string{emptyDir}, opts)
		assert.NoError(t, err)
	})
}

func TestCleanHandler_validateCleanOptions(t *testing.T) {
	logger := logging.NewMockLogger()
	uiOutput := ui.NewUserOutput(ui.DefaultConfig())
	handler := NewCleanHandler(logger, uiOutput)

	t.Run("validates valid options", func(t *testing.T) {
		opts := &CleanOptions{
			DryRun:    true,
			Backup:    true,
			Replace:   "X",
			Recursive: true,
			InPlace:   true, // Add InPlace field
		}

		err := handler.validateCleanOptions(opts)
		assert.NoError(t, err)
	})

	t.Run("validates options with empty replace", func(t *testing.T) {
		opts := &CleanOptions{
			DryRun:    false,
			Backup:    false,
			Replace:   "",
			Recursive: false,
			InPlace:   true, // Required when not DryRun
		}

		err := handler.validateCleanOptions(opts)
		assert.NoError(t, err)
	})

	// Note: Since the current implementation doesn't have validation logic,
	// these tests ensure the function exists and doesn't error
	t.Run("validates options requiring InPlace", func(t *testing.T) {
		opts := &CleanOptions{
			DryRun:  false, // Not dry run
			InPlace: false, // But not in-place either
		}

		err := handler.validateCleanOptions(opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must specify --in-place")
	})
}

// Note: displayResults is tested indirectly through the Execute integration tests
// Direct unit testing would require complex processor.ModifyResult mocking

func TestCleanOptions_Struct(t *testing.T) {
	t.Run("CleanOptions has expected fields", func(t *testing.T) {
		opts := &CleanOptions{
			DryRun:    true,
			Backup:    false,
			Replace:   "REMOVED",
			Recursive: true,
		}

		assert.True(t, opts.DryRun)
		assert.False(t, opts.Backup)
		assert.Equal(t, "REMOVED", opts.Replace)
		assert.True(t, opts.Recursive)
	})
}
