package ebpf

const (
	NumOfPages = 2048

	// JVM metrics eBPF map sizing constants.
	// Uses a hash-of-maps architecture: one outer HashOfMaps keyed by UUID containing
	// per-process inner maps, plus a separate attributes map for resource attributes.
	ProcessKeySize      = 64   // Size of UUID key
	InnerMapIDSize      = 4    // Size of inner map ID (should be 4 bytes hard coded)
	MaxProcessesCount   = 512  // Max number of processes that can have metrics
	AttributesValueSize = 1024 // Size of packed resource attributes value buffer

	// Logs eBPF map sizing constants.
	LogsMaxEntries = 256 * 1024 // 256KB - must match BPF log_events map size

	// Inner Map configuration
	MetricKeySize    = 4   // uint32 metric_key
	MetricValueSize  = 40  // struct metric_value (40 bytes - size of largest union member: histogram_value)
	MaxMetricsPerMap = 256 // MAX_METRICS per process
)
