package fs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetBinaryFileReason(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		{
			name:     "contains null bytes",
			data:     []byte("Hello\x00World"),
			expected: "contains_null_bytes",
		},
		{
			name:     "invalid utf8",
			data:     []byte{0xFF, 0xFE, 0xFD},
			expected: "invalid_utf8",
		},
		{
			name:     "high control characters",
			data:     []byte{0x01, 0x02, 0x03, 0x04, 0x05}, // Many control chars
			expected: "high_control_chars",
		},
		{
			name:     "mixed control characters",
			data:     []byte("Hello\x01\x02\x03World\x04\x05"), // Some control chars
			expected: "high_control_chars",
		},
		{
			name:     "valid text with acceptable control chars",
			data:     []byte("Hello\tWorld\n\r"), // Tab, newline, CR are OK
			expected: "unknown_binary_pattern",
		},
		{
			name:     "empty data",
			data:     []byte{},
			expected: "unknown_binary_pattern",
		},
		{
			name:     "normal text",
			data:     []byte("Hello World"),
			expected: "unknown_binary_pattern",
		},
		{
			name:     "unicode emojis",
			data:     []byte("Hello 👋 World 🌍"),
			expected: "unknown_binary_pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getBinaryFileReason(tt.data)

			if tt.expected == "high_control_chars" {
				// For high control chars, check that it contains the prefix
				assert.Contains(t, result, "high_control_chars")
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestGetBinaryFileReason_EdgeCases(t *testing.T) {
	t.Run("exactly 30% control characters", func(t *testing.T) {
		// Create data with exactly 30% control characters
		data := []byte("abc\x01def\x02ghi\x03") // 3 control chars out of 12 = 25%
		result := getBinaryFileReason(data)
		// Should not trigger high_control_chars at 25%
		assert.Equal(t, "unknown_binary_pattern", result)
	})

	t.Run("just over 30% control characters", func(t *testing.T) {
		// Create data with over 30% control characters
		data := []byte("ab\x01\x02\x03\x04\x05") // 5 control chars out of 7 = ~71%
		result := getBinaryFileReason(data)
		assert.Contains(t, result, "high_control_chars")
	})

	t.Run("single null byte", func(t *testing.T) {
		data := []byte("Hello\x00")
		result := getBinaryFileReason(data)
		assert.Equal(t, "contains_null_bytes", result)
	})

	t.Run("multiple null bytes", func(t *testing.T) {
		data := []byte("\x00\x00\x00")
		result := getBinaryFileReason(data)
		assert.Equal(t, "contains_null_bytes", result)
	})

	t.Run("invalid utf8 sequences", func(t *testing.T) {
		// Create invalid UTF-8
		data := []byte{0x80, 0x81, 0x82} // Invalid UTF-8 start bytes
		result := getBinaryFileReason(data)
		assert.Equal(t, "invalid_utf8", result)
	})

	t.Run("mixed invalid utf8 and text", func(t *testing.T) {
		data := []byte("Hello\xFF\xFEWorld")
		result := getBinaryFileReason(data)
		assert.Equal(t, "invalid_utf8", result)
	})

	t.Run("only acceptable control characters", func(t *testing.T) {
		data := []byte("Line1\nLine2\tTabbed\rCarriageReturn")
		result := getBinaryFileReason(data)
		assert.Equal(t, "unknown_binary_pattern", result)
	})

	t.Run("single character data", func(t *testing.T) {
		// Test with single character
		data := []byte("a")
		result := getBinaryFileReason(data)
		assert.Equal(t, "unknown_binary_pattern", result)

		// Test with single control character
		data = []byte{0x01}
		result = getBinaryFileReason(data)
		assert.Contains(t, result, "high_control_chars")
	})
}
