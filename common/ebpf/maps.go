package ebpf

import (
	"fmt"
	"os"

	cilumebpf "github.com/cilium/ebpf"
	"github.com/cilium/ebpf/features"
)

func CreateTracesMap() (*cilumebpf.Map, error) {
	spec := &cilumebpf.MapSpec{
		Name: "traces",
	}

	// Check if the current kernel supports the ring buffer
	if features.HaveMapType(cilumebpf.RingBuf) == nil {
		spec.Type = cilumebpf.RingBuf
		// Set MaxEntries for ring buffer: MaxEntries = NumOfPages * os.Getpagesize()
		spec.MaxEntries = uint32(NumOfPages * os.Getpagesize())
	} else {
		// If not, default to the old buffer
		spec.Type = cilumebpf.PerfEventArray
	}
	m, err := cilumebpf.NewMap(spec)
	if err != nil {
		return nil, fmt.Errorf("create traces eBPF map: %w", err)
	}
	return m, nil
}

func CreateMetricsMaps() (*cilumebpf.Map, *cilumebpf.Map, error) {
	// Create the metrics eBPF map - always HashOfMaps type
	// The key for the hash of maps is a unique identifier for java process
	// The value for the hash of maps is a pointer to a metrics map
	metricsSpec := &cilumebpf.MapSpec{
		Type:       cilumebpf.HashOfMaps,
		Name:       "metrics",
		KeySize:    ProcessKeySize,
		ValueSize:  InnerMapIDSize,
		MaxEntries: MaxProcessesCount,
		// InnerMap spec should be the same as the ones created in the instrumentations.
		InnerMap: &cilumebpf.MapSpec{
			Name:       "jvm_metrics_inner_map",
			Type:       cilumebpf.Hash,
			KeySize:    MetricKeySize,
			ValueSize:  MetricValueSize,
			MaxEntries: MaxMetricsPerMap,
		},
	}

	metricsMap, err := cilumebpf.NewMap(metricsSpec)
	if err != nil {
		return nil, nil, err
	}

	// Create the metrics attributes eBPF map - simple Hash map for UUID -> packed resource attributes.
	// This map stores resource attributes separately from the HashOfMaps key, allowing attributes
	// to exceed the eBPF key size limit.
	attributesSpec := &cilumebpf.MapSpec{
		Type:       cilumebpf.Hash,
		Name:       "metrics_attributes",
		KeySize:    ProcessKeySize,
		ValueSize:  AttributesValueSize,
		MaxEntries: MaxProcessesCount,
	}

	metricsAttributesMap, err := cilumebpf.NewMap(attributesSpec)
	if err != nil {
		_ = metricsMap.Close()
		return nil, nil, fmt.Errorf("failed to create metrics attributes eBPF map: %w", err)
	}

	return metricsMap, metricsAttributesMap, nil
}

// CreateLogsMaps creates the logs RingBuf eBPF map and the logs ext (attributes)
// Hash map used to pass per-process resource attributes alongside log events.
// Returns (nil, nil, nil) when the kernel does not support RingBuf, since there
// is no PerfEventArray fallback for logs.
func CreateLogsMaps() (*cilumebpf.Map, *cilumebpf.Map, error) {
	if err := features.HaveMapType(cilumebpf.RingBuf); err != nil {
		return nil, nil, nil //nolint:nilerr // graceful: no RingBuf support, no fallback for logs
	}

	logsSpec := &cilumebpf.MapSpec{
		Type:       cilumebpf.RingBuf,
		Name:       "logs",
		MaxEntries: LogsMaxEntries,
	}

	logsMap, err := cilumebpf.NewMap(logsSpec)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create logs eBPF map: %w", err)
	}

	logsExtSpec := &cilumebpf.MapSpec{
		Type:       cilumebpf.Hash,
		Name:       "logs_ext",
		KeySize:    LogsExtKeySize,
		ValueSize:  AttributesValueSize,
		MaxEntries: MaxProcessesCount,
	}

	logsExtMap, err := cilumebpf.NewMap(logsExtSpec)
	if err != nil {
		_ = logsMap.Close()
		return nil, nil, fmt.Errorf("failed to create logs ext eBPF map: %w", err)
	}

	return logsMap, logsExtMap, nil
}
