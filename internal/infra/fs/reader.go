// Package fs provides file system operations with functional programming principles.
package fs

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"unicode/utf8"

	"github.com/antimoji/antimoji/internal/types"
)

// ReadFile reads the entire contents of a file and returns it as a byte slice.
// This is a pure function that does not modify any external state.
func ReadFile(filepath string) types.Result[[]byte] {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return types.Err[[]byte](err)
	}
	return types.Ok(data)
}

// ReadFileStream reads a file in chunks and returns a channel of byte slices.
// This enables memory-efficient processing of large files.
func ReadFileStream(filepath string, chunkSize int) types.Result[<-chan []byte] {
	file, err := os.Open(filepath)
	if err != nil {
		return types.Err[<-chan []byte](err)
	}

	chunks := make(chan []byte)
	
	go func() {
		defer func() {
			_ = file.Close() // Ignore error in goroutine cleanup
		}()
		defer close(chunks)
		
		reader := bufio.NewReader(file)
		buffer := make([]byte, chunkSize)
		
		for {
			n, err := reader.Read(buffer)
			if n > 0 {
				// Create a copy of the data to send through the channel
				chunk := make([]byte, n)
				copy(chunk, buffer[:n])
				chunks <- chunk
			}
			
			if err != nil {
				// For EOF, we just break normally
				// For other errors, we could log them but for now just stop
				break
			}
		}
	}()
	
	return types.Ok((<-chan []byte)(chunks))
}

// IsTextFile determines if a file contains text content by examining its contents.
// It uses heuristics to detect binary vs text files.
func IsTextFile(filepath string) bool {
	file, err := os.Open(filepath)
	if err != nil {
		return false
	}
	defer func() {
		_ = file.Close() // Ignore error in cleanup
	}()
	
	// Read a sample of the file to determine if it's text
	buffer := make([]byte, 1024)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return false
	}
	
	if n == 0 {
		return true // Empty files are considered text
	}
	
	return isTextContent(buffer[:n])
}

// GetFileInfo returns information about a file.
func GetFileInfo(filepath string) types.Result[types.FileInfo] {
	stat, err := os.Stat(filepath)
	if err != nil {
		return types.Err[types.FileInfo](err)
	}
	
	info := types.FileInfo{
		Path: filepath,
		Size: stat.Size(),
	}
	
	return types.Ok(info)
}

// isTextContent determines if the given byte slice contains text content.
func isTextContent(data []byte) bool {
	if len(data) == 0 {
		return true
	}
	
	// Check for null bytes (common in binary files)
	if bytes.Contains(data, []byte{0}) {
		return false
	}
	
	// Check if the content is valid UTF-8
	if !utf8.Valid(data) {
		return false
	}
	
	// Count non-printable characters
	nonPrintable := 0
	total := 0
	
	for _, b := range data {
		total++
		if b < 32 && b != '\t' && b != '\n' && b != '\r' {
			nonPrintable++
		} else if b > 126 {
			// Allow UTF-8 sequences, but count high bytes
			if !utf8.RuneStart(b) {
				nonPrintable++
			}
		}
	}
	
	// If more than 30% of bytes are non-printable, consider it binary
	if total > 0 && float64(nonPrintable)/float64(total) > 0.30 {
		return false
	}
	
	return true
}
