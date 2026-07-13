package rtml

import (
	"math"
	"runtime/metrics"
)

const (
	metricGOMEMLimit = "/gc/gomemlimit:bytes"
	metricHeapGoal   = "/gc/heap/goal:bytes"
	metricHeapLive   = "/gc/heap/live:bytes"
	metricHeapFree   = "/memory/classes/heap/free:bytes"
	metricHeapRel    = "/memory/classes/heap/released:bytes"
	metricMemTotal   = "/memory/classes/total:bytes"
	metricTotalAlloc = "/gc/heap/allocs:bytes"
	metricTotalFree  = "/gc/heap/frees:bytes"
)

// IsMemLimitReached reports whether the Go runtime is at risk of exceeding its
// configured memory limit. It intentionally uses stable runtime metrics instead
// of linkname access to runtime internals, so collector builds are not coupled
// to private Go symbols.
func IsMemLimitReached() bool {
	stats := GetMemLimitRelatedStats()
	if stats.MemoryLimit == 0 || stats.MemoryLimit == math.MaxInt64 {
		return false
	}

	if stats.MemoryLimit > stats.MappedReady {
		return false
	}

	if stats.HeapFree >= stats.MappedReady {
		return false
	}
	if stats.MemoryLimit > stats.MappedReady-stats.HeapFree {
		return false
	}

	return stats.HeapLive >= stats.HeapGoal
}

// MemLimitRelatedStats contains the runtime values used by IsMemLimitReached.
type MemLimitRelatedStats struct {
	MemoryLimit uint64
	HeapGoal    uint64
	HeapLive    uint64
	MappedReady uint64
	HeapFree    uint64
	TotalAlloc  uint64
	TotalFree   uint64
}

// GetMemLimitRelatedStats returns a point-in-time view of Go memory-limit
// metrics. Runtime metrics are sampled together to keep the snapshot as
// consistent as the public API allows.
func GetMemLimitRelatedStats() MemLimitRelatedStats {
	samples := []metrics.Sample{
		{Name: metricGOMEMLimit},
		{Name: metricHeapGoal},
		{Name: metricHeapLive},
		{Name: metricHeapFree},
		{Name: metricHeapRel},
		{Name: metricMemTotal},
		{Name: metricTotalAlloc},
		{Name: metricTotalFree},
	}

	metrics.Read(samples)

	memTotal := uintMetric(samples[5])
	heapReleased := uintMetric(samples[4])
	mappedReady := uint64(0)
	if memTotal > heapReleased {
		mappedReady = memTotal - heapReleased
	}

	return MemLimitRelatedStats{
		MemoryLimit: uintMetric(samples[0]),
		HeapGoal:    uintMetric(samples[1]),
		HeapLive:    uintMetric(samples[2]),
		MappedReady: mappedReady,
		HeapFree:    uintMetric(samples[3]),
		TotalAlloc:  uintMetric(samples[6]),
		TotalFree:   uintMetric(samples[7]),
	}
}

func uintMetric(sample metrics.Sample) uint64 {
	if sample.Value.Kind() != metrics.KindUint64 {
		return 0
	}
	return sample.Value.Uint64()
}
