// Package config provides comprehensive configuration validation with helpful error messages.
package config

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ValidationLevel defines the severity of validation issues.
type ValidationLevel string

const (
	ValidationLevelError   ValidationLevel = "error"
	ValidationLevelWarning ValidationLevel = "warning"
	ValidationLevelInfo    ValidationLevel = "info"
)

// ValidationIssue represents a configuration validation issue with suggestions.
type ValidationIssue struct {
	Level      ValidationLevel `json:"level"`
	Field      string          `json:"field"`
	Value      interface{}     `json:"value,omitempty"`
	Message    string          `json:"message"`
	Suggestion string          `json:"suggestion,omitempty"`
	Example    string          `json:"example,omitempty"`
}

// String returns a human-readable representation of the validation issue.
func (vi ValidationIssue) String() string {
	prefix := strings.ToUpper(string(vi.Level))
	result := fmt.Sprintf("[%s] %s: %s", prefix, vi.Field, vi.Message)
	
	if vi.Suggestion != "" {
		result += fmt.Sprintf("\n  Suggestion: %s", vi.Suggestion)
	}
	
	if vi.Example != "" {
		result += fmt.Sprintf("\n  Example: %s", vi.Example)
	}
	
	return result
}

// ConfigValidator provides comprehensive configuration validation.
type ConfigValidator struct {
	issues []ValidationIssue
}

// NewConfigValidator creates a new configuration validator.
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{
		issues: make([]ValidationIssue, 0),
	}
}

// ValidateConfig performs comprehensive validation of a configuration.
func (cv *ConfigValidator) ValidateConfig(config Config) ValidationResult {
	cv.issues = make([]ValidationIssue, 0)
	
	// Validate basic structure
	if len(config.Profiles) == 0 {
		cv.addError("profiles", nil, "no profiles defined", 
			"add at least one profile", 
			"profiles:\n  default:\n    unicode_emojis: true")
	}
	
	// Validate each profile
	for name, profile := range config.Profiles {
		cv.validateProfile(name, profile)
	}
	
	// Validate cross-profile consistency
	cv.validateCrossProfileConsistency(config)
	
	return ValidationResult{
		IsValid: cv.countErrors() == 0,
		Issues:  cv.issues,
		Summary: cv.generateSummary(),
	}
}

// validateProfile validates a single profile with detailed feedback.
func (cv *ConfigValidator) validateProfile(name string, profile Profile) {
	fieldPrefix := fmt.Sprintf("profiles.%s", name)
	
	// Validate emoji policy consistency
	cv.validateEmojiPolicyConsistency(fieldPrefix, profile)
	
	// Validate file filtering logic
	cv.validateFileFilteringLogic(fieldPrefix, profile)
	
	// Validate performance settings
	cv.validatePerformanceSettings(fieldPrefix, profile)
	
	// Validate output settings
	cv.validateOutputSettings(fieldPrefix, profile)
	
	// Check for common misconfigurations
	cv.checkCommonMisconfigurations(fieldPrefix, profile)
}

// validateEmojiPolicyConsistency validates emoji policy for logical consistency.
func (cv *ConfigValidator) validateEmojiPolicyConsistency(fieldPrefix string, profile Profile) {
	// Check zero tolerance consistency
	if profile.MaxEmojiThreshold == 0 && len(profile.EmojiAllowlist) > 0 {
		cv.addWarning(fieldPrefix+".emoji_allowlist", profile.EmojiAllowlist,
			"zero threshold with non-empty allowlist is contradictory",
			"either increase threshold or remove allowlist",
			"max_emoji_threshold: 5  # or emoji_allowlist: []")
	}
	
	// Check allowlist without threshold
	if len(profile.EmojiAllowlist) > 0 && profile.MaxEmojiThreshold == 0 {
		cv.addWarning(fieldPrefix+".max_emoji_threshold", profile.MaxEmojiThreshold,
			"allowlist specified but threshold is zero",
			"set threshold to match allowlist size",
			fmt.Sprintf("max_emoji_threshold: %d", len(profile.EmojiAllowlist)))
	}
	
	// Check unrealistic thresholds
	if profile.MaxEmojiThreshold > 100 {
		cv.addWarning(fieldPrefix+".max_emoji_threshold", profile.MaxEmojiThreshold,
			"very high emoji threshold may not be effective",
			"consider a lower threshold for better emoji control",
			"max_emoji_threshold: 20")
	}
	
	// Check detection method consistency
	if !profile.UnicodeEmojis && !profile.TextEmoticons && len(profile.CustomPatterns) == 0 {
		cv.addError(fieldPrefix+".emoji_detection", nil,
			"no emoji detection methods enabled",
			"enable at least one detection method",
			"unicode_emojis: true\ntext_emoticons: true")
	}
	
	// Check fail behavior consistency
	if !profile.FailOnFound && profile.ExitCodeOnFound > 0 {
		cv.addWarning(fieldPrefix+".exit_code_on_found", profile.ExitCodeOnFound,
			"exit code set but fail_on_found is false",
			"either enable fail_on_found or set exit_code_on_found to 0",
			"fail_on_found: true  # or exit_code_on_found: 0")
	}
}

// validateFileFilteringLogic validates file filtering for effectiveness.
func (cv *ConfigValidator) validateFileFilteringLogic(fieldPrefix string, profile Profile) {
	// Check for overly restrictive includes
	if len(profile.IncludePatterns) == 1 && profile.IncludePatterns[0] == "*" {
		cv.addInfo(fieldPrefix+".include_patterns", profile.IncludePatterns,
			"include pattern '*' is redundant",
			"remove include_patterns to include all files by default",
			"# include_patterns: []  # or remove this line")
	}
	
	// Check for conflicting patterns
	for _, includePattern := range profile.IncludePatterns {
		for _, excludePattern := range profile.ExcludePatterns {
			if includePattern == excludePattern {
				cv.addError(fieldPrefix+".patterns", 
					fmt.Sprintf("include: %s, exclude: %s", includePattern, excludePattern),
					"same pattern in both include and exclude",
					"remove from one of the lists",
					"decide whether to include or exclude this pattern")
			}
		}
	}
	
	// Check for ineffective patterns
	for i, pattern := range profile.ExcludePatterns {
		if pattern == "" {
			cv.addError(fmt.Sprintf("%s.exclude_patterns[%d]", fieldPrefix, i), pattern,
				"empty exclude pattern",
				"remove empty pattern",
				"exclude_patterns: [\"*.min.js\", \"vendor/*\"]")
		}
		
		if _, err := filepath.Match(pattern, "test"); err != nil {
			cv.addError(fmt.Sprintf("%s.exclude_patterns[%d]", fieldPrefix, i), pattern,
				fmt.Sprintf("invalid pattern syntax: %s", err),
				"use valid glob pattern syntax",
				"exclude_patterns: [\"*.min.js\", \"*_test.go\"]")
		}
	}
	
	// Check for redundant directory ignores
	commonDirs := []string{".git", "node_modules", "vendor"}
	missingCommonDirs := []string{}
	for _, dir := range commonDirs {
		found := false
		for _, ignored := range profile.DirectoryIgnoreList {
			if ignored == dir {
				found = true
				break
			}
		}
		if !found {
			missingCommonDirs = append(missingCommonDirs, dir)
		}
	}
	
	if len(missingCommonDirs) > 0 {
		cv.addInfo(fieldPrefix+".directory_ignore_list", profile.DirectoryIgnoreList,
			fmt.Sprintf("consider ignoring common directories: %s", strings.Join(missingCommonDirs, ", ")),
			"add common directories to ignore list",
			fmt.Sprintf("directory_ignore_list: [\"%s\"]", strings.Join(missingCommonDirs, "\", \"")))
	}
}

// validatePerformanceSettings validates performance configuration.
func (cv *ConfigValidator) validatePerformanceSettings(fieldPrefix string, profile Profile) {
	// Check buffer size
	if profile.BufferSize > 0 && profile.BufferSize < 1024 {
		cv.addWarning(fieldPrefix+".buffer_size", profile.BufferSize,
			"very small buffer size may impact performance",
			"use at least 4KB for better performance",
			"buffer_size: 4096")
	}
	
	if profile.BufferSize > 10*1024*1024 { // 10MB
		cv.addWarning(fieldPrefix+".buffer_size", profile.BufferSize,
			"very large buffer size may impact memory usage",
			"consider smaller buffer size unless processing huge files",
			"buffer_size: 65536  # 64KB")
	}
	
	// Check file size limits
	if profile.MaxFileSize > 0 && profile.MaxFileSize < 1024 {
		cv.addWarning(fieldPrefix+".max_file_size", profile.MaxFileSize,
			"very small max file size may skip legitimate files",
			"increase file size limit",
			"max_file_size: 1048576  # 1MB")
	}
	
	// Check worker configuration
	if profile.MaxWorkers > 32 {
		cv.addWarning(fieldPrefix+".max_workers", profile.MaxWorkers,
			"very high worker count may not improve performance",
			"consider using auto-detection or lower number",
			"max_workers: 0  # auto-detect")
	}
}

// validateOutputSettings validates output configuration.
func (cv *ConfigValidator) validateOutputSettings(fieldPrefix string, profile Profile) {
	validFormats := []string{"table", "json", "csv"}
	if profile.OutputFormat != "" {
		isValid := false
		for _, format := range validFormats {
			if profile.OutputFormat == format {
				isValid = true
				break
			}
		}
		if !isValid {
			cv.addError(fieldPrefix+".output_format", profile.OutputFormat,
				fmt.Sprintf("invalid output format: %s", profile.OutputFormat),
				fmt.Sprintf("use one of: %s", strings.Join(validFormats, ", ")),
				"output_format: \"table\"")
		}
	}
}

// checkCommonMisconfigurations checks for common configuration mistakes.
func (cv *ConfigValidator) checkCommonMisconfigurations(fieldPrefix string, profile Profile) {
	// Check for test files in production config
	if profile.FailOnFound && len(profile.IncludePatterns) > 0 {
		includesTests := false
		for _, pattern := range profile.IncludePatterns {
			if strings.Contains(pattern, "test") {
				includesTests = true
				break
			}
		}
		
		if includesTests {
			cv.addWarning(fieldPrefix+".include_patterns", profile.IncludePatterns,
				"including test files in strict linting may be too restrictive",
				"consider excluding test files for more practical linting",
				"exclude_patterns: [\"*_test.go\", \"test_*\"]")
		}
	}
	
	// Check for missing common exclusions
	if len(profile.FileIgnoreList) == 0 && len(profile.ExcludePatterns) == 0 {
		cv.addInfo(fieldPrefix+".exclusions", nil,
			"no file exclusions configured",
			"consider excluding generated files and dependencies",
			"file_ignore_list: [\"*.min.js\", \"*.generated.*\"]\ndirectory_ignore_list: [\"vendor\", \"node_modules\"]")
	}
}

// validateCrossProfileConsistency validates consistency across all profiles.
func (cv *ConfigValidator) validateCrossProfileConsistency(config Config) {
	// Check for profiles with identical configurations
	profileConfigs := make(map[string][]string)
	
	for name, profile := range config.Profiles {
		key := fmt.Sprintf("%d-%d-%t", profile.MaxEmojiThreshold, len(profile.EmojiAllowlist), profile.FailOnFound)
		profileConfigs[key] = append(profileConfigs[key], name)
	}
	
	for _, profiles := range profileConfigs {
		if len(profiles) > 1 {
			cv.addInfo("profiles", profiles,
				fmt.Sprintf("profiles have identical configurations: %s", strings.Join(profiles, ", ")),
				"consider consolidating duplicate profiles",
				"remove duplicate profiles or differentiate their configurations")
		}
	}
}

// Helper methods for adding validation issues with rich context
func (cv *ConfigValidator) addError(field string, value interface{}, message, suggestion, example string) {
	cv.issues = append(cv.issues, ValidationIssue{
		Level:      ValidationLevelError,
		Field:      field,
		Value:      value,
		Message:    message,
		Suggestion: suggestion,
		Example:    example,
	})
}

func (cv *ConfigValidator) addWarning(field string, value interface{}, message, suggestion, example string) {
	cv.issues = append(cv.issues, ValidationIssue{
		Level:      ValidationLevelWarning,
		Field:      field,
		Value:      value,
		Message:    message,
		Suggestion: suggestion,
		Example:    example,
	})
}

func (cv *ConfigValidator) addInfo(field string, value interface{}, message, suggestion, example string) {
	cv.issues = append(cv.issues, ValidationIssue{
		Level:      ValidationLevelInfo,
		Field:      field,
		Value:      value,
		Message:    message,
		Suggestion: suggestion,
		Example:    example,
	})
}

// countErrors returns the number of error-level issues.
func (cv *ConfigValidator) countErrors() int {
	count := 0
	for _, issue := range cv.issues {
		if issue.Level == ValidationLevelError {
			count++
		}
	}
	return count
}

// generateSummary generates a validation summary.
func (cv *ConfigValidator) generateSummary() ValidationSummary {
	summary := ValidationSummary{
		TotalIssues: len(cv.issues),
		Errors:      0,
		Warnings:    0,
		Infos:       0,
	}
	
	for _, issue := range cv.issues {
		switch issue.Level {
		case ValidationLevelError:
			summary.Errors++
		case ValidationLevelWarning:
			summary.Warnings++
		case ValidationLevelInfo:
			summary.Infos++
		}
	}
	
	return summary
}

// ValidationResult represents the result of configuration validation.
type ValidationResult struct {
	IsValid bool               `json:"is_valid"`
	Issues  []ValidationIssue  `json:"issues"`
	Summary ValidationSummary  `json:"summary"`
}

// ValidationSummary provides a summary of validation results.
type ValidationSummary struct {
	TotalIssues int `json:"total_issues"`
	Errors      int `json:"errors"`
	Warnings    int `json:"warnings"`
	Infos       int `json:"infos"`
}

// String returns a human-readable validation summary.
func (vs ValidationSummary) String() string {
	if vs.TotalIssues == 0 {
		return "Configuration is valid with no issues"
	}
	
	parts := []string{}
	if vs.Errors > 0 {
		parts = append(parts, fmt.Sprintf("%d errors", vs.Errors))
	}
	if vs.Warnings > 0 {
		parts = append(parts, fmt.Sprintf("%d warnings", vs.Warnings))
	}
	if vs.Infos > 0 {
		parts = append(parts, fmt.Sprintf("%d suggestions", vs.Infos))
	}
	
	return fmt.Sprintf("Configuration has %s", strings.Join(parts, ", "))
}

// HasErrors checks if there are any error-level issues.
func (vr ValidationResult) HasErrors() bool {
	return vr.Summary.Errors > 0
}

// GetErrorMessages returns all error messages.
func (vr ValidationResult) GetErrorMessages() []string {
	var messages []string
	for _, issue := range vr.Issues {
		if issue.Level == ValidationLevelError {
			messages = append(messages, issue.Message)
		}
	}
	return messages
}

// String returns a human-readable validation result.
func (vr ValidationResult) String() string {
	if vr.IsValid && len(vr.Issues) == 0 {
		return "Configuration is valid"
	}
	
	result := fmt.Sprintf("%s:\n", vr.Summary.String())
	for _, issue := range vr.Issues {
		result += fmt.Sprintf("  %s\n", issue.String())
	}
	
	return result
}

// ValidateConfigFile validates a configuration file and returns detailed results.
func ValidateConfigFile(configPath string) ValidationResult {
	// Load configuration
	configResult := LoadConfig(configPath)
	if configResult.IsErr() {
		return ValidationResult{
			IsValid: false,
			Issues: []ValidationIssue{
				{
					Level:      ValidationLevelError,
					Field:      "file",
					Message:    fmt.Sprintf("failed to load configuration: %s", configResult.Error()),
					Suggestion: "check file syntax and permissions",
					Example:    "ensure valid YAML syntax",
				},
			},
			Summary: ValidationSummary{TotalIssues: 1, Errors: 1},
		}
	}
	
	// Validate configuration
	validator := NewConfigValidator()
	return validator.ValidateConfig(configResult.Unwrap())
}

// SuggestImprovements analyzes a profile and suggests improvements.
func SuggestImprovements(profile Profile) []ValidationIssue {
	validator := NewConfigValidator()
	
	// Add improvement suggestions
	if len(profile.EmojiAllowlist) > 20 {
		validator.addInfo("emoji_allowlist", len(profile.EmojiAllowlist),
			"large allowlist may be hard to maintain",
			"consider reducing allowlist or using permissive mode",
			"max_emoji_threshold: 20\nfail_on_found: false")
	}
	
	if len(profile.IncludePatterns) > 10 {
		validator.addInfo("include_patterns", len(profile.IncludePatterns),
			"many include patterns may impact performance",
			"consider consolidating patterns or using exclude patterns instead",
			"exclude_patterns: [\"vendor/*\", \"*.min.*\"]")
	}
	
	return validator.issues
}
