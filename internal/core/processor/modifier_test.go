package processor

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/antimoji/antimoji/internal/core/allowlist"
	"github.com/antimoji/antimoji/internal/core/detector"
	"github.com/antimoji/antimoji/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestModifyFile(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("removes emojis from file successfully", func(t *testing.T) {
		originalContent := "Hello 😀 world! This is :) a test."
		expectedContent := "Hello  world! This is  a test."
		filePath := filepath.Join(tmpDir, "test.txt")

		err := os.WriteFile(filePath, []byte(originalContent), 0644)
		assert.NoError(t, err)

		patterns := detector.DefaultEmojiPatterns()
		modifyConfig := ModifyConfig{
			Replacement:      "",
			CreateBackup:     false,
			RespectAllowlist: false,
		}

		result := ModifyFile(filePath, patterns, modifyConfig, nil)
		assert.True(t, result.IsOk())

		modifyResult := result.Unwrap()
		assert.True(t, modifyResult.Success)
		assert.True(t, modifyResult.Modified)
		assert.Equal(t, 2, modifyResult.EmojisRemoved)
		assert.NoError(t, modifyResult.Error)

		// Check file content was modified
		modifiedContent, err := os.ReadFile(filePath)
		assert.NoError(t, err)
		assert.Equal(t, expectedContent, string(modifiedContent))
	})

	t.Run("replaces emojis with custom replacement", func(t *testing.T) {
		originalContent := "Hello 😀 world! This is :) a test."
		expectedContent := "Hello [EMOJI] world! This is [EMOJI] a test."
		filePath := filepath.Join(tmpDir, "replace.txt")

		err := os.WriteFile(filePath, []byte(originalContent), 0644)
		assert.NoError(t, err)

		patterns := detector.DefaultEmojiPatterns()
		modifyConfig := ModifyConfig{
			Replacement:      "[EMOJI]",
			CreateBackup:     false,
			RespectAllowlist: false,
		}

		result := ModifyFile(filePath, patterns, modifyConfig, nil)
		assert.True(t, result.IsOk())

		modifyResult := result.Unwrap()
		assert.True(t, modifyResult.Modified)
		assert.Equal(t, 2, modifyResult.EmojisRemoved)

		// Check file content was modified
		modifiedContent, err := os.ReadFile(filePath)
		assert.NoError(t, err)
		assert.Equal(t, expectedContent, string(modifiedContent))
	})

	t.Run("creates backup when requested", func(t *testing.T) {
		originalContent := "Hello 😀 world!"
		filePath := filepath.Join(tmpDir, "backup_test.txt")

		err := os.WriteFile(filePath, []byte(originalContent), 0644)
		assert.NoError(t, err)

		patterns := detector.DefaultEmojiPatterns()
		modifyConfig := ModifyConfig{
			Replacement:      "",
			CreateBackup:     true,
			RespectAllowlist: false,
		}

		result := ModifyFile(filePath, patterns, modifyConfig, nil)
		assert.True(t, result.IsOk())

		modifyResult := result.Unwrap()
		assert.True(t, modifyResult.Modified)
		assert.NotEmpty(t, modifyResult.BackupPath)

		// Check backup file exists and contains original content
		backupContent, err := os.ReadFile(modifyResult.BackupPath)
		assert.NoError(t, err)
		assert.Equal(t, originalContent, string(backupContent))

		// Check original file was modified
		modifiedContent, err := os.ReadFile(filePath)
		assert.NoError(t, err)
		assert.Equal(t, "Hello  world!", string(modifiedContent))
	})

	t.Run("respects allowlist when configured", func(t *testing.T) {
		originalContent := "Status: ✅ done, 😀 happy, ❌ failed"
		expectedContent := "Status: ✅ done,  happy, ❌ failed" // Only 😀 removed
		filePath := filepath.Join(tmpDir, "allowlist_test.txt")

		err := os.WriteFile(filePath, []byte(originalContent), 0644)
		assert.NoError(t, err)

		patterns := detector.DefaultEmojiPatterns()
		emojiAllowlist := allowlist.NewAllowlist([]string{"✅", "❌"}).Unwrap()
		modifyConfig := ModifyConfig{
			Replacement:      "",
			CreateBackup:     false,
			RespectAllowlist: true,
		}

		result := ModifyFile(filePath, patterns, modifyConfig, emojiAllowlist)
		assert.True(t, result.IsOk())

		modifyResult := result.Unwrap()
		assert.True(t, modifyResult.Modified)
		assert.Equal(t, 1, modifyResult.EmojisRemoved) // Only 😀 removed

		// Check file content - allowlisted emojis should remain
		modifiedContent, err := os.ReadFile(filePath)
		assert.NoError(t, err)
		assert.Equal(t, expectedContent, string(modifiedContent))
	})

	t.Run("handles file with no emojis", func(t *testing.T) {
		originalContent := "Hello world! No emojis here."
		filePath := filepath.Join(tmpDir, "no_emojis.txt")

		err := os.WriteFile(filePath, []byte(originalContent), 0644)
		assert.NoError(t, err)

		patterns := detector.DefaultEmojiPatterns()
		modifyConfig := ModifyConfig{
			Replacement:      "",
			CreateBackup:     false,
			RespectAllowlist: false,
		}

		result := ModifyFile(filePath, patterns, modifyConfig, nil)
		assert.True(t, result.IsOk())

		modifyResult := result.Unwrap()
		assert.True(t, modifyResult.Success)
		assert.False(t, modifyResult.Modified) // No modification needed
		assert.Equal(t, 0, modifyResult.EmojisRemoved)

		// File should be unchanged
		content, err := os.ReadFile(filePath)
		assert.NoError(t, err)
		assert.Equal(t, originalContent, string(content))
	})

	t.Run("handles non-existent file", func(t *testing.T) {
		filePath := filepath.Join(tmpDir, "nonexistent.txt")
		patterns := detector.DefaultEmojiPatterns()
		modifyConfig := ModifyConfig{
			Replacement:      "",
			CreateBackup:     false,
			RespectAllowlist: false,
		}

		result := ModifyFile(filePath, patterns, modifyConfig, nil)
		assert.True(t, result.IsOk())

		modifyResult := result.Unwrap()
		assert.False(t, modifyResult.Success)
		assert.False(t, modifyResult.Modified)
		assert.Error(t, modifyResult.Error)
	})

	t.Run("preserves file permissions", func(t *testing.T) {
		originalContent := "Hello 😀 world!"
		filePath := filepath.Join(tmpDir, "permissions.txt")

		err := os.WriteFile(filePath, []byte(originalContent), 0755)
		assert.NoError(t, err)

		// Get original permissions
		originalStat, err := os.Stat(filePath)
		assert.NoError(t, err)
		originalMode := originalStat.Mode()

		patterns := detector.DefaultEmojiPatterns()
		modifyConfig := ModifyConfig{
			Replacement:      "",
			CreateBackup:     false,
			RespectAllowlist: false,
		}

		result := ModifyFile(filePath, patterns, modifyConfig, nil)
		assert.True(t, result.IsOk())

		// Check permissions are preserved
		modifiedStat, err := os.Stat(filePath)
		assert.NoError(t, err)
		assert.Equal(t, originalMode, modifiedStat.Mode())
	})
}

func TestCreateBackup(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("creates backup with timestamp", func(t *testing.T) {
		originalContent := "Hello 😀 world!"
		filePath := filepath.Join(tmpDir, "backup.txt")

		err := os.WriteFile(filePath, []byte(originalContent), 0644)
		assert.NoError(t, err)

		result := CreateBackup(filePath)
		assert.True(t, result.IsOk())

		backupPath := result.Unwrap()
		assert.NotEqual(t, filePath, backupPath)
		assert.Contains(t, backupPath, "backup")
		assert.Contains(t, backupPath, ".backup.")

		// Check backup content matches original
		backupContent, err := os.ReadFile(backupPath)
		assert.NoError(t, err)
		assert.Equal(t, originalContent, string(backupContent))
	})

	t.Run("handles non-existent file", func(t *testing.T) {
		filePath := filepath.Join(tmpDir, "nonexistent.txt")

		result := CreateBackup(filePath)
		assert.True(t, result.IsErr())
	})

	t.Run("handles permission denied", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Skipping permission test on Windows due to different permission model")
		}
		if os.Getuid() == 0 {
			t.Skip("Skipping permission test when running as root")
		}

		filePath := filepath.Join(tmpDir, "noperm.txt")
		err := os.WriteFile(filePath, []byte("test"), 0000)
		assert.NoError(t, err)
		defer func() {
			_ = os.Chmod(filePath, 0644)
		}()

		result := CreateBackup(filePath)
		assert.True(t, result.IsErr())
	})
}

func TestAtomicWriteFile(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("writes file atomically", func(t *testing.T) {
		content := "Hello world!"
		filePath := filepath.Join(tmpDir, "atomic.txt")

		result := AtomicWriteFile(filePath, []byte(content), 0644)
		assert.True(t, result.IsOk())

		// Check file exists and has correct content
		fileContent, err := os.ReadFile(filePath)
		assert.NoError(t, err)
		assert.Equal(t, content, string(fileContent))
	})

	t.Run("overwrites existing file", func(t *testing.T) {
		originalContent := "Original content"
		newContent := "New content"
		filePath := filepath.Join(tmpDir, "overwrite.txt")

		// Create original file
		err := os.WriteFile(filePath, []byte(originalContent), 0644)
		assert.NoError(t, err)

		// Overwrite atomically
		result := AtomicWriteFile(filePath, []byte(newContent), 0644)
		assert.True(t, result.IsOk())

		// Check content was updated
		fileContent, err := os.ReadFile(filePath)
		assert.NoError(t, err)
		assert.Equal(t, newContent, string(fileContent))
	})

	t.Run("preserves permissions on existing file", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Skipping permission preservation test on Windows due to different permission model")
		}
		
		originalContent := "Original content"
		newContent := "New content"
		filePath := filepath.Join(tmpDir, "preserve_perms.txt")

		// Create original file with specific permissions
		err := os.WriteFile(filePath, []byte(originalContent), 0755)
		assert.NoError(t, err)

		// Overwrite with different permissions parameter
		result := AtomicWriteFile(filePath, []byte(newContent), 0644)
		assert.True(t, result.IsOk())

		// Check original permissions are preserved
		stat, err := os.Stat(filePath)
		assert.NoError(t, err)
		assert.Equal(t, os.FileMode(0755), stat.Mode().Perm())
	})

	t.Run("handles permission denied directory", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Skipping permission test on Windows due to different permission model")
		}
		if os.Getuid() == 0 {
			t.Skip("Skipping permission test when running as root")
		}

		// Create directory with no write permissions
		noWriteDir := filepath.Join(tmpDir, "nowrite")
		err := os.Mkdir(noWriteDir, 0555) // Read and execute only
		assert.NoError(t, err)
		defer func() {
			_ = os.Chmod(noWriteDir, 0755)
		}()

		filePath := filepath.Join(noWriteDir, "test.txt")
		result := AtomicWriteFile(filePath, []byte("test"), 0644)
		assert.True(t, result.IsErr())
	})
}

func TestRemoveEmojis(t *testing.T) {
	t.Run("removes all detected emojis", func(t *testing.T) {
		content := "Hello 😀 world! This is :) a test with :smile: patterns."
		expected := "Hello  world! This is  a test with  patterns."

		patterns := detector.DefaultEmojiPatterns()
		detectionResult := detector.DetectEmojis([]byte(content), patterns).Unwrap()

		result := RemoveEmojis(content, detectionResult, "")
		assert.Equal(t, expected, result)
	})

	t.Run("replaces emojis with custom replacement", func(t *testing.T) {
		content := "Hello 😀 world! This is :) a test."
		expected := "Hello [REMOVED] world! This is [REMOVED] a test."

		patterns := detector.DefaultEmojiPatterns()
		detectionResult := detector.DetectEmojis([]byte(content), patterns).Unwrap()

		result := RemoveEmojis(content, detectionResult, "[REMOVED]")
		assert.Equal(t, expected, result)
	})

	t.Run("handles empty detection result", func(t *testing.T) {
		content := "Hello world! No emojis here."

		emptyResult := types.DetectionResult{
			Emojis:     []types.EmojiMatch{},
			TotalCount: 0,
			Success:    true,
		}

		result := RemoveEmojis(content, emptyResult, "")
		assert.Equal(t, content, result) // Should be unchanged
	})

	t.Run("handles overlapping emojis correctly", func(t *testing.T) {
		// Test with emojis that might have overlapping byte ranges
		content := "😀😃😄"  // Adjacent emojis
		expected := "   " // Three spaces (one per emoji)

		patterns := detector.DefaultEmojiPatterns()
		detectionResult := detector.DetectEmojis([]byte(content), patterns).Unwrap()

		result := RemoveEmojis(content, detectionResult, " ")
		assert.Equal(t, expected, result)
	})

	t.Run("preserves non-emoji content exactly", func(t *testing.T) {
		content := "Hello 😀 world!\nLine 2 with :) content.\n\tTabbed content."

		patterns := detector.DefaultEmojiPatterns()
		detectionResult := detector.DetectEmojis([]byte(content), patterns).Unwrap()

		result := RemoveEmojis(content, detectionResult, "")

		// Should preserve whitespace, newlines, tabs exactly
		assert.Contains(t, result, "Hello  world!")
		assert.Contains(t, result, "\nLine 2 with  content.")
		assert.Contains(t, result, "\n\tTabbed content.")
	})
}

func TestModifyConfig(t *testing.T) {
	t.Run("DefaultModifyConfig returns sensible defaults", func(t *testing.T) {
		config := DefaultModifyConfig()

		assert.Equal(t, "", config.Replacement)
		assert.False(t, config.CreateBackup)
		assert.True(t, config.RespectAllowlist)
		assert.True(t, config.PreservePermissions)
		assert.False(t, config.DryRun)
	})

	t.Run("can customize config values", func(t *testing.T) {
		config := ModifyConfig{
			Replacement:         "[EMOJI]",
			CreateBackup:        true,
			RespectAllowlist:    false,
			PreservePermissions: false,
			DryRun:              true,
		}

		assert.Equal(t, "[EMOJI]", config.Replacement)
		assert.True(t, config.CreateBackup)
		assert.False(t, config.RespectAllowlist)
		assert.False(t, config.PreservePermissions)
		assert.True(t, config.DryRun)
	})
}

func TestModifyFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	testFiles := map[string]string{
		"file1.txt": "Hello 😀 world!",
		"file2.txt": "No emojis here",
		"file3.txt": "Multiple 😃😄 emojis",
	}

	var filePaths []string
	for name, content := range testFiles {
		filePath := filepath.Join(tmpDir, name)
		err := os.WriteFile(filePath, []byte(content), 0644)
		assert.NoError(t, err)
		filePaths = append(filePaths, filePath)
	}

	t.Run("modifies multiple files successfully", func(t *testing.T) {
		patterns := detector.DefaultEmojiPatterns()
		modifyConfig := DefaultModifyConfig()

		results := ModifyFiles(filePaths, patterns, modifyConfig, nil)
		assert.Len(t, results, 3)

		totalRemoved := 0
		modifiedCount := 0
		for _, result := range results {
			assert.True(t, result.Success)
			assert.NoError(t, result.Error)
			totalRemoved += result.EmojisRemoved
			if result.Modified {
				modifiedCount++
			}
		}

		assert.Equal(t, 3, totalRemoved)  // 1 + 0 + 2 emojis removed
		assert.Equal(t, 2, modifiedCount) // 2 files actually modified
	})

	t.Run("handles mixed success and failure", func(t *testing.T) {
		mixedPaths := append(filePaths, filepath.Join(tmpDir, "nonexistent.txt"))
		patterns := detector.DefaultEmojiPatterns()
		modifyConfig := DefaultModifyConfig()

		results := ModifyFiles(mixedPaths, patterns, modifyConfig, nil)
		assert.Len(t, results, 4)

		successCount := 0
		errorCount := 0
		for _, result := range results {
			if result.Error == nil {
				successCount++
			} else {
				errorCount++
			}
		}

		assert.Equal(t, 3, successCount)
		assert.Equal(t, 1, errorCount)
	})
}

// Benchmark tests for performance
func BenchmarkModifyFile(b *testing.B) {
	tmpDir := b.TempDir()
	patterns := detector.DefaultEmojiPatterns()
	modifyConfig := DefaultModifyConfig()

	testCases := []struct {
		name    string
		content string
	}{
		{"no_emojis", "This is plain text without any emojis."},
		{"single_emoji", "Hello 😀 world!"},
		{"multiple_emojis", "😀😃😄😁😆"},
		{"mixed_content", "Unicode 😀, emoticon :), custom :smile:"},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			filePath := filepath.Join(tmpDir, tc.name+".txt")

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				// Recreate file for each iteration
				err := os.WriteFile(filePath, []byte(tc.content), 0644)
				if err != nil {
					b.Fatal(err)
				}

				result := ModifyFile(filePath, patterns, modifyConfig, nil)
				if result.IsErr() {
					b.Fatal(result.Error())
				}
			}
		})
	}
}

func BenchmarkRemoveEmojis(b *testing.B) {
	content := "Hello 😀 world! This is :) a test with :smile: patterns and more 😃😄 emojis."
	patterns := detector.DefaultEmojiPatterns()
	detectionResult := detector.DetectEmojis([]byte(content), patterns).Unwrap()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = RemoveEmojis(content, detectionResult, "")
	}
}

// Example usage for documentation
func ExampleModifyFile() {
	// Create a temporary file for the example
	tmpFile, err := os.CreateTemp("", "example.txt")
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	// Write content with emojis
	content := "Hello 😀 world! :)"
	_, err = tmpFile.WriteString(content)
	if err != nil {
		panic(err)
	}
	_ = tmpFile.Close()

	// Modify the file to remove emojis
	patterns := detector.DefaultEmojiPatterns()
	config := DefaultModifyConfig()

	result := ModifyFile(tmpFile.Name(), patterns, config, nil)
	if result.IsOk() {
		modifyResult := result.Unwrap()
		fmt.Printf("Removed %d emojis, modified: %t\n",
			modifyResult.EmojisRemoved, modifyResult.Modified)
	}
	// Output: Removed 2 emojis, modified: true
}
