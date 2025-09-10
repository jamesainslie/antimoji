package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTemplateRegistry(t *testing.T) {
	t.Run("creates registry with built-in templates", func(t *testing.T) {
		registry := NewTemplateRegistry()

		assert.NotNil(t, registry)
		assert.NotNil(t, registry.templates)

		// Check that built-in templates are registered
		templates := registry.ListTemplates()
		assert.Contains(t, templates, "zero-tolerance")
		assert.Contains(t, templates, "allow-list")
		assert.Contains(t, templates, "permissive")
	})
}

func TestTemplateRegistry_ApplyTemplate(t *testing.T) {
	registry := NewTemplateRegistry()

	t.Run("applies zero-tolerance template", func(t *testing.T) {
		options := TemplateOptions{
			AllowedEmojis: []string{"✅", "❌"},
			Threshold:     5,
		}

		profile, err := registry.ApplyTemplate("zero-tolerance", options)

		assert.NoError(t, err)
		assert.True(t, profile.Recursive)
		assert.True(t, profile.UnicodeEmojis)
		assert.True(t, profile.TextEmoticons)
		assert.Equal(t, 0, profile.MaxEmojiThreshold) // Zero tolerance ignores threshold
		assert.True(t, profile.FailOnFound)
		assert.Equal(t, 1, profile.ExitCodeOnFound)
		assert.Empty(t, profile.EmojiAllowlist) // Zero tolerance has empty allowlist
	})

	t.Run("applies allow-list template with customization", func(t *testing.T) {
		options := TemplateOptions{
			AllowedEmojis: []string{"🚀", "✨", "🎉"},
			Threshold:     10,
		}

		profile, err := registry.ApplyTemplate("allow-list", options)

		assert.NoError(t, err)
		assert.True(t, profile.Recursive)
		assert.True(t, profile.UnicodeEmojis)
		assert.True(t, profile.TextEmoticons)
		assert.Equal(t, 10, profile.MaxEmojiThreshold) // Custom threshold applied
		assert.True(t, profile.FailOnFound)
		assert.Equal(t, 1, profile.ExitCodeOnFound)
		assert.Equal(t, []string{"🚀", "✨", "🎉"}, profile.EmojiAllowlist) // Custom allowlist applied
	})

	t.Run("applies allow-list template with default options", func(t *testing.T) {
		options := TemplateOptions{} // Empty options

		profile, err := registry.ApplyTemplate("allow-list", options)

		assert.NoError(t, err)
		assert.Equal(t, 5, profile.MaxEmojiThreshold) // Default threshold
		// The template uses empty emoji strings that render as actual emojis
		assert.Equal(t, 2, len(profile.EmojiAllowlist)) // Default allowlist has 2 items
	})

	t.Run("applies permissive template", func(t *testing.T) {
		options := TemplateOptions{}

		profile, err := registry.ApplyTemplate("permissive", options)

		assert.NoError(t, err)
		assert.True(t, profile.Recursive)
		assert.True(t, profile.UnicodeEmojis)
		assert.True(t, profile.TextEmoticons)
		assert.Equal(t, 20, profile.MaxEmojiThreshold)
		assert.False(t, profile.FailOnFound) // Permissive doesn't fail
		assert.Equal(t, 0, profile.ExitCodeOnFound)
		assert.True(t, len(profile.EmojiAllowlist) > 0) // Has generous allowlist
	})

	t.Run("returns error for unknown template", func(t *testing.T) {
		options := TemplateOptions{}

		profile, err := registry.ApplyTemplate("nonexistent", options)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "template not found")
		assert.Equal(t, Profile{}, profile)
	})
}

func TestTemplateRegistry_ListTemplates(t *testing.T) {
	registry := NewTemplateRegistry()

	t.Run("lists all available templates", func(t *testing.T) {
		templates := registry.ListTemplates()

		assert.Len(t, templates, 3)
		assert.Contains(t, templates, "zero-tolerance")
		assert.Contains(t, templates, "allow-list")
		assert.Contains(t, templates, "permissive")

		// Check descriptions are meaningful
		assert.Contains(t, templates["zero-tolerance"], "Strict")
		assert.Contains(t, templates["allow-list"], "Allow-list")
		assert.Contains(t, templates["permissive"], "Permissive")
	})
}

func TestGetBuiltInProfile(t *testing.T) {
	t.Run("creates zero-tolerance profile", func(t *testing.T) {
		options := TemplateOptions{}

		profile, err := GetBuiltInProfile("zero-tolerance", options)

		assert.NoError(t, err)
		assert.Equal(t, 0, profile.MaxEmojiThreshold)
		assert.True(t, profile.FailOnFound)
		assert.Empty(t, profile.EmojiAllowlist)
	})

	t.Run("creates allow-list profile with custom options", func(t *testing.T) {
		options := TemplateOptions{
			AllowedEmojis: []string{"📝", "🔧"},
			Threshold:     3,
		}

		profile, err := GetBuiltInProfile("allow-list", options)

		assert.NoError(t, err)
		assert.Equal(t, 3, profile.MaxEmojiThreshold)
		assert.Equal(t, []string{"📝", "🔧"}, profile.EmojiAllowlist)
	})

	t.Run("returns error for invalid template", func(t *testing.T) {
		options := TemplateOptions{}

		profile, err := GetBuiltInProfile("invalid", options)

		assert.Error(t, err)
		assert.Equal(t, Profile{}, profile)
	})
}

func TestTemplateConfigurationConsistency(t *testing.T) {
	registry := NewTemplateRegistry()

	t.Run("zero-tolerance template consistency", func(t *testing.T) {
		profile, err := registry.ApplyTemplate("zero-tolerance", TemplateOptions{})
		require.NoError(t, err)

		// Zero tolerance should be strict
		assert.Equal(t, 0, profile.MaxEmojiThreshold)
		assert.True(t, profile.FailOnFound)
		assert.Equal(t, 1, profile.ExitCodeOnFound)
		assert.Empty(t, profile.EmojiAllowlist)

		// Should detect all emoji types
		assert.True(t, profile.UnicodeEmojis)
		assert.True(t, profile.TextEmoticons)
		assert.True(t, len(profile.CustomPatterns) > 0)

		// Should have reasonable file filtering
		assert.True(t, len(profile.IncludePatterns) > 0)
		assert.True(t, len(profile.ExcludePatterns) > 0)
		assert.Contains(t, profile.ExcludePatterns, "vendor/*")
		assert.Contains(t, profile.ExcludePatterns, "node_modules/*")
	})

	t.Run("allow-list template consistency", func(t *testing.T) {
		profile, err := registry.ApplyTemplate("allow-list", TemplateOptions{})
		require.NoError(t, err)

		// Allow-list should be moderate
		assert.True(t, profile.MaxEmojiThreshold > 0)
		assert.True(t, profile.FailOnFound)
		assert.Equal(t, 1, profile.ExitCodeOnFound)
		assert.True(t, len(profile.EmojiAllowlist) > 0)

		// Should have same detection and filtering as zero-tolerance
		assert.True(t, profile.UnicodeEmojis)
		assert.True(t, profile.TextEmoticons)
		assert.True(t, len(profile.IncludePatterns) > 0)
		assert.Contains(t, profile.ExcludePatterns, "vendor/*")
	})

	t.Run("permissive template consistency", func(t *testing.T) {
		profile, err := registry.ApplyTemplate("permissive", TemplateOptions{})
		require.NoError(t, err)

		// Permissive should be lenient
		assert.True(t, profile.MaxEmojiThreshold > 10)
		assert.False(t, profile.FailOnFound) // Key difference
		assert.Equal(t, 0, profile.ExitCodeOnFound)
		assert.True(t, len(profile.EmojiAllowlist) > 10) // Generous allowlist

		// Should still detect emojis but be more forgiving
		assert.True(t, profile.UnicodeEmojis)
		assert.True(t, profile.TextEmoticons)

		// Should have fewer exclusions than strict templates
		assert.True(t, len(profile.ExcludePatterns) < 10) // More permissive filtering
	})
}

func TestTemplateCustomization(t *testing.T) {
	registry := NewTemplateRegistry()

	t.Run("allow-list customizer works correctly", func(t *testing.T) {
		// Test with custom emojis only
		options := TemplateOptions{
			AllowedEmojis: []string{"🎯", "🔥", "💡"},
		}

		profile, err := registry.ApplyTemplate("allow-list", options)
		require.NoError(t, err)

		assert.Equal(t, []string{"🎯", "🔥", "💡"}, profile.EmojiAllowlist)
		assert.Equal(t, 5, profile.MaxEmojiThreshold) // Default threshold
	})

	t.Run("allow-list customizer with threshold only", func(t *testing.T) {
		options := TemplateOptions{
			Threshold: 15,
		}

		profile, err := registry.ApplyTemplate("allow-list", options)
		require.NoError(t, err)

		assert.Equal(t, 15, profile.MaxEmojiThreshold)
		// The template uses empty emoji strings that render as actual emojis
		assert.Equal(t, 2, len(profile.EmojiAllowlist)) // Default allowlist has 2 items
	})

	t.Run("allow-list customizer with both options", func(t *testing.T) {
		options := TemplateOptions{
			AllowedEmojis: []string{"🚀"},
			Threshold:     1,
		}

		profile, err := registry.ApplyTemplate("allow-list", options)
		require.NoError(t, err)

		assert.Equal(t, []string{"🚀"}, profile.EmojiAllowlist)
		assert.Equal(t, 1, profile.MaxEmojiThreshold)
	})

	t.Run("zero-tolerance ignores customization", func(t *testing.T) {
		// Zero tolerance should ignore customization options
		options := TemplateOptions{
			AllowedEmojis: []string{"🚀", "✨"},
			Threshold:     10,
		}

		profile, err := registry.ApplyTemplate("zero-tolerance", options)
		require.NoError(t, err)

		// Should maintain zero-tolerance settings regardless of options
		assert.Equal(t, 0, profile.MaxEmojiThreshold)
		assert.Empty(t, profile.EmojiAllowlist)
		assert.True(t, profile.FailOnFound)
	})

	t.Run("permissive ignores customization", func(t *testing.T) {
		// Permissive should ignore customization options
		options := TemplateOptions{
			AllowedEmojis: []string{"🚀"},
			Threshold:     1,
		}

		profile, err := registry.ApplyTemplate("permissive", options)
		require.NoError(t, err)

		// Should maintain permissive settings
		assert.Equal(t, 20, profile.MaxEmojiThreshold)
		assert.True(t, len(profile.EmojiAllowlist) > 10) // Original generous allowlist
		assert.False(t, profile.FailOnFound)
	})
}

func TestTemplateOptions(t *testing.T) {
	t.Run("TemplateOptions struct", func(t *testing.T) {
		options := TemplateOptions{
			AllowedEmojis: []string{"🎉", "🚀"},
			Threshold:     5,
			TargetDir:     "/test",
			IncludeTests:  true,
			IncludeDocs:   false,
		}

		assert.Equal(t, []string{"🎉", "🚀"}, options.AllowedEmojis)
		assert.Equal(t, 5, options.Threshold)
		assert.Equal(t, "/test", options.TargetDir)
		assert.True(t, options.IncludeTests)
		assert.False(t, options.IncludeDocs)
	})
}

func TestConfigTemplate(t *testing.T) {
	t.Run("ConfigTemplate struct", func(t *testing.T) {
		template := ConfigTemplate{
			Name:        "test",
			Description: "test template",
			BaseProfile: Profile{
				Recursive: true,
			},
			Customizer: func(profile Profile, options TemplateOptions) Profile {
				profile.MaxEmojiThreshold = options.Threshold
				return profile
			},
		}

		assert.Equal(t, "test", template.Name)
		assert.Equal(t, "test template", template.Description)
		assert.True(t, template.BaseProfile.Recursive)
		assert.NotNil(t, template.Customizer)

		// Test customizer function
		options := TemplateOptions{Threshold: 10}
		customized := template.Customizer(template.BaseProfile, options)
		assert.Equal(t, 10, customized.MaxEmojiThreshold)
	})
}

func TestTemplateFileFiltering(t *testing.T) {
	registry := NewTemplateRegistry()

	t.Run("all templates include common source file types", func(t *testing.T) {
		templateNames := []string{"zero-tolerance", "allow-list", "permissive"}
		expectedPatterns := []string{"*.go", "*.js", "*.py", "*.java"}

		for _, templateName := range templateNames {
			profile, err := registry.ApplyTemplate(templateName, TemplateOptions{})
			require.NoError(t, err, "template %s should apply successfully", templateName)

			for _, pattern := range expectedPatterns {
				assert.Contains(t, profile.IncludePatterns, pattern,
					"template %s should include pattern %s", templateName, pattern)
			}
		}
	})

	t.Run("strict templates exclude test files", func(t *testing.T) {
		strictTemplates := []string{"zero-tolerance", "allow-list"}

		for _, templateName := range strictTemplates {
			profile, err := registry.ApplyTemplate(templateName, TemplateOptions{})
			require.NoError(t, err)

			// Should exclude test files
			testExclusions := []string{"*_test.go", "*/test/*", "*/tests/*"}
			for _, exclusion := range testExclusions {
				assert.Contains(t, profile.ExcludePatterns, exclusion,
					"template %s should exclude %s", templateName, exclusion)
			}
		}
	})

	t.Run("all templates exclude common directories", func(t *testing.T) {
		templateNames := []string{"zero-tolerance", "allow-list", "permissive"}
		commonDirs := []string{".git", "node_modules", "vendor"}

		for _, templateName := range templateNames {
			profile, err := registry.ApplyTemplate(templateName, TemplateOptions{})
			require.NoError(t, err)

			for _, dir := range commonDirs {
				assert.Contains(t, profile.DirectoryIgnoreList, dir,
					"template %s should ignore directory %s", templateName, dir)
			}
		}
	})
}

func TestTemplatePerformanceSettings(t *testing.T) {
	registry := NewTemplateRegistry()

	t.Run("all templates have reasonable performance settings", func(t *testing.T) {
		templateNames := []string{"zero-tolerance", "allow-list", "permissive"}

		for _, templateName := range templateNames {
			profile, err := registry.ApplyTemplate(templateName, TemplateOptions{})
			require.NoError(t, err)

			assert.Equal(t, 64*1024, profile.BufferSize,
				"template %s should have 64KB buffer", templateName)
			assert.Equal(t, int64(100*1024*1024), profile.MaxFileSize,
				"template %s should have 100MB max file size", templateName)
			assert.Equal(t, 0, profile.MaxWorkers,
				"template %s should use auto worker detection", templateName)
		}
	})
}
