package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCleanCommand(t *testing.T) {
	t.Run("creates clean command with correct properties", func(t *testing.T) {
		cmd := NewCleanCommand()

		assert.Equal(t, "clean [flags] [path...]", cmd.Use)
		assert.Contains(t, cmd.Short, "Remove emojis from files")
		assert.NotEmpty(t, cmd.Long)
	})

	t.Run("has expected flags", func(t *testing.T) {
		cmd := NewCleanCommand()
		flags := cmd.Flags()

		recursiveFlag := flags.Lookup("recursive")
		assert.NotNil(t, recursiveFlag)

		backupFlag := flags.Lookup("backup")
		assert.NotNil(t, backupFlag)

		replaceFlag := flags.Lookup("replace")
		assert.NotNil(t, replaceFlag)

		inPlaceFlag := flags.Lookup("in-place")
		assert.NotNil(t, inPlaceFlag)

		respectAllowlistFlag := flags.Lookup("respect-allowlist")
		assert.NotNil(t, respectAllowlistFlag)

		statsFlag := flags.Lookup("stats")
		assert.NotNil(t, statsFlag)
	})
}

func TestCleanCommand_Integration(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	testFiles := map[string]string{
		"emoji.go":  "package main\n\n// Hello 😀 world\nfunc main() {}",
		"clean.go":  "package main\n\nfunc main() {\n\tfmt.Println(\"Hello world\")\n}",
		"mixed.txt": "Status: ✅ done, 😀 happy, ❌ failed",
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(tmpDir, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		assert.NoError(t, err)
	}

	t.Run("cleans single file with emojis", func(t *testing.T) {
		cmd := NewRootCommand()

		emojiFile := filepath.Join(tmpDir, "emoji.go")
		cmd.SetArgs([]string{"clean", "--in-place", emojiFile})

		err := cmd.Execute()
		assert.NoError(t, err, "clean command should execute without error")

		// Check that file was modified
		modifiedContent, readErr := os.ReadFile(emojiFile)
		assert.NoError(t, readErr)
		assert.NotContains(t, string(modifiedContent), "😀")
		assert.Contains(t, string(modifiedContent), "// Hello  world") // Emoji removed
	})

	t.Run("creates backup when requested", func(t *testing.T) {
		// Create a fresh test file
		testContent := "Hello 😃 backup test!"
		backupTestFile := filepath.Join(tmpDir, "backup_test.go")
		err := os.WriteFile(backupTestFile, []byte(testContent), 0644)
		assert.NoError(t, err)

		cmd := NewRootCommand()
		cmd.SetArgs([]string{"clean", "--backup", "--in-place", backupTestFile})

		err = cmd.Execute()
		assert.NoError(t, err)

		// Check that backup file exists
		files, err := filepath.Glob(filepath.Join(tmpDir, "backup_test.backup.*"))
		assert.NoError(t, err)
		assert.NotEmpty(t, files, "backup file should be created")

		// Check backup content
		backupContent, err := os.ReadFile(files[0])
		assert.NoError(t, err)
		assert.Equal(t, testContent, string(backupContent))
	})

	t.Run("dry-run mode shows changes without modifying", func(t *testing.T) {
		// Create a fresh test file
		testContent := "Hello 😄 dry run test!"
		dryRunFile := filepath.Join(tmpDir, "dry_run.go")
		err := os.WriteFile(dryRunFile, []byte(testContent), 0644)
		assert.NoError(t, err)

		cmd := NewRootCommand()
		cmd.SetArgs([]string{"clean", "--dry-run", dryRunFile})

		err = cmd.Execute()
		assert.NoError(t, err)

		// File should be unchanged in dry-run mode
		unchangedContent, err := os.ReadFile(dryRunFile)
		assert.NoError(t, err)
		assert.Equal(t, testContent, string(unchangedContent))
	})

	t.Run("respects allowlist configuration", func(t *testing.T) {
		// Create config with allowlist
		configContent := `profiles:
  default:
    emoji_allowlist:
      - "✅"
      - "❌"`

		configPath := filepath.Join(tmpDir, "allowlist_config.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		assert.NoError(t, err)

		// Create test file with mixed emojis
		testContent := "Status: ✅ done, 😀 happy, ❌ failed"
		allowlistFile := filepath.Join(tmpDir, "allowlist_test.go")
		err = os.WriteFile(allowlistFile, []byte(testContent), 0644)
		assert.NoError(t, err)

		cmd := NewRootCommand()
		cmd.SetArgs([]string{"clean", "--config", configPath, "--respect-allowlist", "--in-place", allowlistFile})

		err = cmd.Execute()
		assert.NoError(t, err)

		// Check that allowlisted emojis remain, others are removed
		modifiedContent, err := os.ReadFile(allowlistFile)
		assert.NoError(t, err)
		assert.Contains(t, string(modifiedContent), "✅")    // Should remain
		assert.Contains(t, string(modifiedContent), "❌")    // Should remain
		assert.NotContains(t, string(modifiedContent), "😀") // Should be removed
	})

	t.Run("custom replacement works", func(t *testing.T) {
		// Create a fresh test file
		testContent := "Hello 😅 world with 🎉 celebration!"
		replaceFile := filepath.Join(tmpDir, "replace_test.go")
		err := os.WriteFile(replaceFile, []byte(testContent), 0644)
		assert.NoError(t, err)

		cmd := NewRootCommand()
		cmd.SetArgs([]string{"clean", "--replace", "[EMOJI]", "--in-place", replaceFile})

		err = cmd.Execute()
		assert.NoError(t, err)

		// Check that emojis were replaced with custom text
		modifiedContent, err := os.ReadFile(replaceFile)
		assert.NoError(t, err)
		assert.Contains(t, string(modifiedContent), "[EMOJI]")
		assert.NotContains(t, string(modifiedContent), "😅")
		assert.NotContains(t, string(modifiedContent), "🎉")
	})

	t.Run("processes directory recursively", func(t *testing.T) {
		cmd := NewRootCommand()
		cmd.SetArgs([]string{"clean", "--recursive", "--dry-run", tmpDir})

		err := cmd.Execute()
		assert.NoError(t, err, "recursive clean should execute without error")
	})

	t.Run("handles non-existent file gracefully", func(t *testing.T) {
		cmd := NewRootCommand()
		nonExistentFile := filepath.Join(tmpDir, "nonexistent.go")
		cmd.SetArgs([]string{"clean", "--in-place", nonExistentFile})

		err := cmd.Execute()
		assert.NoError(t, err, "should handle non-existent files gracefully")
	})
}

func TestCleanHelpers(t *testing.T) {
	t.Run("validateCleanOptions works correctly", func(t *testing.T) {
		// Test valid options
		opts := &CleanOptions{InPlace: true}
		assert.NoError(t, validateCleanOptions(opts))

		// Test invalid options (no output method specified)
		invalidOpts := &CleanOptions{InPlace: false}
		assert.Error(t, validateCleanOptions(invalidOpts))
	})
}

// Example usage for documentation
func ExampleNewCleanCommand() {
	cmd := NewCleanCommand()
	cmd.SetArgs([]string{"--help"})

	// This would show the clean command help
	_ = cmd.Execute()
}
