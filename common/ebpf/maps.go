package ebpf

import (
	"fmt"
	"os"

	cilumebpf "github.com/cilium/ebpf"
)

func CreateTracesMap(isRingBufferSupported bool) (*cilumebpf.Map, error) {
	mapType := cilumebpf.PerfEventArray
	spec := &cilumebpf.MapSpec{
		Type: mapType,
		Name: "traces",
	}

	if isRingBufferSupported {
		mapType = cilumebpf.RingBuf
		spec.Type = mapType
		// Set MaxEntries for ring buffer: MaxEntries = NumOfPages * os.Getpagesize()
		spec.MaxEntries = uint32(NumOfPages * os.Getpagesize())
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
		metricsMap.Close()
		return nil, nil, fmt.Errorf("failed to create metrics attributes eBPF map: %w", err)
	}

	return metricsMap, metricsAttributesMap, nil
}
