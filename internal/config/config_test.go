package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("loads valid YAML config", func(t *testing.T) {
		configContent := `profiles:
  default:
    recursive: true
    unicode_emojis: true
    text_emoticons: false
    custom_patterns:
      - ":smile:"
      - ":frown:"
    emoji_allowlist:
      - "✅"
      - "❌"
    max_file_size: 1048576
    buffer_size: 65536`
		configPath := filepath.Join(tmpDir, "config.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		assert.NoError(t, err)

		result := LoadConfig(configPath)
		assert.True(t, result.IsOk())

		config := result.Unwrap()
		assert.Contains(t, config.Profiles, "default")

		profile := config.Profiles["default"]
		assert.True(t, profile.Recursive)
		assert.True(t, profile.UnicodeEmojis)
		assert.False(t, profile.TextEmoticons)
		assert.Equal(t, []string{":smile:", ":frown:"}, profile.CustomPatterns)
		assert.Equal(t, []string{"✅", "❌"}, profile.EmojiAllowlist)
		assert.Equal(t, int64(1048576), profile.MaxFileSize)
		assert.Equal(t, 65536, profile.BufferSize)
	})

	t.Run("handles non-existent config file", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "nonexistent.yaml")

		result := LoadConfig(configPath)
		assert.True(t, result.IsErr())
		// Cross-platform error message check
		errorMsg := result.Error().Error()
		assert.True(t,
			strings.Contains(errorMsg, "no such file") ||
				strings.Contains(errorMsg, "cannot find the file") ||
				strings.Contains(errorMsg, "does not exist"),
			"Expected file not found error, got: %s", errorMsg)
	})

	t.Run("handles invalid YAML", func(t *testing.T) {
		configContent := `
profiles:
  default:
    recursive: true
    invalid_yaml: [unclosed array
`
		configPath := filepath.Join(tmpDir, "invalid.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		assert.NoError(t, err)

		result := LoadConfig(configPath)
		assert.True(t, result.IsErr())
	})

	t.Run("loads empty config file", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "empty.yaml")
		err := os.WriteFile(configPath, []byte{}, 0644)
		assert.NoError(t, err)

		result := LoadConfig(configPath)
		assert.True(t, result.IsOk())

		config := result.Unwrap()
		// Empty config should have empty profiles
		assert.Empty(t, config.Profiles)
	})
}

func TestDefaultConfig(t *testing.T) {
	t.Run("returns sensible defaults", func(t *testing.T) {
		config := DefaultConfig()

		// Default config should have default profile
		assert.Contains(t, config.Profiles, "default")

		profile := config.Profiles["default"]
		assert.True(t, profile.Recursive)
		assert.True(t, profile.UnicodeEmojis)
		assert.True(t, profile.TextEmoticons)
		assert.True(t, profile.CustomPatterns != nil)
		assert.True(t, profile.EmojiAllowlist != nil)
		assert.True(t, profile.MaxFileSize > 0)
		assert.True(t, profile.BufferSize > 0)
	})

	t.Run("has empty include patterns by default", func(t *testing.T) {
		config := DefaultConfig()
		profile := config.Profiles["default"]

		// Empty include patterns means include all files (unless excluded)
		assert.Empty(t, profile.IncludePatterns, "Default config should have empty include patterns to include all files")

		expectedExcludes := []string{"vendor/*", "node_modules/*", ".git/*"}
		for _, pattern := range expectedExcludes {
			assert.Contains(t, profile.ExcludePatterns, pattern)
		}
	})
}

func TestGetProfile(t *testing.T) {
	t.Run("returns existing profile", func(t *testing.T) {
		config := DefaultConfig()

		result := GetProfile(config, "default")
		assert.True(t, result.IsOk())

		profile := result.Unwrap()
		assert.True(t, profile.Recursive)
	})

	t.Run("handles non-existent profile", func(t *testing.T) {
		config := DefaultConfig()

		result := GetProfile(config, "nonexistent")
		assert.True(t, result.IsErr())
		assert.Contains(t, result.Error().Error(), "profile not found")
	})

	t.Run("returns default profile when name is empty", func(t *testing.T) {
		config := DefaultConfig()

		result := GetProfile(config, "")
		assert.True(t, result.IsOk())

		profile := result.Unwrap()
		assert.True(t, profile.Recursive) // Should be default profile
	})
}

func TestValidateConfig(t *testing.T) {
	t.Run("validates correct config", func(t *testing.T) {
		config := DefaultConfig()

		result := ValidateConfig(config)
		assert.True(t, result.IsOk())
	})

	// Version validation removed - no longer needed

	t.Run("rejects negative buffer size", func(t *testing.T) {
		config := DefaultConfig()
		profile := config.Profiles["default"]
		profile.BufferSize = -1
		config.Profiles["default"] = profile

		result := ValidateConfig(config)
		assert.True(t, result.IsErr())
		assert.Contains(t, result.Error().Error(), "buffer size")
	})

	t.Run("rejects negative max file size", func(t *testing.T) {
		config := DefaultConfig()
		profile := config.Profiles["default"]
		profile.MaxFileSize = -1
		config.Profiles["default"] = profile

		result := ValidateConfig(config)
		assert.True(t, result.IsErr())
		assert.Contains(t, result.Error().Error(), "max file size")
	})
}

func TestToProcessingConfig(t *testing.T) {
	t.Run("converts profile to processing config", func(t *testing.T) {
		profile := Profile{
			UnicodeEmojis:  true,
			TextEmoticons:  false,
			CustomPatterns: []string{":smile:"},
			MaxFileSize:    1024,
			BufferSize:     512,
		}

		processingConfig := ToProcessingConfig(profile)
		assert.True(t, processingConfig.EnableUnicode)
		assert.False(t, processingConfig.EnableEmoticons)
		assert.True(t, processingConfig.EnableCustom)
		assert.Equal(t, int64(1024), processingConfig.MaxFileSize)
		assert.Equal(t, 512, processingConfig.BufferSize)
	})

	t.Run("handles empty custom patterns", func(t *testing.T) {
		profile := Profile{
			UnicodeEmojis:  true,
			TextEmoticons:  true,
			CustomPatterns: []string{},
			MaxFileSize:    1024,
			BufferSize:     512,
		}

		processingConfig := ToProcessingConfig(profile)
		assert.True(t, processingConfig.EnableUnicode)
		assert.True(t, processingConfig.EnableEmoticons)
		assert.False(t, processingConfig.EnableCustom) // Should be false for empty patterns
	})
}

func TestMergeProfiles(t *testing.T) {
	t.Run("merges profiles correctly", func(t *testing.T) {
		base := Profile{
			Recursive:     true,
			UnicodeEmojis: true,
			BufferSize:    1024,
			OutputFormat:  "table",
		}

		override := Profile{
			Recursive:     false,  // Override
			UnicodeEmojis: true,   // Explicitly set to maintain
			BufferSize:    2048,   // Override
			OutputFormat:  "json", // Override
		}

		merged := MergeProfiles(base, override)
		assert.False(t, merged.Recursive)            // Overridden
		assert.True(t, merged.UnicodeEmojis)         // Explicitly set in override
		assert.Equal(t, 2048, merged.BufferSize)     // Overridden
		assert.Equal(t, "json", merged.OutputFormat) // Overridden
	})

	t.Run("handles empty override", func(t *testing.T) {
		base := Profile{
			Recursive:     true,
			UnicodeEmojis: true,
			BufferSize:    1024,
		}

		override := Profile{} // Empty override (zero values)

		merged := MergeProfiles(base, override)
		// With our current implementation, boolean zero values (false) will override
		assert.False(t, merged.Recursive)                   // Zero value overrides
		assert.False(t, merged.UnicodeEmojis)               // Zero value overrides
		assert.Equal(t, base.BufferSize, merged.BufferSize) // Non-zero values preserved
	})
}

func TestValidateProfile(t *testing.T) {
	t.Run("validates output format", func(t *testing.T) {
		config := DefaultConfig()
		profile := config.Profiles["default"]
		profile.OutputFormat = "invalid"
		config.Profiles["default"] = profile

		result := ValidateConfig(config)
		assert.True(t, result.IsErr())
		assert.Contains(t, result.Error().Error(), "invalid output format")
	})

	t.Run("validates max workers", func(t *testing.T) {
		config := DefaultConfig()
		profile := config.Profiles["default"]
		profile.MaxWorkers = -5
		config.Profiles["default"] = profile

		result := ValidateConfig(config)
		assert.True(t, result.IsErr())
		assert.Contains(t, result.Error().Error(), "max workers")
	})

	t.Run("validates max emoji threshold", func(t *testing.T) {
		config := DefaultConfig()
		profile := config.Profiles["default"]
		profile.MaxEmojiThreshold = -1
		config.Profiles["default"] = profile

		result := ValidateConfig(config)
		assert.True(t, result.IsErr())
		assert.Contains(t, result.Error().Error(), "max emoji threshold")
	})

	t.Run("allows valid output formats", func(t *testing.T) {
		validFormats := []string{"table", "json", "csv", ""}

		for _, format := range validFormats {
			config := DefaultConfig()
			profile := config.Profiles["default"]
			profile.OutputFormat = format
			config.Profiles["default"] = profile

			result := ValidateConfig(config)
			assert.True(t, result.IsOk(), "format %s should be valid", format)
		}
	})
}

// Example usage for documentation
func ExampleLoadConfig() {
	// Create a temporary config file
	tmpFile, err := os.CreateTemp("", "config.yaml")
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	// Write config content
	configContent := `version: "0.9.0"
profiles:
  default:
    recursive: true
    unicode_emojis: true
    text_emoticons: true`
	_, err = tmpFile.WriteString(configContent)
	if err != nil {
		panic(err)
	}
	_ = tmpFile.Close()

	// Load the config
	result := LoadConfig(tmpFile.Name())
	if result.IsOk() {
		config := result.Unwrap()
		fmt.Println("Config loaded with profiles:", len(config.Profiles))
	}
	// Output: Config loaded with profiles: 1
}
