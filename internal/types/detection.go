// Package types provides core types for emoji detection and processing.
package types

import "time"

// EmojiMatch represents a detected emoji in text content.
type EmojiMatch struct {
	// Emoji is the actual emoji string (e.g., "😀", "👍🏽")
	Emoji string `json:"emoji"`

	// Start is the byte position where the emoji begins
	Start int `json:"start"`

	// End is the byte position where the emoji ends (exclusive)
	End int `json:"end"`

	// Line is the line number where the emoji appears (1-based)
	Line int `json:"line"`

	// Column is the column position where the emoji starts (1-based)
	Column int `json:"column"`

	// Category describes the type of emoji (Unicode, Emoticon, Custom)
	Category EmojiCategory `json:"category"`
}

// EmojiCategory represents the type of emoji detected.
type EmojiCategory string

const (
	// CategoryUnicode represents Unicode emoji characters (e.g., 😀, 👍)
	CategoryUnicode EmojiCategory = "unicode"

	// CategoryEmoticon represents text-based emoticons (e.g., :), :()
	CategoryEmoticon EmojiCategory = "emoticon"

	// CategoryCustom represents custom emoji patterns (e.g., :smile:, :thumbs_up:)
	CategoryCustom EmojiCategory = "custom"
)

// DetectionResult contains the results of emoji detection on content.
type DetectionResult struct {
	// Emojis contains all detected emoji matches
	Emojis []EmojiMatch `json:"emojis"`

	// TotalCount is the total number of emojis detected
	TotalCount int `json:"total_count"`

	// UniqueCount is the number of unique emojis detected
	UniqueCount int `json:"unique_count"`

	// ProcessedBytes is the number of bytes that were processed
	ProcessedBytes int64 `json:"processed_bytes"`

	// Duration is how long the detection took
	Duration time.Duration `json:"duration"`

	// Success indicates if detection completed successfully
	Success bool `json:"success"`
}

// Reset clears the DetectionResult for reuse.
func (dr *DetectionResult) Reset() {
	dr.Emojis = dr.Emojis[:0]
	dr.TotalCount = 0
	dr.UniqueCount = 0
	dr.ProcessedBytes = 0
	dr.Duration = 0
	dr.Success = false
}

// AddEmoji adds an emoji match to the detection result.
func (dr *DetectionResult) AddEmoji(match EmojiMatch) {
	dr.Emojis = append(dr.Emojis, match)
	dr.TotalCount++
}

// Finalize calculates final statistics for the detection result.
func (dr *DetectionResult) Finalize() {
	// Calculate unique emoji count
	seen := make(map[string]bool)
	for _, emoji := range dr.Emojis {
		seen[emoji.Emoji] = true
	}
	dr.UniqueCount = len(seen)
	dr.Success = true
}

// EmojiPatterns contains compiled patterns for emoji detection.
type EmojiPatterns struct {
	// UnicodeRanges contains Unicode ranges for emoji detection
	UnicodeRanges []UnicodeRange

	// EmoticonPatterns contains regex patterns for text emoticons
	EmoticonPatterns []string

	// CustomPatterns contains patterns for custom emoji syntax
	CustomPatterns []string
}

// UnicodeRange represents a range of Unicode code points for emoji detection.
type UnicodeRange struct {
	Start rune
	End   rune
	Name  string
}

// Contains checks if a rune is within this Unicode range.
func (ur UnicodeRange) Contains(r rune) bool {
	return r >= ur.Start && r <= ur.End
}

// ProcessingConfig contains configuration for emoji detection.
type ProcessingConfig struct {
	// EnableUnicode controls Unicode emoji detection
	EnableUnicode bool

	// EnableEmoticons controls text emoticon detection
	EnableEmoticons bool

	// EnableCustom controls custom pattern detection
	EnableCustom bool

	// MaxFileSize limits the size of files to process (in bytes)
	MaxFileSize int64

	// BufferSize controls the size of read buffers
	BufferSize int
}

// DefaultProcessingConfig returns a default configuration for emoji detection.
func DefaultProcessingConfig() ProcessingConfig {
	return ProcessingConfig{
		EnableUnicode:   true,
		EnableEmoticons: true,
		EnableCustom:    true,
		MaxFileSize:     100 * 1024 * 1024, // 100MB
		BufferSize:      64 * 1024,         // 64KB
	}
}

// FileInfo contains information about a file to be processed.
type FileInfo struct {
	Path string
	Size int64
}

// ProcessResult contains the result of processing a file.
type ProcessResult struct {
	FilePath        string          `json:"file_path"`
	DetectionResult DetectionResult `json:"detection_result"`
	Error           error           `json:"error,omitempty"`
	Modified        bool            `json:"modified"`
	BackupPath      string          `json:"backup_path,omitempty"`
}
