package profiles

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// These names and defaults are documented as user-facing overrides for the UI deployment
// (docs/snippets/enterprise/instrumentations/profiling), so they are pinned to literals rather
// than to the constants under test.
func TestTheDocumentedProfileStoreOverridesKeepTheirNamesAndDefaults(t *testing.T) {
	assert.Equal(t, "PROFILES_MAX_SLOTS", envMaxSlots)
	assert.Equal(t, "PROFILES_SLOT_TTL_SECONDS", envSlotTTLSeconds)
	assert.Equal(t, "PROFILES_SLOT_MAX_BYTES", envSlotMaxBytes)
	assert.Equal(t, "PROFILES_CLEANUP_INTERVAL_SECONDS", envCleanupIntervalSeconds)

	assert.Equal(t, 24, DefaultProfilingMaxSlots)
	assert.Equal(t, 120, DefaultProfilingSlotTTLSeconds)
	assert.Equal(t, 8*1024*1024, DefaultProfilingSlotMaxBytes)
	assert.Equal(t, 15, DefaultProfilingCleanupIntervalSeconds)
}

func TestStoreLimitsFromEnvFallsBackToTheDefaults(t *testing.T) {
	maxSlots, ttlSeconds, slotMaxBytes, cleanupInterval := StoreLimitsFromEnv()

	assert.Equal(t, DefaultProfilingMaxSlots, maxSlots)
	assert.Equal(t, DefaultProfilingSlotTTLSeconds, ttlSeconds)
	assert.Equal(t, DefaultProfilingSlotMaxBytes, slotMaxBytes)
	assert.Equal(t, time.Duration(DefaultProfilingCleanupIntervalSeconds)*time.Second, cleanupInterval)
}

func TestStoreLimitsFromEnvReadsEachOverrideIndependently(t *testing.T) {
	t.Setenv(envMaxSlots, "7")
	t.Setenv(envSlotTTLSeconds, "30")
	t.Setenv(envSlotMaxBytes, "1024")
	t.Setenv(envCleanupIntervalSeconds, "5")

	maxSlots, ttlSeconds, slotMaxBytes, cleanupInterval := StoreLimitsFromEnv()

	assert.Equal(t, 7, maxSlots)
	assert.Equal(t, 30, ttlSeconds)
	assert.Equal(t, 1024, slotMaxBytes)
	assert.Equal(t, 5*time.Second, cleanupInterval)
}

// Each override must map to its own limit; a crossed wire would silently apply the wrong bound
// (for example capping slots at the byte budget).
func TestStoreLimitsFromEnvMapsEachVariableToItsOwnLimit(t *testing.T) {
	sentinels := map[string]struct {
		value int
		read  func(maxSlots, ttlSeconds, slotMaxBytes int, cleanup time.Duration) int
	}{
		envMaxSlots: {value: 11, read: func(maxSlots, _, _ int, _ time.Duration) int { return maxSlots }},
		envSlotTTLSeconds: {
			value: 12,
			read:  func(_, ttlSeconds, _ int, _ time.Duration) int { return ttlSeconds },
		},
		envSlotMaxBytes: {
			value: 13,
			read:  func(_, _, slotMaxBytes int, _ time.Duration) int { return slotMaxBytes },
		},
		envCleanupIntervalSeconds: {
			value: 14,
			read:  func(_, _, _ int, cleanup time.Duration) int { return int(cleanup / time.Second) },
		},
	}

	for envVar, expected := range sentinels {
		t.Run(envVar, func(t *testing.T) {
			t.Setenv(envVar, strconv.Itoa(expected.value))

			maxSlots, ttlSeconds, slotMaxBytes, cleanup := StoreLimitsFromEnv()

			assert.Equal(t, expected.value, expected.read(maxSlots, ttlSeconds, slotMaxBytes, cleanup))

			// Exactly one limit moved off its default.
			defaults := 0
			if maxSlots == DefaultProfilingMaxSlots {
				defaults++
			}
			if ttlSeconds == DefaultProfilingSlotTTLSeconds {
				defaults++
			}
			if slotMaxBytes == DefaultProfilingSlotMaxBytes {
				defaults++
			}
			if cleanup == time.Duration(DefaultProfilingCleanupIntervalSeconds)*time.Second {
				defaults++
			}
			assert.Equal(t, 3, defaults)
		})
	}
}

// A misconfigured value must not shrink the buffer to nothing or make the sweep ticker panic.
func TestIntFromEnvOrDefaultRejectsUnusableValues(t *testing.T) {
	for _, raw := range []string{"", "0", "-1", "-1000", "abc", "12abc", "1.5", " 7", "9999999999999999999999"} {
		t.Run("value_"+raw, func(t *testing.T) {
			t.Setenv(envMaxSlots, raw)
			assert.Equal(t, DefaultProfilingMaxSlots, intFromEnvOrDefault(envMaxSlots, DefaultProfilingMaxSlots))
		})
	}
}

func TestIntFromEnvOrDefaultAcceptsAPositiveValue(t *testing.T) {
	t.Setenv(envMaxSlots, "42")
	assert.Equal(t, 42, intFromEnvOrDefault(envMaxSlots, DefaultProfilingMaxSlots))
}

func TestIntFromEnvOrDefaultUsesTheDefaultWhenUnset(t *testing.T) {
	assert.Equal(t, 99, intFromEnvOrDefault("PROFILES_TEST_UNSET_VARIABLE", 99))
}
