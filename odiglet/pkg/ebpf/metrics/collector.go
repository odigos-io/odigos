package metrics

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/cilium/ebpf"
	"github.com/go-logr/logr"
	"golang.org/x/sync/singleflight"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/attribute"
)

// EBPFMetricsConfig provides configuration for eBPF metrics collection
type EBPFMetricsConfig struct {
	// Memory Management
	MaxMemoryBytes   int64         `json:"max_memory_bytes" yaml:"max_memory_bytes"`
	CollectionInterval time.Duration `json:"collection_interval" yaml:"collection_interval"`
	
	// Filtering
	ProgramPrefixes  []string      `json:"program_prefixes" yaml:"program_prefixes"`     // Filter programs by name prefix
	EnablePerMapMetrics bool       `json:"enable_per_map_metrics" yaml:"enable_per_map_metrics"` // Enable detailed per-map metrics
	EnablePerProgMetrics bool      `json:"enable_per_prog_metrics" yaml:"enable_per_prog_metrics"` // Enable detailed per-program metrics
	
	// Performance
	MaxMapsToTrack   int           `json:"max_maps_to_track" yaml:"max_maps_to_track"`   // Limit number of maps tracked individually
	MaxProgsToTrack  int           `json:"max_progs_to_track" yaml:"max_progs_to_track"` // Limit number of programs tracked individually
}

// DefaultEBPFMetricsConfig returns production-optimized configuration
func DefaultEBPFMetricsConfig() *EBPFMetricsConfig {
	return &EBPFMetricsConfig{
		MaxMemoryBytes:       10 * 1024 * 1024, // 10MB
		CollectionInterval:   60 * time.Second,
		ProgramPrefixes:      []string{"odigos_", "trace_", "uprobe_", "uretprobe_"}, // odiglet prefixes
		EnablePerMapMetrics:  false, // Aggregate only for performance
		EnablePerProgMetrics: false, // Aggregate only for performance  
		MaxMapsToTrack:       500,
		MaxProgsToTrack:      200,
	}
}

// HighPerformanceConfig returns minimal resource usage configuration
func HighPerformanceConfig() *EBPFMetricsConfig {
	return &EBPFMetricsConfig{
		MaxMemoryBytes:       5 * 1024 * 1024, // 5MB
		CollectionInterval:   120 * time.Second,
		ProgramPrefixes:      []string{"odigos_"}, // Only odigos programs
		EnablePerMapMetrics:  false,
		EnablePerProgMetrics: false,
		MaxMapsToTrack:       100,
		MaxProgsToTrack:      50,
	}
}

// DetailedConfig returns configuration with per-object metrics for debugging
func DetailedConfig() *EBPFMetricsConfig {
	return &EBPFMetricsConfig{
		MaxMemoryBytes:       50 * 1024 * 1024, // 50MB
		CollectionInterval:   30 * time.Second,
		ProgramPrefixes:      nil, // All programs
		EnablePerMapMetrics:  true,
		EnablePerProgMetrics: true,
		MaxMapsToTrack:       1000,
		MaxProgsToTrack:      500,
	}
}

// bpfUsage tracks eBPF resource usage similar to Cilium but with more detail
type bpfUsage struct {
	// Aggregate counters
	programs     uint64
	programBytes uint64
	maps         uint64
	mapBytes     uint64
	
	// Detailed information for granular metrics
	programDetails []ProgramDetails
	mapDetails     []MapDetails
}

type ProgramDetails struct {
	ID          ebpf.ProgramID
	Name        string
	Type        string
	MemoryBytes uint64
	MapIDs      []ebpf.MapID
}

type MapDetails struct {
	ID          ebpf.MapID
	Name        string
	Type        string
	MemoryBytes uint64
	KeySize     uint32
	ValueSize   uint32
	MaxEntries  uint32
}

// bpfVisitor implements efficient eBPF object discovery like Cilium
type bpfVisitor struct {
	bpfUsage
	config *EBPFMetricsConfig
	
	programsVisited map[ebpf.ProgramID]struct{}
	mapsVisited     map[ebpf.MapID]struct{}
}

func newBPFVisitor(config *EBPFMetricsConfig) *bpfVisitor {
	return &bpfVisitor{
		config:          config,
		programsVisited: make(map[ebpf.ProgramID]struct{}),
		mapsVisited:     make(map[ebpf.MapID]struct{}),
		bpfUsage: bpfUsage{
			programDetails: make([]ProgramDetails, 0),
			mapDetails:     make([]MapDetails, 0),
		},
	}
}

// Usage discovers and measures all relevant eBPF objects
func (v *bpfVisitor) Usage() (*bpfUsage, error) {
	var id ebpf.ProgramID
	for {
		id, err := ebpf.ProgramGetNextID(id)
		if errors.Is(err, os.ErrNotExist) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("get next program: %w", err)
		}

		if err := v.visitProgram(id); err != nil {
			// Log error but continue - don't fail entire collection for one object
			continue
		}
		
		// Respect memory/object limits
		if len(v.programDetails) >= v.config.MaxProgsToTrack {
			break
		}
	}

	return &v.bpfUsage, nil
}

// visitProgram processes a single eBPF program
func (v *bpfVisitor) visitProgram(id ebpf.ProgramID) error {
	if _, ok := v.programsVisited[id]; ok {
		return nil
	}
	v.programsVisited[id] = struct{}{}

	prog, err := ebpf.NewProgramFromID(id)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("open program by id: %w", err)
	}
	defer prog.Close()

	info, err := prog.Info()
	if err != nil {
		return fmt.Errorf("get program info: %w", err)
	}

	// Filter by program name prefixes if configured
	if len(v.config.ProgramPrefixes) > 0 {
		hasPrefix := func(prefix string) bool { 
			return strings.HasPrefix(info.Name, prefix) 
		}
		if !slices.ContainsFunc(v.config.ProgramPrefixes, hasPrefix) {
			return nil
		}
	}

	mem, ok := info.Memlock()
	if !ok {
		return fmt.Errorf("program %s has zero memlock", info.Name)
	}

	// Update aggregate counters
	v.programs++
	v.programBytes += mem

	// Collect detailed program information if enabled
	if v.config.EnablePerProgMetrics {
		progDetail := ProgramDetails{
			ID:          id,
			Name:        info.Name,
			Type:        info.Type.String(),
			MemoryBytes: mem,
		}
		
		// Get associated map IDs
		if mapIDs, ok := info.MapIDs(); ok {
			progDetail.MapIDs = mapIDs
		}
		
		v.programDetails = append(v.programDetails, progDetail)
	}

	// Visit all maps used by this program
	if mapIDs, ok := info.MapIDs(); ok {
		for _, mapID := range mapIDs {
			if err := v.visitMap(mapID); err != nil {
				// Log but continue
				continue
			}
			
			// Respect map limits
			if len(v.mapDetails) >= v.config.MaxMapsToTrack {
				break
			}
		}
	}

	return nil
}

// visitMap processes a single eBPF map
func (v *bpfVisitor) visitMap(id ebpf.MapID) error {
	if _, ok := v.mapsVisited[id]; ok {
		return nil
	}
	v.mapsVisited[id] = struct{}{}

	m, err := ebpf.NewMapFromID(id)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("open map by id: %w", err)
	}
	defer m.Close()

	info, err := m.Info()
	if err != nil {
		return fmt.Errorf("get map info: %w", err)
	}

	// Get memory usage (may be 0 for maps with BPF_F_NO_PREALLOC)
	mem, _ := info.Memlock()

	// Update aggregate counters
	v.maps++
	v.mapBytes += mem

	// Collect detailed map information if enabled
	if v.config.EnablePerMapMetrics {
		mapDetail := MapDetails{
			ID:          id,
			Name:        info.Name,
			Type:        info.Type.String(),
			MemoryBytes: mem,
			KeySize:     info.KeySize,
			ValueSize:   info.ValueSize,
			MaxEntries:  info.MaxEntries,
		}
		
		v.mapDetails = append(v.mapDetails, mapDetail)
	}

	return nil
}

// EBPFMetricsCollector implements efficient eBPF metrics collection using cilium/ebpf
type EBPFMetricsCollector struct {
	logger logr.Logger
	config *EBPFMetricsConfig
	meter  otelmetric.Meter
	
	// Singleflight to prevent concurrent collection
	sfg singleflight.Group
	
	// OpenTelemetry metrics - aggregate only for performance
	totalPrograms      otelmetric.Int64UpDownCounter
	totalProgramMemory otelmetric.Int64UpDownCounter
	totalMaps          otelmetric.Int64UpDownCounter
	totalMapMemory     otelmetric.Int64UpDownCounter
	
	// Collection status metrics
	collectionErrors   otelmetric.Int64Counter
	collectionDuration otelmetric.Int64Histogram
	memoryLimitHit     otelmetric.Int64Counter
	
	// Detailed per-object metrics (only if enabled)
	mapMemoryBytes     otelmetric.Int64UpDownCounter
	programMemoryBytes otelmetric.Int64UpDownCounter
	
	// Status tracking
	isRunning int64
	
	// Previous values for delta calculation
	prevPrograms     int64
	prevProgramMemory int64
	prevMaps         int64
	prevMapMemory    int64
}

// NewEBPFMetricsCollector creates a new eBPF metrics collector using the cilium/ebpf approach
func NewEBPFMetricsCollector(logger logr.Logger, meter otelmetric.Meter) (*EBPFMetricsCollector, error) {
	return NewEBPFMetricsCollectorWithConfig(logger, meter, DefaultEBPFMetricsConfig())
}

func NewEBPFMetricsCollectorWithConfig(logger logr.Logger, meter otelmetric.Meter, config *EBPFMetricsConfig) (*EBPFMetricsCollector, error) {
	c := &EBPFMetricsCollector{
		logger: logger,
		config: config,
		meter:  meter,
	}

	var err error
	
	// Initialize aggregate metrics (always enabled)
	c.totalPrograms, err = meter.Int64UpDownCounter(
		"odigos.ebpf.programs.total",
		otelmetric.WithDescription("Total number of eBPF programs"),
		otelmetric.WithUnit("{program}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create programs counter: %w", err)
	}

	c.totalProgramMemory, err = meter.Int64UpDownCounter(
		"odigos.ebpf.programs.memory_bytes",
		otelmetric.WithDescription("Total memory usage of eBPF programs"),
		otelmetric.WithUnit("By"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create program memory counter: %w", err)
	}

	c.totalMaps, err = meter.Int64UpDownCounter(
		"odigos.ebpf.maps.total",
		otelmetric.WithDescription("Total number of eBPF maps"),
		otelmetric.WithUnit("{map}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create maps counter: %w", err)
	}

	c.totalMapMemory, err = meter.Int64UpDownCounter(
		"odigos.ebpf.maps.memory_bytes",
		otelmetric.WithDescription("Total memory usage of eBPF maps"),
		otelmetric.WithUnit("By"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create map memory counter: %w", err)
	}

	// Collection status metrics
	c.collectionErrors, err = meter.Int64Counter(
		"odigos.ebpf.collection.errors_total",
		otelmetric.WithDescription("Total eBPF metrics collection errors"),
		otelmetric.WithUnit("{error}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create errors counter: %w", err)
	}

	c.collectionDuration, err = meter.Int64Histogram(
		"odigos.ebpf.collection.duration_ms",
		otelmetric.WithDescription("eBPF metrics collection duration"),
		otelmetric.WithUnit("ms"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create duration histogram: %w", err)
	}

	c.memoryLimitHit, err = meter.Int64Counter(
		"odigos.ebpf.collection.memory_limit_hit_total",
		otelmetric.WithDescription("Times collection stopped due to memory limits"),
		otelmetric.WithUnit("{event}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create memory limit counter: %w", err)
	}

	// Detailed per-object metrics (only if enabled)
	if config.EnablePerMapMetrics {
		c.mapMemoryBytes, err = meter.Int64UpDownCounter(
			"odigos.ebpf.map.memory_bytes",
			otelmetric.WithDescription("Memory usage of individual eBPF maps"),
			otelmetric.WithUnit("By"),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create per-map memory counter: %w", err)
		}
	}

	if config.EnablePerProgMetrics {
		c.programMemoryBytes, err = meter.Int64UpDownCounter(
			"odigos.ebpf.program.memory_bytes",
			otelmetric.WithDescription("Memory usage of individual eBPF programs"),
			otelmetric.WithUnit("By"),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create per-program memory counter: %w", err)
		}
	}

	return c, nil
}

// Start begins the metrics collection loop
func (c *EBPFMetricsCollector) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt64(&c.isRunning, 0, 1) {
		return fmt.Errorf("collector already running")
	}
	defer atomic.StoreInt64(&c.isRunning, 0)

	ticker := time.NewTicker(c.config.CollectionInterval)
	defer ticker.Stop()

	// Collect immediately on start
	c.collectMetrics(ctx)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			c.collectMetrics(ctx)
		}
	}
}

// collectMetrics performs the actual eBPF object discovery and measurement
func (c *EBPFMetricsCollector) collectMetrics(ctx context.Context) {
	startTime := time.Now()
	
	// Use singleflight to prevent concurrent collection
	result, err, _ := c.sfg.Do("collect", func() (interface{}, error) {
		visitor := newBPFVisitor(c.config)
		return visitor.Usage()
	})

	duration := time.Since(startTime)
	c.collectionDuration.Record(ctx, duration.Milliseconds())

	if err != nil {
		c.logger.Error(err, "failed to collect eBPF metrics")
		c.collectionErrors.Add(ctx, 1)
		return
	}

	usage := result.(*bpfUsage)

	// Calculate deltas and update UpDownCounters
	currentPrograms := int64(usage.programs)
	currentProgramMemory := int64(usage.programBytes)
	currentMaps := int64(usage.maps)
	currentMapMemory := int64(usage.mapBytes)
	
	prevPrograms := atomic.LoadInt64(&c.prevPrograms)
	prevProgramMemory := atomic.LoadInt64(&c.prevProgramMemory)
	prevMaps := atomic.LoadInt64(&c.prevMaps)
	prevMapMemory := atomic.LoadInt64(&c.prevMapMemory)
	
	c.totalPrograms.Add(ctx, currentPrograms - prevPrograms)
	c.totalProgramMemory.Add(ctx, currentProgramMemory - prevProgramMemory)
	c.totalMaps.Add(ctx, currentMaps - prevMaps)
	c.totalMapMemory.Add(ctx, currentMapMemory - prevMapMemory)
	
	// Store current values for next iteration
	atomic.StoreInt64(&c.prevPrograms, currentPrograms)
	atomic.StoreInt64(&c.prevProgramMemory, currentProgramMemory)
	atomic.StoreInt64(&c.prevMaps, currentMaps)
	atomic.StoreInt64(&c.prevMapMemory, currentMapMemory)

	// Update detailed metrics if enabled
	if c.config.EnablePerMapMetrics && c.mapMemoryBytes != nil {
		for _, mapDetail := range usage.mapDetails {
			c.mapMemoryBytes.Add(ctx, int64(mapDetail.MemoryBytes),
				otelmetric.WithAttributes(
					attribute.String("map_id", fmt.Sprintf("%d", mapDetail.ID)),
					attribute.String("map_name", mapDetail.Name),
					attribute.String("map_type", mapDetail.Type),
					attribute.Int("key_size", int(mapDetail.KeySize)),
					attribute.Int("value_size", int(mapDetail.ValueSize)),
					attribute.Int("max_entries", int(mapDetail.MaxEntries)),
				))
		}
	}

	if c.config.EnablePerProgMetrics && c.programMemoryBytes != nil {
		for _, progDetail := range usage.programDetails {
			c.programMemoryBytes.Add(ctx, int64(progDetail.MemoryBytes),
				otelmetric.WithAttributes(
					attribute.String("program_id", fmt.Sprintf("%d", progDetail.ID)),
					attribute.String("program_name", progDetail.Name),
					attribute.String("program_type", progDetail.Type),
					attribute.Int("associated_maps", len(progDetail.MapIDs)),
				))
		}
	}

	c.logger.V(1).Info("eBPF metrics collected",
		"programs", usage.programs,
		"program_memory", usage.programBytes,
		"maps", usage.maps,
		"map_memory", usage.mapBytes,
		"duration_ms", duration.Milliseconds(),
	)
}

// GetMemoryUsage returns estimated memory usage of the collector
func (c *EBPFMetricsCollector) GetMemoryUsage() int64 {
	// Estimate based on configuration
	const (
		baseMemory = 1024 * 1024 // 1MB base
		progMemoryPerItem = 200  // bytes per program detail
		mapMemoryPerItem = 150   // bytes per map detail
	)
	
	estimated := baseMemory
	if c.config.EnablePerProgMetrics {
		estimated += c.config.MaxProgsToTrack * progMemoryPerItem
	}
	if c.config.EnablePerMapMetrics {
		estimated += c.config.MaxMapsToTrack * mapMemoryPerItem
	}
	
	return int64(estimated)
}

// SetCollectionInterval updates the collection interval
func (c *EBPFMetricsCollector) SetCollectionInterval(interval time.Duration) {
	c.config.CollectionInterval = interval
}