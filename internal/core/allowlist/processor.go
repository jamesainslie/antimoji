// Package allowlist provides unified allowlist processing functionality for consistent behavior across commands.
package allowlist

import (
	"context"

	"github.com/antimoji/antimoji/internal/config"
	"github.com/antimoji/antimoji/internal/observability/logging"
)

// ProcessingOptions contains options for allowlist processing.
type ProcessingOptions struct {
	// IgnoreAllowlist completely ignores the allowlist (takes precedence)
	IgnoreAllowlist bool
	// RespectAllowlist uses the allowlist (default behavior)
	RespectAllowlist bool
	// Operation name for logging purposes
	Operation string
}

// CreateAllowlistForProcessing creates an allowlist based on processing options and profile configuration.
// This function provides consistent allowlist creation logic for both clean and scan commands.
func CreateAllowlistForProcessing(ctx context.Context, profile config.Profile, opts ProcessingOptions) (*Allowlist, error) {
	// If explicitly ignoring allowlist, return nil
	if opts.IgnoreAllowlist {
		logging.Info(ctx, "Allowlist ignored",
			"operation", opts.Operation,
			"ignore_allowlist", true)
		return nil, nil
	}

	// If not respecting allowlist or no allowlist configured, return nil
	if !opts.RespectAllowlist || len(profile.EmojiAllowlist) == 0 {
		logging.Info(ctx, "No allowlist configured or not respecting allowlist",
			"operation", opts.Operation,
			"respect_allowlist", opts.RespectAllowlist,
			"allowlist_size", len(profile.EmojiAllowlist))
		return nil, nil
	}

	// Create allowlist
	allowlistResult := NewAllowlist(profile.EmojiAllowlist)
	if allowlistResult.IsErr() {
		return nil, allowlistResult.Error()
	}

	emojiAllowlist := allowlistResult.Unwrap()
	logging.Info(ctx, "Allowlist configured",
		"operation", opts.Operation,
		"patterns_count", emojiAllowlist.Size(),
		"ignore_allowlist", opts.IgnoreAllowlist,
		"respect_allowlist", opts.RespectAllowlist)

	return emojiAllowlist, nil
}

// ShouldUseAllowlist determines whether an allowlist should be used based on processing options and profile.
// This provides a consistent way to resolve allowlist usage across commands.
func ShouldUseAllowlist(opts ProcessingOptions, profile config.Profile) bool {
	// --ignore-allowlist takes precedence over --respect-allowlist
	if opts.IgnoreAllowlist {
		return false
	}
	// Must both respect allowlist AND have allowlist configured
	return opts.RespectAllowlist && len(profile.EmojiAllowlist) > 0
}

// ValidateConsistentOptions validates that allowlist options are consistent and warns about potential issues.
// This helps catch common pre-commit configuration mistakes.
func ValidateConsistentOptions(ctx context.Context, cleanOpts, scanOpts ProcessingOptions, cleanProfile, scanProfile config.Profile) []string {
	var warnings []string

	// Check for profile inconsistency patterns common in pre-commit setups
	if cleanOpts.RespectAllowlist && !cleanOpts.IgnoreAllowlist && scanOpts.IgnoreAllowlist {
		warnings = append(warnings, "Pre-commit configuration issue: clean respects allowlist while scan ignores it. This may cause clean to report '0 modified' while scan still finds emojis.")
	}

	cleanShouldUse := ShouldUseAllowlist(cleanOpts, cleanProfile)
	scanShouldUse := ShouldUseAllowlist(scanOpts, scanProfile)

	if !cleanShouldUse && scanShouldUse {
		warnings = append(warnings, "Allowlist behavior inconsistency: clean ignores allowlist while scan respects it. Consider using the same profile and flags for both commands.")
	}

	// Log warnings
	for _, warning := range warnings {
		logging.Warn(ctx, "Configuration validation warning", "warning", warning)
	}

	return warnings
}
