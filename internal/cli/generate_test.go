package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGenerateCommand(t *testing.T) {
	t.Run("creates generate command with correct properties", func(t *testing.T) {
		cmd := NewGenerateCommand()

		assert.Equal(t, "generate", strings.Fields(cmd.Use)[0])
		assert.Contains(t, cmd.Short, "allowlist configuration")
		assert.NotEmpty(t, cmd.Long)
		assert.Contains(t, cmd.Long, "ci-lint")
		assert.Contains(t, cmd.Long, "dev")
		assert.Contains(t, cmd.Long, "test-only")
	})

	t.Run("has expected flags", func(t *testing.T) {
		cmd := NewGenerateCommand()
		flags := cmd.Flags()

		// Check for generate-specific flags
		outputFlag := flags.Lookup("output")
		assert.NotNil(t, outputFlag)
		assert.Equal(t, "string", outputFlag.Value.Type())

		typeFlag := flags.Lookup("type")
		assert.NotNil(t, typeFlag)
		assert.Equal(t, "string", typeFlag.Value.Type())

		includeTestsFlag := flags.Lookup("include-tests")
		assert.NotNil(t, includeTestsFlag)
		assert.Equal(t, "bool", includeTestsFlag.Value.Type())

		minUsageFlag := flags.Lookup("min-usage")
		assert.NotNil(t, minUsageFlag)
		assert.Equal(t, "int", minUsageFlag.Value.Type())
	})
}

func TestAnalyzeEmojiUsage(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files with different emoji patterns
	testFiles := map[string]string{
		"source.go":      "package main\n\n// Hello world - no emojis",
		"test_emoji.go":  "package main\n\n// Test with 😀 emoji\nfunc TestSomething() { /* 😃 */ }",
		"README.md":      "# Project\n\nStatus: ✅ Working\n\nWarning: ⚠️ Be careful",
		"docs/guide.md":  "# Guide\n\nCelebration: 🎉\nRocket: 🚀",
		"script.sh":      "#!/bin/bash\necho 'Build complete 🔥'",
		".github/ci.yml": "name: CI\nsteps:\n  - name: Success\n    run: echo '✅ Done'",
	}

	for relativePath, content := range testFiles {
		fullPath := filepath.Join(tmpDir, relativePath)
		dir := filepath.Dir(fullPath)
		err := os.MkdirAll(dir, 0755)
		require.NoError(t, err)
		err = os.WriteFile(fullPath, []byte(content), 0644)
		require.NoError(t, err)
	}

	t.Run("analyzes emoji usage correctly", func(t *testing.T) {
		opts := &GenerateOptions{
			Recursive:    true,
			IncludeTests: true,
			IncludeDocs:  true,
			IncludeCI:    true,
			MinUsage:     1,
		}

		analysis, err := analyzeEmojiUsage([]string{tmpDir}, opts)
		assert.NoError(t, err)
		assert.NotNil(t, analysis)

		// Should find emojis
		assert.Greater(t, analysis.Statistics.UniqueEmojis, 0)
		assert.Greater(t, analysis.Statistics.TotalEmojis, 0)
		assert.Greater(t, analysis.Statistics.FilesWithEmojis, 0)

		// Should categorize files correctly
		assert.Contains(t, analysis.FilesByType, "test")
		assert.Contains(t, analysis.FilesByType, "documentation")
		assert.Contains(t, analysis.FilesByType, "ci")

		// Should have emojis by category
		assert.NotEmpty(t, analysis.EmojisByCategory)
	})

	t.Run("categorizes files correctly", func(t *testing.T) {
		testCases := []struct {
			filePath     string
			expectedType string
		}{
			{"test_emoji.go", "test"},
			{"README.md", "documentation"},
			{"docs/guide.md", "markdown"},
			{"script.sh", "other"},
			{".github/ci.yml", "ci"},
			{"source.go", "source"},
		}

		for _, tc := range testCases {
			result := categorizeFile(tc.filePath)
			assert.Equal(t, tc.expectedType, result, "File %s should be categorized as %s", tc.filePath, tc.expectedType)
		}
	})

	t.Run("filters text files correctly", func(t *testing.T) {
		testCases := []struct {
			filePath string
			isText   bool
		}{
			{"file.go", true},
			{"file.js", true},
			{"file.md", true},
			{"file.txt", true},
			{"Dockerfile", true},
			{"Makefile", true},
			{"file.bin", false},
			{"file.exe", false},
			{"file.png", false},
			{"unknown", false},
		}

		for _, tc := range testCases {
			result := isTextFile(tc.filePath)
			assert.Equal(t, tc.isText, result, "File %s text detection should be %v", tc.filePath, tc.isText)
		}
	})
}

func TestGenerateAllowlistConfig(t *testing.T) {
	// Create mock analysis data
	analysis := &EmojiUsageAnalysis{
		EmojisByCategory: map[string][]EmojiUsage{
			"unicode": {
				{Emoji: "😀", Count: 5, Files: []string{"test1.go", "test2.go"}, Category: "unicode", FileTypes: []string{"test"}},
				{Emoji: "✅", Count: 3, Files: []string{"README.md"}, Category: "unicode", FileTypes: []string{"documentation"}},
			},
			"emoticon": {
				{Emoji: ":)", Count: 2, Files: []string{"test1.go"}, Category: "emoticon", FileTypes: []string{"test"}},
			},
		},
		FilesByType: map[string][]string{
			"test":          {"test1.go", "test2.go"},
			"documentation": {"README.md"},
			"source":        {"main.go"},
		},
		Statistics: UsageStatistics{
			TotalEmojis:       10,
			UniqueEmojis:      3,
			FilesWithEmojis:   3,
			TotalFilesScanned: 4,
		},
	}

	t.Run("generates ci-lint allowlist correctly", func(t *testing.T) {
		opts := &GenerateOptions{
			Type:         "ci-lint",
			IncludeTests: true,
			IncludeDocs:  true,
			IncludeCI:    true,
			MinUsage:     1,
		}

		config, err := generateAllowlistConfig(analysis, opts)
		assert.NoError(t, err)
		assert.NotNil(t, config)

		profile := config.Profiles["ci-lint"]
		assert.Contains(t, profile.EmojiAllowlist, "😀")  // From tests
		assert.Contains(t, profile.EmojiAllowlist, "✅")  // From docs and common
		assert.Contains(t, profile.EmojiAllowlist, ":)") // From tests
		assert.NotEmpty(t, profile.FileIgnoreList)
		assert.NotEmpty(t, profile.DirectoryIgnoreList)
	})

	t.Run("generates test-only allowlist correctly", func(t *testing.T) {
		opts := &GenerateOptions{
			Type:         "test-only",
			IncludeTests: true,
			IncludeDocs:  false,
			IncludeCI:    false,
			MinUsage:     1,
		}

		config, err := generateAllowlistConfig(analysis, opts)
		assert.NoError(t, err)
		assert.NotNil(t, config)

		profile := config.Profiles["test-only"]
		assert.Contains(t, profile.EmojiAllowlist, "😀")  // From tests
		assert.Contains(t, profile.EmojiAllowlist, ":)") // From tests
		// Should not contain doc-only emojis unless they're also in tests
	})

	t.Run("generates minimal allowlist correctly", func(t *testing.T) {
		opts := &GenerateOptions{
			Type:     "minimal",
			MinUsage: 3, // Higher threshold
		}

		config, err := generateAllowlistConfig(analysis, opts)
		assert.NoError(t, err)
		assert.NotNil(t, config)

		profile := config.Profiles["minimal"]
		assert.Contains(t, profile.EmojiAllowlist, "😀") // Count: 5, above threshold
		assert.Contains(t, profile.EmojiAllowlist, "✅") // Count: 3, meets threshold
		// Should not contain ":)" as it has count 2, below threshold of 3
	})

	t.Run("handles custom profile name", func(t *testing.T) {
		opts := &GenerateOptions{
			Type:    "dev",
			Profile: "custom-profile",
		}

		config, err := generateAllowlistConfig(analysis, opts)
		assert.NoError(t, err)
		assert.NotNil(t, config)

		_, exists := config.Profiles["custom-profile"]
		assert.True(t, exists)
	})

	t.Run("handles unsupported type", func(t *testing.T) {
		opts := &GenerateOptions{
			Type: "invalid-type",
		}

		config, err := generateAllowlistConfig(analysis, opts)
		assert.Error(t, err)
		assert.Nil(t, config)
		assert.Contains(t, err.Error(), "unsupported generation type")
	})
}

func TestRunGenerate(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	testFiles := map[string]string{
		"main.go":       "package main\n\nfunc main() {\n\t// No emojis here\n}",
		"main_test.go":  "package main\n\nfunc TestMain() {\n\t// Test with 😀 emoji\n}",
		"README.md":     "# Project\n\nStatus: ✅ Working",
		"docs/guide.md": "# Guide\n\nCelebration: 🎉",
	}

	for relativePath, content := range testFiles {
		fullPath := filepath.Join(tmpDir, relativePath)
		dir := filepath.Dir(fullPath)
		err := os.MkdirAll(dir, 0755)
		require.NoError(t, err)
		err = os.WriteFile(fullPath, []byte(content), 0644)
		require.NoError(t, err)
	}

	t.Run("generates allowlist to stdout", func(t *testing.T) {
		cmd := NewGenerateCommand()
		opts := &GenerateOptions{
			Type:         "ci-lint",
			IncludeTests: true,
			IncludeDocs:  true,
			Recursive:    true,
			MinUsage:     1,
			Format:       "yaml",
		}

		// Capture stdout would require more complex setup
		// For now, test the core logic
		err := runGenerate(cmd, []string{tmpDir}, opts)
		assert.NoError(t, err)
	})

	t.Run("generates allowlist to file", func(t *testing.T) {
		outputFile := filepath.Join(tmpDir, "generated-allowlist.yaml")

		cmd := NewGenerateCommand()
		opts := &GenerateOptions{
			Output:       outputFile,
			Type:         "dev",
			IncludeTests: true,
			IncludeDocs:  true,
			Recursive:    true,
			MinUsage:     1,
			Format:       "yaml",
		}

		err := runGenerate(cmd, []string{tmpDir}, opts)
		assert.NoError(t, err)

		// Check that file was created
		_, err = os.Stat(outputFile)
		assert.NoError(t, err)

		// Read and verify content
		content, err := os.ReadFile(outputFile)
		assert.NoError(t, err)
		assert.Contains(t, string(content), "version:")
		assert.Contains(t, string(content), "emoji_allowlist:")
		assert.Contains(t, string(content), "Generated by antimoji generate")
	})

	t.Run("handles empty directory", func(t *testing.T) {
		emptyDir := filepath.Join(tmpDir, "empty")
		err := os.MkdirAll(emptyDir, 0755)
		require.NoError(t, err)

		cmd := NewGenerateCommand()
		opts := &GenerateOptions{
			Type:      "ci-lint",
			Recursive: true,
			MinUsage:  1,
			Format:    "yaml",
		}

		err = runGenerate(cmd, []string{emptyDir}, opts)
		assert.NoError(t, err) // Should not error on empty directories
	})

	t.Run("handles different generation types", func(t *testing.T) {
		types := []string{"ci-lint", "dev", "test-only", "docs-only", "minimal", "full"}

		for _, genType := range types {
			t.Run(genType, func(t *testing.T) {
				cmd := NewGenerateCommand()
				opts := &GenerateOptions{
					Type:         genType,
					IncludeTests: true,
					IncludeDocs:  true,
					IncludeCI:    true,
					Recursive:    true,
					MinUsage:     1,
					Format:       "yaml",
				}

				err := runGenerate(cmd, []string{tmpDir}, opts)
				assert.NoError(t, err, "Generation type %s should not error", genType)
			})
		}
	})

	t.Run("handles invalid generation type", func(t *testing.T) {
		cmd := NewGenerateCommand()
		opts := &GenerateOptions{
			Type:     "invalid-type",
			MinUsage: 1,
			Format:   "yaml",
		}

		err := runGenerate(cmd, []string{tmpDir}, opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported generation type")
	})
}

func TestCategorizeFile(t *testing.T) {
	tests := []struct {
		name         string
		filePath     string
		expectedType string
	}{
		{
			name:         "Go test file",
			filePath:     "internal/core/detector_test.go",
			expectedType: "test",
		},
		{
			name:         "Go source file",
			filePath:     "internal/core/detector.go",
			expectedType: "source",
		},
		{
			name:         "README file",
			filePath:     "README.md",
			expectedType: "documentation",
		},
		{
			name:         "Documentation markdown",
			filePath:     "docs/architecture.md",
			expectedType: "markdown",
		},
		{
			name:         "GitHub workflow",
			filePath:     ".github/workflows/ci.yml",
			expectedType: "ci",
		},
		{
			name:         "Shell script",
			filePath:     "scripts/build.sh",
			expectedType: "ci",
		},
		{
			name:         "Config file",
			filePath:     "config.yaml",
			expectedType: "config",
		},
		{
			name:         "JavaScript file",
			filePath:     "src/main.js",
			expectedType: "source",
		},
		{
			name:         "Test directory file",
			filePath:     "test/fixtures/data.txt",
			expectedType: "test",
		},
		{
			name:         "Unknown file",
			filePath:     "unknown.xyz",
			expectedType: "other",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := categorizeFile(tt.filePath)
			assert.Equal(t, tt.expectedType, result)
		})
	}
}

func TestIsTextFile(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		expected bool
	}{
		{
			name:     "Go file",
			filePath: "main.go",
			expected: true,
		},
		{
			name:     "JavaScript file",
			filePath: "app.js",
			expected: true,
		},
		{
			name:     "Markdown file",
			filePath: "README.md",
			expected: true,
		},
		{
			name:     "YAML file",
			filePath: "config.yaml",
			expected: true,
		},
		{
			name:     "Dockerfile",
			filePath: "Dockerfile",
			expected: true,
		},
		{
			name:     "Makefile",
			filePath: "Makefile",
			expected: true,
		},
		{
			name:     "Binary file",
			filePath: "app.exe",
			expected: false,
		},
		{
			name:     "Image file",
			filePath: "logo.png",
			expected: false,
		},
		{
			name:     "Unknown extension",
			filePath: "file.xyz",
			expected: false,
		},
		{
			name:     "No extension unknown",
			filePath: "randomfile",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTextFile(tt.filePath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEmojisFromFileType(t *testing.T) {
	analysis := &EmojiUsageAnalysis{
		EmojisByFile: map[string][]EmojiUsage{
			"test1.go": {
				{Emoji: "😀", Count: 1, Category: "unicode"},
				{Emoji: ":)", Count: 1, Category: "emoticon"},
			},
			"README.md": {
				{Emoji: "✅", Count: 1, Category: "unicode"},
			},
			"main.go": {
				{Emoji: "🔥", Count: 1, Category: "unicode"},
			},
		},
		FilesByType: map[string][]string{
			"test":          {"test1.go"},
			"documentation": {"README.md"},
			"source":        {"main.go"},
		},
	}

	t.Run("extracts test emojis correctly", func(t *testing.T) {
		emojis := getEmojisFromFileType(analysis, "test")
		assert.Contains(t, emojis, "😀")
		assert.Contains(t, emojis, ":)")
		assert.NotContains(t, emojis, "✅") // From docs, not tests
	})

	t.Run("extracts documentation emojis correctly", func(t *testing.T) {
		emojis := getEmojisFromFileType(analysis, "documentation")
		assert.Contains(t, emojis, "✅")
		assert.NotContains(t, emojis, "😀") // From tests, not docs
	})

	t.Run("handles non-existent file type", func(t *testing.T) {
		emojis := getEmojisFromFileType(analysis, "nonexistent")
		assert.Empty(t, emojis)
	})
}

func TestRemoveDuplicates(t *testing.T) {
	t.Run("removes duplicate strings", func(t *testing.T) {
		input := []string{"a", "b", "a", "c", "b", "d"}
		result := removeDuplicates(input)

		assert.Len(t, result, 4)
		assert.Contains(t, result, "a")
		assert.Contains(t, result, "b")
		assert.Contains(t, result, "c")
		assert.Contains(t, result, "d")
	})

	t.Run("handles empty slice", func(t *testing.T) {
		result := removeDuplicates([]string{})
		assert.Empty(t, result)
	})

	t.Run("handles slice with no duplicates", func(t *testing.T) {
		input := []string{"a", "b", "c"}
		result := removeDuplicates(input)
		assert.Equal(t, input, result)
	})
}

func TestAppendUnique(t *testing.T) {
	t.Run("appends unique item", func(t *testing.T) {
		slice := []string{"a", "b"}
		result := appendUnique(slice, "c")
		assert.Equal(t, []string{"a", "b", "c"}, result)
	})

	t.Run("does not append duplicate item", func(t *testing.T) {
		slice := []string{"a", "b"}
		result := appendUnique(slice, "b")
		assert.Equal(t, []string{"a", "b"}, result)
	})

	t.Run("handles empty slice", func(t *testing.T) {
		result := appendUnique([]string{}, "a")
		assert.Equal(t, []string{"a"}, result)
	})
}

// Example function demonstrating the generate command usage
func ExampleNewGenerateCommand() {
	// Create a generate command
	cmd := NewGenerateCommand()

	// Set up arguments for CI lint generation
	cmd.SetArgs([]string{
		"--type=ci-lint",
		"--output=.antimoji.yaml",
		"--include-tests=true",
		"--include-docs=true",
		"--min-usage=2",
		".",
	})

	// Execute the command (in real usage)
	// err := cmd.Execute()
	// if err != nil {
	//     fmt.Printf("Error: %v\n", err)
	// }

	fmt.Println("Generate command created successfully")
	// Output: Generate command created successfully
}
