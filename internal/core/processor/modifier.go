// Package processor provides file modification functionality with atomic operations and backup support.
package processor

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/antimoji/antimoji/internal/core/allowlist"
	"github.com/antimoji/antimoji/internal/core/detector"
	"github.com/antimoji/antimoji/internal/infra/fs"
	"github.com/antimoji/antimoji/internal/types"
)

// ModifyConfig contains configuration for file modification operations.
type ModifyConfig struct {
	// Replacement is the string to replace emojis with
	Replacement string

	// CreateBackup creates a backup file before modification
	CreateBackup bool

	// RespectAllowlist applies allowlist filtering before removal
	RespectAllowlist bool

	// PreservePermissions maintains original file permissions
	PreservePermissions bool

	// DryRun shows what would be changed without modifying files
	DryRun bool
}

// ModifyResult contains the result of a file modification operation.
type ModifyResult struct {
	FilePath      string `json:"file_path"`
	Success       bool   `json:"success"`
	Modified      bool   `json:"modified"`
	EmojisRemoved int    `json:"emojis_removed"`
	BackupPath    string `json:"backup_path,omitempty"`
	Error         error  `json:"error,omitempty"`
}

// DefaultModifyConfig returns a default configuration for file modification.
func DefaultModifyConfig() ModifyConfig {
	return ModifyConfig{
		Replacement:         "",
		CreateBackup:        false,
		RespectAllowlist:    true,
		PreservePermissions: true,
		DryRun:              false,
	}
}

// ModifyFile modifies a single file to remove emojis according to the configuration.
// This function performs atomic file operations to prevent data corruption.
func ModifyFile(filePath string, patterns types.EmojiPatterns, config ModifyConfig,
	emojiAllowlist *allowlist.Allowlist) types.Result[ModifyResult] {

	result := ModifyResult{
		FilePath: filePath,
		Success:  false,
		Modified: false,
	}

	// Read original file content
	contentResult := fs.ReadFile(filePath)
	if contentResult.IsErr() {
		result.Error = contentResult.Error()
		return types.Ok(result)
	}

	originalContent := string(contentResult.Unwrap())

	// Detect emojis in the content
	detectionResult := detector.DetectEmojis([]byte(originalContent), patterns)
	if detectionResult.IsErr() {
		result.Error = detectionResult.Error()
		return types.Ok(result)
	}

	detection := detectionResult.Unwrap()

	// Apply allowlist filtering if configured
	// When respecting allowlist, we need to create a new detection result with only non-allowed emojis
	if config.RespectAllowlist && emojiAllowlist != nil {
		filteredEmojis := make([]types.EmojiMatch, 0)
		for _, emoji := range detection.Emojis {
			if !emojiAllowlist.IsAllowed(emoji.Emoji) {
				filteredEmojis = append(filteredEmojis, emoji)
			}
		}

		// Create new detection result with only non-allowed emojis (these will be removed)
		detection = types.DetectionResult{
			Emojis:         filteredEmojis,
			TotalCount:     len(filteredEmojis),
			ProcessedBytes: detection.ProcessedBytes,
			Duration:       detection.Duration,
			Success:        detection.Success,
		}
		detection.Finalize()
	}

	// If no emojis to remove, return success without modification
	if detection.TotalCount == 0 {
		result.Success = true
		return types.Ok(result)
	}

	// Create backup if requested
	if config.CreateBackup {
		backupResult := CreateBackup(filePath)
		if backupResult.IsErr() {
			result.Error = fmt.Errorf("failed to create backup: %w", backupResult.Error())
			return types.Ok(result)
		}
		result.BackupPath = backupResult.Unwrap()
	}

	// Remove emojis from content
	modifiedContent := RemoveEmojis(originalContent, detection, config.Replacement)

	// In dry-run mode, don't actually modify the file
	if config.DryRun {
		result.Success = true
		result.Modified = true
		result.EmojisRemoved = detection.TotalCount
		return types.Ok(result)
	}

	// Get original file permissions
	var fileMode os.FileMode = 0644
	if config.PreservePermissions {
		if stat, err := os.Stat(filePath); err == nil {
			fileMode = stat.Mode().Perm()
		}
	}

	// Write modified content atomically
	writeResult := AtomicWriteFile(filePath, []byte(modifiedContent), fileMode)
	if writeResult.IsErr() {
		result.Error = fmt.Errorf("failed to write file: %w", writeResult.Error())
		return types.Ok(result)
	}

	result.Success = true
	result.Modified = true
	result.EmojisRemoved = detection.TotalCount

	return types.Ok(result)
}

// ModifyFiles modifies multiple files to remove emojis.
func ModifyFiles(filePaths []string, patterns types.EmojiPatterns, config ModifyConfig,
	emojiAllowlist *allowlist.Allowlist) []ModifyResult {

	results := make([]ModifyResult, 0, len(filePaths))

	for _, filePath := range filePaths {
		modifyResult := ModifyFile(filePath, patterns, config, emojiAllowlist)
		if modifyResult.IsOk() {
			results = append(results, modifyResult.Unwrap())
		} else {
			// This shouldn't happen with current implementation
			errorResult := ModifyResult{
				FilePath: filePath,
				Success:  false,
				Error:    modifyResult.Error(),
			}
			results = append(results, errorResult)
		}
	}

	return results
}

// CreateBackup creates a backup copy of the specified file.
func CreateBackup(filePath string) types.Result[string] {
	// Generate backup filename with timestamp
	timestamp := time.Now().Format("20060102-150405")
	dir := filepath.Dir(filePath)
	base := filepath.Base(filePath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	backupPath := filepath.Join(dir, fmt.Sprintf("%s.backup.%s%s", name, timestamp, ext))

	// Read original content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return types.Err[string](err)
	}

	// Get original permissions
	stat, err := os.Stat(filePath)
	if err != nil {
		return types.Err[string](err)
	}

	// Write backup file
	err = os.WriteFile(backupPath, content, stat.Mode().Perm())
	if err != nil {
		return types.Err[string](err)
	}

	return types.Ok(backupPath)
}

// AtomicWriteFile writes data to a file atomically by writing to a temporary file first.
func AtomicWriteFile(filePath string, data []byte, perm os.FileMode) types.Result[struct{}] {
	dir := filepath.Dir(filePath)

	// Check if file exists and get its permissions
	var existingMode os.FileMode
	if stat, err := os.Stat(filePath); err == nil {
		existingMode = stat.Mode().Perm()
	} else {
		existingMode = perm
	}

	// Create temporary file in the same directory
	tmpFile, err := os.CreateTemp(dir, ".antimoji-tmp-*")
	if err != nil {
		return types.Err[struct{}](err)
	}
	tmpPath := tmpFile.Name()

	// Clean up on error
	defer func() {
		if tmpFile != nil {
			_ = tmpFile.Close()
			_ = os.Remove(tmpPath)
		}
	}()

	// Write data to temporary file
	_, err = tmpFile.Write(data)
	if err != nil {
		return types.Err[struct{}](err)
	}

	// Sync to ensure data is written
	err = tmpFile.Sync()
	if err != nil {
		return types.Err[struct{}](err)
	}

	// Close temporary file
	err = tmpFile.Close()
	tmpFile = nil // Mark as closed
	if err != nil {
		return types.Err[struct{}](err)
	}

	// Set permissions on temporary file (use existing file permissions if available)
	err = os.Chmod(tmpPath, existingMode)
	if err != nil {
		return types.Err[struct{}](err)
	}

	// Atomically replace original file
	err = os.Rename(tmpPath, filePath)
	if err != nil {
		return types.Err[struct{}](err)
	}

	return types.Ok(struct{}{})
}

// RemoveEmojis removes detected emojis from content and replaces them with the specified replacement.
// This is a pure function that does not modify external state.
func RemoveEmojis(content string, detectionResult types.DetectionResult, replacement string) string {
	if len(detectionResult.Emojis) == 0 {
		return content
	}

	// Sort emojis by position in reverse order to avoid position shifts
	emojis := make([]types.EmojiMatch, len(detectionResult.Emojis))
	copy(emojis, detectionResult.Emojis)
	sort.Slice(emojis, func(i, j int) bool {
		return emojis[i].Start > emojis[j].Start
	})

	// Remove emojis from end to beginning to avoid position shifts
	result := content
	for _, emoji := range emojis {
		if emoji.Start >= 0 && emoji.End <= len(result) && emoji.End > emoji.Start {
			result = result[:emoji.Start] + replacement + result[emoji.End:]
		}
	}

	return result
}
