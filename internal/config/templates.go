// Package config provides configuration templates for common use cases.
package config

import (
	"fmt"
)

// ConfigTemplate represents a reusable configuration template.
type ConfigTemplate struct {
	Name        string
	Description string
	BaseProfile Profile
	Customizer  func(profile Profile, options TemplateOptions) Profile
}

// TemplateOptions holds customization options for templates.
type TemplateOptions struct {
	AllowedEmojis []string
	Threshold     int
	TargetDir     string
	IncludeTests  bool
	IncludeDocs   bool
}

// TemplateRegistry manages configuration templates.
type TemplateRegistry struct {
	templates map[string]ConfigTemplate
}

// NewTemplateRegistry creates a new template registry with built-in templates.
func NewTemplateRegistry() *TemplateRegistry {
	registry := &TemplateRegistry{
		templates: make(map[string]ConfigTemplate),
	}

	// Register built-in templates
	registry.registerBuiltInTemplates()

	return registry
}

// ApplyTemplate applies a template with the given options to create a profile.
func (tr *TemplateRegistry) ApplyTemplate(templateName string, options TemplateOptions) (Profile, error) {
	template, exists := tr.templates[templateName]
	if !exists {
		return Profile{}, fmt.Errorf("template not found: %s", templateName)
	}

	// Start with base profile
	profile := template.BaseProfile

	// Apply customizations if customizer exists
	if template.Customizer != nil {
		profile = template.Customizer(profile, options)
	}

	return profile, nil
}

// ListTemplates returns available template names and descriptions.
func (tr *TemplateRegistry) ListTemplates() map[string]string {
	result := make(map[string]string)
	for name, template := range tr.templates {
		result[name] = template.Description
	}
	return result
}

// registerBuiltInTemplates registers the built-in configuration templates.
func (tr *TemplateRegistry) registerBuiltInTemplates() {
	// Zero Tolerance Template
	tr.templates["zero-tolerance"] = ConfigTemplate{
		Name:        "zero-tolerance",
		Description: "Strict emoji-free codebase configuration",
		BaseProfile: Profile{
			Recursive: true,

			// Emoji detection
			UnicodeEmojis:  true,
			TextEmoticons:  true,
			CustomPatterns: []string{"", "", "", "", "", "", "", ":x:"},

			// Zero tolerance policy
			EmojiAllowlist:    []string{}, // No emojis allowed
			MaxEmojiThreshold: 0,
			FailOnFound:       true,
			ExitCodeOnFound:   1,

			// File filtering - focus on source code
			IncludePatterns: []string{
				"*.go", "*.js", "*.ts", "*.jsx", "*.tsx", "*.py", "*.rb",
				"*.java", "*.c", "*.cpp", "*.h", "*.hpp", "*.rs", "*.php",
				"*.swift", "*.kt", "*.scala",
			},
			ExcludePatterns: []string{
				"vendor/*", "node_modules/*", ".git/*", "dist/*", "build/*",
				"*_test.go", "*/test/*", "*/tests/*", "*/testdata/*", "*/fixtures/*",
				"*.md", "docs/*",
			},
			FileIgnoreList: []string{
				"*.min.js", "*.min.css", "vendor/**/*", "node_modules/**/*",
				".git/**/*", "**/*.generated.*", "**/*.pb.go", "**/wire_gen.go",
				"README.md", "CHANGELOG.md", "*.md", "docs/**/*",
			},
			DirectoryIgnoreList: []string{
				".git", "node_modules", "vendor", "dist", "build", "docs",
			},

			// Performance
			MaxWorkers:  0,
			BufferSize:  64 * 1024,
			MaxFileSize: 100 * 1024 * 1024,

			// Output
			OutputFormat:  "table",
			ShowProgress:  false,
			ColoredOutput: true,
		},
	}

	// Allow List Template
	tr.templates["allow-list"] = ConfigTemplate{
		Name:        "allow-list",
		Description: "Allow-list configuration with specific emojis",
		BaseProfile: Profile{
			Recursive: true,

			// Emoji detection
			UnicodeEmojis:  true,
			TextEmoticons:  true,
			CustomPatterns: []string{"", "", "", "", "", "", "", ":x:"},

			// Allow-list policy (will be customized)
			EmojiAllowlist:    []string{"", ""}, // Default allowlist
			MaxEmojiThreshold: 5,
			FailOnFound:       true,
			ExitCodeOnFound:   1,

			// File filtering - same as zero tolerance
			IncludePatterns: []string{
				"*.go", "*.js", "*.ts", "*.jsx", "*.tsx", "*.py", "*.rb",
				"*.java", "*.c", "*.cpp", "*.h", "*.hpp", "*.rs", "*.php",
				"*.swift", "*.kt", "*.scala",
			},
			ExcludePatterns: []string{
				"vendor/*", "node_modules/*", ".git/*", "dist/*", "build/*",
				"*_test.go", "*/test/*", "*/tests/*", "*/testdata/*", "*/fixtures/*",
				"*.md", "docs/*",
			},
			FileIgnoreList: []string{
				"*.min.js", "*.min.css", "vendor/**/*", "node_modules/**/*",
				".git/**/*", "**/*.generated.*", "**/*.pb.go", "**/wire_gen.go",
				"README.md", "CHANGELOG.md", "*.md", "docs/**/*",
			},
			DirectoryIgnoreList: []string{
				".git", "node_modules", "vendor", "dist", "build", "docs",
			},

			// Performance
			MaxWorkers:  0,
			BufferSize:  64 * 1024,
			MaxFileSize: 100 * 1024 * 1024,

			// Output
			OutputFormat:  "table",
			ShowProgress:  false,
			ColoredOutput: true,
		},
		Customizer: func(profile Profile, options TemplateOptions) Profile {
			// Apply custom allowlist
			if len(options.AllowedEmojis) > 0 {
				profile.EmojiAllowlist = options.AllowedEmojis
			}

			// Apply custom threshold
			if options.Threshold > 0 {
				profile.MaxEmojiThreshold = options.Threshold
			}

			return profile
		},
	}

	// Permissive Template
	tr.templates["permissive"] = ConfigTemplate{
		Name:        "permissive",
		Description: "Permissive configuration that warns but doesn't fail",
		BaseProfile: Profile{
			Recursive: true,

			// Emoji detection
			UnicodeEmojis:  true,
			TextEmoticons:  true,
			CustomPatterns: []string{"", "", "", "", "", "", "", ":x:"},

			// Permissive policy
			EmojiAllowlist: []string{
				"", "", "", "", "", "", "", "", "", "",
				"", "", "", "", "", "", "", "", "", "",
			},
			MaxEmojiThreshold: 20,
			FailOnFound:       false, // Don't fail, just warn
			ExitCodeOnFound:   0,     // Don't exit with error

			// More lenient file filtering
			IncludePatterns: []string{
				"*.go", "*.js", "*.ts", "*.jsx", "*.tsx", "*.py", "*.rb",
				"*.java", "*.c", "*.cpp", "*.h", "*.hpp", "*.rs", "*.php",
				"*.swift", "*.kt", "*.scala",
			},
			ExcludePatterns: []string{
				"vendor/*", "node_modules/*", ".git/*", "dist/*", "build/*",
			},
			FileIgnoreList: []string{
				"*.min.js", "*.min.css", "vendor/**/*", "node_modules/**/*",
				".git/**/*", "**/*.generated.*", "**/*.pb.go", "**/wire_gen.go",
			},
			DirectoryIgnoreList: []string{
				".git", "node_modules", "vendor", "dist", "build",
			},

			// Performance
			MaxWorkers:  0,
			BufferSize:  64 * 1024,
			MaxFileSize: 100 * 1024 * 1024,

			// Output
			OutputFormat:  "table",
			ShowProgress:  false,
			ColoredOutput: true,
		},
	}
}

// GetBuiltInProfile creates a profile from a built-in template with custom options.
func GetBuiltInProfile(templateName string, options TemplateOptions) (Profile, error) {
	registry := NewTemplateRegistry()
	return registry.ApplyTemplate(templateName, options)
}
