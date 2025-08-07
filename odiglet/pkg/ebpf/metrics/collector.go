package metrics

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/attribute"
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

// EBPFMetricsCollector collects comprehensive eBPF metrics
type EBPFMetricsCollector struct {
	logger             logr.Logger
	meter              otelmetric.Meter
	
	// Metrics
	ebpfObjectsTotal           otelmetric.Int64UpDownCounter
	ebpfMapMemoryBytes         otelmetric.Int64UpDownCounter
	ebpfMapEntries             otelmetric.Int64UpDownCounter
	ebpfProgMemoryBytes        otelmetric.Int64UpDownCounter
	ebpfProgInstructions       otelmetric.Int64UpDownCounter
	ebpfProgRuntime            otelmetric.Int64Counter
	ebpfProgRunCount           otelmetric.Int64Counter
	ebpfTotalMemoryBytes       otelmetric.Int64UpDownCounter
	ebpfMapOperationsTotal     otelmetric.Int64Counter
	ebpfProgVerificationTime   otelmetric.Int64Histogram
	ebpfPerfBufferEvents       otelmetric.Int64Counter
	ebpfRingBufferEvents       otelmetric.Int64Counter
	ebpfPerfBufferLost         otelmetric.Int64Counter
	ebpfSystemResourceUsage    otelmetric.Float64UpDownCounter
	
	// Collection state
	mutex              sync.RWMutex
	collectionInterval time.Duration
	enableSystemStats  bool
	
	// Tracked objects
	trackedMaps        map[uint32]*EBPFMapInfo
	trackedProgs       map[uint32]*EBPFProgInfo
	trackedLinks       map[uint32]*EBPFLinkInfo
}

// NewEBPFMetricsCollector creates a new eBPF metrics collector
func NewEBPFMetricsCollector(logger logr.Logger, meter otelmetric.Meter) (*EBPFMetricsCollector, error) {
	var err, errs error
	c := &EBPFMetricsCollector{
		logger:             logger,
		meter:              meter,
		collectionInterval: 30 * time.Second, // Default collection interval
		enableSystemStats:  true,
		trackedMaps:        make(map[uint32]*EBPFMapInfo),
		trackedProgs:       make(map[uint32]*EBPFProgInfo),
		trackedLinks:       make(map[uint32]*EBPFLinkInfo),
	}

	// Initialize metrics
	c.ebpfObjectsTotal, err = meter.Int64UpDownCounter(
		"odigos.ebpf.objects.total",
		otelmetric.WithDescription("Total number of eBPF objects allocated"),
		otelmetric.WithUnit("{object}"),
	)
	errs = appendError(errs, err)

	c.ebpfMapMemoryBytes, err = meter.Int64UpDownCounter(
		"odigos.ebpf.map.memory_bytes",
		otelmetric.WithDescription("Memory used by eBPF maps in bytes"),
		otelmetric.WithUnit("By"),
	)
	errs = appendError(errs, err)

	c.ebpfMapEntries, err = meter.Int64UpDownCounter(
		"odigos.ebpf.map.entries",
		otelmetric.WithDescription("Number of entries in eBPF maps"),
		otelmetric.WithUnit("{entry}"),
	)
	errs = appendError(errs, err)

	c.ebpfProgMemoryBytes, err = meter.Int64UpDownCounter(
		"odigos.ebpf.program.memory_bytes",
		otelmetric.WithDescription("Memory used by eBPF programs in bytes"),
		otelmetric.WithUnit("By"),
	)
	errs = appendError(errs, err)

	c.ebpfProgInstructions, err = meter.Int64UpDownCounter(
		"odigos.ebpf.program.instructions",
		otelmetric.WithDescription("Number of instructions in eBPF programs"),
		otelmetric.WithUnit("{instruction}"),
	)
	errs = appendError(errs, err)

	c.ebpfProgRuntime, err = meter.Int64Counter(
		"odigos.ebpf.program.runtime_ns_total",
		otelmetric.WithDescription("Total runtime of eBPF programs in nanoseconds"),
		otelmetric.WithUnit("ns"),
	)
	errs = appendError(errs, err)

	c.ebpfProgRunCount, err = meter.Int64Counter(
		"odigos.ebpf.program.runs_total",
		otelmetric.WithDescription("Total number of eBPF program executions"),
		otelmetric.WithUnit("{execution}"),
	)
	errs = appendError(errs, err)

	c.ebpfTotalMemoryBytes, err = meter.Int64UpDownCounter(
		"odigos.ebpf.total_memory_bytes",
		otelmetric.WithDescription("Total memory allocated to eBPF objects"),
		otelmetric.WithUnit("By"),
	)
	errs = appendError(errs, err)

	c.ebpfMapOperationsTotal, err = meter.Int64Counter(
		"odigos.ebpf.map.operations_total",
		otelmetric.WithDescription("Total number of eBPF map operations"),
		otelmetric.WithUnit("{operation}"),
	)
	errs = appendError(errs, err)

	c.ebpfProgVerificationTime, err = meter.Int64Histogram(
		"odigos.ebpf.program.verification_duration_ms",
		otelmetric.WithDescription("Time spent verifying eBPF programs"),
		otelmetric.WithUnit("ms"),
	)
	errs = appendError(errs, err)

	c.ebpfPerfBufferEvents, err = meter.Int64Counter(
		"odigos.ebpf.perf_buffer.events_total",
		otelmetric.WithDescription("Total events processed through perf buffers"),
		otelmetric.WithUnit("{event}"),
	)
	errs = appendError(errs, err)

	c.ebpfRingBufferEvents, err = meter.Int64Counter(
		"odigos.ebpf.ring_buffer.events_total",
		otelmetric.WithDescription("Total events processed through ring buffers"),
		otelmetric.WithUnit("{event}"),
	)
	errs = appendError(errs, err)

	c.ebpfPerfBufferLost, err = meter.Int64Counter(
		"odigos.ebpf.perf_buffer.lost_events_total",
		otelmetric.WithDescription("Total events lost in perf buffers"),
		otelmetric.WithUnit("{event}"),
	)
	errs = appendError(errs, err)

	c.ebpfSystemResourceUsage, err = meter.Float64UpDownCounter(
		"odigos.ebpf.system.resource_usage_percent",
		otelmetric.WithDescription("eBPF system resource usage percentage"),
		otelmetric.WithUnit("%"),
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

// collectMetrics gathers all eBPF metrics
func (c *EBPFMetricsCollector) collectMetrics(ctx context.Context) error {
	var totalErrors []error

	// Collect map metrics
	if err := c.collectMapMetrics(ctx); err != nil {
		totalErrors = append(totalErrors, fmt.Errorf("map metrics: %w", err))
	}

	// Collect program metrics
	if err := c.collectProgMetrics(ctx); err != nil {
		totalErrors = append(totalErrors, fmt.Errorf("program metrics: %w", err))
	}

	// Collect link metrics
	if err := c.collectLinkMetrics(ctx); err != nil {
		totalErrors = append(totalErrors, fmt.Errorf("link metrics: %w", err))
	}

	// Collect system-level metrics
	if c.enableSystemStats {
		if err := c.collectSystemMetrics(ctx); err != nil {
			totalErrors = append(totalErrors, fmt.Errorf("system metrics: %w", err))
		}
	}

	if len(totalErrors) > 0 {
		return fmt.Errorf("collection errors: %v", totalErrors)
	}

	return nil
}

// collectMapMetrics collects metrics about eBPF maps
func (c *EBPFMetricsCollector) collectMapMetrics(ctx context.Context) error {
	maps, err := c.getBPFMaps()
	if err != nil {
		return fmt.Errorf("failed to get BPF maps: %w", err)
	}

	// Update tracked maps with lock protection
	c.mutex.Lock()
	// Clear previous tracked maps
	c.trackedMaps = make(map[uint32]*EBPFMapInfo)
	for _, mapInfo := range maps {
		c.trackedMaps[mapInfo.ID] = mapInfo
	}
	c.mutex.Unlock()

	var totalMapMemory int64
	var totalMapEntries int64

	for _, mapInfo := range maps {
		attrs := []attribute.KeyValue{
			attribute.String("map_id", fmt.Sprintf("%d", mapInfo.ID)),
			attribute.String("map_name", mapInfo.Name),
			attribute.String("map_type", mapInfo.Type),
		}

		// Update map memory usage
		c.ebpfMapMemoryBytes.Add(ctx, int64(mapInfo.MemoryUsage), otelmetric.WithAttributes(attrs...))
		totalMapMemory += int64(mapInfo.MemoryUsage)

		// Estimate entries (max_entries for now, could be improved with actual usage)
		c.ebpfMapEntries.Add(ctx, int64(mapInfo.MaxEntries), otelmetric.WithAttributes(attrs...))
		totalMapEntries += int64(mapInfo.MaxEntries)

		// Track object count
		c.ebpfObjectsTotal.Add(ctx, 1, otelmetric.WithAttributes(
			attribute.String("object_type", string(BPFMapType)),
			attribute.String("object_name", mapInfo.Name),
		))
	}

	return nil
}

// collectProgMetrics collects metrics about eBPF programs
func (c *EBPFMetricsCollector) collectProgMetrics(ctx context.Context) error {
	progs, err := c.getBPFProgs()
	if err != nil {
		return fmt.Errorf("failed to get BPF programs: %w", err)
	}

	// Update tracked programs with lock protection
	c.mutex.Lock()
	// Clear previous tracked programs
	c.trackedProgs = make(map[uint32]*EBPFProgInfo)
	for _, progInfo := range progs {
		c.trackedProgs[progInfo.ID] = progInfo
	}
	c.mutex.Unlock()

	var totalProgMemory int64
	var totalInstructions int64

	for _, progInfo := range progs {
		attrs := []attribute.KeyValue{
			attribute.String("prog_id", fmt.Sprintf("%d", progInfo.ID)),
			attribute.String("prog_name", progInfo.Name),
			attribute.String("prog_type", progInfo.Type),
		}

		// Update program memory usage
		progMemory := int64(progInfo.JitedProgLen + progInfo.XlatedProgLen)
		c.ebpfProgMemoryBytes.Add(ctx, progMemory, otelmetric.WithAttributes(attrs...))
		totalProgMemory += progMemory

		// Update instruction count
		c.ebpfProgInstructions.Add(ctx, int64(progInfo.InsnCnt), otelmetric.WithAttributes(attrs...))
		totalInstructions += int64(progInfo.InsnCnt)

		// Update runtime stats
		c.ebpfProgRuntime.Add(ctx, int64(progInfo.RunTimeBs), otelmetric.WithAttributes(attrs...))
		c.ebpfProgRunCount.Add(ctx, int64(progInfo.RunCnt), otelmetric.WithAttributes(attrs...))

		// Track object count
		c.ebpfObjectsTotal.Add(ctx, 1, otelmetric.WithAttributes(
			attribute.String("object_type", string(BPFProgType)),
			attribute.String("object_name", progInfo.Name),
		))
	}

	return nil
}

// collectLinkMetrics collects metrics about eBPF links
func (c *EBPFMetricsCollector) collectLinkMetrics(ctx context.Context) error {
	links, err := c.getBPFLinks()
	if err != nil {
		return fmt.Errorf("failed to get BPF links: %w", err)
	}

	// Update tracked links with lock protection
	c.mutex.Lock()
	// Clear previous tracked links
	c.trackedLinks = make(map[uint32]*EBPFLinkInfo)
	for _, linkInfo := range links {
		c.trackedLinks[linkInfo.ID] = linkInfo
	}
	c.mutex.Unlock()

	for _, linkInfo := range links {
		// Track object count
		c.ebpfObjectsTotal.Add(ctx, 1, otelmetric.WithAttributes(
			attribute.String("object_type", string(BPFLinkType)),
			attribute.String("link_type", linkInfo.Type),
			attribute.String("link_id", fmt.Sprintf("%d", linkInfo.ID)),
		))
	}

	return nil
}

// collectSystemMetrics collects system-level eBPF metrics
func (c *EBPFMetricsCollector) collectSystemMetrics(ctx context.Context) error {
	// Collect memory usage from /proc/meminfo if available
	if memUsage, err := c.getEBPFSystemMemoryUsage(); err == nil {
		c.ebpfTotalMemoryBytes.Add(ctx, memUsage, otelmetric.WithAttributes(
			attribute.String("resource_type", "kernel_memory"),
		))
	}

	// Collect eBPF resource limits and usage
	if resourceUsage, err := c.getEBPFResourceUsage(); err == nil {
		c.ebpfSystemResourceUsage.Add(ctx, resourceUsage, otelmetric.WithAttributes(
			attribute.String("resource", "memory_limit"),
		))
	}

	return nil
}

// Helper functions to get eBPF object information
// These would use bpftool-like functionality or libbpf APIs

func (c *EBPFMetricsCollector) getBPFMaps() ([]*EBPFMapInfo, error) {
	// This is a simplified implementation
	// In production, this would use proper BPF APIs to enumerate maps
	return c.parseMapInfo()
}

func (c *EBPFMetricsCollector) getBPFProgs() ([]*EBPFProgInfo, error) {
	// This is a simplified implementation
	// In production, this would use proper BPF APIs to enumerate programs
	return c.parseProgInfo()
}

func (c *EBPFMetricsCollector) getBPFLinks() ([]*EBPFLinkInfo, error) {
	// This is a simplified implementation
	// In production, this would use proper BPF APIs to enumerate links
	return c.parseLinkInfo()
}

// These methods are implemented in bpf_objects.go

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
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.collectionInterval = interval
}

// EnableSystemStats enables or disables system-level eBPF statistics collection
func (c *EBPFMetricsCollector) EnableSystemStats(enable bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.enableSystemStats = enable
}