package allowlist

import (
	"context"
	"testing"

	"github.com/antimoji/antimoji/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestCreateAllowlistForProcessing(t *testing.T) {
	ctx := context.Background()

	t.Run("with allowlist in profile", func(t *testing.T) {
		profile := config.Profile{
			EmojiAllowlist: []string{"✅", "❌", "⚠️"},
		}

		opts := ProcessingOptions{
			IgnoreAllowlist:  false,
			RespectAllowlist: true,
			Operation:        "scan",
		}

		allowlist, err := CreateAllowlistForProcessing(ctx, profile, opts)

		assert.NoError(t, err)
		assert.NotNil(t, allowlist)
		assert.False(t, allowlist.IsEmpty())
		assert.True(t, allowlist.IsAllowed("✅"))
		assert.True(t, allowlist.IsAllowed("❌"))
		assert.True(t, allowlist.IsAllowed("⚠️"))
	})

	t.Run("ignore allowlist option", func(t *testing.T) {
		profile := config.Profile{
			EmojiAllowlist: []string{"✅", "❌"},
		}

		opts := ProcessingOptions{
			IgnoreAllowlist:  true,
			RespectAllowlist: true,
			Operation:        "scan",
		}

		allowlist, err := CreateAllowlistForProcessing(ctx, profile, opts)

		assert.NoError(t, err)
		assert.Nil(t, allowlist) // Should be nil when ignoring allowlist
	})

	t.Run("empty allowlist", func(t *testing.T) {
		profile := config.Profile{
			EmojiAllowlist: []string{},
		}

		opts := ProcessingOptions{
			IgnoreAllowlist:  false,
			RespectAllowlist: true,
			Operation:        "clean",
		}

		allowlist, err := CreateAllowlistForProcessing(ctx, profile, opts)

		assert.NoError(t, err)
		assert.Nil(t, allowlist) // Should be nil for empty allowlist
	})

	t.Run("not respecting allowlist", func(t *testing.T) {
		profile := config.Profile{
			EmojiAllowlist: []string{"✅", "❌"},
		}

		opts := ProcessingOptions{
			IgnoreAllowlist:  false,
			RespectAllowlist: false,
			Operation:        "scan",
		}

		allowlist, err := CreateAllowlistForProcessing(ctx, profile, opts)

		assert.NoError(t, err)
		assert.Nil(t, allowlist) // Should be nil when not respecting allowlist
	})
}

func TestShouldUseAllowlist(t *testing.T) {
	tests := []struct {
		name     string
		opts     ProcessingOptions
		profile  config.Profile
		expected bool
	}{
		{
			name: "ignore allowlist takes precedence",
			opts: ProcessingOptions{
				IgnoreAllowlist:  true,
				RespectAllowlist: true,
			},
			profile: config.Profile{
				EmojiAllowlist: []string{"✅"},
			},
			expected: false,
		},
		{
			name: "respect allowlist with non-empty list",
			opts: ProcessingOptions{
				IgnoreAllowlist:  false,
				RespectAllowlist: true,
			},
			profile: config.Profile{
				EmojiAllowlist: []string{"✅"},
			},
			expected: true,
		},
		{
			name: "respect allowlist with empty list",
			opts: ProcessingOptions{
				IgnoreAllowlist:  false,
				RespectAllowlist: true,
			},
			profile: config.Profile{
				EmojiAllowlist: []string{},
			},
			expected: false,
		},
		{
			name: "not respecting allowlist",
			opts: ProcessingOptions{
				IgnoreAllowlist:  false,
				RespectAllowlist: false,
			},
			profile: config.Profile{
				EmojiAllowlist: []string{"✅"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldUseAllowlist(tt.opts, tt.profile)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateConsistentOptions(t *testing.T) {
	ctx := context.Background()

	cleanOpts := ProcessingOptions{
		IgnoreAllowlist:  false,
		RespectAllowlist: true,
		Operation:        "clean",
	}

	scanOpts := ProcessingOptions{
		IgnoreAllowlist:  true,
		RespectAllowlist: false,
		Operation:        "scan",
	}

	cleanProfile := config.Profile{
		EmojiAllowlist: []string{"✅"},
	}

	scanProfile := config.Profile{
		EmojiAllowlist: []string{},
	}

	warnings := ValidateConsistentOptions(ctx, cleanOpts, scanOpts, cleanProfile, scanProfile)

	// Should have warnings about inconsistent configuration
	assert.NotEmpty(t, warnings)
}
