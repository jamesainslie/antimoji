// Package filtering provides tests for the unified file filtering engine.
package filtering

import (
	"path/filepath"
	"testing"

	"github.com/antimoji/antimoji/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestFileFilterEngine_Precedence(t *testing.T) {
	// Create a profile with various filter rules
	profile := config.Profile{
		IncludePatterns:     []string{"*.go"},
		ExcludePatterns:     []string{"*_test.go"},
		FileIgnoreList:      []string{"*.min.js"},
		DirectoryIgnoreList: []string{"vendor", "node_modules"},
	}

	tests := []struct {
		name          string
		cmdInclude    string
		cmdExclude    string
		filePath      string
		expectInclude bool
		expectStage   string
		expectRule    string
	}{
		{
			name:          "command-line exclude overrides everything",
			cmdExclude:    "*.go",
			filePath:      "main.go",
			expectInclude: false,
			expectStage:   "command_line",
			expectRule:    "command_line.exclude",
		},
		{
			name:          "command-line include overrides profile exclude",
			cmdInclude:    "*_test.go",
			filePath:      "main_test.go",
			expectInclude: true,
			expectStage:   "command_line",
			expectRule:    "command_line.include",
		},
		{
			name:          "command-line include rejects non-matching",
			cmdInclude:    "*.go",
			filePath:      "app.js",
			expectInclude: false,
			expectStage:   "command_line",
			expectRule:    "command_line.include_mismatch",
		},
		{
			name:          "profile exclude works without command-line override",
			filePath:      "main_test.go",
			expectInclude: false,
			expectStage:   "profile",
			expectRule:    "profile.exclude_patterns",
		},
		{
			name:          "profile include works with matching pattern",
			filePath:      "main.go",
			expectInclude: true,
			expectStage:   "profile",
			expectRule:    "profile.include_patterns",
		},
		{
			name:          "directory ignore works",
			filePath:      "vendor/package/file.go",
			expectInclude: false,
			expectStage:   "profile",
			expectRule:    "profile.directory_ignore_list",
		},
		{
			name:          "file ignore works",
			filePath:      "app.min.js",
			expectInclude: false,
			expectStage:   "profile",
			expectRule:    "profile.file_ignore_list",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewFileFilterEngine(profile).
				WithCommandLineFilters(tt.cmdInclude, tt.cmdExclude)

			decision := engine.ShouldInclude(tt.filePath)

			assert.Equal(t, tt.expectInclude, decision.Include, "Include decision mismatch")
			assert.Equal(t, tt.expectStage, decision.Stage, "Stage mismatch")
			assert.Contains(t, decision.Rule, tt.expectRule, "Rule mismatch")
			assert.NotEmpty(t, decision.Reason, "Reason should not be empty")
		})
	}
}

func TestFileFilterEngine_DefaultBehavior(t *testing.T) {
	tests := []struct {
		name          string
		profile       config.Profile
		filePath      string
		expectInclude bool
		expectReason  string
	}{
		{
			name: "default allow when no include patterns",
			profile: config.Profile{
				ExcludePatterns: []string{}, // No excludes
				IncludePatterns: []string{}, // No includes
			},
			filePath:      "anything.txt",
			expectInclude: true,
			expectReason:  "no include patterns specified, default allow",
		},
		{
			name: "exclude takes precedence over default allow",
			profile: config.Profile{
				ExcludePatterns: []string{"*.txt"},
				IncludePatterns: []string{}, // No includes
			},
			filePath:      "readme.txt",
			expectInclude: false,
			expectReason:  "matches profile exclude pattern: *.txt",
		},
		{
			name: "include patterns reject non-matching files",
			profile: config.Profile{
				ExcludePatterns: []string{},
				IncludePatterns: []string{"*.go"}, // Only Go files
			},
			filePath:      "app.js",
			expectInclude: false,
			expectReason:  "does not match any profile include patterns",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewFileFilterEngine(tt.profile)
			decision := engine.ShouldInclude(tt.filePath)

			assert.Equal(t, tt.expectInclude, decision.Include)
			assert.Contains(t, decision.Reason, tt.expectReason)
		})
	}
}

func TestGlobMatcher(t *testing.T) {
	matcher := NewGlobMatcher()

	tests := []struct {
		name     string
		pattern  string
		target   string
		expected bool
	}{
		{"simple glob", "*.go", "main.go", true},
		{"simple glob no match", "*.go", "main.js", false},
		{"question mark", "test?.go", "test1.go", true},
		{"question mark no match", "test?.go", "test12.go", false},
		{"exact match", "main.go", "main.go", true},
		{"exact no match", "main.go", "test.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.Match(tt.pattern, tt.target)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGlobMatcher_PathMatching(t *testing.T) {
	matcher := NewGlobMatcher()

	tests := []struct {
		name     string
		pattern  string
		path     string
		expected bool
	}{
		{"directory pattern", "vendor", "vendor/package/file.go", true},
		{"directory pattern with slash", "vendor/", "vendor/package/file.go", true},
		{"directory pattern exact", "vendor", "src/vendor/file.go", true},
		{"directory pattern basename", "vendor", "some/path/vendor", true},
		{"directory pattern no match", "vendor", "src/lib/file.go", false},
		{"file pattern in path", "*.go", "src/main.go", true},
		{"file pattern no match", "*.js", "src/main.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.MatchPath(tt.pattern, tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterAnalyzer(t *testing.T) {
	profile := config.Profile{
		IncludePatterns: []string{"*.go"},
		ExcludePatterns: []string{"*_test.go"},
	}

	engine := NewFileFilterEngine(profile)
	analyzer := NewFilterAnalyzer(engine)

	tests := []struct {
		name          string
		filePath      string
		expectInclude bool
	}{
		{"included go file", "main.go", true},
		{"excluded test file", "main_test.go", false},
		{"non-matching extension", "app.js", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analysis := analyzer.AnalyzeFile(tt.filePath)

			assert.Equal(t, tt.filePath, analysis.FilePath)
			assert.Equal(t, tt.expectInclude, analysis.Decision.Include)
			assert.NotEmpty(t, analysis.Decision.Reason)
			assert.NotEmpty(t, analysis.FileInfo.Name)
			assert.Equal(t, filepath.Base(tt.filePath), analysis.FileInfo.Name)
			assert.Equal(t, filepath.Ext(tt.filePath), analysis.FileInfo.Extension)
			assert.Equal(t, filepath.Dir(tt.filePath), analysis.FileInfo.Directory)
		})
	}
}

func TestFilterDecision_String(t *testing.T) {
	decision := FilterDecision{
		Include: true,
		Reason:  "matches include pattern",
		Rule:    "profile.include_patterns",
		Stage:   "profile",
	}

	str := decision.String()
	assert.Contains(t, str, "INCLUDE")
	assert.Contains(t, str, "matches include pattern")
	assert.Contains(t, str, "profile.include_patterns")
	assert.Contains(t, str, "profile")
}

func TestFileFilterEngine_RealWorldScenarios(t *testing.T) {
	// Simulate a real Go project configuration
	goProjectProfile := config.Profile{
		IncludePatterns:     []string{"*.go", "*.mod", "*.sum"},
		ExcludePatterns:     []string{"*_test.go", "*.pb.go"},
		FileIgnoreList:      []string{"*.min.js", "*.min.css"},
		DirectoryIgnoreList: []string{"vendor", "dist", ".git"},
	}

	engine := NewFileFilterEngine(goProjectProfile)

	tests := []struct {
		name          string
		filePath      string
		expectInclude bool
		expectReason  string
	}{
		{
			name:          "include main go file",
			filePath:      "cmd/main.go",
			expectInclude: true,
			expectReason:  "matches profile include pattern: *.go",
		},
		{
			name:          "exclude test file",
			filePath:      "pkg/util_test.go",
			expectInclude: false,
			expectReason:  "matches profile exclude pattern: *_test.go",
		},
		{
			name:          "exclude vendor directory",
			filePath:      "vendor/github.com/pkg/file.go",
			expectInclude: false,
			expectReason:  "in ignored directory: vendor",
		},
		{
			name:          "exclude protobuf generated file",
			filePath:      "api/service.pb.go",
			expectInclude: false,
			expectReason:  "matches profile exclude pattern: *.pb.go",
		},
		{
			name:          "include go.mod file",
			filePath:      "go.mod",
			expectInclude: true,
			expectReason:  "matches profile include pattern: *.mod",
		},
		{
			name:          "exclude non-go file",
			filePath:      "README.md",
			expectInclude: false,
			expectReason:  "does not match any profile include patterns",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := engine.ShouldInclude(tt.filePath)

			assert.Equal(t, tt.expectInclude, decision.Include)
			assert.Contains(t, decision.Reason, tt.expectReason)
		})
	}
}
