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

	// Pyroscope JSON contract for single CPU-style flamebearer payloads (historical Grafana/Pyroscope shape).
	pyroscopeFlamebearerJSONVersion = 1
	pyroscopeMetadataFormatSingle   = "single"
	pyroscopeMetadataUnitsSamples   = "samples"
	pyroscopeMetadataProfileNameCPU = "cpu"
	// pyroscopeMetadataSampleRate is a display hint for the UI, not OTLP sample timing.
	pyroscopeMetadataSampleRate = 100
	// pyroscopeTimelineDurationDeltaSec is a minimal timeline width heuristic when merging chunks.
	pyroscopeTimelineDurationDeltaSec = 15
)
