// Package processor provides file processing functionality with functional programming principles.
package processor

import (
	"errors"
	"time"

	"github.com/antimoji/antimoji/internal/core/detector"
	"github.com/antimoji/antimoji/internal/infra/fs"
	"github.com/antimoji/antimoji/internal/types"
)

// ProcessingPipeline represents a configured processing pipeline.
type ProcessingPipeline struct {
	Config types.ProcessingConfig
}

// ProcessFile processes a single file for emoji detection.
// This is a pure function that does not modify files (scan mode only for now).
func ProcessFile(filePath string, patterns types.EmojiPatterns, config types.ProcessingConfig) types.Result[types.ProcessResult] {
	startTime := time.Now()
	
	// Initialize result
	result := types.ProcessResult{
		FilePath: filePath,
		Modified: false,
	}
	
	// Get file info first
	fileInfoResult := fs.GetFileInfo(filePath)
	if fileInfoResult.IsErr() {
		result.Error = fileInfoResult.Error()
		return types.Ok(result)
	}
	
	fileInfo := fileInfoResult.Unwrap()
	
	// Check file size limit
	if fileInfo.Size > config.MaxFileSize {
		result.Error = errors.New("file too large")
		return types.Ok(result)
	}
	
	// Check if it's a text file
	if !fs.IsTextFile(filePath) {
		// Skip binary files
		result.DetectionResult = types.DetectionResult{
			ProcessedBytes: fileInfo.Size,
			Duration:       time.Since(startTime),
			Success:        false,
		}
		return types.Ok(result)
	}
	
	// Read file content
	contentResult := fs.ReadFile(filePath)
	if contentResult.IsErr() {
		result.Error = contentResult.Error()
		return types.Ok(result)
	}
	
	content := contentResult.Unwrap()
	
	// Filter patterns based on configuration
	filteredPatterns := filterPatterns(patterns, config)
	
	// Detect emojis
	detectionResult := detector.DetectEmojis(content, filteredPatterns)
	if detectionResult.IsErr() {
		result.Error = detectionResult.Error()
		return types.Ok(result)
	}
	
	detection := detectionResult.Unwrap()
	detection.Duration = time.Since(startTime)
	result.DetectionResult = detection
	
	return types.Ok(result)
}

// ProcessFiles processes multiple files and returns results for all files.
func ProcessFiles(filePaths []string, patterns types.EmojiPatterns, config types.ProcessingConfig) []types.ProcessResult {
	results := make([]types.ProcessResult, 0, len(filePaths))
	
	for _, filePath := range filePaths {
		processResult := ProcessFile(filePath, patterns, config)
		if processResult.IsOk() {
			results = append(results, processResult.Unwrap())
		} else {
			// This shouldn't happen with current implementation, but handle it
			errorResult := types.ProcessResult{
				FilePath: filePath,
				Error:    processResult.Error(),
				Modified: false,
			}
			results = append(results, errorResult)
		}
	}
	
	return results
}

// CreateProcessingPipeline creates a new processing pipeline with the given configuration.
func CreateProcessingPipeline(config types.ProcessingConfig) *ProcessingPipeline {
	return &ProcessingPipeline{
		Config: config,
	}
}

// Process processes files using the pipeline configuration.
func (p *ProcessingPipeline) Process(filePaths []string, patterns types.EmojiPatterns) []types.ProcessResult {
	return ProcessFiles(filePaths, patterns, p.Config)
}

// filterPatterns filters emoji patterns based on processing configuration.
func filterPatterns(patterns types.EmojiPatterns, config types.ProcessingConfig) types.EmojiPatterns {
	filtered := types.EmojiPatterns{}
	
	if config.EnableUnicode {
		filtered.UnicodeRanges = patterns.UnicodeRanges
	}
	
	if config.EnableEmoticons {
		filtered.EmoticonPatterns = patterns.EmoticonPatterns
	}
	
	if config.EnableCustom {
		filtered.CustomPatterns = patterns.CustomPatterns
	}
	
	return filtered
}
