// Package allowlist provides emoji allowlist functionality with pattern matching and Unicode normalization.
package allowlist

import (
	"strings"
	"unicode"

	"github.com/antimoji/antimoji/internal/types"
)

// Allowlist represents a compiled allowlist of emoji patterns.
type Allowlist struct {
	patterns         map[string]bool // Normalized patterns for fast lookup
	originalPatterns []string        // Original patterns for reference
}

// NewAllowlist creates a new allowlist from the given patterns.
// It normalizes Unicode emojis and builds an optimized lookup structure.
func NewAllowlist(patterns []string) types.Result[*Allowlist] {
	allowlist := &Allowlist{
		patterns:         make(map[string]bool),
		originalPatterns: make([]string, len(patterns)),
	}

	// Copy original patterns
	copy(allowlist.originalPatterns, patterns)

	// Normalize and deduplicate patterns
	for _, pattern := range patterns {
		normalized := normalizeEmoji(pattern)
		allowlist.patterns[normalized] = true
	}

	return types.Ok(allowlist)
}

// IsAllowed checks if the given emoji is in the allowlist.
// This operation is optimized for performance with O(1) lookup.
func (a *Allowlist) IsAllowed(emoji string) bool {
	if emoji == "" {
		return false
	}

	normalized := normalizeEmoji(emoji)
	return a.patterns[normalized]
}

// GetPatterns returns a copy of the original patterns used to create this allowlist.
func (a *Allowlist) GetPatterns() []string {
	result := make([]string, len(a.originalPatterns))
	copy(result, a.originalPatterns)
	return result
}

// Size returns the number of patterns in the allowlist.
func (a *Allowlist) Size() int {
	return len(a.patterns)
}

// Contains is an alias for IsAllowed for better readability.
func (a *Allowlist) Contains(emoji string) bool {
	return a.IsAllowed(emoji)
}

// ApplyAllowlist filters a DetectionResult to only include allowed emojis.
// This is a pure function that does not modify the input.
func ApplyAllowlist(detectionResult types.DetectionResult, allowlist *Allowlist) types.Result[types.DetectionResult] {
	if allowlist == nil {
		return types.Ok(detectionResult)
	}

	// Create new result with filtered emojis
	filtered := types.DetectionResult{
		ProcessedBytes: detectionResult.ProcessedBytes,
		Duration:       detectionResult.Duration,
		Success:        detectionResult.Success,
	}

	// Filter emojis through allowlist
	for _, emoji := range detectionResult.Emojis {
		if allowlist.IsAllowed(emoji.Emoji) {
			filtered.AddEmoji(emoji)
		}
	}

	// Recalculate final statistics
	filtered.Finalize()

	return types.Ok(filtered)
}

// normalizeEmoji normalizes an emoji string for consistent matching.
func normalizeEmoji(emoji string) string {
	// Remove variation selectors and other invisible characters
	var normalized strings.Builder

	for _, r := range emoji {
		// Skip variation selectors
		if r == 0xFE0F || r == 0xFE0E {
			continue
		}
		// Skip zero-width characters
		if r == 0x200D || r == 0x200C {
			continue
		}
		// Keep visible characters
		if !isInvisibleUnicode(r) {
			normalized.WriteRune(r)
		}
	}

	return strings.TrimSpace(normalized.String())
}

// isInvisibleUnicode checks if a rune represents an invisible Unicode character.
func isInvisibleUnicode(r rune) bool {
	// Check for various invisible Unicode categories
	switch unicode.In(r, unicode.Cf, unicode.Mn, unicode.Me) {
	case true:
		return true
	}

	// Check for specific invisible characters
	switch r {
	case 0x00AD, // Soft hyphen
		0x034F,         // Combining grapheme joiner
		0x061C,         // Arabic letter mark
		0x115F, 0x1160, // Hangul fillers
		0x17B4, 0x17B5, // Khmer vowel inherent
		0x180E, // Mongolian vowel separator
		0x3164, // Hangul filler
		0xFEFF: // Zero width no-break space
		return true
	}

	return false
}

// CreateDefaultAllowlist creates a default allowlist with commonly allowed emojis.
func CreateDefaultAllowlist() *Allowlist {
	patterns := []string{
		// Common status indicators
		"", "", "", "ℹ️",
		// Version control and CI/CD
		"", "", "⭐", "", "", "",
		// Documentation
		"", "", "", "",
		// Common custom patterns
		":white_check_mark:", ":x:", ":warning:", ":information_source:",
		"", "", "", "", ":bug:", ":sparkles:",
	}

	return NewAllowlist(patterns).Unwrap()
}

// IsEmpty returns true if the allowlist contains no patterns.
func (a *Allowlist) IsEmpty() bool {
	return len(a.patterns) == 0
}

// Merge combines two allowlists into a new allowlist.
func Merge(a1, a2 *Allowlist) *Allowlist {
	if a1 == nil && a2 == nil {
		return NewAllowlist([]string{}).Unwrap()
	}
	if a1 == nil {
		return a2
	}
	if a2 == nil {
		return a1
	}

	// Combine patterns from both allowlists
	combined := make([]string, 0, len(a1.originalPatterns)+len(a2.originalPatterns))
	combined = append(combined, a1.originalPatterns...)
	combined = append(combined, a2.originalPatterns...)

	return NewAllowlist(combined).Unwrap()
}
