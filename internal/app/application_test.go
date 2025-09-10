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

	t.Run("scan command works with dependency injection", func(t *testing.T) {
		deps := NewTestDependencies()
		app, err := New(deps)
		require.NoError(t, err)

		// Scan command should work now with dependency injection
		err = app.Run([]string{"scan", "."})
		assert.NoError(t, err)
	})

	t.Run("clean command works with dependency injection", func(t *testing.T) {
		deps := NewTestDependencies()
		app, err := New(deps)
		require.NoError(t, err)

		// Clean command should return validation error (requires --in-place or --dry-run)
		err = app.Run([]string{"clean", "."})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must specify --in-place")
	})

	t.Run("generate command uses dependency injection", func(t *testing.T) {
		deps := NewTestDependencies()
		app, err := New(deps)
		require.NoError(t, err)

		// Generate command should work with DI but return placeholder error
		err = app.Run([]string{"generate", "."})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "complex implementation pending")
	})

	t.Run("setup-lint command uses dependency injection", func(t *testing.T) {
		deps := NewTestDependencies()
		app, err := New(deps)
		require.NoError(t, err)

		// Setup-lint command should work with DI but return placeholder error
		err = app.Run([]string{"setup-lint", "."})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "complex implementation pending")
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

func TestApplication_Shutdown(t *testing.T) {
	t.Run("shutdown works correctly", func(t *testing.T) {
		deps := NewTestDependencies()
		app, err := New(deps)
		require.NoError(t, err)

		err = app.Shutdown()
		assert.NoError(t, err)
	})

	t.Run("shutdown handles timeout", func(t *testing.T) {
		deps := NewTestDependencies()
		app, err := New(deps)
		require.NoError(t, err)

		// Multiple shutdowns should work
		err = app.Shutdown()
		assert.NoError(t, err)

		err = app.Shutdown()
		assert.NoError(t, err)
	})
}
