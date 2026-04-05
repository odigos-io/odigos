package collectorprofiles

import (
	"os"
	"strconv"
	"time"
)

// Optional env overrides for the in-memory profile store (local dev / ops). In-cluster defaults usually
// come from odigos-configuration (profiling.ui) via ResolveProfilingFromEffectiveConfig, not from the UI pod env.
const (
	envSlotTTLSeconds         = "PROFILES_SLOT_TTL_SECONDS"
	envMaxSlots               = "PROFILES_MAX_SLOTS"
	envSlotMaxBytes           = "PROFILES_SLOT_MAX_BYTES"
	envCleanupIntervalSeconds = "PROFILES_CLEANUP_INTERVAL_SECONDS"
)

// StoreConfigFromEnv reads profiling store limits from the environment (defaults on unset/invalid).
func StoreConfigFromEnv() (maxSlots, ttlSeconds, slotMaxBytes int, cleanupInterval time.Duration) {
	maxSlots = DefaultProfilingMaxSlots
	if s := os.Getenv(envMaxSlots); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v >= 0 {
			maxSlots = v
		}
	}
	ttlSeconds = DefaultProfilingSlotTTLSeconds
	if s := os.Getenv(envSlotTTLSeconds); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v >= 0 {
			ttlSeconds = v
		}
	}
	slotMaxBytes = DefaultProfilingSlotMaxBytes
	if s := os.Getenv(envSlotMaxBytes); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v >= 0 {
			slotMaxBytes = v
		}
	}
	sec := DefaultProfilingCleanupIntervalSeconds
	if s := os.Getenv(envCleanupIntervalSeconds); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v >= 0 {
			sec = v
		}
	}
	cleanupInterval = time.Duration(sec) * time.Second
	return maxSlots, ttlSeconds, slotMaxBytes, cleanupInterval
}
