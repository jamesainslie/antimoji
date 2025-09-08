// Package filtering provides file discovery utilities using the unified filtering engine.
package filtering

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/antimoji/antimoji/internal/config"
)

// DiscoveryOptions holds options for file discovery.
type DiscoveryOptions struct {
	Recursive      bool
	IncludePattern string // Command-line include override
	ExcludePattern string // Command-line exclude override
}

// DiscoverFiles discovers files to process using the unified filtering engine.
func DiscoverFiles(args []string, opts DiscoveryOptions, profile config.Profile) ([]string, error) {
	// Create filtering engine
	engine := NewFileFilterEngine(profile).
		WithCommandLineFilters(opts.IncludePattern, opts.ExcludePattern)

	var filePaths []string

	for _, arg := range args {
		stat, err := os.Stat(arg)
		if err != nil {
			// For non-existent files, include them so they show up as errors in results
			filePaths = append(filePaths, arg)
			continue
		}

		if stat.IsDir() {
			if opts.Recursive {
				err := filepath.WalkDir(arg, func(path string, d os.DirEntry, err error) error {
					if err != nil {
						return err
					}

					if d.IsDir() {
						// Check if directory should be ignored using engine
						// Test with a dummy file to check directory rules
						decision := engine.ShouldInclude(filepath.Join(path, "dummy.go"))
						if !decision.Include && (strings.Contains(decision.Rule, "directory") ||
							strings.Contains(decision.Rule, "exclude") ||
							strings.Contains(decision.Rule, "ignore")) {
							return filepath.SkipDir
						}
						return nil
					}

					// Check if file should be included using engine
					decision := engine.ShouldInclude(path)
					if decision.Include {
						filePaths = append(filePaths, path)
					}

					return nil
				})
				if err != nil {
					return nil, err
				}
			} else {
				return nil, fmt.Errorf("directory %s requires --recursive flag", arg)
			}
		} else {
			// Single file - check with engine
			decision := engine.ShouldInclude(arg)
			if decision.Include {
				filePaths = append(filePaths, arg)
			}
		}
	}

	return filePaths, nil
}

// AnalyzeDiscovery provides detailed analysis of file discovery decisions.
func AnalyzeDiscovery(args []string, opts DiscoveryOptions, profile config.Profile) ([]FilterAnalysis, error) {
	engine := NewFileFilterEngine(profile).
		WithCommandLineFilters(opts.IncludePattern, opts.ExcludePattern)
	analyzer := NewFilterAnalyzer(engine)

	var analyses []FilterAnalysis

	for _, arg := range args {
		stat, err := os.Stat(arg)
		if err != nil {
			// Add analysis for non-existent files
			analyses = append(analyses, FilterAnalysis{
				FilePath: arg,
				Decision: FilterDecision{
					Include: true, // Will be included to show error
					Reason:  "file does not exist, included for error reporting",
					Rule:    "error_handling",
					Stage:   "discovery",
				},
				FileInfo: FileInfo{
					Name:      filepath.Base(arg),
					Extension: filepath.Ext(arg),
					Directory: filepath.Dir(arg),
					FullPath:  arg,
				},
			})
			continue
		}

		if stat.IsDir() {
			if opts.Recursive {
				err := filepath.WalkDir(arg, func(path string, d os.DirEntry, err error) error {
					if err != nil {
						return err
					}

					if !d.IsDir() {
						analysis := analyzer.AnalyzeFile(path)
						analyses = append(analyses, analysis)
					}

					return nil
				})
				if err != nil {
					return nil, err
				}
			}
		} else {
			analysis := analyzer.AnalyzeFile(arg)
			analyses = append(analyses, analysis)
		}
	}

	return analyses, nil
}
