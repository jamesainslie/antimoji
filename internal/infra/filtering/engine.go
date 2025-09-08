// Package filtering provides unified file filtering with clear precedence and comprehensive pattern support.
package filtering

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/antimoji/antimoji/internal/config"
)

// FileFilterEngine provides unified file filtering with clear precedence rules.
type FileFilterEngine struct {
	profile    config.Profile
	cmdInclude string // Command-line include pattern
	cmdExclude string // Command-line exclude pattern
	matcher    PatternMatcher
}

// NewFileFilterEngine creates a new file filter engine.
func NewFileFilterEngine(profile config.Profile) *FileFilterEngine {
	return &FileFilterEngine{
		profile: profile,
		matcher: NewGlobMatcher(),
	}
}

// WithCommandLineFilters adds command-line filter overrides.
func (ffe *FileFilterEngine) WithCommandLineFilters(include, exclude string) *FileFilterEngine {
	ffe.cmdInclude = include
	ffe.cmdExclude = exclude
	return ffe
}

// WithMatcher sets a custom pattern matcher.
func (ffe *FileFilterEngine) WithMatcher(matcher PatternMatcher) *FileFilterEngine {
	ffe.matcher = matcher
	return ffe
}

// ShouldInclude determines if a file should be included based on all filter rules.
// Clear precedence order:
// 1. Command-line excludes (highest priority - absolute exclusion)
// 2. Command-line includes (override profile excludes)
// 3. Profile excludes
// 4. Profile includes (default allow if empty)
func (ffe *FileFilterEngine) ShouldInclude(filePath string) FilterDecision {
	fileName := filepath.Base(filePath)
	fileExt := strings.ToLower(filepath.Ext(filePath))
	dirPath := filepath.Dir(filePath)

	// 1. Command-line excludes - absolute priority
	if ffe.cmdExclude != "" {
		if ffe.matcher.Match(ffe.cmdExclude, fileName) || ffe.matcher.Match(ffe.cmdExclude, filePath) {
			return FilterDecision{
				Include: false,
				Reason:  fmt.Sprintf("matches command-line exclude pattern: %s", ffe.cmdExclude),
				Rule:    "command_line.exclude",
				Stage:   "command_line",
			}
		}
	}

	// 2. Command-line includes - override profile excludes
	if ffe.cmdInclude != "" {
		if ffe.matcher.Match(ffe.cmdInclude, fileName) || ffe.matcher.Match(ffe.cmdInclude, filePath) {
			return FilterDecision{
				Include: true,
				Reason:  fmt.Sprintf("matches command-line include pattern: %s", ffe.cmdInclude),
				Rule:    "command_line.include",
				Stage:   "command_line",
			}
		}
		// If command-line include is specified but doesn't match, exclude
		return FilterDecision{
			Include: false,
			Reason:  fmt.Sprintf("does not match command-line include pattern: %s", ffe.cmdInclude),
			Rule:    "command_line.include_mismatch",
			Stage:   "command_line",
		}
	}

	// 3. Profile excludes
	if decision := ffe.checkProfileExcludes(filePath, fileName, fileExt, dirPath); !decision.Include {
		return decision
	}

	// 4. Profile includes (if specified) or default allow
	return ffe.checkProfileIncludes(filePath, fileName, fileExt, dirPath)
}

// checkProfileExcludes checks all profile exclude rules.
func (ffe *FileFilterEngine) checkProfileExcludes(filePath, fileName, fileExt, dirPath string) FilterDecision {
	// Check exclude patterns (legacy field)
	for _, pattern := range ffe.profile.ExcludePatterns {
		if ffe.matcher.Match(pattern, fileName) || ffe.matcher.Match(pattern, filePath) {
			return FilterDecision{
				Include: false,
				Reason:  fmt.Sprintf("matches profile exclude pattern: %s", pattern),
				Rule:    "profile.exclude_patterns",
				Stage:   "profile",
			}
		}
	}

	// Check file ignore list (legacy field)
	for _, pattern := range ffe.profile.FileIgnoreList {
		if ffe.matcher.Match(pattern, fileName) || ffe.matcher.Match(pattern, filePath) {
			return FilterDecision{
				Include: false,
				Reason:  fmt.Sprintf("matches profile file ignore: %s", pattern),
				Rule:    "profile.file_ignore_list",
				Stage:   "profile",
			}
		}
	}

	// Check directory ignore list (legacy field)
	for _, dir := range ffe.profile.DirectoryIgnoreList {
		if ffe.matcher.MatchPath(dir, dirPath) || ffe.matcher.MatchPath(dir, filePath) {
			return FilterDecision{
				Include: false,
				Reason:  fmt.Sprintf("in ignored directory: %s", dir),
				Rule:    "profile.directory_ignore_list",
				Stage:   "profile",
			}
		}
	}

	return FilterDecision{Include: true}
}

// checkProfileIncludes checks profile include rules.
func (ffe *FileFilterEngine) checkProfileIncludes(filePath, fileName, fileExt, dirPath string) FilterDecision {
	// If no include patterns are specified, default to include
	if len(ffe.profile.IncludePatterns) == 0 {
		return FilterDecision{
			Include: true,
			Reason:  "no include patterns specified, default allow",
			Rule:    "profile.include_default",
			Stage:   "profile",
		}
	}

	// Check include patterns
	for _, pattern := range ffe.profile.IncludePatterns {
		if ffe.matcher.Match(pattern, fileName) || ffe.matcher.Match(pattern, filePath) {
			return FilterDecision{
				Include: true,
				Reason:  fmt.Sprintf("matches profile include pattern: %s", pattern),
				Rule:    "profile.include_patterns",
				Stage:   "profile",
			}
		}
	}

	// If include patterns are specified but none match, exclude
	return FilterDecision{
		Include: false,
		Reason:  "does not match any profile include patterns",
		Rule:    "profile.include_mismatch",
		Stage:   "profile",
	}
}

// FilterDecision represents the result of a file filtering decision.
type FilterDecision struct {
	Include bool   `json:"include"`
	Reason  string `json:"reason"`
	Rule    string `json:"rule"`
	Stage   string `json:"stage"` // "command_line", "profile", "default"
}

// String returns a human-readable representation of the filter decision.
func (fd FilterDecision) String() string {
	action := "EXCLUDE"
	if fd.Include {
		action = "INCLUDE"
	}
	return fmt.Sprintf("%s: %s (rule: %s, stage: %s)", action, fd.Reason, fd.Rule, fd.Stage)
}

// PatternMatcher interface defines different pattern matching strategies.
type PatternMatcher interface {
	Match(pattern, target string) bool
	MatchPath(pattern, path string) bool
	MatchGlob(pattern, path string) bool
}

// GlobMatcher implements pattern matching using filepath.Match and enhanced glob support.
type GlobMatcher struct {
	regexCache map[string]*regexp.Regexp
}

// NewGlobMatcher creates a new glob matcher.
func NewGlobMatcher() *GlobMatcher {
	return &GlobMatcher{
		regexCache: make(map[string]*regexp.Regexp),
	}
}

// Match performs basic filepath.Match pattern matching.
func (gm *GlobMatcher) Match(pattern, target string) bool {
	matched, err := filepath.Match(pattern, target)
	return err == nil && matched
}

// MatchPath performs path-aware pattern matching for directories.
func (gm *GlobMatcher) MatchPath(pattern, path string) bool {
	// Handle directory patterns
	if strings.HasSuffix(pattern, "/") || !strings.Contains(pattern, ".") {
		// Directory pattern - check if path contains this directory
		cleanPattern := strings.TrimSuffix(pattern, "/")
		return strings.Contains(path, cleanPattern) ||
			strings.HasPrefix(path, cleanPattern+"/") ||
			filepath.Base(path) == cleanPattern
	}

	// File pattern - check both basename and full path
	return gm.Match(pattern, filepath.Base(path)) || gm.Match(pattern, path)
}

// MatchGlob performs advanced glob pattern matching with ** support.
func (gm *GlobMatcher) MatchGlob(pattern, path string) bool {
	// For now, use basic matching - can be enhanced later
	return gm.Match(pattern, path) || gm.MatchPath(pattern, path)
}

// FilterAnalyzer provides detailed analysis of filter decisions.
type FilterAnalyzer struct {
	engine *FileFilterEngine
}

// NewFilterAnalyzer creates a new filter analyzer.
func NewFilterAnalyzer(engine *FileFilterEngine) *FilterAnalyzer {
	return &FilterAnalyzer{engine: engine}
}

// AnalyzeFile provides detailed analysis of why a file was included or excluded.
func (fa *FilterAnalyzer) AnalyzeFile(filePath string) FilterAnalysis {
	decision := fa.engine.ShouldInclude(filePath)

	return FilterAnalysis{
		FilePath: filePath,
		Decision: decision,
		FileInfo: FileInfo{
			Name:      filepath.Base(filePath),
			Extension: filepath.Ext(filePath),
			Directory: filepath.Dir(filePath),
			FullPath:  filePath,
		},
	}
}

// FilterAnalysis provides detailed analysis of a filter decision.
type FilterAnalysis struct {
	FilePath string         `json:"file_path"`
	Decision FilterDecision `json:"decision"`
	FileInfo FileInfo       `json:"file_info"`
}

// FileInfo provides structured information about a file.
type FileInfo struct {
	Name      string `json:"name"`
	Extension string `json:"extension"`
	Directory string `json:"directory"`
	FullPath  string `json:"full_path"`
}

// String returns a human-readable analysis.
func (fa FilterAnalysis) String() string {
	return fmt.Sprintf("%s: %s", fa.FilePath, fa.Decision.String())
}
