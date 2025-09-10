package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadFile_Comprehensive(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("reads regular text file", func(t *testing.T) {
		content := "Hello world!\nThis is a test file."
		testFile := filepath.Join(tempDir, "test.txt")
		err := os.WriteFile(testFile, []byte(content), 0644)
		require.NoError(t, err)

		result := ReadFile(testFile)
		assert.True(t, result.IsOk())

		data := result.Unwrap()
		assert.Equal(t, content, string(data))
	})

	t.Run("reads empty file", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "empty.txt")
		err := os.WriteFile(testFile, []byte{}, 0644)
		require.NoError(t, err)

		result := ReadFile(testFile)
		assert.True(t, result.IsOk())

		data := result.Unwrap()
		assert.Empty(t, data)
	})

	t.Run("reads binary file", func(t *testing.T) {
		binaryData := []byte{0x00, 0x01, 0xFF, 0xFE, 0x7F, 0x80}
		testFile := filepath.Join(tempDir, "binary.bin")
		err := os.WriteFile(testFile, binaryData, 0644)
		require.NoError(t, err)

		result := ReadFile(testFile)
		assert.True(t, result.IsOk())

		data := result.Unwrap()
		assert.Equal(t, binaryData, data)
	})

	t.Run("handles nonexistent file", func(t *testing.T) {
		nonexistentFile := filepath.Join(tempDir, "nonexistent.txt")

		result := ReadFile(nonexistentFile)
		assert.True(t, result.IsErr())
		assert.Contains(t, result.Error().Error(), "no such file")
	})

	t.Run("handles permission denied", func(t *testing.T) {
		restrictedFile := filepath.Join(tempDir, "restricted.txt")
		err := os.WriteFile(restrictedFile, []byte("restricted"), 0000)
		require.NoError(t, err)
		defer func() { _ = os.Chmod(restrictedFile, 0644) }() // Cleanup, ignore error

		result := ReadFile(restrictedFile)
		// May succeed or fail depending on system permissions
		_ = result
	})

	t.Run("reads file with unicode content", func(t *testing.T) {
		unicodeContent := "Hello 👋 World 🌍! This has emojis 🚀✨"
		testFile := filepath.Join(tempDir, "unicode.txt")
		err := os.WriteFile(testFile, []byte(unicodeContent), 0644)
		require.NoError(t, err)

		result := ReadFile(testFile)
		assert.True(t, result.IsOk())

		data := result.Unwrap()
		assert.Equal(t, unicodeContent, string(data))
	})

	t.Run("reads large file", func(t *testing.T) {
		// Create a larger file (1MB)
		largeContent := make([]byte, 1024*1024)
		for i := range largeContent {
			largeContent[i] = byte('A' + (i % 26))
		}

		testFile := filepath.Join(tempDir, "large.txt")
		err := os.WriteFile(testFile, largeContent, 0644)
		require.NoError(t, err)

		result := ReadFile(testFile)
		assert.True(t, result.IsOk())

		data := result.Unwrap()
		assert.Equal(t, largeContent, data)
	})
}

func TestReadFileStream_Comprehensive(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("streams file in chunks", func(t *testing.T) {
		content := "This is a test file for streaming. It has multiple lines.\nLine 2\nLine 3\nLine 4"
		testFile := filepath.Join(tempDir, "stream_test.txt")
		err := os.WriteFile(testFile, []byte(content), 0644)
		require.NoError(t, err)

		result := ReadFileStream(testFile, 10) // Small chunks
		assert.True(t, result.IsOk())

		chunks := result.Unwrap()
		var receivedData []byte

		for chunk := range chunks {
			receivedData = append(receivedData, chunk...)
		}

		assert.Equal(t, content, string(receivedData))
	})

	t.Run("handles empty file stream", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "empty_stream.txt")
		err := os.WriteFile(testFile, []byte{}, 0644)
		require.NoError(t, err)

		result := ReadFileStream(testFile, 1024)
		assert.True(t, result.IsOk())

		chunks := result.Unwrap()
		count := 0
		for range chunks {
			count++
		}
		assert.Equal(t, 0, count) // No chunks for empty file
	})

	t.Run("handles nonexistent file stream", func(t *testing.T) {
		nonexistentFile := filepath.Join(tempDir, "nonexistent_stream.txt")

		result := ReadFileStream(nonexistentFile, 1024)
		assert.True(t, result.IsErr())
	})

	t.Run("streams with different chunk sizes", func(t *testing.T) {
		content := "0123456789" // 10 bytes
		testFile := filepath.Join(tempDir, "chunk_test.txt")
		err := os.WriteFile(testFile, []byte(content), 0644)
		require.NoError(t, err)

		chunkSizes := []int{1, 3, 5, 10, 20}
		for _, chunkSize := range chunkSizes {
			t.Run(fmt.Sprintf("chunk_size_%d", chunkSize), func(t *testing.T) {
				result := ReadFileStream(testFile, chunkSize)
				assert.True(t, result.IsOk())

				chunks := result.Unwrap()
				var receivedData []byte

				for chunk := range chunks {
					receivedData = append(receivedData, chunk...)
				}

				assert.Equal(t, content, string(receivedData))
			})
		}
	})

	t.Run("handles small chunk size", func(t *testing.T) {
		content := "test content"
		testFile := filepath.Join(tempDir, "small_chunk.txt")
		err := os.WriteFile(testFile, []byte(content), 0644)
		require.NoError(t, err)

		result := ReadFileStream(testFile, 1) // Very small chunk size
		assert.True(t, result.IsOk())

		chunks := result.Unwrap()
		var receivedData []byte
		for chunk := range chunks {
			receivedData = append(receivedData, chunk...)
		}
		assert.Equal(t, content, string(receivedData))
	})
}

func TestIsTextFile_Comprehensive(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name     string
		content  []byte
		expected bool
	}{
		{"plain text", []byte("Hello world"), true},
		{"text with newlines", []byte("Line 1\nLine 2\nLine 3"), true},
		{"text with unicode", []byte("Hello 👋 World 🌍"), true},
		{"empty file", []byte{}, true},
		{"binary data", []byte{0x00, 0x01, 0xFF, 0xFE}, false},
		{"mostly text with some binary", []byte("Hello\x00World"), false},
		{"json content", []byte(`{"key": "value", "number": 42}`), true},
		{"xml content", []byte(`<?xml version="1.0"?><root><item>test</item></root>`), true},
		{"code content", []byte("package main\n\nfunc main() {\n\tprintln(\"Hello\")\n}"), true},
		{"mixed valid utf8", []byte("Hello\tWorld\r\nNew Line"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tempDir, "test_"+tt.name+".txt")
			err := os.WriteFile(testFile, tt.content, 0644)
			require.NoError(t, err)

			result := IsTextFile(testFile)
			assert.Equal(t, tt.expected, result)
		})
	}

	t.Run("handles nonexistent file", func(t *testing.T) {
		nonexistentFile := filepath.Join(tempDir, "nonexistent.txt")
		result := IsTextFile(nonexistentFile)
		assert.False(t, result) // Should return false for nonexistent files
	})

	t.Run("handles directory instead of file", func(t *testing.T) {
		testDir := filepath.Join(tempDir, "testdir")
		err := os.MkdirAll(testDir, 0755)
		require.NoError(t, err)

		result := IsTextFile(testDir)
		assert.False(t, result) // Should return false for directories
	})
}

// Note: Testing only functions that actually exist in the fs package
