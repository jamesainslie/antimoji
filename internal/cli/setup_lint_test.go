// Package cli provides tests for the setup-lint command.
package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSetupLintCommand(t *testing.T) {
	cmd := NewSetupLintCommand()

	assert.Equal(t, "setup-lint", cmd.Use[:10])
	assert.Contains(t, cmd.Short, "linting configuration")
	assert.Contains(t, cmd.Long, "zero-tolerance")
	assert.Contains(t, cmd.Long, "allow-list")
	assert.Contains(t, cmd.Long, "permissive")
}

func TestSetupLintModes(t *testing.T) {
	tests := []struct {
		mode     string
		isValid  bool
		expected LintMode
	}{
		{"zero-tolerance", true, ZeroToleranceMode},
		{"allow-list", true, AllowListMode},
		{"permissive", true, PermissiveMode},
		{"invalid-mode", false, ""},
		{"", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			mode := LintMode(tt.mode)
			valid := isValidLintMode(mode)

			assert.Equal(t, tt.isValid, valid)
			if tt.isValid {
				assert.Equal(t, tt.expected, mode)
			}
		})
	}
}

func TestGenerateConfigForMode(t *testing.T) {
	tests := []struct {
		mode                LintMode
		allowedEmojis       []string
		expectedProfile     string
		expectedThreshold   int
		expectedFailOnFound bool
	}{
		{
			mode:                ZeroToleranceMode,
			allowedEmojis:       []string{},
			expectedProfile:     "ci-lint",
			expectedThreshold:   0,
			expectedFailOnFound: true,
		},
		{
			mode:                AllowListMode,
			allowedEmojis:       []string{"✅", "❌"},
			expectedProfile:     "allow-list",
			expectedThreshold:   5,
			expectedFailOnFound: true,
		},
		{
			mode:                PermissiveMode,
			allowedEmojis:       []string{},
			expectedProfile:     "permissive",
			expectedThreshold:   20,
			expectedFailOnFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			opts := &SetupLintOptions{
				AllowedEmojis: tt.allowedEmojis,
			}

			cfg := generateConfigForMode(tt.mode, opts)

			// Check that appropriate profiles exist
			require.Contains(t, cfg.Profiles, tt.expectedProfile)
			profile := cfg.Profiles[tt.expectedProfile]

			assert.Equal(t, tt.expectedThreshold, profile.MaxEmojiThreshold)
			assert.Equal(t, tt.expectedFailOnFound, profile.FailOnFound)

			// Test specific mode behaviors
			switch tt.mode {
			case ZeroToleranceMode:
				assert.Empty(t, profile.EmojiAllowlist)
				assert.True(t, profile.FailOnFound)
				assert.Equal(t, 0, profile.MaxEmojiThreshold)
			case AllowListMode:
				assert.Equal(t, tt.allowedEmojis, profile.EmojiAllowlist)
				assert.True(t, profile.FailOnFound)
				assert.Greater(t, profile.MaxEmojiThreshold, 0)
			case PermissiveMode:
				assert.NotEmpty(t, profile.EmojiAllowlist)
				assert.False(t, profile.FailOnFound)
				assert.Greater(t, profile.MaxEmojiThreshold, 10)
			}
		})
	}
}

func TestGenerateZeroToleranceConfig(t *testing.T) {
	baseConfig := generateConfigForMode(ZeroToleranceMode, &SetupLintOptions{})

	require.Contains(t, baseConfig.Profiles, "ci-lint")
	require.Contains(t, baseConfig.Profiles, "zero-tolerance")

	profile := baseConfig.Profiles["ci-lint"]

	// Should have empty allowlist
	assert.Empty(t, profile.EmojiAllowlist)

	// Should be strict
	assert.True(t, profile.FailOnFound)
	assert.Equal(t, 0, profile.MaxEmojiThreshold)
	assert.Equal(t, 1, profile.ExitCodeOnFound)

	// Should detect all emoji types
	assert.True(t, profile.UnicodeEmojis)
	assert.True(t, profile.TextEmoticons)
	assert.NotEmpty(t, profile.CustomPatterns)
}

func TestGenerateAllowListConfig(t *testing.T) {
	allowedEmojis := []string{"✅", "❌", "⚠️"}
	baseConfig := generateConfigForMode(AllowListMode, &SetupLintOptions{
		AllowedEmojis: allowedEmojis,
	})

	require.Contains(t, baseConfig.Profiles, "allow-list")
	profile := baseConfig.Profiles["allow-list"]

	// Should have specified allowlist
	assert.Equal(t, allowedEmojis, profile.EmojiAllowlist)

	// Should be moderately strict
	assert.True(t, profile.FailOnFound)
	assert.Equal(t, 5, profile.MaxEmojiThreshold)
	assert.Equal(t, 1, profile.ExitCodeOnFound)
}

func TestGeneratePermissiveConfig(t *testing.T) {
	baseConfig := generateConfigForMode(PermissiveMode, &SetupLintOptions{})

	require.Contains(t, baseConfig.Profiles, "permissive")
	profile := baseConfig.Profiles["permissive"]

	// Should have generous allowlist
	assert.NotEmpty(t, profile.EmojiAllowlist)
	assert.GreaterOrEqual(t, len(profile.EmojiAllowlist), 10)

	// Should be lenient
	assert.False(t, profile.FailOnFound)
	assert.Equal(t, 20, profile.MaxEmojiThreshold)
	assert.Equal(t, 0, profile.ExitCodeOnFound)
}

func TestGeneratePreCommitConfigForMode(t *testing.T) {
	tests := []struct {
		mode     LintMode
		contains []string
	}{
		{
			mode: ZeroToleranceMode,
			contains: []string{
				"zero-tolerance",
				"--threshold=0",
				"--fail-on-found",
				"Strict emoji linting",
			},
		},
		{
			mode: AllowListMode,
			contains: []string{
				"allow-list",
				"--threshold=5",
				"--fail-on-found",
				"Allow-list emoji linting",
			},
		},
		{
			mode: PermissiveMode,
			contains: []string{
				"permissive",
				"--threshold=20",
				"Permissive emoji linting",
			},
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			config := generatePreCommitConfigForMode(tt.mode)

			for _, expected := range tt.contains {
				assert.Contains(t, config, expected)
			}

			// Should contain standard structure
			assert.Contains(t, config, "repos:")
			assert.Contains(t, config, "antimoji-lint")
			assert.Contains(t, config, "build-antimoji")
		})
	}
}

func TestGenerateGolangCIConfigForMode(t *testing.T) {
	tests := []struct {
		mode     LintMode
		contains []string
	}{
		{
			mode: ZeroToleranceMode,
			contains: []string{
				"antimoji:",
				"profile: ci-lint",
				"mode: zero-tolerance",
			},
		},
		{
			mode: AllowListMode,
			contains: []string{
				"antimoji:",
				"profile: allow-list",
				"mode: allow-list",
			},
		},
		{
			mode: PermissiveMode,
			contains: []string{
				"antimoji:",
				"profile: permissive",
				"mode: permissive",
			},
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			config := generateGolangCIConfigForMode(tt.mode)

			for _, expected := range tt.contains {
				assert.Contains(t, config, expected)
			}
		})
	}
}

func TestRunSetupLintValidation(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	tests := []struct {
		name      string
		opts      *SetupLintOptions
		args      []string
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid zero-tolerance mode",
			opts: &SetupLintOptions{
				Mode:      "zero-tolerance",
				OutputDir: tempDir,
				Force:     true,
			},
			args:      []string{},
			expectErr: false,
		},
		{
			name: "invalid mode",
			opts: &SetupLintOptions{
				Mode:      "invalid-mode",
				OutputDir: tempDir,
			},
			args:      []string{},
			expectErr: true,
			errMsg:    "invalid linting mode",
		},
		{
			name: "non-existent directory",
			opts: &SetupLintOptions{
				Mode:      "zero-tolerance",
				OutputDir: "/non/existent/directory",
			},
			args:      []string{},
			expectErr: true,
			errMsg:    "target directory does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set quiet mode for tests
			originalQuiet := quiet
			quiet = true
			defer func() { quiet = originalQuiet }()

			cmd := NewSetupLintCommand()
			err := runSetupLint(cmd, tt.args, tt.opts)

			if tt.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGenerateAntimojiConfig(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name      string
		mode      LintMode
		opts      *SetupLintOptions
		expectErr bool
		errMsg    string
	}{
		{
			name: "successful config generation",
			mode: ZeroToleranceMode,
			opts: &SetupLintOptions{
				Force: true,
			},
			expectErr: false,
		},
		{
			name: "file exists without force",
			mode: AllowListMode,
			opts: &SetupLintOptions{
				Force: false,
			},
			expectErr: true,
			errMsg:    "configuration file already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set quiet mode for tests
			originalQuiet := quiet
			quiet = true
			defer func() { quiet = originalQuiet }()

			configPath := filepath.Join(tempDir, ".antimoji.yaml")

			// For the second test, create the file first
			if tt.name == "file exists without force" {
				err := os.WriteFile(configPath, []byte("existing config"), 0644)
				require.NoError(t, err)
			}

			err := generateAntimojiConfig(tempDir, tt.mode, tt.opts)

			if tt.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)

				// Check that file was created
				assert.FileExists(t, configPath)

				// Check that file contains valid YAML
				data, err := os.ReadFile(configPath)
				require.NoError(t, err)
				assert.Contains(t, string(data), "version:")
				assert.Contains(t, string(data), "profiles:")
			}
		})
	}
}

func TestUpdatePreCommitConfig(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name      string
		mode      LintMode
		opts      *SetupLintOptions
		setup     func()
		expectErr bool
	}{
		{
			name: "create new config",
			mode: ZeroToleranceMode,
			opts: &SetupLintOptions{
				Force: true,
			},
			setup:     func() {},
			expectErr: false,
		},
		{
			name: "existing config without force",
			mode: AllowListMode,
			opts: &SetupLintOptions{
				Force: false,
			},
			setup: func() {
				configPath := filepath.Join(tempDir, ".pre-commit-config.yaml")
				err := os.WriteFile(configPath, []byte("existing config"), 0644)
				require.NoError(t, err)
			},
			expectErr: false, // Should not error, just skip
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set quiet mode for tests
			originalQuiet := quiet
			quiet = true
			defer func() { quiet = originalQuiet }()

			tt.setup()

			err := updatePreCommitConfig(tempDir, tt.mode, tt.opts)

			if tt.expectErr {
				require.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUpdateGolangCIConfig(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name      string
		mode      LintMode
		opts      *SetupLintOptions
		setup     func()
		expectErr bool
	}{
		{
			name: "create new config",
			mode: PermissiveMode,
			opts: &SetupLintOptions{
				Force: true,
			},
			setup:     func() {},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set quiet mode for tests
			originalQuiet := quiet
			quiet = true
			defer func() { quiet = originalQuiet }()

			tt.setup()

			err := updateGolangCIConfig(tempDir, tt.mode, tt.opts)

			if tt.expectErr {
				require.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Check that file was created/updated
				configPath := filepath.Join(tempDir, ".golangci.yml")
				assert.FileExists(t, configPath)

				data, err := os.ReadFile(configPath)
				require.NoError(t, err)
				content := string(data)

				assert.Contains(t, content, "antimoji:")
				assert.Contains(t, content, string(tt.mode))
			}
		})
	}
}

// Integration test for the complete setup-lint workflow
func TestSetupLintIntegration(t *testing.T) {
	tempDir := t.TempDir()

	// Test each mode
	modes := []LintMode{ZeroToleranceMode, AllowListMode, PermissiveMode}

	for _, mode := range modes {
		t.Run(string(mode), func(t *testing.T) {
			// Set quiet mode for tests
			originalQuiet := quiet
			quiet = true
			defer func() { quiet = originalQuiet }()

			opts := &SetupLintOptions{
				Mode:              string(mode),
				OutputDir:         tempDir,
				PreCommitConfig:   true,
				GolangCIConfig:    true,
				AllowedEmojis:     []string{"✅", "❌"},
				Force:             true,
				SkipPreCommitHook: true, // Skip since pre-commit might not be installed
			}

			cmd := NewSetupLintCommand()
			err := runSetupLint(cmd, []string{}, opts)
			require.NoError(t, err)

			// Check that all expected files were created
			antimojiConfig := filepath.Join(tempDir, ".antimoji.yaml")
			preCommitConfig := filepath.Join(tempDir, ".pre-commit-config.yaml")
			golangCIConfig := filepath.Join(tempDir, ".golangci.yml")

			assert.FileExists(t, antimojiConfig)
			assert.FileExists(t, preCommitConfig)
			assert.FileExists(t, golangCIConfig)

			// Verify antimoji config content
			data, err := os.ReadFile(antimojiConfig)
			require.NoError(t, err)
			antimojiContent := string(data)
			assert.Contains(t, antimojiContent, "version:")
			assert.Contains(t, antimojiContent, "profiles:")

			// Verify pre-commit config content
			data, err = os.ReadFile(preCommitConfig)
			require.NoError(t, err)
			preCommitContent := string(data)
			assert.Contains(t, preCommitContent, "antimoji-lint")
			assert.Contains(t, preCommitContent, string(mode))

			// Verify golangci config content
			data, err = os.ReadFile(golangCIConfig)
			require.NoError(t, err)
			golangCIContent := string(data)
			assert.Contains(t, golangCIContent, "antimoji:")
			assert.Contains(t, golangCIContent, string(mode))
		})
	}
}

func TestPrintSetupSummary(t *testing.T) {
	// This is mainly for coverage - testing output formatting
	opts := &SetupLintOptions{
		AllowedEmojis:   []string{"✅", "❌"},
		PreCommitConfig: true,
		GolangCIConfig:  true,
	}

	// Set quiet mode to false to test output
	originalQuiet := quiet
	quiet = false
	defer func() { quiet = originalQuiet }()

	// Test doesn't fail - mainly for coverage
	printSetupSummary(ZeroToleranceMode, opts)
	printSetupSummary(AllowListMode, opts)
	printSetupSummary(PermissiveMode, opts)
}
