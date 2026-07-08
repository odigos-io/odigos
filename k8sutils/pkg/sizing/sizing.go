package sizing

import (
	"github.com/odigos-io/odigos/common"
)

type ResourceSizePreset struct {
	CollectorGatewayConfig common.CollectorGatewayConfiguration
	CollectorNodeConfig    common.CollectorNodeConfiguration
}

type Sizing string

const (
	SizeSmall  Sizing = "size_s"
	SizeMedium Sizing = "size_m"
	SizeLarge  Sizing = "size_l"
)

var configs = map[Sizing]ResourceSizePreset{
	SizeSmall: {
		CollectorGatewayConfig: common.CollectorGatewayConfiguration{
			MinReplicas:                1,
			MaxReplicas:                5,
			RequestCPUm:                150,
			LimitCPUm:                  300,
			RequestMemoryMiB:           300,
			LimitMemoryMiB:             300,
			MemoryLimiterLimitMiB:      250, // LimitMemoryMiB - 50
			MemoryLimiterSpikeLimitMiB: 50,  // 20% of MemoryLimiterLimitMiB
			GoMemLimitMib:              200, // 80% of MemoryLimiterLimitMiB
		},
		CollectorNodeConfig: common.CollectorNodeConfiguration{
			RequestMemoryMiB:           150,
			LimitMemoryMiB:             300,
			RequestCPUm:                150,
			LimitCPUm:                  300,
			MemoryLimiterLimitMiB:      250, // LimitMemoryMiB - 50
			MemoryLimiterSpikeLimitMiB: 50,  // 20% of MemoryLimiterSpikeLimitMiB
			GoMemLimitMib:              200, // 80% of MemoryLimiterSpikeLimitMiB
		},
	},
	SizeMedium: {
		CollectorGatewayConfig: common.CollectorGatewayConfiguration{
			MinReplicas:                2,
			MaxReplicas:                8,
			RequestCPUm:                500,
			LimitCPUm:                  1000,
			RequestMemoryMiB:           600,
			LimitMemoryMiB:             600,
			MemoryLimiterLimitMiB:      550, // LimitMemoryMiB - 50
			MemoryLimiterSpikeLimitMiB: 110, // 20% of MemoryLimiterLimitMiB
			GoMemLimitMib:              440, // 80% of MemoryLimiterLimitMiB
		},
		CollectorNodeConfig: common.CollectorNodeConfiguration{
			RequestMemoryMiB:           250,
			LimitMemoryMiB:             500,
			RequestCPUm:                250,
			LimitCPUm:                  500,
			MemoryLimiterLimitMiB:      450, // LimitMemoryMiB - 50
			MemoryLimiterSpikeLimitMiB: 90,  // 20% of MemoryLimiterLimitMiB
			GoMemLimitMib:              360, // 80% of MemoryLimiterLimitMiB
		},
	},
	SizeLarge: {
		CollectorGatewayConfig: common.CollectorGatewayConfiguration{
			MinReplicas:                3,
			MaxReplicas:                12,
			RequestCPUm:                750,
			LimitCPUm:                  1250,
			RequestMemoryMiB:           850,
			LimitMemoryMiB:             850,
			MemoryLimiterLimitMiB:      800, // LimitMemoryMiB - 50
			MemoryLimiterSpikeLimitMiB: 160, // 20% of MemoryLimiterLimitMiB
			GoMemLimitMib:              640, // 80% of MemoryLimiterLimitMiB
		},
		CollectorNodeConfig: common.CollectorNodeConfiguration{
			RequestMemoryMiB:           500,
			LimitMemoryMiB:             750,
			RequestCPUm:                500,
			LimitCPUm:                  750,
			MemoryLimiterLimitMiB:      700, // LimitMemoryMiB - 50
			MemoryLimiterSpikeLimitMiB: 140, // 20% of MemoryLimiterLimitMiB
			GoMemLimitMib:              560, // 80% of MemoryLimiterLimitMiB
		},
	},
}

// GetResourceSizePreset returns the resource size preset for the given sizing
// if the sizing is not valid, it will return the medium size preset
func GetResourceSizePreset(sizing string) ResourceSizePreset {
	if !IsValidSizing(sizing) {
		sizing = string(SizeMedium)
	}

	return configs[Sizing(sizing)]
}

// memoryProfilingNodeFloor is the minimum node-collector sizing applied when
// continuous memory profiling is enabled. Heap-dump parse + central symbolize is a
// bursty, allocation-heavy workload on the node collector, so the trace-only presets
// (e.g. medium's 250/500) OOM under it. This floor keeps the k8s container limit and
// the OTel memory_limiter config in sync with the Helm chart's profiling floor
// (collector.node.memory* in _sizing-helpers.tpl). Larger presets keep their larger
// values (applied field-wise via max); explicit collectorNode overrides still win.
var memoryProfilingNodeFloor = common.CollectorNodeConfiguration{
	RequestMemoryMiB:           384,
	LimitMemoryMiB:             1024,
	MemoryLimiterLimitMiB:      896, // ~128MiB under the container limit for burst headroom
	MemoryLimiterSpikeLimitMiB: 179, // 20% of the hard limit
	GoMemLimitMib:              716, // 80% of the hard limit
}

// memoryProfilingEnabled reports whether cluster-wide continuous memory profiling is
// on, which materially raises the node collector's working set.
func memoryProfilingEnabled(c *common.OdigosConfiguration) bool {
	return c.Profiling != nil &&
		c.Profiling.Enabled != nil && *c.Profiling.Enabled &&
		c.Profiling.Memory != nil &&
		c.Profiling.Memory.Enabled != nil && *c.Profiling.Memory.Enabled
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// applyMemoryProfilingNodeFloor raises each node memory field to at least the
// profiling floor, leaving larger preset values (e.g. size_l/xl) untouched.
func applyMemoryProfilingNodeFloor(n common.CollectorNodeConfiguration) common.CollectorNodeConfiguration {
	f := memoryProfilingNodeFloor
	n.RequestMemoryMiB = maxInt(n.RequestMemoryMiB, f.RequestMemoryMiB)
	n.LimitMemoryMiB = maxInt(n.LimitMemoryMiB, f.LimitMemoryMiB)
	n.MemoryLimiterLimitMiB = maxInt(n.MemoryLimiterLimitMiB, f.MemoryLimiterLimitMiB)
	n.MemoryLimiterSpikeLimitMiB = maxInt(n.MemoryLimiterSpikeLimitMiB, f.MemoryLimiterSpikeLimitMiB)
	n.GoMemLimitMib = maxInt(n.GoMemLimitMib, f.GoMemLimitMib)
	return n
}

// ComputeResourceSizePreset computes the resource size preset for the given Odigos configuration.
func ComputeResourceSizePreset(c *common.OdigosConfiguration) ResourceSizePreset {
	// pick preset (default to medium if invalid/missing)
	if !IsValidSizing(c.ResourceSizePreset) {
		c.ResourceSizePreset = string(SizeMedium)
	}

	// start from preset
	base := configs[Sizing(c.ResourceSizePreset)]
	node := base.CollectorNodeConfig

	// memory profiling raises the node-collector floor (before user overrides, which still win)
	if memoryProfilingEnabled(c) {
		node = applyMemoryProfilingNodeFloor(node)
	}

	// overlay user overrides (non-zero only)
	gw := copyNonZeroGateway(&base.CollectorGatewayConfig, c.CollectorGateway)
	node = copyNonZeroNode(node, c.CollectorNode)

	return ResourceSizePreset{
		CollectorGatewayConfig: *gw,
		CollectorNodeConfig:    node,
	}
}

// MergeSizing lets you reuse the merge logic outside OdigosConfiguration.
// You pass a base preset and optional override structs.
func MergeSizing(preset string, gwOverride *common.CollectorGatewayConfiguration,
	nodeOverride *common.CollectorNodeConfiguration) ResourceSizePreset {
	if !IsValidSizing(preset) {
		preset = string(SizeMedium)
	}
	base := configs[Sizing(preset)]
	gw := copyNonZeroGateway(&base.CollectorGatewayConfig, gwOverride)
	node := copyNonZeroNode(base.CollectorNodeConfig, nodeOverride)
	return ResourceSizePreset{
		CollectorGatewayConfig: *gw,
		CollectorNodeConfig:    node,
	}
}

var validSizings = map[Sizing]struct{}{
	SizeSmall:  {},
	SizeMedium: {},
	SizeLarge:  {},
}

func IsValidSizing(s string) bool {
	_, ok := validSizings[Sizing(s)]
	return ok
}

// ComputeEffectiveCollectorConfig computes the effective collector configuration by merging
// the sizing preset with the existing configuration, preserving all non-sizing attributes.
// This replaces the pattern of overriding configurations and manually preserving specific fields.
func ComputeEffectiveCollectorConfig(c *common.OdigosConfiguration) (
	*common.CollectorGatewayConfiguration,
	*common.CollectorNodeConfiguration,
) {
	// Get the sizing preset configuration
	effectiveSizing := ComputeResourceSizePreset(c)

	// Merge gateway configuration: preserve existing config, update only sizing fields
	effectiveGateway := mergeGatewayConfiguration(c.CollectorGateway, &effectiveSizing.CollectorGatewayConfig)

	// Merge node configuration: preserve existing config, update only sizing fields
	effectiveNode := mergeNodeConfiguration(c.CollectorNode, effectiveSizing.CollectorNodeConfig)

	return effectiveGateway, effectiveNode
}

// mergeGatewayConfiguration merges sizing information from SizingPreset into existing gateway configuration
// while preserving non-sizing configuration fields like ServiceGraphDisabled, ClusterMetricsEnabled, etc.
func mergeGatewayConfiguration(existing *common.CollectorGatewayConfiguration,
	sizingPreset *common.CollectorGatewayConfiguration) *common.CollectorGatewayConfiguration {
	if existing == nil {
		return sizingPreset
	}

	// Create a copy of existing config to preserve all non-sizing fields
	merged := *existing

	// Update only sizing-related fields from preset
	merged.MinReplicas = sizingPreset.MinReplicas
	merged.MaxReplicas = sizingPreset.MaxReplicas
	merged.RequestMemoryMiB = sizingPreset.RequestMemoryMiB
	merged.LimitMemoryMiB = sizingPreset.LimitMemoryMiB
	merged.RequestCPUm = sizingPreset.RequestCPUm
	merged.LimitCPUm = sizingPreset.LimitCPUm
	merged.MemoryLimiterLimitMiB = sizingPreset.MemoryLimiterLimitMiB
	merged.MemoryLimiterSpikeLimitMiB = sizingPreset.MemoryLimiterSpikeLimitMiB
	merged.GoMemLimitMib = sizingPreset.GoMemLimitMib

	// All other fields (ServiceGraphDisabled, ClusterMetricsEnabled, HttpsProxyAddress)
	// are preserved from the existing configuration

	return &merged
}

// mergeNodeConfiguration merges sizing information from SizingPreset into existing node configuration
// while preserving non-sizing configuration fields like CollectorOwnMetricsPort, EnableDataCompression, etc.
func mergeNodeConfiguration(existing *common.CollectorNodeConfiguration,
	sizingPreset common.CollectorNodeConfiguration) *common.CollectorNodeConfiguration {
	if existing == nil {
		return &sizingPreset
	}

	// Create a copy of existing config to preserve all non-sizing fields
	merged := *existing

	// Update only sizing-related fields from preset
	merged.RequestMemoryMiB = sizingPreset.RequestMemoryMiB
	merged.LimitMemoryMiB = sizingPreset.LimitMemoryMiB
	merged.RequestCPUm = sizingPreset.RequestCPUm
	merged.LimitCPUm = sizingPreset.LimitCPUm
	merged.MemoryLimiterLimitMiB = sizingPreset.MemoryLimiterLimitMiB
	merged.MemoryLimiterSpikeLimitMiB = sizingPreset.MemoryLimiterSpikeLimitMiB
	merged.GoMemLimitMib = sizingPreset.GoMemLimitMib

	// All other fields (CollectorOwnMetricsPort, EnableDataCompression)
	// are preserved from the existing configuration

	return &merged
}

// copyNonZeroGateway overlays only non-zero numeric fields from src onto dst.
func copyNonZeroGateway(dst *common.CollectorGatewayConfiguration, src *common.CollectorGatewayConfiguration) *common.CollectorGatewayConfiguration {
	if src == nil {
		return dst
	}

	// Replicas
	if src.MinReplicas != 0 {
		dst.MinReplicas = src.MinReplicas
	}
	if src.MaxReplicas != 0 {
		dst.MaxReplicas = src.MaxReplicas
	}

	// Memory (MiB)
	if src.RequestMemoryMiB != 0 {
		dst.RequestMemoryMiB = src.RequestMemoryMiB
	}
	if src.LimitMemoryMiB != 0 {
		dst.LimitMemoryMiB = src.LimitMemoryMiB
	}

	// CPU (m)
	if src.RequestCPUm != 0 {
		dst.RequestCPUm = src.RequestCPUm
	}
	if src.LimitCPUm != 0 {
		dst.LimitCPUm = src.LimitCPUm
	}

	if src.MemoryLimiterLimitMiB != 0 {
		dst.MemoryLimiterLimitMiB = src.MemoryLimiterLimitMiB
	}
	if src.MemoryLimiterSpikeLimitMiB != 0 {
		dst.MemoryLimiterSpikeLimitMiB = src.MemoryLimiterSpikeLimitMiB
	}
	if src.GoMemLimitMib != 0 {
		dst.GoMemLimitMib = src.GoMemLimitMib
	}
	return dst
}

// copyNonZeroNode overlays only non-zero numeric fields from src onto dst.
func copyNonZeroNode(dst common.CollectorNodeConfiguration, src *common.CollectorNodeConfiguration) common.CollectorNodeConfiguration {
	if src == nil {
		return dst
	}

	// Memory (MiB)
	if src.RequestMemoryMiB != 0 {
		dst.RequestMemoryMiB = src.RequestMemoryMiB
	}
	if src.LimitMemoryMiB != 0 {
		dst.LimitMemoryMiB = src.LimitMemoryMiB
	}

	// CPU (m)
	if src.RequestCPUm != 0 {
		dst.RequestCPUm = src.RequestCPUm
	}
	if src.LimitCPUm != 0 {
		dst.LimitCPUm = src.LimitCPUm
	}

	// Optional: memory-limiter trio (only if your struct has these fields)
	if src.MemoryLimiterLimitMiB != 0 {
		dst.MemoryLimiterLimitMiB = src.MemoryLimiterLimitMiB
	}
	if src.MemoryLimiterSpikeLimitMiB != 0 {
		dst.MemoryLimiterSpikeLimitMiB = src.MemoryLimiterSpikeLimitMiB
	}
	if src.GoMemLimitMib != 0 {
		dst.GoMemLimitMib = src.GoMemLimitMib
	}

	return dst
}
