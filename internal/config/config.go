// Package config provides configuration management with XDG compliance and Viper integration.
package config

import (
	"fmt"

	"github.com/antimoji/antimoji/internal/types"
	"github.com/spf13/viper"
)

// Config represents the complete application configuration.
type Config struct {
	Profiles map[string]Profile `yaml:"profiles" json:"profiles"`
}

// Profile represents a configuration profile with specific settings.
type Profile struct {
	// File processing
	Recursive      bool `yaml:"recursive" json:"recursive"`
	FollowSymlinks bool `yaml:"follow_symlinks" json:"follow_symlinks"`
	BackupFiles    bool `yaml:"backup_files" json:"backup_files"`

	// Emoji detection
	UnicodeEmojis  bool     `yaml:"unicode_emojis" json:"unicode_emojis"`
	TextEmoticons  bool     `yaml:"text_emoticons" json:"text_emoticons"`
	CustomPatterns []string `yaml:"custom_patterns" json:"custom_patterns"`

	// Allowlist and ignore functionality
	EmojiAllowlist      []string `yaml:"emoji_allowlist" json:"emoji_allowlist"`
	FileIgnoreList      []string `yaml:"file_ignore_list" json:"file_ignore_list"`
	DirectoryIgnoreList []string `yaml:"directory_ignore_list" json:"directory_ignore_list"`

	// Replacement behavior
	Replacement        string `yaml:"replacement" json:"replacement"`
	PreserveWhitespace bool   `yaml:"preserve_whitespace" json:"preserve_whitespace"`

	// File filters
	IncludePatterns []string `yaml:"include_patterns" json:"include_patterns"`
	ExcludePatterns []string `yaml:"exclude_patterns" json:"exclude_patterns"`

	// CI/CD and linting
	FailOnFound       bool `yaml:"fail_on_found" json:"fail_on_found"`
	MaxEmojiThreshold int  `yaml:"max_emoji_threshold" json:"max_emoji_threshold"`
	ExitCodeOnFound   int  `yaml:"exit_code_on_found" json:"exit_code_on_found"`

	// Performance
	MaxWorkers  int   `yaml:"max_workers" json:"max_workers"`
	BufferSize  int   `yaml:"buffer_size" json:"buffer_size"`
	MaxFileSize int64 `yaml:"max_file_size" json:"max_file_size"`

	// Output
	OutputFormat  string `yaml:"output_format" json:"output_format"`
	ShowProgress  bool   `yaml:"show_progress" json:"show_progress"`
	ColoredOutput bool   `yaml:"colored_output" json:"colored_output"`
}

// LoadConfig loads configuration from the specified file path.
func LoadConfig(configPath string) types.Result[Config] {
	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return types.Err[Config](err)
	}

	config := Config{
		Profiles: make(map[string]Profile),
	}

	// Load profiles manually to handle the nested structure
	profilesMap := v.GetStringMap("profiles")
	for profileName := range profilesMap {
		profile, err := loadProfile(v, profileName)
		if err != nil {
			return types.Err[Config](err)
		}
		config.Profiles[profileName] = profile
	}

	return types.Ok(config)
}

// loadProfile loads a single profile from Viper.
func loadProfile(v *viper.Viper, profileName string) (Profile, error) {
	prefix := "profiles." + profileName

	profile := Profile{
		// File processing
		Recursive:      v.GetBool(prefix + ".recursive"),
		FollowSymlinks: v.GetBool(prefix + ".follow_symlinks"),
		BackupFiles:    v.GetBool(prefix + ".backup_files"),

		// Emoji detection
		UnicodeEmojis:  v.GetBool(prefix + ".unicode_emojis"),
		TextEmoticons:  v.GetBool(prefix + ".text_emoticons"),
		CustomPatterns: v.GetStringSlice(prefix + ".custom_patterns"),

		// Allowlist and ignore functionality
		EmojiAllowlist:      v.GetStringSlice(prefix + ".emoji_allowlist"),
		FileIgnoreList:      v.GetStringSlice(prefix + ".file_ignore_list"),
		DirectoryIgnoreList: v.GetStringSlice(prefix + ".directory_ignore_list"),

		// Replacement behavior
		Replacement:        v.GetString(prefix + ".replacement"),
		PreserveWhitespace: v.GetBool(prefix + ".preserve_whitespace"),

		// File filters
		IncludePatterns: v.GetStringSlice(prefix + ".include_patterns"),
		ExcludePatterns: v.GetStringSlice(prefix + ".exclude_patterns"),

		// CI/CD and linting
		FailOnFound:       v.GetBool(prefix + ".fail_on_found"),
		MaxEmojiThreshold: v.GetInt(prefix + ".max_emoji_threshold"),
		ExitCodeOnFound:   v.GetInt(prefix + ".exit_code_on_found"),

		// Performance
		MaxWorkers:  v.GetInt(prefix + ".max_workers"),
		BufferSize:  v.GetInt(prefix + ".buffer_size"),
		MaxFileSize: v.GetInt64(prefix + ".max_file_size"),

		// Output
		OutputFormat:  v.GetString(prefix + ".output_format"),
		ShowProgress:  v.GetBool(prefix + ".show_progress"),
		ColoredOutput: v.GetBool(prefix + ".colored_output"),
	}

	return profile, nil
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	return Config{
		Profiles: map[string]Profile{
			"default": {
				// File processing
				Recursive:      true,
				FollowSymlinks: false,
				BackupFiles:    false,

				// Emoji detection
				UnicodeEmojis:  true,
				TextEmoticons:  true,
				CustomPatterns: []string{}, // No custom patterns by default

				// Allowlist and ignore functionality
				EmojiAllowlist: []string{}, // No default allowlist - will be empty by default
				FileIgnoreList: []string{
					"*.min.js", "*.min.css", "vendor/**/*", "node_modules/**/*",
					".git/**/*", "**/*.generated.*",
				},
				DirectoryIgnoreList: []string{
					".git", "node_modules", "vendor", "dist", "build",
				},

				// Replacement behavior
				Replacement:        "",
				PreserveWhitespace: true,

				// File filters - empty include patterns means include all files
				IncludePatterns: []string{}, // Empty = include all files (unless excluded)
				ExcludePatterns: []string{"vendor/*", "node_modules/*", ".git/*"},

				// CI/CD and linting
				FailOnFound:       false,
				MaxEmojiThreshold: 0,
				ExitCodeOnFound:   1,

				// Performance
				MaxWorkers:  0, // Auto-detect CPU cores
				BufferSize:  64 * 1024,
				MaxFileSize: 100 * 1024 * 1024, // 100MB

				// Output
				OutputFormat:  "table",
				ShowProgress:  true,
				ColoredOutput: true,
			},
		},
	}
}

// GetProfile retrieves a specific profile from the configuration.
func GetProfile(config Config, profileName string) types.Result[Profile] {
	if profileName == "" {
		profileName = "default"
	}

	profile, exists := config.Profiles[profileName]
	if !exists {
		return types.Err[Profile](fmt.Errorf("profile not found: %s", profileName))
	}

	return types.Ok(profile)
}

// ValidateConfig validates the configuration for correctness.
func ValidateConfig(config Config) types.Result[Config] {
	// Validate each profile
	for name, profile := range config.Profiles {
		if err := validateProfile(name, profile); err != nil {
			return types.Err[Config](err)
		}
	}

	return types.Ok(config)
}

// validateProfile validates a single profile configuration.
func validateProfile(name string, profile Profile) error {
	if profile.BufferSize < 0 {
		return fmt.Errorf("profile %s: buffer size cannot be negative", name)
	}

	if profile.MaxFileSize < 0 {
		return fmt.Errorf("profile %s: max file size cannot be negative", name)
	}

	if profile.MaxWorkers < 0 {
		return fmt.Errorf("profile %s: max workers cannot be negative", name)
	}

	if profile.MaxEmojiThreshold < 0 {
		return fmt.Errorf("profile %s: max emoji threshold cannot be negative", name)
	}

	// Validate output format
	validFormats := []string{"table", "json", "csv"}
	validFormat := false
	for _, format := range validFormats {
		if profile.OutputFormat == format {
			validFormat = true
			break
		}
	}
	if !validFormat && profile.OutputFormat != "" {
		return fmt.Errorf("profile %s: invalid output format: %s", name, profile.OutputFormat)
	}

	return nil
}

// ToProcessingConfig converts a Profile to a ProcessingConfig.
func ToProcessingConfig(profile Profile) types.ProcessingConfig {
	return types.ProcessingConfig{
		EnableUnicode:   profile.UnicodeEmojis,
		EnableEmoticons: profile.TextEmoticons,
		EnableCustom:    len(profile.CustomPatterns) > 0,
		MaxFileSize:     profile.MaxFileSize,
		BufferSize:      profile.BufferSize,
	}
}

// MergeProfiles merges two profiles, with the override taking precedence.
// For this implementation, we'll use explicit field copying for clarity.
func MergeProfiles(base, override Profile) Profile {
	result := base

	// For boolean fields, we need to check if they were explicitly set
	// Since Go doesn't have optional types, we'll assume any difference means it was set
	// This is a simplified approach for now

	// Override specific fields that are different from zero values
	if len(override.CustomPatterns) > 0 {
		result.CustomPatterns = override.CustomPatterns
	}
	if len(override.EmojiAllowlist) > 0 {
		result.EmojiAllowlist = override.EmojiAllowlist
	}
	if override.MaxFileSize > 0 {
		result.MaxFileSize = override.MaxFileSize
	}
	if override.BufferSize > 0 {
		result.BufferSize = override.BufferSize
	}
	if override.OutputFormat != "" {
		result.OutputFormat = override.OutputFormat
	}

	// For this simple implementation, we'll just override boolean fields explicitly
	// In a real implementation, we might use pointers or a more sophisticated approach
	result.Recursive = override.Recursive
	result.UnicodeEmojis = override.UnicodeEmojis
	result.TextEmoticons = override.TextEmoticons

	return result
}
