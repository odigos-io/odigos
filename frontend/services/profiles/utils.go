package profiles

import (
	"strconv"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosconsts "github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/services/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"go.opentelemetry.io/collector/pdata/pcommon"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// NormalizeWorkloadKind maps API/UI strings to canonical WorkloadKind values for source keys.
// GraphQL and resource attributes may use mixed casing; unknown strings fall back to k8sconsts.WorkloadKind(kindStr).
func NormalizeWorkloadKind(kindStr string) k8sconsts.WorkloadKind {
	if k := workload.WorkloadKindFromString(kindStr); k != "" {
		return k
	}
	return k8sconsts.WorkloadKind(kindStr)
}

// SourceKeyFromSourceID returns a stable string key for the given SourceID.
// Format: "namespace/kind/name" so it matches keys derived from profile resource attributes.
func SourceKeyFromSourceID(id common.SourceID) string {
	return id.Namespace + "/" + string(id.Kind) + "/" + id.Name
}

// SourceKeyFromResource extracts namespace, kind and name from OTLP resource attributes
func SourceKeyFromResource(attrs pcommon.Map) (string, bool) {
	ns, ok := attrs.Get(string(semconv.K8SNamespaceNameKey))
	if !ok || ns.Str() == "" {
		return "", false
	}
	namespace := ns.Str()

	var kind k8sconsts.WorkloadKind
	var name string
	var found bool

	for _, pair := range workload.OTLPWorkloadNameAttrKindPairs {
		if n, ok := getStr(attrs, pair.Key); ok {
			kind = pair.Kind
			name = n
			found = true
			break
		}
	}
	if !found {
		odigosKind, kindFound := getStr(attrs, odigosconsts.OdigosWorkloadKindAttribute)
		odigosName, nameFound := getStr(attrs, odigosconsts.OdigosWorkloadNameAttribute)
		if kindFound && nameFound && odigosName != "" {
			// Odigos attributes are free-form strings; normalize so the key matches SourceKeyFromSourceID.
			kind = NormalizeWorkloadKind(odigosKind)
			name = odigosName
			found = true
		}
	}
	if !found || name == "" {
		return "", false
	}

	return namespace + "/" + string(kind) + "/" + name, true
}

func getStr(attrs pcommon.Map, key string) (string, bool) {
	v, ok := attrs.Get(key)
	if !ok {
		return "", false
	}
	return v.Str(), true
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
