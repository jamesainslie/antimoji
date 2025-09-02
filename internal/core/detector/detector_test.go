package detector

import (
	"fmt"
	"testing"

	"github.com/antimoji/antimoji/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestDetectEmojis(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []types.EmojiMatch
		wantErr  bool
	}{
		{
			name:     "empty content returns no emojis",
			content:  "",
			expected: []types.EmojiMatch{},
			wantErr:  false,
		},
		{
			name:    "plain text without emojis",
			content: "Hello world! This is plain text.",
			expected: []types.EmojiMatch{},
			wantErr: false,
		},
		{
			name:    "single unicode emoji",
			content: "Hello 😀 world",
			expected: []types.EmojiMatch{
				{
					Emoji:    "😀",
					Start:    6,
					End:      10,
					Line:     1,
					Column:   7,
					Category: types.CategoryUnicode,
				},
			},
			wantErr: false,
		},
		{
			name:    "multiple unicode emojis",
			content: "😀😃😄",
			expected: []types.EmojiMatch{
				{
					Emoji:    "😀",
					Start:    0,
					End:      4,
					Line:     1,
					Column:   1,
					Category: types.CategoryUnicode,
				},
				{
					Emoji:    "😃",
					Start:    4,
					End:      8,
					Line:     1,
					Column:   2,
					Category: types.CategoryUnicode,
				},
				{
					Emoji:    "😄",
					Start:    8,
					End:      12,
					Line:     1,
					Column:   3,
					Category: types.CategoryUnicode,
				},
			},
			wantErr: false,
		},
		{
			name:    "emoji with skin tone modifier",
			content: "👍🏽",
			expected: []types.EmojiMatch{
				{
					Emoji:    "👍🏽",
					Start:    0,
					End:      8,
					Line:     1,
					Column:   1,
					Category: types.CategoryUnicode,
				},
			},
			wantErr: false,
		},
		{
			name:    "multiline content with emojis",
			content: "Line 1 😀\nLine 2 😃\nLine 3",
			expected: []types.EmojiMatch{
				{
					Emoji:    "😀",
					Start:    7,
					End:      11,
					Line:     1,
					Column:   8,
					Category: types.CategoryUnicode,
				},
				{
					Emoji:    "😃",
					Start:    19,
					End:      23,
					Line:     2,
					Column:   8,
					Category: types.CategoryUnicode,
				},
			},
			wantErr: false,
		},
		{
			name:    "text emoticons",
			content: "Happy :) and sad :(",
			expected: []types.EmojiMatch{
				{
					Emoji:    ":)",
					Start:    6,
					End:      8,
					Line:     1,
					Column:   7,
					Category: types.CategoryEmoticon,
				},
				{
					Emoji:    ":(",
					Start:    17,
					End:      19,
					Line:     1,
					Column:   18,
					Category: types.CategoryEmoticon,
				},
			},
			wantErr: false,
		},
		{
			name:    "custom emoji patterns",
			content: "I'm :smile: and you're :thumbs_up:",
			expected: []types.EmojiMatch{
				{
					Emoji:    ":smile:",
					Start:    4,
					End:      11,
					Line:     1,
					Column:   5,
					Category: types.CategoryCustom,
				},
				{
					Emoji:    ":thumbs_up:",
					Start:    23,
					End:      34,
					Line:     1,
					Column:   24,
					Category: types.CategoryCustom,
				},
			},
			wantErr: false,
		},
		{
			name:    "mixed emoji types",
			content: "Unicode 😀, emoticon :), custom :smile:",
			expected: []types.EmojiMatch{
				{
					Emoji:    "😀",
					Start:    8,
					End:      12,
					Line:     1,
					Column:   9,
					Category: types.CategoryUnicode,
				},
				{
					Emoji:    ":)",
					Start:    23,
					End:      25,
					Line:     1,
					Column:   21,
					Category: types.CategoryEmoticon,
				},
				{
					Emoji:    ":smile:",
					Start:    34,
					End:      41,
					Line:     1,
					Column:   32,
					Category: types.CategoryCustom,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patterns := DefaultEmojiPatterns()
			result := DetectEmojis([]byte(tt.content), patterns)

			if tt.wantErr {
				assert.True(t, result.IsErr())
				return
			}

			assert.True(t, result.IsOk())
			detection := result.Unwrap()
			assert.Equal(t, len(tt.expected), len(detection.Emojis))
			
			for i, expected := range tt.expected {
				if i < len(detection.Emojis) {
					actual := detection.Emojis[i]
					assert.Equal(t, expected.Emoji, actual.Emoji, "emoji mismatch at index %d", i)
					assert.Equal(t, expected.Start, actual.Start, "start position mismatch at index %d", i)
					assert.Equal(t, expected.End, actual.End, "end position mismatch at index %d", i)
					assert.Equal(t, expected.Line, actual.Line, "line mismatch at index %d", i)
					assert.Equal(t, expected.Column, actual.Column, "column mismatch at index %d", i)
					assert.Equal(t, expected.Category, actual.Category, "category mismatch at index %d", i)
				}
			}
		})
	}
}

func TestDetectEmojis_Properties(t *testing.T) {
	patterns := DefaultEmojiPatterns()

	t.Run("detection is deterministic", func(t *testing.T) {
		testInputs := []string{
			"Hello 😀 world",
			"😀😃😄",
			"No emojis here",
			"Mixed :) 😀 :smile:",
		}

		for _, input := range testInputs {
			result1 := DetectEmojis([]byte(input), patterns)
			result2 := DetectEmojis([]byte(input), patterns)

			assert.True(t, result1.IsOk())
			assert.True(t, result2.IsOk())
			assert.Equal(t, result1.Unwrap().Emojis, result2.Unwrap().Emojis)
		}
	})

	t.Run("empty input produces empty result", func(t *testing.T) {
		result := DetectEmojis([]byte{}, patterns)
		assert.True(t, result.IsOk())
		detection := result.Unwrap()
		assert.Empty(t, detection.Emojis)
		assert.Equal(t, 0, detection.TotalCount)
	})

	t.Run("detection preserves input content", func(t *testing.T) {
		original := []byte("Hello 😀 world")
		backup := make([]byte, len(original))
		copy(backup, original)

		result := DetectEmojis(original, patterns)
		assert.True(t, result.IsOk())
		assert.Equal(t, backup, original, "input content should not be modified")
	})
}

func TestDefaultEmojiPatterns(t *testing.T) {
	t.Run("returns valid default patterns", func(t *testing.T) {
		patterns := DefaultEmojiPatterns()

		assert.NotEmpty(t, patterns.UnicodeRanges, "should include Unicode ranges")
		assert.NotEmpty(t, patterns.EmoticonPatterns, "should include emoticon patterns")
		assert.NotEmpty(t, patterns.CustomPatterns, "should include custom patterns")
	})

	t.Run("includes common unicode ranges", func(t *testing.T) {
		patterns := DefaultEmojiPatterns()

		// Check for basic emoticons range
		hasEmoticons := false
		for _, urange := range patterns.UnicodeRanges {
			if urange.Start == 0x1F600 && urange.End == 0x1F64F {
				hasEmoticons = true
				break
			}
		}
		assert.True(t, hasEmoticons, "should include basic emoticons range")
	})

	t.Run("includes common emoticon patterns", func(t *testing.T) {
		patterns := DefaultEmojiPatterns()

		commonEmoticons := []string{`:)`, `:(`}
		for _, emoticon := range commonEmoticons {
			found := false
			for _, pattern := range patterns.EmoticonPatterns {
				if pattern == emoticon {
					found = true
					break
				}
			}
			assert.True(t, found, "should include emoticon pattern: %s", emoticon)
		}
	})
}

// Benchmark tests for performance
func BenchmarkDetectEmojis(b *testing.B) {
	patterns := DefaultEmojiPatterns()
	testCases := []struct {
		name    string
		content string
	}{
		{"empty", ""},
		{"no_emojis", "This is plain text without any emojis or special characters."},
		{"single_emoji", "Hello 😀 world"},
		{"multiple_emojis", "😀😃😄😁😆😅😂🤣"},
		{"mixed_content", "Unicode 😀, emoticon :), custom :smile: mixed together"},
		{"large_text", generateLargeText(1024)},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			content := []byte(tc.content)
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				result := DetectEmojis(content, patterns)
				_ = result.Unwrap() // Force evaluation
			}

			b.ReportMetric(float64(len(content)*b.N)/b.Elapsed().Seconds(), "bytes/sec")
		})
	}
}

func BenchmarkDetectEmojis_LargeFile(b *testing.B) {
	patterns := DefaultEmojiPatterns()
	sizes := []int{1024, 64 * 1024, 1024 * 1024} // 1KB, 64KB, 1MB

	for _, size := range sizes {
		b.Run(fmt.Sprintf("%dKB", size/1024), func(b *testing.B) {
			content := []byte(generateLargeText(size))
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				result := DetectEmojis(content, patterns)
				_ = result.Unwrap()
			}

			b.ReportMetric(float64(size*b.N)/b.Elapsed().Seconds(), "bytes/sec")
		})
	}
}

// Helper function to generate large test content
func generateLargeText(size int) string {
	content := "Hello 😀 world! This is some test content with emojis. :) "
	result := ""
	for len(result) < size {
		result += content
	}
	return result[:size]
}

// Example usage for documentation
func ExampleDetectEmojis() {
	patterns := DefaultEmojiPatterns()
	content := []byte("Hello 😀 world! :)")

	result := DetectEmojis(content, patterns)
	if result.IsOk() {
		detection := result.Unwrap()
		fmt.Printf("Found %d emojis\n", detection.TotalCount)
		for _, emoji := range detection.Emojis {
			fmt.Printf("  %s at position %d-%d\n", emoji.Emoji, emoji.Start, emoji.End)
		}
	}
	// Output:
	// Found 2 emojis
	//   😀 at position 6-10
	//   :) at position 18-20
}
