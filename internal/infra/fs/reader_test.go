package fs

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadFile(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	t.Run("reads small file successfully", func(t *testing.T) {
		content := "Hello 😀 world!"
		filePath := filepath.Join(tmpDir, "small.txt")
		err := os.WriteFile(filePath, []byte(content), 0644)
		assert.NoError(t, err)

		result := ReadFile(filePath)
		assert.True(t, result.IsOk())

		data := result.Unwrap()
		assert.Equal(t, content, string(data))
	})

	t.Run("reads empty file successfully", func(t *testing.T) {
		filePath := filepath.Join(tmpDir, "empty.txt")
		err := os.WriteFile(filePath, []byte{}, 0644)
		assert.NoError(t, err)

		result := ReadFile(filePath)
		assert.True(t, result.IsOk())

		data := result.Unwrap()
		assert.Empty(t, data)
	})

	t.Run("handles non-existent file", func(t *testing.T) {
		filePath := filepath.Join(tmpDir, "nonexistent.txt")

		result := ReadFile(filePath)
		assert.True(t, result.IsErr())
		// Cross-platform error message check
		errorMsg := result.Error().Error()
		assert.True(t, 
			strings.Contains(errorMsg, "no such file") || 
			strings.Contains(errorMsg, "cannot find the file") ||
			strings.Contains(errorMsg, "does not exist"),
			"Expected file not found error, got: %s", errorMsg)
	})

	t.Run("reads large file successfully", func(t *testing.T) {
		// Create a 1MB file
		content := strings.Repeat("Hello 😀 world!\n", 64*1024)
		filePath := filepath.Join(tmpDir, "large.txt")
		err := os.WriteFile(filePath, []byte(content), 0644)
		assert.NoError(t, err)

		result := ReadFile(filePath)
		assert.True(t, result.IsOk())

		data := result.Unwrap()
		assert.Equal(t, content, string(data))
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
			_ = os.Chmod(filePath, 0644) // Cleanup, ignore error
		}()

		result := ReadFile(filePath)
		assert.True(t, result.IsErr())
		assert.Contains(t, result.Error().Error(), "permission denied")
	})
}

func TestReadFileStream(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("streams file content in chunks", func(t *testing.T) {
		content := "Hello 😀 world! This is a test file with some content."
		filePath := filepath.Join(tmpDir, "stream.txt")
		err := os.WriteFile(filePath, []byte(content), 0644)
		assert.NoError(t, err)

		chunkSize := 16
		result := ReadFileStream(filePath, chunkSize)
		assert.True(t, result.IsOk())

		stream := result.Unwrap()
		var chunks [][]byte
		for chunk := range stream {
			chunks = append(chunks, chunk)
		}

		// Reconstruct content from chunks
		var reconstructed bytes.Buffer
		for _, chunk := range chunks {
			reconstructed.Write(chunk)
		}

		assert.Equal(t, content, reconstructed.String())
		assert.True(t, len(chunks) > 1, "should have multiple chunks")
	})

	t.Run("handles empty file", func(t *testing.T) {
		filePath := filepath.Join(tmpDir, "empty_stream.txt")
		err := os.WriteFile(filePath, []byte{}, 0644)
		assert.NoError(t, err)

		result := ReadFileStream(filePath, 1024)
		assert.True(t, result.IsOk())

		stream := result.Unwrap()
		chunks := make([][]byte, 0)
		for chunk := range stream {
			chunks = append(chunks, chunk)
		}

		assert.Empty(t, chunks)
	})

	t.Run("handles non-existent file", func(t *testing.T) {
		filePath := filepath.Join(tmpDir, "nonexistent_stream.txt")

		result := ReadFileStream(filePath, 1024)
		assert.True(t, result.IsErr())
	})
}

func TestIsTextFile(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("identifies text files correctly", func(t *testing.T) {
		testCases := []struct {
			name     string
			content  string
			expected bool
		}{
			{"plain text", "Hello world", true},
			{"text with emojis", "Hello 😀 world", true},
			{"code file", "package main\n\nfunc main() {}", true},
			{"empty file", "", true},
			{"binary data", string([]byte{0x00, 0x01, 0x02, 0xFF}), false},
			{"mostly text with null", "Hello\x00world", false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				filePath := filepath.Join(tmpDir, tc.name+".txt")
				err := os.WriteFile(filePath, []byte(tc.content), 0644)
				assert.NoError(t, err)

				result := IsTextFile(filePath)
				assert.Equal(t, tc.expected, result)
			})
		}
	})

	t.Run("handles non-existent file", func(t *testing.T) {
		filePath := filepath.Join(tmpDir, "nonexistent.txt")
		result := IsTextFile(filePath)
		assert.False(t, result)
	})
}

func TestGetFileInfo(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("returns correct file info", func(t *testing.T) {
		content := "Hello world"
		filePath := filepath.Join(tmpDir, "info.txt")
		err := os.WriteFile(filePath, []byte(content), 0644)
		assert.NoError(t, err)

		result := GetFileInfo(filePath)
		assert.True(t, result.IsOk())

		info := result.Unwrap()
		assert.Equal(t, filePath, info.Path)
		assert.Equal(t, int64(len(content)), info.Size)
	})

	t.Run("handles non-existent file", func(t *testing.T) {
		filePath := filepath.Join(tmpDir, "nonexistent.txt")

		result := GetFileInfo(filePath)
		assert.True(t, result.IsErr())
	})
}

// Benchmark tests for performance
func BenchmarkReadFile(b *testing.B) {
	tmpDir := b.TempDir()

	testCases := []struct {
		name string
		size int
	}{
		{"1KB", 1024},
		{"64KB", 64 * 1024},
		{"1MB", 1024 * 1024},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			content := strings.Repeat("a", tc.size)
			filePath := filepath.Join(tmpDir, "bench.txt")
			err := os.WriteFile(filePath, []byte(content), 0644)
			assert.NoError(b, err)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				result := ReadFile(filePath)
				if result.IsErr() {
					b.Fatal(result.Error())
				}
				_ = result.Unwrap()
			}

			b.ReportMetric(float64(tc.size*b.N)/b.Elapsed().Seconds(), "bytes/sec")
		})
	}
}

func BenchmarkReadFileStream(b *testing.B) {
	tmpDir := b.TempDir()
	content := strings.Repeat("Hello world\n", 8192) // ~96KB
	filePath := filepath.Join(tmpDir, "stream_bench.txt")
	err := os.WriteFile(filePath, []byte(content), 0644)
	assert.NoError(b, err)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		result := ReadFileStream(filePath, 4096)
		if result.IsErr() {
			b.Fatal(result.Error())
		}

		stream := result.Unwrap()
		for chunk := range stream {
			_ = chunk // Consume the chunk
		}
	}
}

// Example usage for documentation
func ExampleReadFile() {
	// Create a temporary file for the example
	tmpFile, err := os.CreateTemp("", "example.txt")
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = os.Remove(tmpFile.Name()) // Ignore cleanup error
	}()

	// Write some content
	content := "Hello 😀 world!"
	_, err = tmpFile.WriteString(content)
	if err != nil {
		panic(err)
	}
	_ = tmpFile.Close() // Ignore error in test cleanup

	// Read the file
	result := ReadFile(tmpFile.Name())
	if result.IsOk() {
		data := result.Unwrap()
		fmt.Println("File content:", string(data))
	}
	// Output: File content: Hello 😀 world!
}

func ExampleReadFileStream() {
	// Create a temporary file for the example
	tmpFile, err := os.CreateTemp("", "stream_example.txt")
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = os.Remove(tmpFile.Name()) // Ignore cleanup error
	}()

	// Write some content
	content := "This is a longer file that will be read in chunks."
	_, err = tmpFile.WriteString(content)
	if err != nil {
		panic(err)
	}
	_ = tmpFile.Close() // Ignore error in test cleanup

	// Stream the file in 16-byte chunks
	result := ReadFileStream(tmpFile.Name(), 16)
	if result.IsOk() {
		stream := result.Unwrap()
		chunkCount := 0
		for chunk := range stream {
			chunkCount++
			fmt.Println("Chunk", chunkCount, ":", string(chunk))
		}
	}
}
