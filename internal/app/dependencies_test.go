package app

import (
	"os"
	"testing"

	"github.com/antimoji/antimoji/internal/observability/logging"
	"github.com/antimoji/antimoji/internal/ui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDependencies(t *testing.T) {
	t.Run("creates dependencies with valid config", func(t *testing.T) {
		config := &Config{
			LogLevel:       logging.LevelInfo,
			LogFormat:      logging.FormatJSON,
			LogOutput:      os.Stderr,
			UILevel:        ui.OutputNormal,
			UIWriter:       os.Stdout,
			UIErrorWriter:  os.Stderr,
			UIEnableColors: true,
			ServiceName:    "test-service",
			ServiceVersion: "1.0.0",
		}

		deps, err := NewDependencies(config)
		require.NoError(t, err)
		require.NotNil(t, deps)

		assert.NotNil(t, deps.Logger)
		assert.NotNil(t, deps.UI)
	})

	t.Run("returns error with nil config", func(t *testing.T) {
		deps, err := NewDependencies(nil)
		assert.Error(t, err)
		assert.Nil(t, deps)
		assert.Contains(t, err.Error(), "config cannot be nil")
	})
}

func TestNewTestDependencies(t *testing.T) {
	t.Run("creates test dependencies", func(t *testing.T) {
		deps := NewTestDependencies()
		require.NotNil(t, deps)

		assert.NotNil(t, deps.Logger)
		assert.NotNil(t, deps.UI)

		// Verify logger is a mock
		mockLogger, ok := deps.Logger.(*logging.MockLogger)
		assert.True(t, ok, "Expected MockLogger")
		assert.NotNil(t, mockLogger)
	})
}

func TestDependencies_Validate(t *testing.T) {
	t.Run("validates complete dependencies", func(t *testing.T) {
		deps := NewTestDependencies()
		err := deps.Validate()
		assert.NoError(t, err)
	})

	t.Run("returns error for nil logger", func(t *testing.T) {
		deps := &Dependencies{
			Logger: nil,
			UI:     ui.NewUserOutput(ui.DefaultConfig()),
		}
		err := deps.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "logger dependency is nil")
	})

	t.Run("returns error for nil UI", func(t *testing.T) {
		deps := &Dependencies{
			Logger: logging.NewMockLogger(),
			UI:     nil,
		}
		err := deps.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "UI dependency is nil")
	})
}

func TestDependencies_Close(t *testing.T) {
	t.Run("closes without error", func(t *testing.T) {
		deps := NewTestDependencies()
		err := deps.Close(nil)
		assert.NoError(t, err)
	})
}
