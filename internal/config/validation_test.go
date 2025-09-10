package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfigValidator(t *testing.T) {
	t.Run("creates new validator", func(t *testing.T) {
		validator := NewConfigValidator()

		assert.NotNil(t, validator)
		assert.NotNil(t, validator.issues)
		assert.Empty(t, validator.issues)
	})
}

func TestConfigValidator_ValidateConfig(t *testing.T) {
	validator := NewConfigValidator()

	t.Run("validates empty config", func(t *testing.T) {
		config := Config{
			Profiles: map[string]Profile{},
		}

		result := validator.ValidateConfig(config)

		assert.False(t, result.IsValid)
		assert.True(t, result.HasErrors())
		assert.True(t, result.Summary.Errors > 0)

		errorMessages := result.GetErrorMessages()
		assert.Contains(t, errorMessages[0], "no profiles defined")
	})

	t.Run("validates valid config", func(t *testing.T) {
		config := DefaultConfig()

		result := validator.ValidateConfig(config)

		// Should be valid or have only warnings/info
		if !result.IsValid {
			assert.Equal(t, 0, result.Summary.Errors, "Should have no errors: %v", result.GetErrorMessages())
		}
		assert.False(t, result.HasErrors())
	})

	t.Run("validates config with invalid profile", func(t *testing.T) {
		config := Config{
			Profiles: map[string]Profile{
				"test": {
					BufferSize:        -1,        // Invalid
					MaxEmojiThreshold: -5,        // Invalid
					OutputFormat:      "invalid", // Invalid
				},
			},
		}

		result := validator.ValidateConfig(config)

		assert.False(t, result.IsValid)
		assert.True(t, result.HasErrors())
		assert.True(t, result.Summary.Errors >= 2) // At least 2 errors (buffer size and threshold)
	})
}

func TestConfigValidator_validateProfile(t *testing.T) {
	t.Run("validates profile with emoji policy issues", func(t *testing.T) {
		validator := NewConfigValidator()
		profile := Profile{
			MaxEmojiThreshold: 0,
			EmojiAllowlist:    []string{"✅", "❌"}, // Contradictory with zero threshold
			UnicodeEmojis:     false,
			TextEmoticons:     false,
			CustomPatterns:    []string{}, // No detection methods
		}

		validator.validateProfile("test", profile)

		assert.True(t, len(validator.issues) > 0)

		// Should have warning about contradictory settings
		foundContradictionWarning := false
		foundNoDetectionError := false

		for _, issue := range validator.issues {
			if issue.Level == ValidationLevelWarning && issue.Field == "profiles.test.emoji_allowlist" {
				foundContradictionWarning = true
			}
			if issue.Level == ValidationLevelError && issue.Field == "profiles.test.emoji_detection" {
				foundNoDetectionError = true
			}
		}

		assert.True(t, foundContradictionWarning, "Should warn about contradictory allowlist/threshold")
		assert.True(t, foundNoDetectionError, "Should error about no detection methods")
	})

	t.Run("validates profile with file filtering issues", func(t *testing.T) {
		validator := NewConfigValidator()
		profile := Profile{
			IncludePatterns: []string{"*.go"},
			ExcludePatterns: []string{"*.go", ""}, // Conflicting and empty pattern
			UnicodeEmojis:   true,                 // Valid detection method
		}

		validator.validateProfile("test", profile)

		assert.True(t, len(validator.issues) > 0)

		// Should have error about conflicting patterns and empty pattern
		foundConflictError := false
		foundEmptyPatternError := false

		for _, issue := range validator.issues {
			if issue.Level == ValidationLevelError && issue.Field == "profiles.test.patterns" {
				foundConflictError = true
			}
			if issue.Level == ValidationLevelError && issue.Field == "profiles.test.exclude_patterns[1]" {
				foundEmptyPatternError = true
			}
		}

		assert.True(t, foundConflictError, "Should error about conflicting patterns")
		assert.True(t, foundEmptyPatternError, "Should error about empty pattern")
	})

	t.Run("validates profile with performance issues", func(t *testing.T) {
		validator := NewConfigValidator()
		profile := Profile{
			BufferSize:    100, // Very small
			MaxFileSize:   500, // Very small
			MaxWorkers:    50,  // Very high
			UnicodeEmojis: true,
		}

		validator.validateProfile("test", profile)

		assert.True(t, len(validator.issues) > 0)

		// Should have warnings about performance settings
		warningCount := 0
		for _, issue := range validator.issues {
			if issue.Level == ValidationLevelWarning {
				warningCount++
			}
		}

		assert.True(t, warningCount >= 3, "Should have warnings about buffer size, file size, and workers")
	})
}

func TestConfigValidator_validateEmojiPolicyConsistency(t *testing.T) {
	tests := []struct {
		name          string
		profile       Profile
		expectWarning bool
		expectError   bool
	}{
		{
			name: "consistent zero tolerance",
			profile: Profile{
				MaxEmojiThreshold: 0,
				EmojiAllowlist:    []string{},
				UnicodeEmojis:     true,
			},
			expectWarning: false,
			expectError:   false,
		},
		{
			name: "consistent allow list",
			profile: Profile{
				MaxEmojiThreshold: 5,
				EmojiAllowlist:    []string{"✅", "❌"},
				UnicodeEmojis:     true,
			},
			expectWarning: false,
			expectError:   false,
		},
		{
			name: "contradictory zero threshold with allowlist",
			profile: Profile{
				MaxEmojiThreshold: 0,
				EmojiAllowlist:    []string{"✅"},
				UnicodeEmojis:     true,
			},
			expectWarning: true,
			expectError:   false,
		},
		{
			name: "no detection methods",
			profile: Profile{
				MaxEmojiThreshold: 5,
				UnicodeEmojis:     false,
				TextEmoticons:     false,
				CustomPatterns:    []string{},
			},
			expectWarning: false,
			expectError:   true,
		},
		{
			name: "unrealistic threshold",
			profile: Profile{
				MaxEmojiThreshold: 200,
				UnicodeEmojis:     true,
			},
			expectWarning: true,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewConfigValidator()
			validator.validateEmojiPolicyConsistency("test", tt.profile)

			hasWarning := false
			hasError := false

			for _, issue := range validator.issues {
				if issue.Level == ValidationLevelWarning {
					hasWarning = true
				}
				if issue.Level == ValidationLevelError {
					hasError = true
				}
			}

			assert.Equal(t, tt.expectWarning, hasWarning, "Warning expectation mismatch")
			assert.Equal(t, tt.expectError, hasError, "Error expectation mismatch")
		})
	}
}

func TestConfigValidator_validateFileFilteringLogic(t *testing.T) {
	t.Run("detects redundant include patterns", func(t *testing.T) {
		validator := NewConfigValidator()
		profile := Profile{
			IncludePatterns: []string{"*"},
			UnicodeEmojis:   true,
		}

		validator.validateFileFilteringLogic("test", profile)

		// Should have info about redundant pattern
		hasRedundantInfo := false
		for _, issue := range validator.issues {
			if issue.Level == ValidationLevelInfo && issue.Field == "test.include_patterns" {
				hasRedundantInfo = true
				break
			}
		}
		assert.True(t, hasRedundantInfo)
	})

	t.Run("detects invalid glob patterns", func(t *testing.T) {
		validator := NewConfigValidator()
		profile := Profile{
			ExcludePatterns: []string{"[invalid"},
			UnicodeEmojis:   true,
		}

		validator.validateFileFilteringLogic("test", profile)

		// Should have error about invalid pattern
		hasPatternError := false
		for _, issue := range validator.issues {
			if issue.Level == ValidationLevelError && issue.Field == "test.exclude_patterns[0]" {
				hasPatternError = true
				break
			}
		}
		assert.True(t, hasPatternError)
	})

	t.Run("suggests common directory ignores", func(t *testing.T) {
		validator := NewConfigValidator()
		profile := Profile{
			DirectoryIgnoreList: []string{}, // Missing common directories
			UnicodeEmojis:       true,
		}

		validator.validateFileFilteringLogic("test", profile)

		// Should have info about missing common directories
		hasDirectoryInfo := false
		for _, issue := range validator.issues {
			if issue.Level == ValidationLevelInfo && issue.Field == "test.directory_ignore_list" {
				hasDirectoryInfo = true
				break
			}
		}
		assert.True(t, hasDirectoryInfo)
	})
}

func TestValidationIssue_String(t *testing.T) {
	tests := []struct {
		name  string
		issue ValidationIssue
		want  []string // Strings that should be present
	}{
		{
			name: "error with suggestion and example",
			issue: ValidationIssue{
				Level:      ValidationLevelError,
				Field:      "test.field",
				Message:    "test error",
				Suggestion: "fix it",
				Example:    "field: value",
			},
			want: []string{"[ERROR]", "test.field", "test error", "Suggestion: fix it", "Example: field: value"},
		},
		{
			name: "warning without example",
			issue: ValidationIssue{
				Level:      ValidationLevelWarning,
				Field:      "test.field",
				Message:    "test warning",
				Suggestion: "consider fixing",
			},
			want: []string{"[WARNING]", "test.field", "test warning", "Suggestion: consider fixing"},
		},
		{
			name: "info without suggestion",
			issue: ValidationIssue{
				Level:   ValidationLevelInfo,
				Field:   "test.field",
				Message: "test info",
			},
			want: []string{"[INFO]", "test.field", "test info"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.issue.String()

			for _, expected := range tt.want {
				assert.Contains(t, result, expected)
			}
		})
	}
}

func TestValidationSummary_String(t *testing.T) {
	tests := []struct {
		name    string
		summary ValidationSummary
		want    string
	}{
		{
			name:    "no issues",
			summary: ValidationSummary{TotalIssues: 0},
			want:    "Configuration is valid with no issues",
		},
		{
			name: "errors only",
			summary: ValidationSummary{
				TotalIssues: 2,
				Errors:      2,
			},
			want: "Configuration has 2 errors",
		},
		{
			name: "mixed issues",
			summary: ValidationSummary{
				TotalIssues: 5,
				Errors:      1,
				Warnings:    2,
				Infos:       2,
			},
			want: "Configuration has 1 errors, 2 warnings, 2 suggestions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.summary.String()
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestValidationResult_Methods(t *testing.T) {
	t.Run("HasErrors", func(t *testing.T) {
		result := ValidationResult{
			Summary: ValidationSummary{Errors: 1},
		}
		assert.True(t, result.HasErrors())

		result.Summary.Errors = 0
		assert.False(t, result.HasErrors())
	})

	t.Run("GetErrorMessages", func(t *testing.T) {
		result := ValidationResult{
			Issues: []ValidationIssue{
				{Level: ValidationLevelError, Message: "error 1"},
				{Level: ValidationLevelWarning, Message: "warning 1"},
				{Level: ValidationLevelError, Message: "error 2"},
			},
		}

		messages := result.GetErrorMessages()
		assert.Len(t, messages, 2)
		assert.Contains(t, messages, "error 1")
		assert.Contains(t, messages, "error 2")
	})

	t.Run("String", func(t *testing.T) {
		result := ValidationResult{
			IsValid: false,
			Issues: []ValidationIssue{
				{Level: ValidationLevelError, Field: "test", Message: "test error"},
			},
			Summary: ValidationSummary{TotalIssues: 1, Errors: 1},
		}

		str := result.String()
		assert.Contains(t, str, "Configuration has 1 errors")
		assert.Contains(t, str, "[ERROR]")
		assert.Contains(t, str, "test error")
	})
}

func TestValidateConfigFile(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("validates existing config file", func(t *testing.T) {
		configContent := `profiles:
  default:
    unicode_emojis: true
    text_emoticons: true`

		configPath := filepath.Join(tempDir, "valid.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		result := ValidateConfigFile(configPath)
		assert.False(t, result.HasErrors()) // Should be valid or have only warnings
	})

	t.Run("handles nonexistent config file", func(t *testing.T) {
		configPath := filepath.Join(tempDir, "nonexistent.yaml")

		result := ValidateConfigFile(configPath)

		assert.False(t, result.IsValid)
		assert.True(t, result.HasErrors())
		assert.Len(t, result.Issues, 1)
		assert.Equal(t, ValidationLevelError, result.Issues[0].Level)
		assert.Equal(t, "file", result.Issues[0].Field)
	})

	t.Run("handles invalid YAML", func(t *testing.T) {
		configContent := `profiles:
  default:
    unicode_emojis: true
    invalid_yaml: [unclosed`

		configPath := filepath.Join(tempDir, "invalid.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		result := ValidateConfigFile(configPath)

		assert.False(t, result.IsValid)
		assert.True(t, result.HasErrors())
	})
}

func TestSuggestImprovements(t *testing.T) {
	t.Run("suggests improvements for large allowlist", func(t *testing.T) {
		profile := Profile{
			EmojiAllowlist: make([]string, 25), // Large allowlist
		}

		suggestions := SuggestImprovements(profile)

		assert.True(t, len(suggestions) > 0)

		hasAllowlistSuggestion := false
		for _, suggestion := range suggestions {
			if suggestion.Level == ValidationLevelInfo && suggestion.Field == "emoji_allowlist" {
				hasAllowlistSuggestion = true
				break
			}
		}
		assert.True(t, hasAllowlistSuggestion)
	})

	t.Run("suggests improvements for many include patterns", func(t *testing.T) {
		profile := Profile{
			IncludePatterns: make([]string, 15), // Many patterns
		}

		suggestions := SuggestImprovements(profile)

		assert.True(t, len(suggestions) > 0)

		hasPatternSuggestion := false
		for _, suggestion := range suggestions {
			if suggestion.Level == ValidationLevelInfo && suggestion.Field == "include_patterns" {
				hasPatternSuggestion = true
				break
			}
		}
		assert.True(t, hasPatternSuggestion)
	})

	t.Run("no suggestions for optimal profile", func(t *testing.T) {
		profile := Profile{
			EmojiAllowlist:  []string{"✅", "❌"}, // Reasonable size
			IncludePatterns: []string{"*.go"},   // Reasonable number
		}

		suggestions := SuggestImprovements(profile)
		assert.Empty(t, suggestions)
	})
}

func TestValidationLevels(t *testing.T) {
	t.Run("validation levels are defined", func(t *testing.T) {
		assert.Equal(t, ValidationLevel("error"), ValidationLevelError)
		assert.Equal(t, ValidationLevel("warning"), ValidationLevelWarning)
		assert.Equal(t, ValidationLevel("info"), ValidationLevelInfo)
	})
}

func TestConfigValidator_checkCommonMisconfigurations(t *testing.T) {
	t.Run("warns about test files in strict config", func(t *testing.T) {
		validator := NewConfigValidator()
		profile := Profile{
			FailOnFound:     true,
			IncludePatterns: []string{"*.go", "*_test.go"},
		}

		validator.checkCommonMisconfigurations("test", profile)

		hasTestWarning := false
		for _, issue := range validator.issues {
			if issue.Level == ValidationLevelWarning && issue.Field == "test.include_patterns" {
				hasTestWarning = true
				break
			}
		}
		assert.True(t, hasTestWarning)
	})

	t.Run("suggests exclusions when none configured", func(t *testing.T) {
		validator := NewConfigValidator()
		profile := Profile{
			FileIgnoreList:      []string{},
			ExcludePatterns:     []string{},
			DirectoryIgnoreList: []string{},
		}

		validator.checkCommonMisconfigurations("test", profile)

		hasExclusionInfo := false
		for _, issue := range validator.issues {
			if issue.Level == ValidationLevelInfo && issue.Field == "test.exclusions" {
				hasExclusionInfo = true
				break
			}
		}
		assert.True(t, hasExclusionInfo)
	})
}

func TestConfigValidator_validateCrossProfileConsistency(t *testing.T) {
	t.Run("detects identical profiles", func(t *testing.T) {
		validator := NewConfigValidator()
		config := Config{
			Profiles: map[string]Profile{
				"profile1": {
					MaxEmojiThreshold: 5,
					EmojiAllowlist:    []string{"✅"},
					FailOnFound:       true,
				},
				"profile2": {
					MaxEmojiThreshold: 5,
					EmojiAllowlist:    []string{"✅"},
					FailOnFound:       true,
				},
			},
		}

		validator.validateCrossProfileConsistency(config)

		hasDuplicateInfo := false
		for _, issue := range validator.issues {
			if issue.Level == ValidationLevelInfo && issue.Field == "profiles" {
				hasDuplicateInfo = true
				break
			}
		}
		assert.True(t, hasDuplicateInfo)
	})
}
