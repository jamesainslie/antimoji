// Package cli provides tests for pre-commit integration scenarios.
package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/antimoji/antimoji/internal/config"
	"github.com/antimoji/antimoji/internal/core/allowlist"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPreCommitZeroToleranceIntegration tests the zero-tolerance pre-commit workflow
// that was reported as buggy in the GitHub issue.
func TestPreCommitZeroToleranceIntegration(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "antimoji-precommit-test-")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create test files with emojis
	testFiles := map[string]string{
		"test1.txt": "Hello world 😀 this has emojis 🎉",
		"test2.txt": "Another file with 🚀 emojis ✨",
		"test3.txt": "Clean this ✅ but keep allowed ones ❌",
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(tempDir, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err)
	}

	// Create zero-tolerance profile
	profile := config.Profile{
		EmojiAllowlist:      []string{},
		FileIgnoreList:      []string{},
		DirectoryIgnoreList: []string{".git", "vendor", "dist", "bin", "node_modules"},
		IncludePatterns:     []string{},
		ExcludePatterns:     []string{},
	}

	t.Run("ZeroToleranceWorkflow", func(t *testing.T) {
		// Step 1: Clean with zero profile (should remove all emojis)
		cleanOpts := &CleanOptions{
			Recursive:        true,
			InPlace:          true,
			RespectAllowlist: true,
			IgnoreAllowlist:  false, // Use default behavior
		}

		// Simulate clean command execution
		ctx := context.Background()
		allowlistOpts := allowlist.ProcessingOptions{
			IgnoreAllowlist:  cleanOpts.IgnoreAllowlist,
			RespectAllowlist: cleanOpts.RespectAllowlist,
			Operation:        "clean",
		}

		emojiAllowlist, err := allowlist.CreateAllowlistForProcessing(ctx, profile, allowlistOpts)
		require.NoError(t, err)
		assert.Nil(t, emojiAllowlist, "Zero profile should create no allowlist")

		// Verify that clean would process all files
		shouldUseAllowlist := allowlist.ShouldUseAllowlist(allowlistOpts, profile)
		assert.False(t, shouldUseAllowlist, "Zero profile should not use allowlist")

		// Step 2: Verify scan would find no emojis after clean
		scanOpts := &ScanOptions{
			Recursive:       true,
			Threshold:       0,
			IgnoreAllowlist: false, // Same as clean for consistency
		}

		scanAllowlistOpts := allowlist.ProcessingOptions{
			IgnoreAllowlist:  scanOpts.IgnoreAllowlist,
			RespectAllowlist: true, // scan default
			Operation:        "scan",
		}

		scanEmojiAllowlist, err := allowlist.CreateAllowlistForProcessing(ctx, profile, scanAllowlistOpts)
		require.NoError(t, err)
		assert.Nil(t, scanEmojiAllowlist, "Zero profile should create no allowlist for scan too")

		// Verify consistent allowlist behavior
		scanShouldUseAllowlist := allowlist.ShouldUseAllowlist(scanAllowlistOpts, profile)
		assert.False(t, scanShouldUseAllowlist, "Zero profile should not use allowlist in scan")
		assert.Equal(t, shouldUseAllowlist, scanShouldUseAllowlist, "Clean and scan should have consistent allowlist behavior")
	})
}

// TestPreCommitProfileInconsistencyDetection tests the detection of profile inconsistencies
// that cause the "0 modified but still finds emojis" bug.
func TestPreCommitProfileInconsistencyDetection(t *testing.T) {
	ctx := context.Background()

	// Create profiles that would cause inconsistency
	ciLintProfile := config.Profile{
		EmojiAllowlist: []string{"✅", "❌", "⚠️", "✨"},
	}

	zeroProfile := config.Profile{
		EmojiAllowlist: []string{},
	}

	t.Run("DetectInconsistentProfiles", func(t *testing.T) {
		// Clean uses ci-lint profile (has allowlist)
		cleanOpts := allowlist.ProcessingOptions{
			IgnoreAllowlist:  false,
			RespectAllowlist: true,
			Operation:        "clean",
		}

		// Scan uses zero profile with ignore-allowlist (problematic combination from bug report)
		scanOpts := allowlist.ProcessingOptions{
			IgnoreAllowlist:  true, // This causes the inconsistency
			RespectAllowlist: true,
			Operation:        "scan",
		}

		// Test clean with ci-lint profile
		cleanAllowlist, err := allowlist.CreateAllowlistForProcessing(ctx, ciLintProfile, cleanOpts)
		require.NoError(t, err)
		assert.NotNil(t, cleanAllowlist, "Clean should create allowlist with ci-lint profile")

		// Test scan with zero profile but ignore-allowlist
		scanAllowlist, err := allowlist.CreateAllowlistForProcessing(ctx, zeroProfile, scanOpts)
		require.NoError(t, err)
		assert.Nil(t, scanAllowlist, "Scan should ignore allowlist when flag is set")

		// This combination would cause the bug: clean respects allowlist, scan ignores it
		cleanUsesAllowlist := allowlist.ShouldUseAllowlist(cleanOpts, ciLintProfile)
		scanUsesAllowlist := allowlist.ShouldUseAllowlist(scanOpts, zeroProfile)

		assert.True(t, cleanUsesAllowlist, "Clean should use allowlist")
		assert.False(t, scanUsesAllowlist, "Scan should ignore allowlist")
		assert.NotEqual(t, cleanUsesAllowlist, scanUsesAllowlist, "This inconsistency causes the bug")

		// Test validation function
		warnings := allowlist.ValidateConsistentOptions(ctx, cleanOpts, scanOpts, ciLintProfile, zeroProfile)
		assert.NotEmpty(t, warnings, "Should detect inconsistent configuration")
		assert.Contains(t, warnings[0], "clean respects allowlist while scan ignores it", "Should identify the specific issue")
	})

	t.Run("ConsistentConfiguration", func(t *testing.T) {
		// Both use zero profile with consistent settings
		cleanOpts := allowlist.ProcessingOptions{
			IgnoreAllowlist:  false,
			RespectAllowlist: true,
			Operation:        "clean",
		}

		scanOpts := allowlist.ProcessingOptions{
			IgnoreAllowlist:  false,
			RespectAllowlist: true,
			Operation:        "scan",
		}

		// Test both with zero profile
		cleanAllowlist, err := allowlist.CreateAllowlistForProcessing(ctx, zeroProfile, cleanOpts)
		require.NoError(t, err)
		assert.Nil(t, cleanAllowlist, "Clean should create no allowlist with zero profile")

		scanAllowlist, err := allowlist.CreateAllowlistForProcessing(ctx, zeroProfile, scanOpts)
		require.NoError(t, err)
		assert.Nil(t, scanAllowlist, "Scan should create no allowlist with zero profile")

		// Verify consistent behavior
		cleanUsesAllowlist := allowlist.ShouldUseAllowlist(cleanOpts, zeroProfile)
		scanUsesAllowlist := allowlist.ShouldUseAllowlist(scanOpts, zeroProfile)

		assert.False(t, cleanUsesAllowlist, "Clean should not use allowlist with zero profile")
		assert.False(t, scanUsesAllowlist, "Scan should not use allowlist with zero profile")
		assert.Equal(t, cleanUsesAllowlist, scanUsesAllowlist, "Clean and scan should have consistent behavior")

		// Test validation function
		warnings := allowlist.ValidateConsistentOptions(ctx, cleanOpts, scanOpts, zeroProfile, zeroProfile)
		assert.Empty(t, warnings, "Should not detect issues with consistent configuration")
	})
}

// TestIgnoreAllowlistFlagConsistency tests that the --ignore-allowlist flag works consistently
// across both clean and scan commands.
func TestIgnoreAllowlistFlagConsistency(t *testing.T) {
	ctx := context.Background()

	// Create profile with allowlist
	profileWithAllowlist := config.Profile{
		EmojiAllowlist: []string{"✅", "❌", "⚠️"},
	}

	t.Run("IgnoreAllowlistInClean", func(t *testing.T) {
		// Test clean with ignore-allowlist flag
		cleanOpts := allowlist.ProcessingOptions{
			IgnoreAllowlist:  true, // This should work now (was the bug)
			RespectAllowlist: true,
			Operation:        "clean",
		}

		allowlistResult, err := allowlist.CreateAllowlistForProcessing(ctx, profileWithAllowlist, cleanOpts)
		require.NoError(t, err)
		assert.Nil(t, allowlistResult, "Clean should ignore allowlist when flag is set")

		shouldUse := allowlist.ShouldUseAllowlist(cleanOpts, profileWithAllowlist)
		assert.False(t, shouldUse, "Clean should not use allowlist when ignore flag is set")
	})

	t.Run("IgnoreAllowlistInScan", func(t *testing.T) {
		// Test scan with ignore-allowlist flag (this always worked)
		scanOpts := allowlist.ProcessingOptions{
			IgnoreAllowlist:  true,
			RespectAllowlist: true,
			Operation:        "scan",
		}

		allowlistResult, err := allowlist.CreateAllowlistForProcessing(ctx, profileWithAllowlist, scanOpts)
		require.NoError(t, err)
		assert.Nil(t, allowlistResult, "Scan should ignore allowlist when flag is set")

		shouldUse := allowlist.ShouldUseAllowlist(scanOpts, profileWithAllowlist)
		assert.False(t, shouldUse, "Scan should not use allowlist when ignore flag is set")
	})

	t.Run("ConsistentIgnoreBehavior", func(t *testing.T) {
		// Both commands with ignore-allowlist should behave identically
		cleanOpts := allowlist.ProcessingOptions{
			IgnoreAllowlist:  true,
			RespectAllowlist: true,
			Operation:        "clean",
		}

		scanOpts := allowlist.ProcessingOptions{
			IgnoreAllowlist:  true,
			RespectAllowlist: true,
			Operation:        "scan",
		}

		cleanAllowlist, err := allowlist.CreateAllowlistForProcessing(ctx, profileWithAllowlist, cleanOpts)
		require.NoError(t, err)

		scanAllowlist, err := allowlist.CreateAllowlistForProcessing(ctx, profileWithAllowlist, scanOpts)
		require.NoError(t, err)

		// Both should ignore the allowlist
		assert.Nil(t, cleanAllowlist, "Clean should ignore allowlist")
		assert.Nil(t, scanAllowlist, "Scan should ignore allowlist")

		// Both should have consistent behavior
		cleanShouldUse := allowlist.ShouldUseAllowlist(cleanOpts, profileWithAllowlist)
		scanShouldUse := allowlist.ShouldUseAllowlist(scanOpts, profileWithAllowlist)

		assert.False(t, cleanShouldUse, "Clean should not use allowlist")
		assert.False(t, scanShouldUse, "Scan should not use allowlist")
		assert.Equal(t, cleanShouldUse, scanShouldUse, "Both commands should behave consistently")
	})
}

// TestRespectAllowlistBackwardCompatibility tests that the existing --respect-allowlist flag
// still works but is properly overridden by --ignore-allowlist for consistency.
func TestRespectAllowlistBackwardCompatibility(t *testing.T) {
	ctx := context.Background()

	profileWithAllowlist := config.Profile{
		EmojiAllowlist: []string{"✅", "❌"},
	}

	t.Run("RespectAllowlistTrue", func(t *testing.T) {
		opts := allowlist.ProcessingOptions{
			IgnoreAllowlist:  false,
			RespectAllowlist: true,
			Operation:        "clean",
		}

		allowlistResult, err := allowlist.CreateAllowlistForProcessing(ctx, profileWithAllowlist, opts)
		require.NoError(t, err)
		assert.NotNil(t, allowlistResult, "Should create allowlist when respect-allowlist is true")

		shouldUse := allowlist.ShouldUseAllowlist(opts, profileWithAllowlist)
		assert.True(t, shouldUse, "Should use allowlist when respect-allowlist is true")
	})

	t.Run("RespectAllowlistFalse", func(t *testing.T) {
		opts := allowlist.ProcessingOptions{
			IgnoreAllowlist:  false,
			RespectAllowlist: false,
			Operation:        "clean",
		}

		allowlistResult, err := allowlist.CreateAllowlistForProcessing(ctx, profileWithAllowlist, opts)
		require.NoError(t, err)
		assert.Nil(t, allowlistResult, "Should not create allowlist when respect-allowlist is false")

		shouldUse := allowlist.ShouldUseAllowlist(opts, profileWithAllowlist)
		assert.False(t, shouldUse, "Should not use allowlist when respect-allowlist is false")
	})

	t.Run("IgnoreAllowlistOverridesRespectAllowlist", func(t *testing.T) {
		// ignore-allowlist=true should override respect-allowlist=true
		opts := allowlist.ProcessingOptions{
			IgnoreAllowlist:  true, // This takes precedence
			RespectAllowlist: true, // This should be overridden
			Operation:        "clean",
		}

		allowlistResult, err := allowlist.CreateAllowlistForProcessing(ctx, profileWithAllowlist, opts)
		require.NoError(t, err)
		assert.Nil(t, allowlistResult, "ignore-allowlist should override respect-allowlist")

		shouldUse := allowlist.ShouldUseAllowlist(opts, profileWithAllowlist)
		assert.False(t, shouldUse, "ignore-allowlist should take precedence")
	})
}
