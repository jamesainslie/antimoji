package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("creates application with valid dependencies", func(t *testing.T) {
		deps := NewTestDependencies()
		app, err := New(deps)
		require.NoError(t, err)
		require.NotNil(t, app)

		assert.Equal(t, deps, app.GetDependencies())
		assert.NotNil(t, app.GetRootCommand())
		assert.Equal(t, "antimoji", app.GetRootCommand().Use)
	})

	t.Run("returns error with nil dependencies", func(t *testing.T) {
		app, err := New(nil)
		assert.Error(t, err)
		assert.Nil(t, app)
		assert.Contains(t, err.Error(), "dependencies cannot be nil")
	})

	t.Run("returns error with invalid dependencies", func(t *testing.T) {
		deps := &Dependencies{
			Logger: nil, // Invalid - missing logger
			UI:     nil, // Invalid - missing UI
		}
		app, err := New(deps)
		assert.Error(t, err)
		assert.Nil(t, app)
		assert.Contains(t, err.Error(), "invalid dependencies")
	})
}

func TestApplication_Run(t *testing.T) {
	t.Run("runs version command successfully", func(t *testing.T) {
		deps := NewTestDependencies()
		app, err := New(deps)
		require.NoError(t, err)

		err = app.Run([]string{"version"})
		assert.NoError(t, err)
	})

	t.Run("returns error for placeholder commands", func(t *testing.T) {
		deps := NewTestDependencies()
		app, err := New(deps)
		require.NoError(t, err)

		err = app.Run([]string{"scan", "."})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "scan command not yet refactored")
	})
}

func TestApplication_GetBuildVersion(t *testing.T) {
	t.Run("returns correct version", func(t *testing.T) {
		deps := NewTestDependencies()
		app, err := New(deps)
		require.NoError(t, err)

		version := app.getBuildVersion()
		assert.Equal(t, "0.9.16-refactor", version)
	})
}
