package processor

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/antimoji/antimoji/internal/core/detector"
	"github.com/antimoji/antimoji/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestProcessFile(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("processes file with emojis successfully", func(t *testing.T) {
		content := "Hello 😀 world! This is :) a test."
		filePath := filepath.Join(tmpDir, "test.txt")
		err := os.WriteFile(filePath, []byte(content), 0644)
		assert.NoError(t, err)

		patterns := detector.DefaultEmojiPatterns()
		config := types.DefaultProcessingConfig()

		result := ProcessFile(filePath, patterns, config)
		assert.True(t, result.IsOk())

		processResult := result.Unwrap()
		assert.Equal(t, filePath, processResult.FilePath)
		assert.True(t, processResult.DetectionResult.Success)
		assert.Equal(t, 2, processResult.DetectionResult.TotalCount)
		assert.False(t, processResult.Modified) // No modification in scan mode
		assert.NoError(t, processResult.Error)
	})

	t.Run("processes file without emojis", func(t *testing.T) {
		content := "Hello world! This is a test without any emojis."
		filePath := filepath.Join(tmpDir, "no_emojis.txt")
		err := os.WriteFile(filePath, []byte(content), 0644)
		assert.NoError(t, err)

		patterns := detector.DefaultEmojiPatterns()
		config := types.DefaultProcessingConfig()

		result := ProcessFile(filePath, patterns, config)
		assert.True(t, result.IsOk())

		processResult := result.Unwrap()
		assert.Equal(t, filePath, processResult.FilePath)
		assert.True(t, processResult.DetectionResult.Success)
		assert.Equal(t, 0, processResult.DetectionResult.TotalCount)
		assert.False(t, processResult.Modified)
		assert.NoError(t, processResult.Error)
	})

	t.Run("handles non-existent file", func(t *testing.T) {
		filePath := filepath.Join(tmpDir, "nonexistent.txt")
		patterns := detector.DefaultEmojiPatterns()
		config := types.DefaultProcessingConfig()

		result := ProcessFile(filePath, patterns, config)
		assert.True(t, result.IsOk()) // We return a result with error info

		processResult := result.Unwrap()
		assert.Equal(t, filePath, processResult.FilePath)
		assert.Error(t, processResult.Error)
		assert.False(t, processResult.DetectionResult.Success)
	})

	t.Run("handles binary file", func(t *testing.T) {
		// Create a binary file
		binaryData := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE}
		filePath := filepath.Join(tmpDir, "binary.bin")
		err := os.WriteFile(filePath, binaryData, 0644)
		assert.NoError(t, err)

		patterns := detector.DefaultEmojiPatterns()
		config := types.DefaultProcessingConfig()

		result := ProcessFile(filePath, patterns, config)
		assert.True(t, result.IsOk())

		processResult := result.Unwrap()
		assert.Equal(t, filePath, processResult.FilePath)
		assert.False(t, processResult.DetectionResult.Success) // Should skip binary files
		assert.Equal(t, 0, processResult.DetectionResult.TotalCount)
	})

	t.Run("respects max file size limit", func(t *testing.T) {
		content := "Hello 😀 world!"
		filePath := filepath.Join(tmpDir, "large.txt")
		err := os.WriteFile(filePath, []byte(content), 0644)
		assert.NoError(t, err)

		patterns := detector.DefaultEmojiPatterns()
		config := types.DefaultProcessingConfig()
		config.MaxFileSize = 5 // Very small limit

		result := ProcessFile(filePath, patterns, config)
		assert.True(t, result.IsOk())

		processResult := result.Unwrap()
		assert.Error(t, processResult.Error)
		assert.Contains(t, processResult.Error.Error(), "file too large")
	})

	t.Run("processes empty file", func(t *testing.T) {
		filePath := filepath.Join(tmpDir, "empty.txt")
		err := os.WriteFile(filePath, []byte{}, 0644)
		assert.NoError(t, err)

		patterns := detector.DefaultEmojiPatterns()
		config := types.DefaultProcessingConfig()

		result := ProcessFile(filePath, patterns, config)
		assert.True(t, result.IsOk())

		processResult := result.Unwrap()
		assert.True(t, processResult.DetectionResult.Success)
		assert.Equal(t, 0, processResult.DetectionResult.TotalCount)
		assert.Equal(t, int64(0), processResult.DetectionResult.ProcessedBytes)
	})

	t.Run("respects configuration flags", func(t *testing.T) {
		content := "Unicode 😀, emoticon :), custom :smile:"
		filePath := filepath.Join(tmpDir, "config_test.txt")
		err := os.WriteFile(filePath, []byte(content), 0644)
		assert.NoError(t, err)

		patterns := detector.DefaultEmojiPatterns()
		
		// Test with only Unicode enabled
		config := types.DefaultProcessingConfig()
		config.EnableEmoticons = false
		config.EnableCustom = false

		result := ProcessFile(filePath, patterns, config)
		assert.True(t, result.IsOk())

		processResult := result.Unwrap()
		assert.Equal(t, 1, processResult.DetectionResult.TotalCount) // Only Unicode emoji
		assert.Equal(t, "😀", processResult.DetectionResult.Emojis[0].Emoji)
	})

	t.Run("handles detector error gracefully", func(t *testing.T) {
		content := "Test content"
		filePath := filepath.Join(tmpDir, "detector_error.txt")
		err := os.WriteFile(filePath, []byte(content), 0644)
		assert.NoError(t, err)

		// Use empty patterns that might cause issues
		patterns := types.EmojiPatterns{}
		config := types.DefaultProcessingConfig()

		result := ProcessFile(filePath, patterns, config)
		assert.True(t, result.IsOk()) // Should still return Ok, but with empty results
		
		processResult := result.Unwrap()
		assert.Equal(t, 0, processResult.DetectionResult.TotalCount)
	})
}

func TestProcessFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	files := map[string]string{
		"file1.txt": "Hello 😀 world!",
		"file2.txt": "No emojis here",
		"file3.txt": "Multiple 😃😄 emojis",
	}

	var filePaths []string
	for name, content := range files {
		filePath := filepath.Join(tmpDir, name)
		err := os.WriteFile(filePath, []byte(content), 0644)
		assert.NoError(t, err)
		filePaths = append(filePaths, filePath)
	}

	t.Run("processes multiple files successfully", func(t *testing.T) {
		patterns := detector.DefaultEmojiPatterns()
		config := types.DefaultProcessingConfig()

		results := ProcessFiles(filePaths, patterns, config)
		assert.Len(t, results, 3)

		totalEmojis := 0
		for _, result := range results {
			assert.True(t, result.DetectionResult.Success)
			assert.NoError(t, result.Error)
			totalEmojis += result.DetectionResult.TotalCount
		}

		assert.Equal(t, 3, totalEmojis) // 1 + 0 + 2 emojis
	})

	t.Run("handles mixed success and failure", func(t *testing.T) {
		mixedPaths := append(filePaths, filepath.Join(tmpDir, "nonexistent.txt"))
		patterns := detector.DefaultEmojiPatterns()
		config := types.DefaultProcessingConfig()

		results := ProcessFiles(mixedPaths, patterns, config)
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

func TestCreateProcessingPipeline(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("creates pipeline with correct configuration", func(t *testing.T) {
		config := types.ProcessingConfig{
			EnableUnicode:   true,
			EnableEmoticons: false,
			EnableCustom:    true,
			MaxFileSize:     1024,
			BufferSize:      512,
		}

		pipeline := CreateProcessingPipeline(config)
		assert.NotNil(t, pipeline)
		assert.Equal(t, config, pipeline.Config)
	})

	t.Run("pipeline processes files using its configuration", func(t *testing.T) {
		content := "Hello 😀 world! :)"
		filePath := filepath.Join(tmpDir, "pipeline_test.txt")
		err := os.WriteFile(filePath, []byte(content), 0644)
		assert.NoError(t, err)

		config := types.DefaultProcessingConfig()
		config.EnableEmoticons = false // Disable emoticons

		pipeline := CreateProcessingPipeline(config)
		patterns := detector.DefaultEmojiPatterns()
		
		results := pipeline.Process([]string{filePath}, patterns)
		assert.Len(t, results, 1)
		assert.Equal(t, 1, results[0].DetectionResult.TotalCount) // Only Unicode, no emoticon
	})
}

// Benchmark tests for performance
func BenchmarkProcessFile(b *testing.B) {
	tmpDir := b.TempDir()
	patterns := detector.DefaultEmojiPatterns()
	config := types.DefaultProcessingConfig()

	testCases := []struct {
		name    string
		content string
	}{
		{"no_emojis", "This is plain text without any emojis or special characters."},
		{"single_emoji", "Hello 😀 world!"},
		{"multiple_emojis", "😀😃😄😁😆😅😂🤣"},
		{"mixed_content", "Unicode 😀, emoticon :), custom :smile: mixed together"},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			filePath := filepath.Join(tmpDir, tc.name+".txt")
			err := os.WriteFile(filePath, []byte(tc.content), 0644)
			assert.NoError(b, err)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				result := ProcessFile(filePath, patterns, config)
				if result.IsErr() {
					b.Fatal(result.Error())
				}
				_ = result.Unwrap()
			}
		})
	}
}

func BenchmarkProcessFiles(b *testing.B) {
	tmpDir := b.TempDir()
	patterns := detector.DefaultEmojiPatterns()
	config := types.DefaultProcessingConfig()

	// Create multiple test files
	var filePaths []string
	for i := 0; i < 10; i++ {
		content := "Hello 😀 world! This is test file number " + string(rune('0'+i))
		filePath := filepath.Join(tmpDir, "file"+string(rune('0'+i))+".txt")
		err := os.WriteFile(filePath, []byte(content), 0644)
		assert.NoError(b, err)
		filePaths = append(filePaths, filePath)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		results := ProcessFiles(filePaths, patterns, config)
		if len(results) != len(filePaths) {
			b.Fatal("unexpected number of results")
		}
	}
}

// Example usage for documentation
func ExampleProcessFile() {
	// Create a temporary file for the example
	tmpFile, err := os.CreateTemp("", "example.txt")
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = os.Remove(tmpFile.Name()) // Ignore cleanup error
	}()

	// Write some content with emojis
	content := "Hello 😀 world! :)"
	_, err = tmpFile.WriteString(content)
	if err != nil {
		panic(err)
	}
	_ = tmpFile.Close() // Ignore error in test cleanup

	// Process the file
	patterns := detector.DefaultEmojiPatterns()
	config := types.DefaultProcessingConfig()
	
	result := ProcessFile(tmpFile.Name(), patterns, config)
	if result.IsOk() {
		processResult := result.Unwrap()
		fmt.Println("Found", processResult.DetectionResult.TotalCount, "emojis")
	}
	// Output: Found 2 emojis
}
