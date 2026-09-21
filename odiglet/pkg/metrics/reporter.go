package metrics

// MapMemoryEntry is one accounted eBPF map allocation, however many probes
// or processes reference it. Plain types on purpose: the implementation
// lives in a module OSS odigos does not depend on.
type MapMemoryEntry struct {
	MapName string
	// MapType is the stringified cilium/ebpf type, e.g. "Hash".
	MapType   string
	Component string
	// ProbeID is empty when the map is shared more widely than one probe,
	// and PID is nil when it is not scoped to a single process. Both are
	// normal values for node-wide and probe-wide shared maps.
	ProbeID string
	PID     *int
	// Bytes is what the kernel commits for the map once full, fixed at load.
	Bytes int64
	// Refs is how many registrations hold this allocation — the sharing
	// structure, not a count of open file descriptors.
	Refs int
}

// MapMemoryReporter supplies map memory recorded when each map was loaded,
// rather than rediscovered from the kernel on every read. Builds without an
// implementation leave it nil and fall back to reading /proc, which is the
// only option for maps loaded outside that accounting anyway.
//
// MapMemorySnapshot is called from a metrics scrape callback: it must be
// safe for concurrent use and must not block.
type MapMemoryReporter interface {
	MapMemorySnapshot() []MapMemoryEntry
}
