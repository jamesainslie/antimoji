package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/antimoji/antimoji/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestNewScanCommand(t *testing.T) {
	t.Run("creates scan command with correct properties", func(t *testing.T) {
		cmd := NewScanCommand()
		
		assert.Equal(t, "scan [flags] [path...]", cmd.Use)
		assert.Contains(t, cmd.Short, "Scan files for emojis")
		assert.NotEmpty(t, cmd.Long)
	})

	t.Run("has expected flags", func(t *testing.T) {
		cmd := NewScanCommand()
		flags := cmd.Flags()

		recursiveFlag := flags.Lookup("recursive")
		assert.NotNil(t, recursiveFlag)
		assert.Equal(t, "bool", recursiveFlag.Value.Type())

		includeFlag := flags.Lookup("include")
		assert.NotNil(t, includeFlag)
		assert.Equal(t, "string", includeFlag.Value.Type())

		excludeFlag := flags.Lookup("exclude")
		assert.NotNil(t, excludeFlag)
		assert.Equal(t, "string", excludeFlag.Value.Type())

		formatFlag := flags.Lookup("format")
		assert.NotNil(t, formatFlag)
		assert.Equal(t, "string", formatFlag.Value.Type())

		countOnlyFlag := flags.Lookup("count-only")
		assert.NotNil(t, countOnlyFlag)
		assert.Equal(t, "bool", countOnlyFlag.Value.Type())

		statsFlag := flags.Lookup("stats")
		assert.NotNil(t, statsFlag)
		assert.Equal(t, "bool", statsFlag.Value.Type())
	})
}

func TestScanCommand_Integration(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	testFiles := map[string]string{
		"emoji.go":    "package main\n\n// Hello 😀 world\nfunc main() {}",
		"clean.go":    "package main\n\nfunc main() {\n\tfmt.Println(\"Hello world\")\n}",
		"mixed.md":    "# Title\n\nThis has :) emoticons and 😀 unicode.",
		"custom.txt":  "I'm :smile: about this :thumbs_up: feature!",
		"binary.bin":  string([]byte{0x00, 0x01, 0x02, 0xFF}),
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(tmpDir, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		assert.NoError(t, err)
	}

	t.Run("scans single file with emojis", func(t *testing.T) {
		cmd := NewRootCommand()
		
		emojiFile := filepath.Join(tmpDir, "emoji.go")
		cmd.SetArgs([]string{"scan", emojiFile})
		
		err := cmd.Execute()
		assert.NoError(t, err, "scan command should execute without error")
	})

	t.Run("scans directory recursively", func(t *testing.T) {
		cmd := NewRootCommand()
		cmd.SetArgs([]string{"scan", "--recursive", tmpDir})
		
		err := cmd.Execute()
		assert.NoError(t, err, "recursive scan should execute without error")
	})

	t.Run("count-only mode executes successfully", func(t *testing.T) {
		cmd := NewRootCommand()
		cmd.SetArgs([]string{"scan", "--count-only", tmpDir})
		
		err := cmd.Execute()
		assert.NoError(t, err, "count-only mode should execute without error")
	})

	t.Run("JSON output format executes successfully", func(t *testing.T) {
		cmd := NewRootCommand()
		emojiFile := filepath.Join(tmpDir, "emoji.go")
		cmd.SetArgs([]string{"scan", "--format", "json", emojiFile})
		
		err := cmd.Execute()
		assert.NoError(t, err, "JSON format should execute without error")
	})

	t.Run("CSV output format executes successfully", func(t *testing.T) {
		cmd := NewRootCommand()
		emojiFile := filepath.Join(tmpDir, "emoji.go")
		cmd.SetArgs([]string{"scan", "--format", "csv", emojiFile})
		
		err := cmd.Execute()
		assert.NoError(t, err, "CSV format should execute without error")
	})

	t.Run("stats mode executes successfully", func(t *testing.T) {
		cmd := NewRootCommand()
		emojiFile := filepath.Join(tmpDir, "emoji.go")
		cmd.SetArgs([]string{"scan", "--stats", emojiFile})
		
		err := cmd.Execute()
		assert.NoError(t, err, "stats mode should execute without error")
	})

	t.Run("handles non-existent file gracefully", func(t *testing.T) {
		cmd := NewRootCommand()
		nonExistentFile := filepath.Join(tmpDir, "nonexistent.txt")
		cmd.SetArgs([]string{"scan", nonExistentFile})
		
		err := cmd.Execute()
		assert.NoError(t, err, "should handle non-existent files gracefully")
	})

	t.Run("handles directory without recursive flag", func(t *testing.T) {
		cmd := NewRootCommand()
		cmd.SetArgs([]string{"scan", "--recursive=false", tmpDir})
		
		err := cmd.Execute()
		assert.Error(t, err, "should error when scanning directory without recursive flag")
		assert.Contains(t, err.Error(), "requires --recursive")
	})

	t.Run("uses default path when no args provided", func(t *testing.T) {
		cmd := NewRootCommand()
		cmd.SetArgs([]string{"scan"})
		
		err := cmd.Execute()
		assert.NoError(t, err, "should use current directory by default")
	})

	t.Run("threshold mode passes when under limit", func(t *testing.T) {
		cmd := NewRootCommand()
		cleanFile := filepath.Join(tmpDir, "clean.go")
		cmd.SetArgs([]string{"scan", "--threshold", "5", cleanFile})
		
		err := cmd.Execute()
		assert.NoError(t, err, "should pass when emoji count is under threshold")
	})

	t.Run("handles invalid output format", func(t *testing.T) {
		cmd := NewRootCommand()
		emojiFile := filepath.Join(tmpDir, "emoji.go")
		cmd.SetArgs([]string{"scan", "--format", "invalid", emojiFile})
		
		err := cmd.Execute()
		assert.Error(t, err, "should error with invalid output format")
		assert.Contains(t, err.Error(), "unsupported output format")
	})
}

func TestScanHelpers(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("shouldIgnoreDirectory works correctly", func(t *testing.T) {
		ignoreList := []string{".git", "node_modules", "vendor"}
		
		assert.True(t, shouldIgnoreDirectory(filepath.Join(tmpDir, ".git"), ignoreList))
		assert.True(t, shouldIgnoreDirectory(filepath.Join(tmpDir, "node_modules"), ignoreList))
		assert.False(t, shouldIgnoreDirectory(filepath.Join(tmpDir, "src"), ignoreList))
	})

	t.Run("shouldIncludeFile works correctly", func(t *testing.T) {
		profile := config.Profile{
			IncludePatterns: []string{"*.go", "*.md"},
			ExcludePatterns: []string{"*_test.go"},
			FileIgnoreList:  []string{"*.min.js"},
		}
		opts := &ScanOptions{}

		assert.True(t, shouldIncludeFile("main.go", opts, profile))
		assert.True(t, shouldIncludeFile("README.md", opts, profile))
		assert.False(t, shouldIncludeFile("main_test.go", opts, profile)) // Excluded
		assert.False(t, shouldIncludeFile("app.min.js", opts, profile))  // Ignored
		assert.False(t, shouldIncludeFile("main.py", opts, profile))     // Not included
	})

	t.Run("truncateString works correctly", func(t *testing.T) {
		assert.Equal(t, "hello", truncateString("hello", 10))
		assert.Equal(t, "hello...", truncateString("hello world", 8))
		assert.Equal(t, "abc", truncateString("abc", 3))  // No truncation needed
		assert.Equal(t, "...", truncateString("hello", 2)) // Very short maxLen
	})
}

// Example usage for documentation  
func ExampleNewScanCommand() {
	cmd := NewScanCommand()
	cmd.SetArgs([]string{"--help"})
	
	var output bytes.Buffer
	cmd.SetOut(&output)
	
	_ = cmd.Execute()
	
	outputStr := output.String()
	if len(outputStr) > 0 {
		println("Scan command help displayed")
	}
}
