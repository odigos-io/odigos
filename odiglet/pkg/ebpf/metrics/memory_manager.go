package metrics

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

const (
	// Maximum memory allocated for metrics collection (configurable)
	DefaultMaxMemoryBytes = 10 * 1024 * 1024 // 10MB default
	
	// Pre-allocated object pools to avoid frequent allocations
	MaxTrackedMaps     = 500  // Pre-allocate for 500 maps
	MaxTrackedProgs    = 200  // Pre-allocate for 200 programs  
	MaxTrackedLinks    = 100  // Pre-allocate for 100 links
	
	// Batch processing limits to reduce syscall overhead
	MaxBatchSize = 50 // Process up to 50 objects per batch
)

// MemoryPool manages pre-allocated objects to avoid runtime allocations
type MemoryPool struct {
	// Pre-allocated object pools
	mapInfoPool   []EBPFMapInfo
	progInfoPool  []EBPFProgInfo
	linkInfoPool  []EBPFLinkInfo
	
	// Pool usage tracking
	mapPoolUsed   int32
	progPoolUsed  int32
	linkPoolUsed  int32
	
	// Current memory usage
	currentMemoryUsage int64
	maxMemoryLimit     int64
	
	// Memory usage tracking
	mu sync.RWMutex
}

// NewMemoryPool creates a memory pool with strict limits
func NewMemoryPool(maxMemoryBytes int64) *MemoryPool {
	if maxMemoryBytes <= 0 {
		maxMemoryBytes = DefaultMaxMemoryBytes
	}
	
	pool := &MemoryPool{
		mapInfoPool:        make([]EBPFMapInfo, MaxTrackedMaps),
		progInfoPool:       make([]EBPFProgInfo, MaxTrackedProgs),
		linkInfoPool:       make([]EBPFLinkInfo, MaxTrackedLinks),
		maxMemoryLimit:     maxMemoryBytes,
		currentMemoryUsage: 0,
	}
	
	// Calculate pre-allocated memory usage
	poolMemory := int64(len(pool.mapInfoPool)*int(unsafe.Sizeof(EBPFMapInfo{}))) +
		int64(len(pool.progInfoPool)*int(unsafe.Sizeof(EBPFProgInfo{}))) +
		int64(len(pool.linkInfoPool)*int(unsafe.Sizeof(EBPFLinkInfo{})))
	
	atomic.StoreInt64(&pool.currentMemoryUsage, poolMemory)
	
	return pool
}

// GetMapInfo returns a pre-allocated map info object or nil if pool exhausted
func (p *MemoryPool) GetMapInfo() *EBPFMapInfo {
	used := atomic.LoadInt32(&p.mapPoolUsed)
	if int(used) >= len(p.mapInfoPool) {
		return nil // Pool exhausted
	}
	
	// Try to increment atomically
	if !atomic.CompareAndSwapInt32(&p.mapPoolUsed, used, used+1) {
		return nil // Race condition, try again next time
	}
	
	return &p.mapInfoPool[used]
}

// GetProgInfo returns a pre-allocated prog info object or nil if pool exhausted
func (p *MemoryPool) GetProgInfo() *EBPFProgInfo {
	used := atomic.LoadInt32(&p.progPoolUsed)
	if int(used) >= len(p.progInfoPool) {
		return nil // Pool exhausted
	}
	
	if !atomic.CompareAndSwapInt32(&p.progPoolUsed, used, used+1) {
		return nil
	}
	
	return &p.progInfoPool[used]
}

// GetLinkInfo returns a pre-allocated link info object or nil if pool exhausted
func (p *MemoryPool) GetLinkInfo() *EBPFLinkInfo {
	used := atomic.LoadInt32(&p.linkPoolUsed)
	if int(used) >= len(p.linkInfoPool) {
		return nil // Pool exhausted
	}
	
	if !atomic.CompareAndSwapInt32(&p.linkPoolUsed, used, used+1) {
		return nil
	}
	
	return &p.linkInfoPool[used]
}

// Reset clears the pools for reuse without deallocating memory
func (p *MemoryPool) Reset() {
	atomic.StoreInt32(&p.mapPoolUsed, 0)
	atomic.StoreInt32(&p.progPoolUsed, 0)
	atomic.StoreInt32(&p.linkPoolUsed, 0)
}

// GetCurrentMemoryUsage returns current memory usage in bytes
func (p *MemoryPool) GetCurrentMemoryUsage() int64 {
	return atomic.LoadInt64(&p.currentMemoryUsage)
}

// GetMaxMemoryLimit returns the maximum allowed memory
func (p *MemoryPool) GetMaxMemoryLimit() int64 {
	return atomic.LoadInt64(&p.maxMemoryLimit)
}

// IsMemoryLimitExceeded checks if we're close to memory limit
func (p *MemoryPool) IsMemoryLimitExceeded() bool {
	current := atomic.LoadInt64(&p.currentMemoryUsage)
	limit := atomic.LoadInt64(&p.maxMemoryLimit)
	return current >= limit*90/100 // 90% threshold
}

// GetPoolUtilization returns pool usage statistics
func (p *MemoryPool) GetPoolUtilization() (mapUsed, progUsed, linkUsed int32) {
	return atomic.LoadInt32(&p.mapPoolUsed),
		atomic.LoadInt32(&p.progPoolUsed),
		atomic.LoadInt32(&p.linkPoolUsed)
}

// MetricsCache provides fast read access for /metrics endpoint
type MetricsCache struct {
	// Pre-computed metric values for fast access
	totalObjects      int64
	totalMemoryBytes  int64
	mapCount         int64
	progCount        int64
	linkCount        int64
	
	// Aggregated memory by type
	totalMapMemory    int64
	totalProgMemory   int64
	
	// Performance counters
	totalProgRuns     int64
	totalProgRuntime  int64
	
	// Collection status
	collectionActive  int64
	collectionErrors  int64
	memoryLimitHit    int64
	
	// Last update timestamp
	lastUpdateTime    int64
	
	// Read-write mutex for cache updates
	mu sync.RWMutex
}

// NewMetricsCache creates a new metrics cache
func NewMetricsCache() *MetricsCache {
	return &MetricsCache{}
}

// UpdateCache atomically updates all cached values
func (c *MetricsCache) UpdateCache(
	totalObjects, totalMemory, mapCount, progCount, linkCount int64,
	mapMemory, progMemory, progRuns, progRuntime int64,
	collectionActive, collectionErrors, memoryLimitHit int64,
	timestamp int64) {
	
	// Use atomic operations for thread-safe updates
	atomic.StoreInt64(&c.totalObjects, totalObjects)
	atomic.StoreInt64(&c.totalMemoryBytes, totalMemory)
	atomic.StoreInt64(&c.mapCount, mapCount)
	atomic.StoreInt64(&c.progCount, progCount)
	atomic.StoreInt64(&c.linkCount, linkCount)
	atomic.StoreInt64(&c.totalMapMemory, mapMemory)
	atomic.StoreInt64(&c.totalProgMemory, progMemory)
	atomic.StoreInt64(&c.totalProgRuns, progRuns)
	atomic.StoreInt64(&c.totalProgRuntime, progRuntime)
	atomic.StoreInt64(&c.collectionActive, collectionActive)
	atomic.StoreInt64(&c.collectionErrors, collectionErrors)
	atomic.StoreInt64(&c.memoryLimitHit, memoryLimitHit)
	atomic.StoreInt64(&c.lastUpdateTime, timestamp)
}

// GetCachedValues returns all cached values atomically for fast /metrics response
func (c *MetricsCache) GetCachedValues() (
	totalObjects, totalMemory, mapCount, progCount, linkCount int64,
	mapMemory, progMemory, progRuns, progRuntime int64,
	collectionActive, collectionErrors, memoryLimitHit, lastUpdate int64) {
	
	return atomic.LoadInt64(&c.totalObjects),
		atomic.LoadInt64(&c.totalMemoryBytes),
		atomic.LoadInt64(&c.mapCount),
		atomic.LoadInt64(&c.progCount),
		atomic.LoadInt64(&c.linkCount),
		atomic.LoadInt64(&c.totalMapMemory),
		atomic.LoadInt64(&c.totalProgMemory),
		atomic.LoadInt64(&c.totalProgRuns),
		atomic.LoadInt64(&c.totalProgRuntime),
		atomic.LoadInt64(&c.collectionActive),
		atomic.LoadInt64(&c.collectionErrors),
		atomic.LoadInt64(&c.memoryLimitHit),
		atomic.LoadInt64(&c.lastUpdateTime)
}

// BatchProcessor handles efficient batch processing of eBPF objects
type BatchProcessor struct {
	maxBatchSize    int
	syscallLimiter  chan struct{} // Rate limiter for syscalls
}

// NewBatchProcessor creates a new batch processor with rate limiting
func NewBatchProcessor(maxBatchSize int, maxConcurrentSyscalls int) *BatchProcessor {
	if maxBatchSize <= 0 {
		maxBatchSize = MaxBatchSize
	}
	if maxConcurrentSyscalls <= 0 {
		maxConcurrentSyscalls = 5 // Conservative default
	}
	
	return &BatchProcessor{
		maxBatchSize:   maxBatchSize,
		syscallLimiter: make(chan struct{}, maxConcurrentSyscalls),
	}
}

// ProcessInBatches processes object IDs in batches to minimize syscall overhead
func (bp *BatchProcessor) ProcessInBatches(objectIDs []uint32, processFn func(id uint32) error) error {
	for i := 0; i < len(objectIDs); i += bp.maxBatchSize {
		end := i + bp.maxBatchSize
		if end > len(objectIDs) {
			end = len(objectIDs)
		}
		
		// Rate limit syscalls
		bp.syscallLimiter <- struct{}{}
		
		// Process batch
		for j := i; j < end; j++ {
			if err := processFn(objectIDs[j]); err != nil {
				// Log error but continue processing
				// Don't fail entire batch for one object
			}
		}
		
		<-bp.syscallLimiter
	}
	
	return nil
}

// EfficientObjectTracker tracks objects with minimal memory overhead
type EfficientObjectTracker struct {
	// Use slices instead of maps for better memory efficiency
	// and faster iteration
	mapIDs    []uint32
	progIDs   []uint32
	linkIDs   []uint32
	
	// Compact storage for basic metrics
	mapMemoryUsage  []uint64 // Parallel arrays for memory efficiency
	progMemoryUsage []uint64
	progRunCounts   []uint64
	progRuntimes    []uint64
	
	// Single mutex for all operations
	mu sync.RWMutex
}

// NewEfficientObjectTracker creates a memory-efficient object tracker
func NewEfficientObjectTracker() *EfficientObjectTracker {
	return &EfficientObjectTracker{
		mapIDs:          make([]uint32, 0, MaxTrackedMaps),
		progIDs:         make([]uint32, 0, MaxTrackedProgs),
		linkIDs:         make([]uint32, 0, MaxTrackedLinks),
		mapMemoryUsage:  make([]uint64, 0, MaxTrackedMaps),
		progMemoryUsage: make([]uint64, 0, MaxTrackedProgs),
		progRunCounts:   make([]uint64, 0, MaxTrackedProgs),
		progRuntimes:    make([]uint64, 0, MaxTrackedProgs),
	}
}

// UpdateMaps efficiently updates map tracking with minimal allocations
func (t *EfficientObjectTracker) UpdateMaps(maps []EBPFMapInfo) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	// Reset slices but keep underlying arrays
	t.mapIDs = t.mapIDs[:0]
	t.mapMemoryUsage = t.mapMemoryUsage[:0]
	
	// Add new data
	for i := range maps {
		if len(t.mapIDs) >= cap(t.mapIDs) {
			break // Don't exceed pre-allocated capacity
		}
		t.mapIDs = append(t.mapIDs, maps[i].ID)
		t.mapMemoryUsage = append(t.mapMemoryUsage, maps[i].MemoryUsage)
	}
}

// UpdateProgs efficiently updates program tracking
func (t *EfficientObjectTracker) UpdateProgs(progs []EBPFProgInfo) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	t.progIDs = t.progIDs[:0]
	t.progMemoryUsage = t.progMemoryUsage[:0]
	t.progRunCounts = t.progRunCounts[:0]
	t.progRuntimes = t.progRuntimes[:0]
	
	for i := range progs {
		if len(t.progIDs) >= cap(t.progIDs) {
			break
		}
		t.progIDs = append(t.progIDs, progs[i].ID)
		t.progMemoryUsage = append(t.progMemoryUsage, progs[i].MemoryUsage)
		t.progRunCounts = append(t.progRunCounts, progs[i].RunCnt)
		t.progRuntimes = append(t.progRuntimes, progs[i].RunTimeBs)
	}
}

// UpdateLinks efficiently updates link tracking
func (t *EfficientObjectTracker) UpdateLinks(links []EBPFLinkInfo) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	t.linkIDs = t.linkIDs[:0]
	
	for i := range links {
		if len(t.linkIDs) >= cap(t.linkIDs) {
			break
		}
		t.linkIDs = append(t.linkIDs, links[i].ID)
	}
}

// GetAggregatedStats returns aggregated statistics without copying data
func (t *EfficientObjectTracker) GetAggregatedStats() (
	mapCount, progCount, linkCount int,
	totalMapMemory, totalProgMemory, totalProgRuns, totalProgRuntime uint64) {
	
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	mapCount = len(t.mapIDs)
	progCount = len(t.progIDs)
	linkCount = len(t.linkIDs)
	
	// Calculate aggregates
	for i := range t.mapMemoryUsage {
		totalMapMemory += t.mapMemoryUsage[i]
	}
	
	for i := range t.progMemoryUsage {
		totalProgMemory += t.progMemoryUsage[i]
		totalProgRuns += t.progRunCounts[i]
		totalProgRuntime += t.progRuntimes[i]
	}
	
	return
}