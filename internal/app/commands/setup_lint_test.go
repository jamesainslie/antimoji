package commands

import (
	"context"
	"testing"

	"github.com/antimoji/antimoji/internal/observability/logging"
	"github.com/antimoji/antimoji/internal/ui"
	"github.com/stretchr/testify/assert"
)

func TestNewSetupLintHandler(t *testing.T) {
	t.Run("creates handler with dependencies", func(t *testing.T) {
		logger := logging.NewMockLogger()
		uiOutput := ui.NewUserOutput(ui.DefaultConfig())

		handler := NewSetupLintHandler(logger, uiOutput)

		assert.NotNil(t, handler)
		assert.Equal(t, logger, handler.logger)
		assert.Equal(t, uiOutput, handler.ui)
	})
}

func TestSetupLintHandler_CreateCommand(t *testing.T) {
	logger := logging.NewMockLogger()
	uiOutput := ui.NewUserOutput(ui.DefaultConfig())
	handler := NewSetupLintHandler(logger, uiOutput)

	t.Run("creates command with correct properties", func(t *testing.T) {
		cmd := handler.CreateCommand()

		assert.Equal(t, "setup-lint", cmd.Use[:10])
		assert.Contains(t, cmd.Short, "linting")
		assert.NotEmpty(t, cmd.Long)
		assert.NotNil(t, cmd.RunE)
	})

	t.Run("has expected flags", func(t *testing.T) {
		cmd := handler.CreateCommand()
		flags := cmd.Flags()

		assert.NotNil(t, flags.Lookup("mode"))
		assert.NotNil(t, flags.Lookup("output-dir"))
		assert.NotNil(t, flags.Lookup("precommit"))
		assert.NotNil(t, flags.Lookup("golangci"))
		assert.NotNil(t, flags.Lookup("allowed-emojis"))
		assert.NotNil(t, flags.Lookup("force"))
		assert.NotNil(t, flags.Lookup("skip-precommit"))
		assert.NotNil(t, flags.Lookup("repair"))
		assert.NotNil(t, flags.Lookup("review"))
		assert.NotNil(t, flags.Lookup("validate"))
	})

	t.Run("flag defaults are correct", func(t *testing.T) {
		cmd := handler.CreateCommand()
		flags := cmd.Flags()

		mode, _ := flags.GetString("mode")
		outputDir, _ := flags.GetString("output-dir")
		precommit, _ := flags.GetBool("precommit")
		golangci, _ := flags.GetBool("golangci")
		allowedEmojis, _ := flags.GetStringSlice("allowed-emojis")
		force, _ := flags.GetBool("force")
		skipPrecommit, _ := flags.GetBool("skip-precommit")
		repair, _ := flags.GetBool("repair")
		review, _ := flags.GetBool("review")
		validate, _ := flags.GetBool("validate")

		assert.Equal(t, "zero-tolerance", mode)
		assert.Equal(t, ".", outputDir)
		assert.True(t, precommit)
		assert.True(t, golangci)
		// The template uses empty emoji strings that render as actual emojis
		assert.Equal(t, 2, len(allowedEmojis)) // Should have 2 default emojis
		assert.False(t, force)
		assert.False(t, skipPrecommit)
		assert.False(t, repair)
		assert.False(t, review)
		assert.False(t, validate)
	})
}

func TestSetupLintHandler_Execute(t *testing.T) {
	logger := logging.NewMockLogger()
	uiOutput := ui.NewUserOutput(ui.DefaultConfig())
	handler := NewSetupLintHandler(logger, uiOutput)

	t.Run("executes with placeholder implementation", func(t *testing.T) {
		opts := &SetupLintOptions{
			Mode:              "zero-tolerance",
			OutputDir:         ".",
			PreCommitConfig:   true,
			GolangCIConfig:    true,
			AllowedEmojis:     []string{"✅", "❌"},
			Force:             false,
			SkipPreCommitHook: false,
			Repair:            false,
			Review:            false,
			Validate:          false,
		}

		err := handler.Execute(context.Background(), nil, []string{"."}, opts)

		// Should return placeholder error
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not yet fully refactored")

		// Check logging occurred
		logs := logger.GetLogs()
		assert.True(t, len(logs) > 0)

		foundStart := false
		foundEnd := false
		for _, log := range logs {
			if log.Message == "Starting setup-lint operation" {
				foundStart = true
			}
			if log.Message == "Setup-lint command executed with DI" {
				foundEnd = true
			}
		}
		assert.True(t, foundStart)
		assert.True(t, foundEnd)
	})

	t.Run("handles nil context", func(t *testing.T) {
		opts := &SetupLintOptions{
			Mode:      "allow-list",
			OutputDir: "/tmp/test",
		}

		err := handler.Execute(context.TODO(), nil, []string{"test-path"}, opts)

		assert.Error(t, err) // Placeholder error
	})

	t.Run("logs all operation parameters", func(t *testing.T) {
		opts := &SetupLintOptions{
			Mode:              "allow-list",
			OutputDir:         "/custom/path",
			PreCommitConfig:   false,
			GolangCIConfig:    false,
			AllowedEmojis:     []string{"🚀", "✨"},
			Force:             true,
			SkipPreCommitHook: true,
			Repair:            true,
			Review:            true,
			Validate:          true,
		}

		err := handler.Execute(context.Background(), nil, []string{"/test/path"}, opts)

		assert.Error(t, err) // Placeholder error

		// Verify parameters were logged correctly
		logs := logger.GetLogs()
		foundCorrectMode := false
		foundCorrectOutputDir := false

		for _, log := range logs {
			if log.Message == "Starting setup-lint operation" {
				// Check log contains correct parameters
				for i := 0; i < len(log.KeysAndValues)-1; i += 2 {
					if log.KeysAndValues[i] == "mode" && log.KeysAndValues[i+1] == "allow-list" {
						foundCorrectMode = true
					}
					if log.KeysAndValues[i] == "output_dir" && log.KeysAndValues[i+1] == "/custom/path" {
						foundCorrectOutputDir = true
					}
				}
			}
		}

		assert.True(t, foundCorrectMode)
		assert.True(t, foundCorrectOutputDir)
	})

	t.Run("handles different modes", func(t *testing.T) {
		modes := []string{"zero-tolerance", "allow-list", "permissive"}

		for _, mode := range modes {
			t.Run("mode_"+mode, func(t *testing.T) {
				opts := &SetupLintOptions{
					Mode:      mode,
					OutputDir: ".",
				}

				err := handler.Execute(context.Background(), nil, []string{"."}, opts)

				assert.Error(t, err) // Still placeholder
				assert.Contains(t, err.Error(), "not yet fully refactored")
			})
		}
	})

	t.Run("handles all boolean flags", func(t *testing.T) {
		opts := &SetupLintOptions{
			Mode:              "zero-tolerance",
			OutputDir:         ".",
			PreCommitConfig:   true,
			GolangCIConfig:    true,
			AllowedEmojis:     []string{},
			Force:             true,
			SkipPreCommitHook: true,
			Repair:            true,
			Review:            true,
			Validate:          true,
		}

		err := handler.Execute(context.Background(), nil, []string{"."}, opts)

		assert.Error(t, err) // Placeholder error

		// Just verify it doesn't panic with all flags set to true
		logs := logger.GetLogs()
		assert.True(t, len(logs) > 0)
	})
}

func TestSetupLintOptions_Struct(t *testing.T) {
	t.Run("SetupLintOptions has expected fields", func(t *testing.T) {
		opts := &SetupLintOptions{
			Mode:              "permissive",
			OutputDir:         "/custom/output",
			PreCommitConfig:   false,
			GolangCIConfig:    true,
			AllowedEmojis:     []string{"🎉", "🚀", "✨"},
			Force:             true,
			SkipPreCommitHook: true,
			Repair:            false,
			Review:            true,
			Validate:          false,
		}

		assert.Equal(t, "permissive", opts.Mode)
		assert.Equal(t, "/custom/output", opts.OutputDir)
		assert.False(t, opts.PreCommitConfig)
		assert.True(t, opts.GolangCIConfig)
		assert.Equal(t, []string{"🎉", "🚀", "✨"}, opts.AllowedEmojis)
		assert.True(t, opts.Force)
		assert.True(t, opts.SkipPreCommitHook)
		assert.False(t, opts.Repair)
		assert.True(t, opts.Review)
		assert.False(t, opts.Validate)
	})
}
