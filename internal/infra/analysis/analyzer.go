// Package analysis provides deep configuration analysis capabilities.
package analysis

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/antimoji/antimoji/internal/config"
	"github.com/antimoji/antimoji/internal/infra/filtering"
)

// ConfigAnalyzer provides deep analysis of antimoji configurations.
type ConfigAnalyzer struct {
	profile      config.Profile
	filterEngine *filtering.FileFilterEngine
	targetDir    string
}

// NewConfigAnalyzer creates a new configuration analyzer.
func NewConfigAnalyzer(profile config.Profile, targetDir string) *ConfigAnalyzer {
	return &ConfigAnalyzer{
		profile:      profile,
		filterEngine: filtering.NewFileFilterEngine(profile),
		targetDir:    targetDir,
	}
}

// AnalyzeConfiguration performs comprehensive analysis of the configuration.
func (ca *ConfigAnalyzer) AnalyzeConfiguration() ConfigurationAnalysis {
	analysis := ConfigurationAnalysis{
		Profile:   ca.profile,
		TargetDir: ca.targetDir,
	}

	// Analyze emoji policy
	analysis.PolicyAnalysis = ca.analyzePolicySettings()

	// Analyze file filtering
	analysis.FilterAnalysis = ca.analyzeFileFiltering()

	// Analyze codebase impact
	analysis.ImpactAnalysis = ca.analyzeCodebaseImpact()

	// Generate recommendations
	analysis.Recommendations = ca.generateRecommendations(analysis)

	return analysis
}

// analyzePolicySettings analyzes the emoji policy configuration.
func (ca *ConfigAnalyzer) analyzePolicySettings() PolicyAnalysis {
	analysis := PolicyAnalysis{}

	// Determine policy type
	if ca.profile.MaxEmojiThreshold == 0 && len(ca.profile.EmojiAllowlist) == 0 {
		analysis.PolicyType = "zero-tolerance"
		analysis.Strictness = "maximum"
		analysis.Description = "NO emojis allowed anywhere in source code"
	} else if len(ca.profile.EmojiAllowlist) > 0 {
		analysis.PolicyType = "allow-list"
		analysis.Strictness = "moderate"
		analysis.Description = fmt.Sprintf("Only %d specific emojis allowed", len(ca.profile.EmojiAllowlist))
		analysis.AllowedEmojis = ca.profile.EmojiAllowlist
	} else if ca.profile.MaxEmojiThreshold > 15 {
		analysis.PolicyType = "permissive"
		analysis.Strictness = "low"
		analysis.Description = fmt.Sprintf("Allows up to %d emojis with warnings", ca.profile.MaxEmojiThreshold)
	} else {
		analysis.PolicyType = "custom"
		analysis.Strictness = "variable"
		analysis.Description = "Custom policy with specific threshold and rules"
	}

	// Analyze threshold settings
	analysis.Threshold = ca.profile.MaxEmojiThreshold
	analysis.FailBehavior = ca.profile.FailOnFound
	analysis.ExitCode = ca.profile.ExitCodeOnFound

	// Analyze detection settings
	analysis.DetectionMethods = []string{}
	if ca.profile.UnicodeEmojis {
		analysis.DetectionMethods = append(analysis.DetectionMethods, "unicode-emojis")
	}
	if ca.profile.TextEmoticons {
		analysis.DetectionMethods = append(analysis.DetectionMethods, "text-emoticons")
	}
	if len(ca.profile.CustomPatterns) > 0 {
		analysis.DetectionMethods = append(analysis.DetectionMethods, "custom-patterns")
		analysis.CustomPatternCount = len(ca.profile.CustomPatterns)
	}

	return analysis
}

// analyzeFileFiltering analyzes the file filtering configuration.
func (ca *ConfigAnalyzer) analyzeFileFiltering() FilteringAnalysis {
	analysis := FilteringAnalysis{}

	// Analyze include patterns
	if len(ca.profile.IncludePatterns) > 0 {
		analysis.IncludeStrategy = "explicit-patterns"
		analysis.IncludePatterns = ca.profile.IncludePatterns
		analysis.IncludeDescription = fmt.Sprintf("Explicitly includes %d file patterns", len(ca.profile.IncludePatterns))
	} else {
		analysis.IncludeStrategy = "default-allow"
		analysis.IncludeDescription = "Includes all files unless explicitly excluded"
	}

	// Analyze exclude patterns
	totalExcludes := len(ca.profile.ExcludePatterns) + len(ca.profile.FileIgnoreList) + len(ca.profile.DirectoryIgnoreList)
	if totalExcludes > 0 {
		analysis.ExcludeStrategy = "pattern-based"
		analysis.ExcludePatterns = append(ca.profile.ExcludePatterns, ca.profile.FileIgnoreList...)
		analysis.ExcludeDirectories = ca.profile.DirectoryIgnoreList
		analysis.ExcludeDescription = fmt.Sprintf("Excludes %d patterns and %d directories",
			len(analysis.ExcludePatterns), len(analysis.ExcludeDirectories))
	} else {
		analysis.ExcludeStrategy = "none"
		analysis.ExcludeDescription = "No exclusions configured"
	}

	// Analyze filter complexity
	if totalExcludes > 20 || len(ca.profile.IncludePatterns) > 10 {
		analysis.Complexity = "high"
	} else if totalExcludes > 5 || len(ca.profile.IncludePatterns) > 3 {
		analysis.Complexity = "medium"
	} else {
		analysis.Complexity = "low"
	}

	return analysis
}

// analyzeCodebaseImpact analyzes the impact on the target codebase.
func (ca *ConfigAnalyzer) analyzeCodebaseImpact() ImpactAnalysis {
	analysis := ImpactAnalysis{
		TargetDirectory: ca.targetDir,
	}

	// Scan directory to estimate impact
	fileCount := 0
	emojiCount := 0
	fileTypes := make(map[string]int)

	filepath.Walk(ca.targetDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		// Check if file would be included
		decision := ca.filterEngine.ShouldInclude(path)
		if decision.Include {
			fileCount++

			// Count file types
			ext := strings.ToLower(filepath.Ext(path))
			if ext == "" {
				ext = "no-extension"
			}
			fileTypes[ext]++

			// Simple emoji counting (basic implementation)
			if content, err := os.ReadFile(path); err == nil {
				emojiCount += ca.countEmojisInContent(string(content))
			}
		}

		return nil
	})

	analysis.FilesToScan = fileCount
	analysis.CurrentEmojis = emojiCount
	analysis.FileTypeBreakdown = fileTypes

	// Estimate impact based on policy
	if ca.profile.MaxEmojiThreshold == 0 && len(ca.profile.EmojiAllowlist) == 0 {
		analysis.EstimatedRemovals = emojiCount
		analysis.ImpactLevel = "high"
		analysis.ImpactDescription = fmt.Sprintf("Will remove all %d emojis found", emojiCount)
	} else if len(ca.profile.EmojiAllowlist) > 0 {
		// Estimate how many emojis would be removed (simplified)
		estimatedRemovals := emojiCount / 2 // Rough estimate
		analysis.EstimatedRemovals = estimatedRemovals
		analysis.ImpactLevel = "medium"
		analysis.ImpactDescription = fmt.Sprintf("Will remove approximately %d non-allowed emojis", estimatedRemovals)
	} else {
		analysis.EstimatedRemovals = 0
		analysis.ImpactLevel = "low"
		analysis.ImpactDescription = "Will warn about excessive emoji usage"
	}

	return analysis
}

// countEmojisInContent provides basic emoji counting.
func (ca *ConfigAnalyzer) countEmojisInContent(content string) int {
	count := 0
	// Simplified emoji detection - count common emojis
	commonEmojis := []string{"", "", "", "", "", "", "", "", "", "", "", "", "", ""}

	for _, emoji := range commonEmojis {
		count += strings.Count(content, emoji)
	}

	return count
}

// generateRecommendations generates recommendations based on the analysis.
func (ca *ConfigAnalyzer) generateRecommendations(analysis ConfigurationAnalysis) []Recommendation {
	var recommendations []Recommendation

	// Policy recommendations
	if analysis.PolicyAnalysis.PolicyType == "zero-tolerance" && analysis.ImpactAnalysis.CurrentEmojis > 50 {
		recommendations = append(recommendations, Recommendation{
			Type:        "policy",
			Severity:    "medium",
			Title:       "Consider gradual emoji removal",
			Description: fmt.Sprintf("Found %d emojis. Consider using allow-list mode first to gradually reduce emoji usage.", analysis.ImpactAnalysis.CurrentEmojis),
			Suggestion:  "Run: antimoji setup-lint --mode=allow-list --allowed-emojis=\",\"",
		})
	}

	// Performance recommendations
	if analysis.ImpactAnalysis.FilesToScan > 1000 {
		recommendations = append(recommendations, Recommendation{
			Type:        "performance",
			Severity:    "low",
			Title:       "Consider performance optimization",
			Description: fmt.Sprintf("Scanning %d files. Consider adding more exclude patterns for better performance.", analysis.ImpactAnalysis.FilesToScan),
			Suggestion:  "Add more specific file filters to reduce scan scope",
		})
	}

	// Filter recommendations
	if analysis.FilterAnalysis.Complexity == "high" {
		recommendations = append(recommendations, Recommendation{
			Type:        "configuration",
			Severity:    "low",
			Title:       "Simplify filter rules",
			Description: "Complex filter configuration may be hard to maintain.",
			Suggestion:  "Consider consolidating filter patterns",
		})
	}

	return recommendations
}

// ConfigurationAnalysis represents a comprehensive analysis of a configuration.
type ConfigurationAnalysis struct {
	Profile         config.Profile    `json:"profile"`
	TargetDir       string            `json:"target_dir"`
	PolicyAnalysis  PolicyAnalysis    `json:"policy_analysis"`
	FilterAnalysis  FilteringAnalysis `json:"filter_analysis"`
	ImpactAnalysis  ImpactAnalysis    `json:"impact_analysis"`
	Recommendations []Recommendation  `json:"recommendations"`
}

// PolicyAnalysis provides analysis of the emoji policy.
type PolicyAnalysis struct {
	PolicyType         string   `json:"policy_type"`
	Strictness         string   `json:"strictness"`
	Description        string   `json:"description"`
	Threshold          int      `json:"threshold"`
	AllowedEmojis      []string `json:"allowed_emojis"`
	FailBehavior       bool     `json:"fail_behavior"`
	ExitCode           int      `json:"exit_code"`
	DetectionMethods   []string `json:"detection_methods"`
	CustomPatternCount int      `json:"custom_pattern_count"`
}

// FilteringAnalysis provides analysis of file filtering configuration.
type FilteringAnalysis struct {
	IncludeStrategy    string   `json:"include_strategy"`
	IncludePatterns    []string `json:"include_patterns"`
	IncludeDescription string   `json:"include_description"`
	ExcludeStrategy    string   `json:"exclude_strategy"`
	ExcludePatterns    []string `json:"exclude_patterns"`
	ExcludeDirectories []string `json:"exclude_directories"`
	ExcludeDescription string   `json:"exclude_description"`
	Complexity         string   `json:"complexity"`
}

// ImpactAnalysis provides analysis of the configuration's impact on the codebase.
type ImpactAnalysis struct {
	TargetDirectory   string         `json:"target_directory"`
	FilesToScan       int            `json:"files_to_scan"`
	CurrentEmojis     int            `json:"current_emojis"`
	EstimatedRemovals int            `json:"estimated_removals"`
	ImpactLevel       string         `json:"impact_level"`
	ImpactDescription string         `json:"impact_description"`
	FileTypeBreakdown map[string]int `json:"file_type_breakdown"`
}

// Recommendation provides actionable recommendations.
type Recommendation struct {
	Type        string `json:"type"`     // "policy", "performance", "configuration"
	Severity    string `json:"severity"` // "high", "medium", "low"
	Title       string `json:"title"`
	Description string `json:"description"`
	Suggestion  string `json:"suggestion"`
}

// String returns a human-readable representation of the recommendation.
func (r Recommendation) String() string {
	severityPrefix := ""
	switch r.Severity {
	case "high":
		severityPrefix = "IMPORTANT"
	case "medium":
		severityPrefix = "RECOMMENDED"
	case "low":
		severityPrefix = "SUGGESTION"
	}

	return fmt.Sprintf("[%s] %s: %s\n  %s", severityPrefix, r.Title, r.Description, r.Suggestion)
}
