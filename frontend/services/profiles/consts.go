package profiles

const (
	// Optional env overrides for the UI in-memory profile store.
	envSlotTTLSeconds         = "PROFILES_SLOT_TTL_SECONDS"
	envMaxSlots               = "PROFILES_MAX_SLOTS"
	envSlotMaxBytes           = "PROFILES_SLOT_MAX_BYTES"
	envCleanupIntervalSeconds = "PROFILES_CLEANUP_INTERVAL_SECONDS"

	// Default settings for in-memory profile store.
	DefaultProfilingMaxSlots               = 24
	DefaultProfilingSlotMaxBytes           = 8 * 1024 * 1024 // 8 MiB
	DefaultProfilingSlotTTLSeconds         = 120             // seconds
	DefaultProfilingCleanupIntervalSeconds = 15              // ProfileStore TTL sweep ticker period (pod-local only)

	// Flame-bearer for single CPU-style payloads (UI wire format).
	flamebearerJSONVersion = 1
	metadataFormatSingle   = "single"
	metadataUnitsSamples   = "samples"
	metadataProfileNameCPU = "cpu"
	// metadataSampleRate is a display hint for the UI, not OTLP sample timing.
	metadataSampleRate = 100
	// timelineDurationDeltaSec is a minimal timeline width heuristic when merging chunks.
	timelineDurationDeltaSec = 15
)
