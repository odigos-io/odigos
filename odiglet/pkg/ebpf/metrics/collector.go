package metrics

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/go-logr/logr"
	otelmetric "go.opentelemetry.io/otel/metric"
)

// EBPFObjectType represents the type of eBPF object
type EBPFObjectType string

const (
	// eBPF object types
	BPFMapType        EBPFObjectType = "map"
	BPFProgType       EBPFObjectType = "prog"
	BPFLinkType       EBPFObjectType = "link"
	BPFBTFType        EBPFObjectType = "btf"
	BPFPerfEventType  EBPFObjectType = "perf_event"
	BPFRingBufferType EBPFObjectType = "ring_buffer"
)

// EBPFMapInfo represents information about an eBPF map
type EBPFMapInfo struct {
	ID            uint32
	Type          string
	Name          string
	KeySize       uint32
	ValueSize     uint32
	MaxEntries    uint32
	MapFlags      uint32
	MemoryUsage   uint64
	FrozenFlag    bool
	PinnedPath    string
}

// EBPFProgInfo represents information about an eBPF program
type EBPFProgInfo struct {
	ID               uint32
	Type             string
	Name             string
	Tag              string
	LoadTime         time.Time
	UID              uint32
	GID              uint32
	CreatedByUID     uint32
	MapIDs           []uint32
	InsnCnt          uint32
	JitedProgLen     uint32
	XlatedProgLen    uint32
	MemoryUsage      uint64
	NrMapIDs         uint32
	VerifiedInsnCnt  uint32
	RunTimeBs        uint64
	RunCnt           uint64
}

// EBPFLinkInfo represents information about an eBPF link
type EBPFLinkInfo struct {
	ID       uint32
	Type     string
	ProgID   uint32
	TargetID uint32
}

// EBPFMetricsCollector collects comprehensive eBPF metrics with strict resource limits
type EBPFMetricsCollector struct {
	logger             logr.Logger
	meter              otelmetric.Meter
	
	// High-level aggregate metrics only (for /metrics endpoint performance)
	ebpfTotalObjects        otelmetric.Int64UpDownCounter
	ebpfTotalMemoryBytes    otelmetric.Int64UpDownCounter
	ebpfMapCount           otelmetric.Int64UpDownCounter
	ebpfProgCount          otelmetric.Int64UpDownCounter
	ebpfLinkCount          otelmetric.Int64UpDownCounter
	ebpfCollectionActive   otelmetric.Int64UpDownCounter
	ebpfCollectionErrors   otelmetric.Int64Counter
	ebpfMemoryLimitHit     otelmetric.Int64Counter
	ebpfCollectorMemoryUsage otelmetric.Int64UpDownCounter
	
	// Collection state
	collectionInterval time.Duration
	enableSystemStats  bool
	
	// Efficient memory and resource management
	memoryPool       *MemoryPool
	batchProcessor   *BatchProcessor  
	metricsCache     *MetricsCache
	objectTracker    *EfficientObjectTracker
	
	// Atomic counters for collection status
	memoryLimitHit   *int64
	collectionErrors *int64
}

// NewEBPFMetricsCollector creates a new eBPF metrics collector with strict resource limits
func NewEBPFMetricsCollector(logger logr.Logger, meter otelmetric.Meter) (*EBPFMetricsCollector, error) {
	return NewEBPFMetricsCollectorWithConfig(logger, meter, DefaultEBPFMetricsConfig())
}

// NewEBPFMetricsCollectorWithConfig creates a new eBPF metrics collector with custom configuration
func NewEBPFMetricsCollectorWithConfig(logger logr.Logger, meter otelmetric.Meter, config *EBPFMetricsConfig) (*EBPFMetricsCollector, error) {
	var err, errs error
	
	// Validate and apply configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	
	// Initialize efficient components with config
	memoryPool := NewMemoryPool(config.MaxMemoryBytes)
	batchProcessor := NewBatchProcessor(config.MaxBatchSize, config.MaxConcurrentSyscalls)
	metricsCache := NewMetricsCache()
	objectTracker := NewEfficientObjectTracker()
	
	// Initialize atomic counters
	var memoryLimitHit, collectionErrors int64
	
	c := &EBPFMetricsCollector{
		logger:             logger,
		meter:              meter,
		collectionInterval: config.CollectionInterval,
		enableSystemStats:  config.EnableSystemStats,
		memoryPool:        memoryPool,
		batchProcessor:    batchProcessor,
		metricsCache:      metricsCache,
		objectTracker:     objectTracker,
		memoryLimitHit:    &memoryLimitHit,
		collectionErrors:  &collectionErrors,
	}

	// Initialize high-level aggregate metrics only (for performance)
	c.ebpfTotalObjects, err = meter.Int64UpDownCounter(
		"odigos.ebpf.objects.total",
		otelmetric.WithDescription("Total number of eBPF objects allocated"),
		otelmetric.WithUnit("{object}"),
	)
	errs = appendError(errs, err)

	c.ebpfTotalMemoryBytes, err = meter.Int64UpDownCounter(
		"odigos.ebpf.total_memory_bytes",
		otelmetric.WithDescription("Total memory allocated to eBPF objects"),
		otelmetric.WithUnit("By"),
	)
	errs = appendError(errs, err)

	c.ebpfMapCount, err = meter.Int64UpDownCounter(
		"odigos.ebpf.maps.count",
		otelmetric.WithDescription("Number of eBPF maps"),
		otelmetric.WithUnit("{map}"),
	)
	errs = appendError(errs, err)

	c.ebpfProgCount, err = meter.Int64UpDownCounter(
		"odigos.ebpf.programs.count",
		otelmetric.WithDescription("Number of eBPF programs"),
		otelmetric.WithUnit("{program}"),
	)
	errs = appendError(errs, err)

	c.ebpfLinkCount, err = meter.Int64UpDownCounter(
		"odigos.ebpf.links.count",
		otelmetric.WithDescription("Number of eBPF links"),
		otelmetric.WithUnit("{link}"),
	)
	errs = appendError(errs, err)

	c.ebpfCollectionActive, err = meter.Int64UpDownCounter(
		"odigos.ebpf.collection.active",
		otelmetric.WithDescription("Whether eBPF metrics collection is active (1) or stopped (0)"),
		otelmetric.WithUnit("{status}"),
	)
	errs = appendError(errs, err)

	c.ebpfCollectionErrors, err = meter.Int64Counter(
		"odigos.ebpf.collection.errors_total",
		otelmetric.WithDescription("Total number of eBPF metrics collection errors"),
		otelmetric.WithUnit("{error}"),
	)
	errs = appendError(errs, err)

	c.ebpfMemoryLimitHit, err = meter.Int64Counter(
		"odigos.ebpf.collection.memory_limit_hit_total",
		otelmetric.WithDescription("Total times metrics collection was stopped due to memory limits"),
		otelmetric.WithUnit("{event}"),
	)
	errs = appendError(errs, err)

	c.ebpfCollectorMemoryUsage, err = meter.Int64UpDownCounter(
		"odigos.ebpf.collector.memory_usage_bytes",
		otelmetric.WithDescription("Memory used by the eBPF metrics collector itself"),
		otelmetric.WithUnit("By"),
	)
	errs = appendError(errs, err)

	return c, errs
}

// Start begins collecting eBPF metrics
func (c *EBPFMetricsCollector) Start(ctx context.Context) error {
	c.logger.Info("Starting eBPF metrics collection", "interval", c.collectionInterval)
	
	ticker := time.NewTicker(c.collectionInterval)
	defer ticker.Stop()

	// Collect initial metrics
	if err := c.collectMetrics(ctx); err != nil {
		c.logger.Error(err, "Failed to collect initial eBPF metrics")
	}

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Stopping eBPF metrics collection")
			return ctx.Err()
		case <-ticker.C:
			if err := c.collectMetrics(ctx); err != nil {
				c.logger.Error(err, "Failed to collect eBPF metrics")
			}
		}
	}
}

// collectMetrics efficiently gathers eBPF metrics with strict resource limits
func (c *EBPFMetricsCollector) collectMetrics(ctx context.Context) error {
	// Set collection as active
	c.ebpfCollectionActive.Add(ctx, 1)
	defer c.ebpfCollectionActive.Add(ctx, -1)

	// Update collector's own memory usage metric
	collectorMemory := c.memoryPool.GetCurrentMemoryUsage()
	c.ebpfCollectorMemoryUsage.Add(ctx, collectorMemory)

	// Reset memory pools for reuse
	c.memoryPool.Reset()

	var errorCount int64

	// Collect and aggregate all object information efficiently
	maps, mapErr := c.parseMapInfo()
	if mapErr != nil {
		c.logger.V(1).Info("Map collection error", "error", mapErr)
		errorCount++
	}

	progs, progErr := c.parseProgInfo()
	if progErr != nil {
		c.logger.V(1).Info("Program collection error", "error", progErr)
		errorCount++
	}

	links, linkErr := c.parseLinkInfo()
	if linkErr != nil {
		c.logger.V(1).Info("Link collection error", "error", linkErr)
		errorCount++
	}

	// Update object tracker efficiently
	if maps != nil {
		c.objectTracker.UpdateMaps(convertToMapInfoSlice(maps))
	}
	if progs != nil {
		c.objectTracker.UpdateProgs(convertToProgInfoSlice(progs))
	}
	if links != nil {
		c.objectTracker.UpdateLinks(convertToLinkInfoSlice(links))
	}

	// Get aggregated statistics
	mapCount, progCount, linkCount, totalMapMemory, totalProgMemory, _, _ := 
		c.objectTracker.GetAggregatedStats()

	// Calculate totals
	totalObjects := int64(mapCount + progCount + linkCount)
	totalMemory := int64(totalMapMemory + totalProgMemory)

	// Update high-level metrics efficiently (single atomic operation each)
	c.ebpfTotalObjects.Add(ctx, totalObjects)
	c.ebpfTotalMemoryBytes.Add(ctx, totalMemory)
	c.ebpfMapCount.Add(ctx, int64(mapCount))
	c.ebpfProgCount.Add(ctx, int64(progCount))
	c.ebpfLinkCount.Add(ctx, int64(linkCount))

	// Update error counters
	if errorCount > 0 {
		c.ebpfCollectionErrors.Add(ctx, errorCount)
		atomic.AddInt64(c.collectionErrors, errorCount)
	}

	// Update cache for fast /metrics endpoint response
	c.metricsCache.UpdateCache(
		totalObjects, totalMemory, int64(mapCount), int64(progCount), int64(linkCount),
		int64(totalMapMemory), int64(totalProgMemory), 0, 0, // Skip detailed prog stats for performance
		1, errorCount, atomic.LoadInt64(c.memoryLimitHit), time.Now().Unix(),
	)

	return nil
}

// Helper functions to convert pointer slices to value slices efficiently
func convertToMapInfoSlice(maps []*EBPFMapInfo) []EBPFMapInfo {
	result := make([]EBPFMapInfo, len(maps))
	for i, m := range maps {
		if m != nil {
			result[i] = *m
		}
	}
	return result
}

func convertToProgInfoSlice(progs []*EBPFProgInfo) []EBPFProgInfo {
	result := make([]EBPFProgInfo, len(progs))
	for i, p := range progs {
		if p != nil {
			result[i] = *p
		}
	}
	return result
}

func convertToLinkInfoSlice(links []*EBPFLinkInfo) []EBPFLinkInfo {
	result := make([]EBPFLinkInfo, len(links))
	for i, l := range links {
		if l != nil {
			result[i] = *l
		}
	}
	return result
}

// Helper functions and utilities

func appendError(base error, new error) error {
	if new == nil {
		return base
	}
	if base == nil {
		return new
	}
	return fmt.Errorf("%v; %w", base, new)
}

// SetCollectionInterval sets the interval for metrics collection
func (c *EBPFMetricsCollector) SetCollectionInterval(interval time.Duration) {
	c.collectionInterval = interval
}

// EnableSystemStats enables or disables system-level eBPF statistics collection
func (c *EBPFMetricsCollector) EnableSystemStats(enable bool) {
	c.enableSystemStats = enable
}

// GetMemoryUsage returns current memory usage of the collector
func (c *EBPFMetricsCollector) GetMemoryUsage() int64 {
	return c.memoryPool.GetCurrentMemoryUsage()
}

// GetMemoryLimit returns the maximum memory limit
func (c *EBPFMetricsCollector) GetMemoryLimit() int64 {
	return c.memoryPool.GetMaxMemoryLimit()
}