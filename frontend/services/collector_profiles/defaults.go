package collectorprofiles

// Canonical defaults for the UI in-memory profiling store.
// Keep in sync with helm/odigos values.yaml profiling.ui and odigos-configuration-cm.yaml template defaults.
// Store ticker / env-only knobs that are not in Helm also belong here so config.go and store.go stay aligned.
const (
	DefaultProfilingMaxSlots       = 24
	DefaultProfilingSlotMaxBytes   = 8 * 1024 * 1024 // 8 MiB
	DefaultProfilingSlotTTLSeconds = 120             // seconds
	// DefaultProfilingCleanupIntervalSeconds is the ProfileStore TTL sweep ticker period (pod-local only).
	DefaultProfilingCleanupIntervalSeconds = 15
)

// MaxProfilingBufferBytes is the worst-case sum of per-slot OTLP buffers (excluding Go overhead).
// Chosen so total profile cache stays under ~200 MiB with headroom for runtime, GraphQL, and maps.
const MaxProfilingBufferBytes = DefaultProfilingMaxSlots * DefaultProfilingSlotMaxBytes // 192 MiB
