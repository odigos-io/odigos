package collectorprofiles

import (
	"os"
	"strconv"
	"time"
)

const (
	envSlotTTLSeconds         = "PROFILES_SLOT_TTL_SECONDS"
	envMaxSlots               = "PROFILES_MAX_SLOTS" // max services with profiling enabled at once (default 10)
	envSlotMaxBytes           = "PROFILES_SLOT_MAX_BYTES"
	envCleanupIntervalSeconds = "PROFILES_CLEANUP_INTERVAL_SECONDS"
)

// StoreConfigFromEnv reads profiling store settings from environment variables.
// Unset or invalid values use package defaults (defaultMaxSlots, defaultSlotTTLSeconds, etc.).
func StoreConfigFromEnv() (maxSlots, ttlSeconds, slotMaxBytes int, cleanupInterval time.Duration) {
	maxSlots = intFromEnv(envMaxSlots, defaultMaxSlots)
	ttlSeconds = intFromEnv(envSlotTTLSeconds, defaultSlotTTLSeconds)
	slotMaxBytes = intFromEnv(envSlotMaxBytes, defaultSlotMaxBytes)
	sec := intFromEnv(envCleanupIntervalSeconds, int(defaultCleanupInt/time.Second))
	cleanupInterval = time.Duration(sec) * time.Second
	return maxSlots, ttlSeconds, slotMaxBytes, cleanupInterval
}

func intFromEnv(key string, defaultVal int) int {
	s := os.Getenv(key)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 0 {
		return defaultVal
	}
	return v
}
