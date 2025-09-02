package allowlist

import (
	"fmt"
	"testing"

	"github.com/antimoji/antimoji/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestNewAllowlist(t *testing.T) {
	t.Run("creates allowlist with valid patterns", func(t *testing.T) {
		patterns := []string{"✅", "❌", "⚠️"}

		result := NewAllowlist(patterns)
		assert.True(t, result.IsOk())

		allowlist := result.Unwrap()
		assert.Len(t, allowlist.patterns, 3)
		assert.Equal(t, patterns, allowlist.originalPatterns)
	})

	t.Run("handles empty patterns", func(t *testing.T) {
		patterns := []string{}

		result := NewAllowlist(patterns)
		assert.True(t, result.IsOk())

		allowlist := result.Unwrap()
		assert.Empty(t, allowlist.patterns)
		assert.Empty(t, allowlist.originalPatterns)
	})

	t.Run("normalizes Unicode emojis", func(t *testing.T) {
		// Test with emojis that might have different Unicode representations
		patterns := []string{"✅", "❌️", "⚠️"} // Some with variation selectors

		result := NewAllowlist(patterns)
		assert.True(t, result.IsOk())

		allowlist := result.Unwrap()
		assert.True(t, allowlist.IsAllowed("✅"))
		assert.True(t, allowlist.IsAllowed("❌"))
		assert.True(t, allowlist.IsAllowed("⚠️"))
	})

	t.Run("handles duplicate patterns", func(t *testing.T) {
		patterns := []string{"✅", "✅", "❌", "✅"} // Duplicates

		result := NewAllowlist(patterns)
		assert.True(t, result.IsOk())

		allowlist := result.Unwrap()
		// Should deduplicate internally
		assert.True(t, allowlist.IsAllowed("✅"))
		assert.True(t, allowlist.IsAllowed("❌"))
	})
}

func TestAllowlist_IsAllowed(t *testing.T) {
	patterns := []string{"✅", "❌", "⚠️", ":thumbs_up:", ":warning:"}
	allowlist := NewAllowlist(patterns).Unwrap()

	tests := []struct {
		name     string
		emoji    string
		expected bool
	}{
		{"allowed unicode emoji", "✅", true},
		{"allowed unicode emoji 2", "❌", true},
		{"allowed unicode emoji 3", "⚠️", true},
		{"allowed custom pattern", ":thumbs_up:", true},
		{"allowed custom pattern 2", ":warning:", true},
		{"not allowed unicode emoji", "😀", false},
		{"not allowed emoticon", ":)", false},
		{"not allowed custom pattern", ":smile:", false},
		{"empty string", "", false},
		{"partial match should not be allowed", "✅extra", false},
		{"case sensitive custom pattern", ":THUMBS_UP:", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := allowlist.IsAllowed(tt.emoji)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAllowlist_GetPatterns(t *testing.T) {
	t.Run("returns original patterns", func(t *testing.T) {
		patterns := []string{"✅", "❌", ":smile:"}
		allowlist := NewAllowlist(patterns).Unwrap()

		result := allowlist.GetPatterns()
		assert.Equal(t, patterns, result)
	})

	t.Run("returns copy not reference", func(t *testing.T) {
		patterns := []string{"✅", "❌"}
		allowlist := NewAllowlist(patterns).Unwrap()

		returned := allowlist.GetPatterns()
		returned[0] = "modified" // Modify the returned slice

		// Original should be unchanged
		original := allowlist.GetPatterns()
		assert.Equal(t, "✅", original[0])
		assert.NotEqual(t, "modified", original[0])
	})
}

func TestApplyAllowlist(t *testing.T) {
	allowlistPatterns := []string{"✅", "❌", ":thumbs_up:"}
	allowlist := NewAllowlist(allowlistPatterns).Unwrap()

	t.Run("filters out non-allowed emojis", func(t *testing.T) {
		detectionResult := types.DetectionResult{
			Emojis: []types.EmojiMatch{
				{Emoji: "✅", Category: types.CategoryUnicode},          // Allowed
				{Emoji: "😀", Category: types.CategoryUnicode},          // Not allowed
				{Emoji: ":thumbs_up:", Category: types.CategoryCustom}, // Allowed
				{Emoji: ":)", Category: types.CategoryEmoticon},        // Not allowed
				{Emoji: "❌", Category: types.CategoryUnicode},          // Allowed
			},
			TotalCount:  5,
			UniqueCount: 5,
			Success:     true,
		}

		result := ApplyAllowlist(detectionResult, allowlist)
		assert.True(t, result.IsOk())

		filtered := result.Unwrap()
		assert.Len(t, filtered.Emojis, 3) // Only allowed emojis
		assert.Equal(t, 3, filtered.TotalCount)

		// Check that only allowed emojis remain
		allowedEmojis := make(map[string]bool)
		for _, emoji := range filtered.Emojis {
			allowedEmojis[emoji.Emoji] = true
		}
		assert.True(t, allowedEmojis["✅"])
		assert.True(t, allowedEmojis["❌"])
		assert.True(t, allowedEmojis[":thumbs_up:"])
		assert.False(t, allowedEmojis["😀"])
		assert.False(t, allowedEmojis[":)"])
	})

	t.Run("handles empty detection result", func(t *testing.T) {
		detectionResult := types.DetectionResult{
			Emojis:      []types.EmojiMatch{},
			TotalCount:  0,
			UniqueCount: 0,
			Success:     true,
		}

		result := ApplyAllowlist(detectionResult, allowlist)
		assert.True(t, result.IsOk())

		filtered := result.Unwrap()
		assert.Empty(t, filtered.Emojis)
		assert.Equal(t, 0, filtered.TotalCount)
		assert.Equal(t, 0, filtered.UniqueCount)
	})

	t.Run("handles all emojis allowed", func(t *testing.T) {
		detectionResult := types.DetectionResult{
			Emojis: []types.EmojiMatch{
				{Emoji: "✅", Category: types.CategoryUnicode},
				{Emoji: "❌", Category: types.CategoryUnicode},
			},
			TotalCount:  2,
			UniqueCount: 2,
			Success:     true,
		}

		result := ApplyAllowlist(detectionResult, allowlist)
		assert.True(t, result.IsOk())

		filtered := result.Unwrap()
		assert.Len(t, filtered.Emojis, 2) // All emojis allowed
		assert.Equal(t, 2, filtered.TotalCount)
	})

	t.Run("handles all emojis filtered out", func(t *testing.T) {
		detectionResult := types.DetectionResult{
			Emojis: []types.EmojiMatch{
				{Emoji: "😀", Category: types.CategoryUnicode},
				{Emoji: ":)", Category: types.CategoryEmoticon},
			},
			TotalCount:  2,
			UniqueCount: 2,
			Success:     true,
		}

		result := ApplyAllowlist(detectionResult, allowlist)
		assert.True(t, result.IsOk())

		filtered := result.Unwrap()
		assert.Empty(t, filtered.Emojis) // All filtered out
		assert.Equal(t, 0, filtered.TotalCount)
	})

	t.Run("preserves emoji metadata", func(t *testing.T) {
		detectionResult := types.DetectionResult{
			Emojis: []types.EmojiMatch{
				{
					Emoji:    "✅",
					Start:    5,
					End:      9,
					Line:     2,
					Column:   6,
					Category: types.CategoryUnicode,
				},
			},
			TotalCount:  1,
			UniqueCount: 1,
			Success:     true,
		}

		result := ApplyAllowlist(detectionResult, allowlist)
		assert.True(t, result.IsOk())

		filtered := result.Unwrap()
		assert.Len(t, filtered.Emojis, 1)

		emoji := filtered.Emojis[0]
		assert.Equal(t, "✅", emoji.Emoji)
		assert.Equal(t, 5, emoji.Start)
		assert.Equal(t, 9, emoji.End)
		assert.Equal(t, 2, emoji.Line)
		assert.Equal(t, 6, emoji.Column)
		assert.Equal(t, types.CategoryUnicode, emoji.Category)
	})
}

func TestAllowlist_Properties(t *testing.T) {
	patterns := []string{"✅", "❌", ":smile:"}
	allowlist := NewAllowlist(patterns).Unwrap()

	t.Run("allowlist is deterministic", func(t *testing.T) {
		testEmojis := []string{"✅", "😀", ":smile:", ":)", "❌"}

		// Run multiple times and ensure consistent results
		for i := 0; i < 5; i++ {
			for _, emoji := range testEmojis {
				result1 := allowlist.IsAllowed(emoji)
				result2 := allowlist.IsAllowed(emoji)
				assert.Equal(t, result1, result2, "allowlist should be deterministic for emoji: %s", emoji)
			}
		}
	})

	t.Run("filtering is monotonic", func(t *testing.T) {
		// Create detection result with mixed emojis
		detectionResult := types.DetectionResult{
			Emojis: []types.EmojiMatch{
				{Emoji: "✅", Category: types.CategoryUnicode},
				{Emoji: "😀", Category: types.CategoryUnicode},
				{Emoji: ":smile:", Category: types.CategoryCustom},
				{Emoji: ":)", Category: types.CategoryEmoticon},
			},
			TotalCount:  4,
			UniqueCount: 4,
			Success:     true,
		}

		filtered := ApplyAllowlist(detectionResult, allowlist).Unwrap()

		// Filtering should never increase the number of emojis
		assert.True(t, filtered.TotalCount <= detectionResult.TotalCount)
		assert.True(t, len(filtered.Emojis) <= len(detectionResult.Emojis))
	})
}

func TestAllowlist_AdditionalMethods(t *testing.T) {
	patterns := []string{"✅", "❌", ":smile:"}
	allowlist := NewAllowlist(patterns).Unwrap()

	t.Run("Size returns correct count", func(t *testing.T) {
		assert.Equal(t, 3, allowlist.Size())
	})

	t.Run("Contains is alias for IsAllowed", func(t *testing.T) {
		assert.Equal(t, allowlist.IsAllowed("✅"), allowlist.Contains("✅"))
		assert.Equal(t, allowlist.IsAllowed("😀"), allowlist.Contains("😀"))
	})

	t.Run("IsEmpty works correctly", func(t *testing.T) {
		emptyAllowlist := NewAllowlist([]string{}).Unwrap()
		assert.True(t, emptyAllowlist.IsEmpty())
		assert.False(t, allowlist.IsEmpty())
	})
}

func TestCreateDefaultAllowlist(t *testing.T) {
	t.Run("creates allowlist with common patterns", func(t *testing.T) {
		allowlist := CreateDefaultAllowlist()
		assert.NotNil(t, allowlist)
		assert.False(t, allowlist.IsEmpty())

		// Should include common status indicators
		assert.True(t, allowlist.IsAllowed("✅"))
		assert.True(t, allowlist.IsAllowed("❌"))
		assert.True(t, allowlist.IsAllowed("⚠️"))

		// Should include common development emojis
		assert.True(t, allowlist.IsAllowed("🔥"))
		assert.True(t, allowlist.IsAllowed("🚀"))
		assert.True(t, allowlist.IsAllowed("🐛"))
	})
}

func TestMerge(t *testing.T) {
	t.Run("merges two allowlists", func(t *testing.T) {
		allowlist1 := NewAllowlist([]string{"✅", "❌"}).Unwrap()
		allowlist2 := NewAllowlist([]string{"🔥", "🚀"}).Unwrap()

		merged := Merge(allowlist1, allowlist2)
		assert.Equal(t, 4, merged.Size())
		assert.True(t, merged.IsAllowed("✅"))
		assert.True(t, merged.IsAllowed("❌"))
		assert.True(t, merged.IsAllowed("🔥"))
		assert.True(t, merged.IsAllowed("🚀"))
	})

	t.Run("handles nil allowlists", func(t *testing.T) {
		allowlist1 := NewAllowlist([]string{"✅"}).Unwrap()

		merged1 := Merge(nil, allowlist1)
		assert.Equal(t, allowlist1.Size(), merged1.Size())

		merged2 := Merge(allowlist1, nil)
		assert.Equal(t, allowlist1.Size(), merged2.Size())

		merged3 := Merge(nil, nil)
		assert.True(t, merged3.IsEmpty())
	})

	t.Run("handles overlapping patterns", func(t *testing.T) {
		allowlist1 := NewAllowlist([]string{"✅", "❌", "🔥"}).Unwrap()
		allowlist2 := NewAllowlist([]string{"✅", "🚀"}).Unwrap() // ✅ overlaps

		merged := Merge(allowlist1, allowlist2)
		// Should still work correctly with duplicates
		assert.True(t, merged.IsAllowed("✅"))
		assert.True(t, merged.IsAllowed("❌"))
		assert.True(t, merged.IsAllowed("🔥"))
		assert.True(t, merged.IsAllowed("🚀"))
	})
}

func TestNormalizeEmoji(t *testing.T) {
	t.Run("removes variation selectors", func(t *testing.T) {
		// These test cases would be internal if normalizeEmoji was exported
		// For now, we test through the public API
		allowlist := NewAllowlist([]string{"❌️"}).Unwrap() // With variation selector

		// Should match both with and without variation selector
		assert.True(t, allowlist.IsAllowed("❌"))
		assert.True(t, allowlist.IsAllowed("❌️"))
	})
}

func TestApplyAllowlist_EdgeCases(t *testing.T) {
	t.Run("handles nil allowlist", func(t *testing.T) {
		detectionResult := types.DetectionResult{
			Emojis: []types.EmojiMatch{
				{Emoji: "😀", Category: types.CategoryUnicode},
			},
			TotalCount: 1,
			Success:    true,
		}

		result := ApplyAllowlist(detectionResult, nil)
		assert.True(t, result.IsOk())

		filtered := result.Unwrap()
		assert.Equal(t, detectionResult.TotalCount, filtered.TotalCount) // Should be unchanged
	})

	t.Run("preserves detection metadata", func(t *testing.T) {
		allowlist := NewAllowlist([]string{"✅"}).Unwrap()

		detectionResult := types.DetectionResult{
			Emojis:         []types.EmojiMatch{{Emoji: "✅", Category: types.CategoryUnicode}},
			TotalCount:     1,
			UniqueCount:    1,
			ProcessedBytes: 1024,
			Duration:       100000, // 100µs
			Success:        true,
		}

		result := ApplyAllowlist(detectionResult, allowlist)
		assert.True(t, result.IsOk())

		filtered := result.Unwrap()
		assert.Equal(t, detectionResult.ProcessedBytes, filtered.ProcessedBytes)
		assert.Equal(t, detectionResult.Duration, filtered.Duration)
		assert.Equal(t, detectionResult.Success, filtered.Success)
	})
}

// Benchmark tests for performance
func BenchmarkAllowlist_IsAllowed(b *testing.B) {
	patterns := []string{"✅", "❌", "⚠️", ":smile:", ":frown:", ":thumbs_up:", ":heart:"}
	allowlist := NewAllowlist(patterns).Unwrap()

	testEmojis := []string{"✅", "😀", ":smile:", ":)", "❌", "🚀", ":thumbs_up:", "😃"}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for _, emoji := range testEmojis {
			_ = allowlist.IsAllowed(emoji)
		}
	}
}

func BenchmarkApplyAllowlist(b *testing.B) {
	patterns := []string{"✅", "❌", "⚠️"}
	allowlist := NewAllowlist(patterns).Unwrap()

	// Create detection result with many emojis
	emojis := make([]types.EmojiMatch, 100)
	for i := 0; i < 100; i++ {
		emoji := "😀"
		if i%10 == 0 {
			emoji = "✅" // Some allowed emojis
		}
		emojis[i] = types.EmojiMatch{
			Emoji:    emoji,
			Category: types.CategoryUnicode,
		}
	}

	detectionResult := types.DetectionResult{
		Emojis:      emojis,
		TotalCount:  100,
		UniqueCount: 2,
		Success:     true,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		result := ApplyAllowlist(detectionResult, allowlist)
		_ = result.Unwrap()
	}
}

func BenchmarkNewAllowlist(b *testing.B) {
	patterns := []string{
		"✅", "❌", "⚠️", "🔥", "🚀", "⭐", "❤️", "👍", "👎", "🎉",
		":smile:", ":frown:", ":thumbs_up:", ":heart:", ":fire:", ":rocket:",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		result := NewAllowlist(patterns)
		_ = result.Unwrap()
	}
}

// Example usage for documentation
func ExampleNewAllowlist() {
	patterns := []string{"✅", "❌", ":thumbs_up:"}

	result := NewAllowlist(patterns)
	if result.IsOk() {
		allowlist := result.Unwrap()

		fmt.Println("Is ✅ allowed?", allowlist.IsAllowed("✅"))
		fmt.Println("Is 😀 allowed?", allowlist.IsAllowed("😀"))
	}
	// Output:
	// Is ✅ allowed? true
	// Is 😀 allowed? false
}

func ExampleApplyAllowlist() {
	// Create allowlist
	allowlist := NewAllowlist([]string{"✅", "❌"}).Unwrap()

	// Create detection result with mixed emojis
	detectionResult := types.DetectionResult{
		Emojis: []types.EmojiMatch{
			{Emoji: "✅", Category: types.CategoryUnicode},
			{Emoji: "😀", Category: types.CategoryUnicode},
			{Emoji: "❌", Category: types.CategoryUnicode},
		},
		TotalCount:  3,
		UniqueCount: 3,
		Success:     true,
	}

	// Apply allowlist filtering
	result := ApplyAllowlist(detectionResult, allowlist)
	if result.IsOk() {
		filtered := result.Unwrap()
		fmt.Printf("Original: %d emojis, Filtered: %d emojis\n",
			detectionResult.TotalCount, filtered.TotalCount)
	}
	// Output: Original: 3 emojis, Filtered: 2 emojis
}
