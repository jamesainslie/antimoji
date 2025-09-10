// Package detector provides emoji detection functionality using functional programming principles.
package detector

import (
	"fmt"
	"regexp"
	"sort"
	"time"
	"unicode/utf8"

	"github.com/antimoji/antimoji/internal/types"
)

// DetectEmojis detects emojis in the given content using the provided patterns.
// This is a pure function that does not modify the input content.
func DetectEmojis(content []byte, patterns types.EmojiPatterns) types.Result[types.DetectionResult] {
	if content == nil {
		return types.Ok(types.DetectionResult{Success: true})
	}

	startTime := time.Now()
	result := types.DetectionResult{
		ContentSize: len(content),
		StartTime:   startTime,
	}
	contentStr := string(content)

	// Track line and column positions
	line := 1
	column := 1
	bytePos := 0
	patternsApplied := 0

	// Convert content to runes for proper Unicode handling
	runes := []rune(contentStr)

	for i := 0; i < len(runes); i++ {
		r := runes[i]
		runeStart := bytePos
		runeWidth := utf8.RuneLen(r)

		// Check for Unicode emojis
		if isUnicodeEmoji(r, patterns.UnicodeRanges) {
			patternsApplied++
			// Handle multi-rune emojis (like skin tone modifiers)
			emojiEnd := i + 1
			emojiWidth := runeWidth

			// Check for skin tone modifiers and ZWJ sequences
			for emojiEnd < len(runes) && isEmojiModifier(runes[emojiEnd]) {
				emojiWidth += utf8.RuneLen(runes[emojiEnd])
				emojiEnd++
			}

			emoji := string(runes[i:emojiEnd])
			match := types.EmojiMatch{
				Emoji:    emoji,
				Start:    runeStart,
				End:      runeStart + emojiWidth,
				Line:     line,
				Column:   column,
				Category: types.CategoryUnicode,
			}

			// Store debug information about the Unicode characters detected
			match.DebugInfo = createEmojiDebugInfo(runes[i:emojiEnd], patterns.UnicodeRanges)

			result.AddEmoji(match)

			// Skip processed runes
			i = emojiEnd - 1
			bytePos += emojiWidth
			column += emojiEnd - i
		} else {
			bytePos += runeWidth
			if r == '\n' {
				line++
				column = 1
			} else {
				column++
			}
		}
	}

	// Detect text emoticons
	result, emoticonPatternsApplied := detectEmoticons(contentStr, patterns.EmoticonPatterns, result)
	patternsApplied += emoticonPatternsApplied

	// Detect custom patterns
	result, customPatternsApplied := detectCustomPatterns(contentStr, patterns.CustomPatterns, result)
	patternsApplied += customPatternsApplied

	// Sort emojis by position to ensure consistent ordering
	sort.Slice(result.Emojis, func(i, j int) bool {
		return result.Emojis[i].Start < result.Emojis[j].Start
	})

	// Remove overlapping detections (keep the first one found)
	result.Emojis = removeOverlaps(result.Emojis)
	result.TotalCount = len(result.Emojis)

	result.ProcessedBytes = int64(len(content))
	result.PatternsApplied = patternsApplied
	result.Duration = time.Since(startTime)
	result.Finalize()

	return types.Ok(result)
}

// DefaultEmojiPatterns returns default patterns for emoji detection.
func DefaultEmojiPatterns() types.EmojiPatterns {
	return types.EmojiPatterns{
		UnicodeRanges: []types.UnicodeRange{
			// Basic Emoticons
			{Start: 0x1F600, End: 0x1F64F, Name: "Emoticons"},
			// Miscellaneous Symbols and Pictographs
			{Start: 0x1F300, End: 0x1F5FF, Name: "Miscellaneous Symbols"},
			// Transport and Map Symbols
			{Start: 0x1F680, End: 0x1F6FF, Name: "Transport and Map"},
			// Regional Indicator Symbols
			{Start: 0x1F1E0, End: 0x1F1FF, Name: "Regional Indicators"},
			// Supplemental Symbols and Pictographs
			{Start: 0x1F900, End: 0x1F9FF, Name: "Supplemental Symbols"},
			// Symbols and Pictographs Extended-A
			{Start: 0x1FA70, End: 0x1FAFF, Name: "Extended Symbols A"},
			// Miscellaneous Symbols
			{Start: 0x2600, End: 0x26FF, Name: "Miscellaneous Symbols"},
			// Dingbats
			{Start: 0x2700, End: 0x27BF, Name: "Dingbats"},
		},
		EmoticonPatterns: []string{
			`:)`, `:(`, `:D`, `:P`, `:o`, `:O`, `;)`, `;(`,
			`=)`, `=(`, `=D`, `=P`, `=o`, `=O`, `>:)`, `>:(`,
			`:-)`, `:-(`, `:-D`, `:-P`, `:-o`, `:-O`, `;-)`, `;-(`,
		},
		CustomPatterns: []string{
			`:smile:`, `:frown:`, `:thumbs_up:`, `:thumbs_down:`, `:heart:`,
			`:star:`, `:check:`, `:cross:`, `:warning:`,
			`:fire:`, `:rocket:`, `:tada:`, `:sparkles:`, `:zap:`,
		},
	}
}

// isUnicodeEmoji checks if a rune is a Unicode emoji.
func isUnicodeEmoji(r rune, ranges []types.UnicodeRange) bool {
	for _, urange := range ranges {
		if urange.Contains(r) {
			return true
		}
	}
	return false
}

// isEmojiModifier checks if a rune is an emoji modifier (like skin tone).
func isEmojiModifier(r rune) bool {
	// Skin tone modifiers
	if r >= 0x1F3FB && r <= 0x1F3FF {
		return true
	}
	// Zero Width Joiner
	if r == 0x200D {
		return true
	}
	// Variation Selector-16 (emoji presentation)
	if r == 0xFE0F {
		return true
	}
	return false
}

// detectEmoticons detects text-based emoticons in content.
func detectEmoticons(content string, patterns []string, result types.DetectionResult) (types.DetectionResult, int) {
	for _, pattern := range patterns {
		// Skip empty patterns to avoid zero-length matches causing infinite loops
		if len(pattern) == 0 {
			continue
		}
		// Simple string search for emoticons
		start := 0
		for {
			index := findEmoticonAt(content, pattern, start)
			if index == -1 {
				break
			}

			line, column := calculatePosition(content, index)
			match := types.EmojiMatch{
				Emoji:    pattern,
				Start:    index,
				End:      index + len(pattern),
				Line:     line,
				Column:   column,
				Category: types.CategoryEmoticon,
			}
			result.AddEmoji(match)
			start = index + len(pattern)
		}
	}
	return result, len(patterns)
}

// detectCustomPatterns detects custom emoji patterns in content.
func detectCustomPatterns(content string, patterns []string, result types.DetectionResult) (types.DetectionResult, int) {
	for _, pattern := range patterns {
		// Skip empty patterns to avoid pathological regex behavior
		if len(pattern) == 0 {
			continue
		}
		// Use regex for custom patterns to ensure word boundaries
		regex := regexp.MustCompile(regexp.QuoteMeta(pattern))
		matches := regex.FindAllStringIndex(content, -1)

		for _, match := range matches {
			start, end := match[0], match[1]
			line, column := calculatePosition(content, start)

			emojiMatch := types.EmojiMatch{
				Emoji:    pattern,
				Start:    start,
				End:      end,
				Line:     line,
				Column:   column,
				Category: types.CategoryCustom,
			}
			result.AddEmoji(emojiMatch)
		}
	}
	return result, len(patterns)
}

// findEmoticonAt finds an emoticon pattern at or after the given start position.
func findEmoticonAt(content, pattern string, start int) int {
	if start >= len(content) {
		return -1
	}

	for i := start; i <= len(content)-len(pattern); i++ {
		if content[i:i+len(pattern)] == pattern {
			// Check if it's not part of a larger word (basic boundary check)
			if i > 0 && isAlphanumeric(rune(content[i-1])) {
				continue
			}
			if i+len(pattern) < len(content) && isAlphanumeric(rune(content[i+len(pattern)])) {
				continue
			}
			return i
		}
	}
	return -1
}

// calculatePosition calculates line and column from byte position.
func calculatePosition(content string, bytePos int) (line, column int) {
	line = 1
	column = 1

	for i, r := range content {
		if i >= bytePos {
			break
		}
		if r == '\n' {
			line++
			column = 1
		} else {
			column++
		}
	}
	return line, column
}

// isAlphanumeric checks if a rune is alphanumeric.
func isAlphanumeric(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

// removeOverlaps removes overlapping emoji matches, keeping the first one.
func removeOverlaps(emojis []types.EmojiMatch) []types.EmojiMatch {
	if len(emojis) <= 1 {
		return emojis
	}

	result := make([]types.EmojiMatch, 0, len(emojis))
	result = append(result, emojis[0])

	for i := 1; i < len(emojis); i++ {
		current := emojis[i]
		lastAdded := result[len(result)-1]

		// Check if current emoji overlaps with the last added one
		if current.Start >= lastAdded.End {
			result = append(result, current)
		}
		// If they overlap, skip the current one (keep the first)
	}

	return result
}

// createEmojiDebugInfo creates debug information for detected Unicode emojis.
func createEmojiDebugInfo(runes []rune, ranges []types.UnicodeRange) map[string]interface{} {
	debugInfo := make(map[string]interface{})

	// Add Unicode code points
	var codepoints []string
	var matchedRanges []string

	for _, r := range runes {
		codepoints = append(codepoints, fmt.Sprintf("U+%04X", r))

		// Find which range this rune matches
		for _, urange := range ranges {
			if urange.Contains(r) {
				matchedRanges = append(matchedRanges, urange.Name)
				break
			}
		}
	}

	debugInfo["codepoints"] = codepoints
	debugInfo["matched_ranges"] = matchedRanges
	debugInfo["rune_count"] = len(runes)
	debugInfo["is_multi_rune"] = len(runes) > 1

	// Add information about modifiers if present
	if len(runes) > 1 {
		var modifierTypes []string
		for i := 1; i < len(runes); i++ {
			r := runes[i]
			if r >= 0x1F3FB && r <= 0x1F3FF {
				modifierTypes = append(modifierTypes, "skin_tone")
			} else if r == 0x200D {
				modifierTypes = append(modifierTypes, "zwj")
			} else if r == 0xFE0F {
				modifierTypes = append(modifierTypes, "variation_selector")
			} else {
				modifierTypes = append(modifierTypes, "other")
			}
		}
		debugInfo["modifiers"] = modifierTypes
	}

	return debugInfo
}
