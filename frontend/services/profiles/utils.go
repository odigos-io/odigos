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
