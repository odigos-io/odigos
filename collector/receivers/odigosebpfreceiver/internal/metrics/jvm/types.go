package jvm

import "encoding/binary"

// ============================================================================
// Metric Key Encoding (must match bpf/jvm_metrics.h)
// ============================================================================
// Bits 0-7:   Metric Type (256 possible metrics)
// Bits 8-15:  Attribute 1 (metric-specific primary attribute)
// Bits 16-23: Attribute 2 (metric-specific secondary attribute)
// Bits 24-31: Reserved (future expansion)
// ============================================================================

const (
	metricTypeShift = 0
	metricTypeMask  = 0x000000FF
	attr1Shift      = 8
	attr1Mask       = 0x0000FF00
	attr2Shift      = 16
	attr2Mask       = 0x00FF0000
)

// MetricKey is the encoded map key containing metric type + attributes
type MetricKey uint32

// MetricType extracts the metric type from the key (bits 0-7)
func (k MetricKey) MetricType() MetricType {
	return MetricType((uint32(k) & metricTypeMask) >> metricTypeShift)
}

// Attr1 extracts attribute 1 from the key (bits 8-15)
func (k MetricKey) Attr1() uint8 {
	return uint8((uint32(k) & attr1Mask) >> attr1Shift)
}

// Attr2 extracts attribute 2 from the key (bits 16-23)
func (k MetricKey) Attr2() uint8 {
	return uint8((uint32(k) & attr2Mask) >> attr2Shift)
}

// ============================================================================
// Metric Types (must match enum jvm_metric_type in bpf/jvm_metrics.h)
// ============================================================================

type MetricType uint8

const (
	// Classes (no attributes)
	MetricClassLoaded   MetricType = 1
	MetricClassUnloaded MetricType = 2
	MetricClassCount    MetricType = 3

	// Memory (attr1=type, attr2=pool)
	MetricMemoryUsed        MetricType = 10
	MetricMemoryCommitted   MetricType = 11
	MetricMemoryLimit       MetricType = 12
	MetricMemoryUsedAfterGC MetricType = 13
	MetricMemoryInit        MetricType = 14

	// GC (attr1=action, attr2=gc_name)
	MetricGCDuration MetricType = 20

	// Threads (attr1=daemon, attr2=state)
	MetricThreadCount MetricType = 30

	// CPU (no attributes)
	MetricCPUTime              MetricType = 40
	MetricCPUCount             MetricType = 41
	MetricCPURecentUtilization MetricType = 42
)

// IsGauge returns true if this metric type represents a gauge (current state)
// rather than a counter or histogram (delta/cumulative values)
func (t MetricType) IsGauge() bool {
	switch t {
	case MetricMemoryUsed, MetricMemoryCommitted, MetricMemoryLimit,
		MetricMemoryUsedAfterGC, MetricMemoryInit,
		MetricThreadCount, MetricCPUCount, MetricCPURecentUtilization:
		return true
	default:
		return false
	}
}

// ============================================================================
// GC Action (attr1 for GC_DURATION) - jvm.gc.action
// ============================================================================

type GCAction uint8

const (
	GCActionUnknown GCAction = 0
	GCActionMinor   GCAction = 1 // "end of minor GC"
	GCActionMajor   GCAction = 2 // "end of major GC"
)

// String returns the OTel semantic convention value for jvm.gc.action
func (a GCAction) String() string {
	switch a {
	case GCActionMinor:
		return "end of minor GC"
	case GCActionMajor:
		return "end of major GC"
	default:
		return "unknown"
	}
}

// ============================================================================
// GC Name (attr2 for GC_DURATION) - jvm.gc.name
// ============================================================================

type GCName uint8

const (
	GCNameUnknown GCName = 0
	// G1 Garbage Collector
	GCNameG1Young      GCName = 1
	GCNameG1Old        GCName = 2
	GCNameG1Concurrent GCName = 3
	// Parallel Garbage Collector
	GCNamePSScavenge  GCName = 10
	GCNamePSMarkSweep GCName = 11
	// Serial Garbage Collector
	GCNameCopy             GCName = 20
	GCNameMarkSweepCompact GCName = 21
	// ZGC
	GCNameZGCCycles      GCName = 40
	GCNameZGCPauses      GCName = 41
	GCNameZGCMajorCycles GCName = 42
	GCNameZGCMajorPauses GCName = 43
	GCNameZGCMinorCycles GCName = 44
	GCNameZGCMinorPauses GCName = 45
	// Shenandoah
	GCNameShenandoahCycles GCName = 50
	GCNameShenandoahPauses GCName = 51
)

// String returns the OTel semantic convention value for jvm.gc.name
func (n GCName) String() string {
	switch n {
	case GCNameG1Young:
		return "G1 Young Generation"
	case GCNameG1Old:
		return "G1 Old Generation"
	case GCNameG1Concurrent:
		return "G1 Concurrent GC"
	case GCNamePSScavenge:
		return "PS Scavenge"
	case GCNamePSMarkSweep:
		return "PS MarkSweep"
	case GCNameCopy:
		return "Copy"
	case GCNameMarkSweepCompact:
		return "MarkSweepCompact"
	case GCNameZGCCycles:
		return "ZGC Cycles"
	case GCNameZGCPauses:
		return "ZGC Pauses"
	case GCNameZGCMajorCycles:
		return "ZGC Major Cycles"
	case GCNameZGCMajorPauses:
		return "ZGC Major Pauses"
	case GCNameZGCMinorCycles:
		return "ZGC Minor Cycles"
	case GCNameZGCMinorPauses:
		return "ZGC Minor Pauses"
	case GCNameShenandoahCycles:
		return "Shenandoah Cycles"
	case GCNameShenandoahPauses:
		return "Shenandoah Pauses"
	default:
		return "unknown"
	}
}

// ============================================================================
// Memory Type (attr1 for MEMORY_*) - jvm.memory.type
// ============================================================================

type MemoryType uint8

const (
	MemoryTypeUnknown MemoryType = 0
	MemoryTypeHeap    MemoryType = 1 // "heap"
	MemoryTypeNonHeap MemoryType = 2 // "non_heap"
)

// String returns the OTel semantic convention value for jvm.memory.type
func (t MemoryType) String() string {
	switch t {
	case MemoryTypeHeap:
		return "heap"
	case MemoryTypeNonHeap:
		return "non_heap"
	default:
		return "unknown"
	}
}

// ============================================================================
// Memory Pool Name (attr2 for MEMORY_*) - jvm.memory.pool.name
// ============================================================================

type MemoryPoolName uint8

const (
	PoolNameUnknown MemoryPoolName = 0
	// G1 Heap Pools
	PoolNameG1Eden     MemoryPoolName = 1
	PoolNameG1Survivor MemoryPoolName = 2
	PoolNameG1Old      MemoryPoolName = 3
	// Parallel Heap Pools
	PoolNamePSEden     MemoryPoolName = 10
	PoolNamePSSurvivor MemoryPoolName = 11
	PoolNamePSOld      MemoryPoolName = 12
	// Serial Heap Pools
	PoolNameEden     MemoryPoolName = 20
	PoolNameSurvivor MemoryPoolName = 21
	PoolNameTenured  MemoryPoolName = 22
	// Non-Heap Pools
	PoolNameMetaspace       MemoryPoolName = 100
	PoolNameCodeCache       MemoryPoolName = 101
	PoolNameCompressedClass MemoryPoolName = 102
	// ZGC Heap Pools
	PoolNameZGCOld   MemoryPoolName = 110
	PoolNameZGCYoung MemoryPoolName = 111
	// Shenandoah
	PoolNameShenandoah MemoryPoolName = 120
)

// String returns the OTel semantic convention value for jvm.memory.pool.name
func (p MemoryPoolName) String() string {
	switch p {
	case PoolNameG1Eden:
		return "G1 Eden Space"
	case PoolNameG1Survivor:
		return "G1 Survivor Space"
	case PoolNameG1Old:
		return "G1 Old Gen"
	case PoolNamePSEden:
		return "PS Eden Space"
	case PoolNamePSSurvivor:
		return "PS Survivor Space"
	case PoolNamePSOld:
		return "PS Old Gen"
	case PoolNameEden:
		return "Eden Space"
	case PoolNameSurvivor:
		return "Survivor Space"
	case PoolNameTenured:
		return "Tenured Gen"
	case PoolNameMetaspace:
		return "Metaspace"
	case PoolNameCodeCache:
		return "CodeCache"
	case PoolNameCompressedClass:
		return "Compressed Class Space"
	case PoolNameZGCOld:
		return "ZGC Old Generation"
	case PoolNameZGCYoung:
		return "ZGC Young Generation"
	case PoolNameShenandoah:
		return "Shenandoah"
	default:
		return "unknown"
	}
}

// ============================================================================
// Thread Daemon (attr1 for THREAD_COUNT) - jvm.thread.daemon
// ============================================================================

type ThreadDaemon uint8

const (
	ThreadDaemonUnknown ThreadDaemon = 0
	ThreadDaemonFalse   ThreadDaemon = 1
	ThreadDaemonTrue    ThreadDaemon = 2
)

// String returns the OTel semantic convention value for jvm.thread.daemon
func (d ThreadDaemon) String() string {
	switch d {
	case ThreadDaemonFalse:
		return "false"
	case ThreadDaemonTrue:
		return "true"
	default:
		return "unknown"
	}
}

// ============================================================================
// Thread State (attr2 for THREAD_COUNT) - jvm.thread.state
// ============================================================================

type ThreadState uint8

const (
	ThreadStateUnknown      ThreadState = 0
	ThreadStateNew          ThreadState = 1
	ThreadStateRunnable     ThreadState = 2
	ThreadStateBlocked      ThreadState = 3
	ThreadStateWaiting      ThreadState = 4
	ThreadStateTimedWaiting ThreadState = 5
	ThreadStateTerminated   ThreadState = 6
)

// String returns the OTel semantic convention value for jvm.thread.state
func (s ThreadState) String() string {
	switch s {
	case ThreadStateNew:
		return "new"
	case ThreadStateRunnable:
		return "runnable"
	case ThreadStateBlocked:
		return "blocked"
	case ThreadStateWaiting:
		return "waiting"
	case ThreadStateTimedWaiting:
		return "timed_waiting"
	case ThreadStateTerminated:
		return "terminated"
	default:
		return "unknown"
	}
}

// ============================================================================
// Value Types (must match structs in bpf/jvm_metrics.h)
// ============================================================================

// CounterValue matches struct counter_value in BPF
type CounterValue struct {
	Count uint64
}

// GaugeValue matches struct gauge_value in BPF
type GaugeValue struct {
	Value uint64
}

// HistogramValue matches struct histogram_value in BPF
type HistogramValue struct {
	Bucket1ms   uint32
	Bucket10ms  uint32
	Bucket100ms uint32
	Bucket1s    uint32
	BucketInf   uint32
	SumNs       uint64
	TotalCount  uint32
	_           uint32 // padding
}

// MetricValue is a raw byte representation of union metric_value
// Size must match BPF map value size: 40 bytes
// Layout: histogram_value = 5×u32(20) + pad(4) + u64(8) + 2×u32(8) = 40
type MetricValue [40]byte

func (v *MetricValue) AsCounter() CounterValue {
	return CounterValue{
		Count: binary.LittleEndian.Uint64(v[:8]),
	}
}

func (v *MetricValue) AsGauge() GaugeValue {
	return GaugeValue{
		Value: binary.LittleEndian.Uint64(v[:8]),
	}
}

func (v *MetricValue) AsHistogram() HistogramValue {
	return HistogramValue{
		Bucket1ms:   binary.LittleEndian.Uint32(v[0:4]),
		Bucket10ms:  binary.LittleEndian.Uint32(v[4:8]),
		Bucket100ms: binary.LittleEndian.Uint32(v[8:12]),
		Bucket1s:    binary.LittleEndian.Uint32(v[12:16]),
		BucketInf:   binary.LittleEndian.Uint32(v[16:20]),
		SumNs:       binary.LittleEndian.Uint64(v[24:32]),
		TotalCount:  binary.LittleEndian.Uint32(v[32:36]),
	}
}

// HistogramBuckets defines the bucket boundaries in seconds for Prometheus
var HistogramBuckets = []float64{0.001, 0.01, 0.1, 1.0} // 1ms, 10ms, 100ms, 1s
