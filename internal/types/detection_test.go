package types

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEmojiMatch(t *testing.T) {
	t.Run("creates emoji match with all fields", func(t *testing.T) {
		match := EmojiMatch{
			Emoji:    "😀",
			Start:    5,
			End:      9,
			Line:     2,
			Column:   6,
			Category: CategoryUnicode,
		}

		assert.Equal(t, "😀", match.Emoji)
		assert.Equal(t, 5, match.Start)
		assert.Equal(t, 9, match.End)
		assert.Equal(t, 2, match.Line)
		assert.Equal(t, 6, match.Column)
		assert.Equal(t, CategoryUnicode, match.Category)
	})
}

func TestEmojiCategory(t *testing.T) {
	t.Run("has correct category constants", func(t *testing.T) {
		assert.Equal(t, EmojiCategory("unicode"), CategoryUnicode)
		assert.Equal(t, EmojiCategory("emoticon"), CategoryEmoticon)
		assert.Equal(t, EmojiCategory("custom"), CategoryCustom)
	})
}

func TestDetectionResult(t *testing.T) {
	t.Run("initializes with zero values", func(t *testing.T) {
		result := DetectionResult{}

		assert.Empty(t, result.Emojis)
		assert.Equal(t, 0, result.TotalCount)
		assert.Equal(t, 0, result.UniqueCount)
		assert.Equal(t, int64(0), result.ProcessedBytes)
		assert.Equal(t, time.Duration(0), result.Duration)
		assert.False(t, result.Success)
	})

	t.Run("Reset clears all fields", func(t *testing.T) {
		result := DetectionResult{
			Emojis:         []EmojiMatch{{Emoji: "😀"}},
			TotalCount:     5,
			UniqueCount:    3,
			ProcessedBytes: 1024,
			Duration:       time.Millisecond * 100,
			Success:        true,
		}

		result.Reset()

		assert.Empty(t, result.Emojis)
		assert.Equal(t, 0, result.TotalCount)
		assert.Equal(t, 0, result.UniqueCount)
		assert.Equal(t, int64(0), result.ProcessedBytes)
		assert.Equal(t, time.Duration(0), result.Duration)
		assert.False(t, result.Success)
	})

	t.Run("AddEmoji increments count and appends emoji", func(t *testing.T) {
		result := DetectionResult{}
		match := EmojiMatch{
			Emoji:    "😀",
			Start:    0,
			End:      4,
			Category: CategoryUnicode,
		}

		result.AddEmoji(match)

		assert.Len(t, result.Emojis, 1)
		assert.Equal(t, 1, result.TotalCount)
		assert.Equal(t, match, result.Emojis[0])
	})

	t.Run("Finalize calculates unique count and sets success", func(t *testing.T) {
		result := DetectionResult{}

		// Add duplicate emojis
		result.AddEmoji(EmojiMatch{Emoji: "😀", Category: CategoryUnicode})
		result.AddEmoji(EmojiMatch{Emoji: "😃", Category: CategoryUnicode})
		result.AddEmoji(EmojiMatch{Emoji: "😀", Category: CategoryUnicode}) // duplicate
		result.AddEmoji(EmojiMatch{Emoji: "👍", Category: CategoryUnicode})

		result.Finalize()

		assert.Equal(t, 4, result.TotalCount)
		assert.Equal(t, 3, result.UniqueCount) // 😀, 😃, 👍
		assert.True(t, result.Success)
	})

	t.Run("Finalize handles empty result", func(t *testing.T) {
		result := DetectionResult{}
		result.Finalize()

		assert.Equal(t, 0, result.TotalCount)
		assert.Equal(t, 0, result.UniqueCount)
		assert.True(t, result.Success)
	})
}

func TestUnicodeRange(t *testing.T) {
	t.Run("creates unicode range", func(t *testing.T) {
		urange := UnicodeRange{
			Start: 0x1F600,
			End:   0x1F64F,
			Name:  "Emoticons",
		}

		assert.Equal(t, rune(0x1F600), urange.Start)
		assert.Equal(t, rune(0x1F64F), urange.End)
		assert.Equal(t, "Emoticons", urange.Name)
	})

	t.Run("Contains returns true for rune in range", func(t *testing.T) {
		urange := UnicodeRange{Start: 0x1F600, End: 0x1F64F, Name: "Test"}

		assert.True(t, urange.Contains(0x1F600)) // Start boundary
		assert.True(t, urange.Contains(0x1F620)) // Middle
		assert.True(t, urange.Contains(0x1F64F)) // End boundary
	})

	t.Run("Contains returns false for rune outside range", func(t *testing.T) {
		urange := UnicodeRange{Start: 0x1F600, End: 0x1F64F, Name: "Test"}

		assert.False(t, urange.Contains(0x1F5FF)) // Before start
		assert.False(t, urange.Contains(0x1F650)) // After end
		assert.False(t, urange.Contains(0x0041))  // Way outside (letter A)
	})
}

func TestEmojiPatterns(t *testing.T) {
	t.Run("creates emoji patterns", func(t *testing.T) {
		patterns := EmojiPatterns{
			UnicodeRanges: []UnicodeRange{
				{Start: 0x1F600, End: 0x1F64F, Name: "Emoticons"},
			},
			EmoticonPatterns: []string{`:\)`, `:\(`},
			CustomPatterns:   []string{`:smile:`, `:frown:`},
		}

		assert.Len(t, patterns.UnicodeRanges, 1)
		assert.Len(t, patterns.EmoticonPatterns, 2)
		assert.Len(t, patterns.CustomPatterns, 2)
		assert.Equal(t, "Emoticons", patterns.UnicodeRanges[0].Name)
	})
}

func TestProcessingConfig(t *testing.T) {
	t.Run("DefaultProcessingConfig returns sensible defaults", func(t *testing.T) {
		config := DefaultProcessingConfig()

		assert.True(t, config.EnableUnicode)
		assert.True(t, config.EnableEmoticons)
		assert.True(t, config.EnableCustom)
		assert.Equal(t, int64(100*1024*1024), config.MaxFileSize) // 100MB
		assert.Equal(t, 64*1024, config.BufferSize)               // 64KB
	})

	t.Run("can customize config values", func(t *testing.T) {
		config := ProcessingConfig{
			EnableUnicode:   false,
			EnableEmoticons: true,
			EnableCustom:    false,
			MaxFileSize:     1024,
			BufferSize:      512,
		}

		assert.False(t, config.EnableUnicode)
		assert.True(t, config.EnableEmoticons)
		assert.False(t, config.EnableCustom)
		assert.Equal(t, int64(1024), config.MaxFileSize)
		assert.Equal(t, 512, config.BufferSize)
	})
}

func TestFileInfo(t *testing.T) {
	t.Run("creates file info", func(t *testing.T) {
		info := FileInfo{
			Path: "/path/to/file.go",
			Size: 1024,
		}

		assert.Equal(t, "/path/to/file.go", info.Path)
		assert.Equal(t, int64(1024), info.Size)
	})
}

func TestProcessResult(t *testing.T) {
	t.Run("creates process result with detection result", func(t *testing.T) {
		detectionResult := DetectionResult{
			TotalCount:     3,
			UniqueCount:    2,
			ProcessedBytes: 512,
			Success:        true,
		}

		result := ProcessResult{
			FilePath:        "/path/to/file.go",
			DetectionResult: detectionResult,
			Error:           nil,
			Modified:        false,
			BackupPath:      "",
		}

		assert.Equal(t, "/path/to/file.go", result.FilePath)
		assert.Equal(t, 3, result.DetectionResult.TotalCount)
		assert.Equal(t, 2, result.DetectionResult.UniqueCount)
		assert.NoError(t, result.Error)
		assert.False(t, result.Modified)
		assert.Empty(t, result.BackupPath)
	})

	t.Run("can represent error result", func(t *testing.T) {
		err := assert.AnError
		result := ProcessResult{
			FilePath: "/path/to/file.go",
			Error:    err,
		}

		assert.Equal(t, "/path/to/file.go", result.FilePath)
		assert.Equal(t, err, result.Error)
	})
}

// Benchmark tests for performance-critical operations
func BenchmarkDetectionResult_AddEmoji(b *testing.B) {
	result := DetectionResult{}
	match := EmojiMatch{
		Emoji:    "😀",
		Start:    0,
		End:      4,
		Category: CategoryUnicode,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result.Reset()
		result.AddEmoji(match)
	}
}

func BenchmarkDetectionResult_Finalize(b *testing.B) {
	// Create result with many duplicate emojis
	result := DetectionResult{}
	emojis := []string{"😀", "😃", "😄", "😁", "😆", "😅", "😂", "🤣"}

	for i := 0; i < 100; i++ {
		for _, emoji := range emojis {
			result.AddEmoji(EmojiMatch{
				Emoji:    emoji,
				Category: CategoryUnicode,
			})
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create a copy to avoid modifying the benchmark data
		testResult := result
		testResult.Finalize()
	}
}

func BenchmarkUnicodeRange_Contains(b *testing.B) {
	urange := UnicodeRange{Start: 0x1F600, End: 0x1F64F, Name: "Emoticons"}
	testRune := rune(0x1F620)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = urange.Contains(testRune)
	}
}

// Example usage for documentation
func ExampleDetectionResult_AddEmoji() {
	result := DetectionResult{}

	match := EmojiMatch{
		Emoji:    "😀",
		Start:    5,
		End:      9,
		Line:     1,
		Column:   6,
		Category: CategoryUnicode,
	}

	result.AddEmoji(match)
	result.Finalize()

	fmt.Printf("Found %d emojis (%d unique)\n", result.TotalCount, result.UniqueCount)
	// Output: Found 1 emojis (1 unique)
}

func ExampleUnicodeRange_Contains() {
	// Define the basic emoticons Unicode range
	emoticons := UnicodeRange{
		Start: 0x1F600,
		End:   0x1F64F,
		Name:  "Emoticons",
	}

	// Test if a grinning face emoji is in the range
	grinningFace := rune(0x1F600) // 😀
	fmt.Printf("Is 😀 an emoticon? %t\n", emoticons.Contains(grinningFace))
	// Output: Is 😀 an emoticon? true
}
