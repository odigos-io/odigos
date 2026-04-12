package profiles

import (
	"strconv"
	"time"

	"github.com/odigos-io/odigos/frontend/services/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

// SourceKeyFromResource extracts namespace, kind and name from OTLP resource attributes
// using the same resolution rules as collector traffic metrics (ResourceAttributesToSourceID).
func SourceKeyFromResource(attrs pcommon.Map) (string, bool) {
	sID, err := common.ResourceAttributesToSourceID(attrs)
	if err != nil || sID.Name == "" {
		return "", false
	}
	return sID.Namespace + "/" + string(sID.Kind) + "/" + sID.Name, true
}

func intFromEnvOrDefault(key string, def int) int {
	if v, err := strconv.Atoi(env.GetEnvVarOrDefault(key, strconv.Itoa(def))); err == nil && v > 0 {
		return v
	}
	return def
}

// StoreLimitsFromEnv returns store tuning from environment, falling back to defaults.go.
func StoreLimitsFromEnv() (maxSlots, ttlSeconds, slotMaxBytes int, cleanupInterval time.Duration) {
	maxSlots = intFromEnvOrDefault(envMaxSlots, DefaultProfilingMaxSlots)
	ttlSeconds = intFromEnvOrDefault(envSlotTTLSeconds, DefaultProfilingSlotTTLSeconds)
	slotMaxBytes = intFromEnvOrDefault(envSlotMaxBytes, DefaultProfilingSlotMaxBytes)
	cleanupInterval = time.Duration(intFromEnvOrDefault(envCleanupIntervalSeconds, DefaultProfilingCleanupIntervalSeconds)) * time.Second
	return
}

// allNamesArePlaceholders reports whether every frame name is synthetic (no resolved symbols).
func allNamesArePlaceholders(names []string) bool {
	for _, n := range names {
		if n == "" || n == "total" || n == "other" {
			continue
		}
		if isSyntheticFrameName(n) {
			continue
		}
		return false
	}
	return true
}

func isSyntheticFrameName(n string) bool {
	if len(n) > 6 && n[:6] == "frame_" {
		return true
	}
	if len(n) > 2 && n[:2] == "0x" {
		return true
	}
	return false
}

// ChunksForSourceKey returns a shallow snapshot of buffered OTLP profile chunks for the given
// source key. Each element is one protobuf-encoded ExportProfilesServiceRequest
func ChunksForSourceKey(store common.ProfileStoreRef, sourceKey string) [][]byte {
	if store == nil {
		return nil
	}
	return store.GetProfileData(sourceKey)
}
