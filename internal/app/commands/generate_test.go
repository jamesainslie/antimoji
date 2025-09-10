package commands

import (
	"context"
	"testing"

	"github.com/antimoji/antimoji/internal/observability/logging"
	"github.com/antimoji/antimoji/internal/ui"
	"github.com/stretchr/testify/assert"
)

func TestNewGenerateHandler(t *testing.T) {
	t.Run("creates handler with dependencies", func(t *testing.T) {
		logger := logging.NewMockLogger()
		uiOutput := ui.NewUserOutput(ui.DefaultConfig())

		handler := NewGenerateHandler(logger, uiOutput)

		assert.NotNil(t, handler)
		assert.Equal(t, logger, handler.logger)
		assert.Equal(t, uiOutput, handler.ui)
	})
}

func TestGenerateHandler_CreateCommand(t *testing.T) {
	logger := logging.NewMockLogger()
	uiOutput := ui.NewUserOutput(ui.DefaultConfig())
	handler := NewGenerateHandler(logger, uiOutput)

	t.Run("creates command with correct properties", func(t *testing.T) {
		cmd := handler.CreateCommand()

		assert.Equal(t, "generate", cmd.Use[:8])
		assert.Contains(t, cmd.Short, "Generate")
		assert.NotEmpty(t, cmd.Long)
		assert.NotNil(t, cmd.RunE)
	})

	t.Run("has expected flags", func(t *testing.T) {
		cmd := handler.CreateCommand()
		flags := cmd.Flags()

		assert.NotNil(t, flags.Lookup("type"))
		assert.NotNil(t, flags.Lookup("output"))
		assert.NotNil(t, flags.Lookup("include-tests"))
		assert.NotNil(t, flags.Lookup("include-docs"))
		assert.NotNil(t, flags.Lookup("include-ci"))
		assert.NotNil(t, flags.Lookup("recursive"))
		assert.NotNil(t, flags.Lookup("min-usage"))
		assert.NotNil(t, flags.Lookup("format"))
		assert.NotNil(t, flags.Lookup("profile-name"))
	})

	t.Run("flag defaults are correct", func(t *testing.T) {
		cmd := handler.CreateCommand()
		flags := cmd.Flags()

		typeVal, _ := flags.GetString("type")
		output, _ := flags.GetString("output")
		includeTests, _ := flags.GetBool("include-tests")
		includeDocs, _ := flags.GetBool("include-docs")
		includeCI, _ := flags.GetBool("include-ci")
		recursive, _ := flags.GetBool("recursive")
		minUsage, _ := flags.GetInt("min-usage")
		format, _ := flags.GetString("format")
		profileName, _ := flags.GetString("profile-name")

		assert.Equal(t, "ci-lint", typeVal)
		assert.Equal(t, "", output)
		assert.True(t, includeTests)
		assert.True(t, includeDocs)
		assert.True(t, includeCI)
		assert.True(t, recursive)
		assert.Equal(t, 1, minUsage)
		assert.Equal(t, "yaml", format)
		assert.Equal(t, "", profileName)
	})
}

func TestGenerateHandler_Execute(t *testing.T) {
	logger := logging.NewMockLogger()
	uiOutput := ui.NewUserOutput(ui.DefaultConfig())
	handler := NewGenerateHandler(logger, uiOutput)

	t.Run("executes with placeholder implementation", func(t *testing.T) {
		opts := &GenerateOptions{
			Type:         "allowlist",
			Output:       "output.yaml",
			IncludeTests: true,
			IncludeDocs:  true,
			IncludeCI:    true,
			Recursive:    true,
			MinUsage:     1,
			Format:       "yaml",
			Profile:      "custom",
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
			if log.Message == "Starting emoji analysis for allowlist generation" {
				foundStart = true
			}
			if log.Message == "Generate command executed with DI" {
				foundEnd = true
			}
		}
		assert.True(t, foundStart)
		assert.True(t, foundEnd)
	})

	t.Run("uses current directory when no args", func(t *testing.T) {
		opts := &GenerateOptions{
			Type:   "allowlist",
			Format: "json",
		}

		err := handler.Execute(context.Background(), nil, []string{}, opts)

		assert.Error(t, err) // Still placeholder implementation

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
		opts := &GenerateOptions{
			Type: "allowlist",
		}

		err := handler.Execute(context.TODO(), nil, []string{"."}, opts)

		assert.Error(t, err) // Placeholder error
	})

	t.Run("logs correct parameters", func(t *testing.T) {
		opts := &GenerateOptions{
			Type:         "ci-lint",
			Output:       "test.json",
			IncludeTests: false,
			IncludeDocs:  false,
			IncludeCI:    false,
			Recursive:    false,
			MinUsage:     5,
			Format:       "json",
			Profile:      "test-profile",
		}

		err := handler.Execute(context.Background(), nil, []string{"/test/path"}, opts)

		assert.Error(t, err) // Placeholder error

		// Verify parameters were logged correctly
		logs := logger.GetLogs()
		foundCorrectType := false
		foundCorrectFormat := false

		for _, log := range logs {
			if log.Message == "Starting emoji analysis for allowlist generation" {
				// Check log contains correct type
				for i := 0; i < len(log.KeysAndValues)-1; i += 2 {
					if log.KeysAndValues[i] == "type" && log.KeysAndValues[i+1] == "ci-lint" {
						foundCorrectType = true
					}
				}
			}
			if log.Message == "Generate command executed with DI" {
				// Check log contains correct format
				for i := 0; i < len(log.KeysAndValues)-1; i += 2 {
					if log.KeysAndValues[i] == "format" && log.KeysAndValues[i+1] == "json" {
						foundCorrectFormat = true
					}
				}
			}
		}

		assert.True(t, foundCorrectType)
		assert.True(t, foundCorrectFormat)
	})
}

func TestGenerateOptions_Struct(t *testing.T) {
	t.Run("GenerateOptions has expected fields", func(t *testing.T) {
		opts := &GenerateOptions{
			Type:         "ci-lint",
			Output:       "output.json",
			IncludeTests: false,
			IncludeDocs:  true,
			IncludeCI:    false,
			Recursive:    true,
			MinUsage:     3,
			Format:       "json",
			Profile:      "test-profile",
		}

		assert.Equal(t, "ci-lint", opts.Type)
		assert.Equal(t, "output.json", opts.Output)
		assert.False(t, opts.IncludeTests)
		assert.True(t, opts.IncludeDocs)
		assert.False(t, opts.IncludeCI)
		assert.True(t, opts.Recursive)
		assert.Equal(t, 3, opts.MinUsage)
		assert.Equal(t, "json", opts.Format)
		assert.Equal(t, "test-profile", opts.Profile)
	})
}
