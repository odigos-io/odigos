package ebpf

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/features"
	"github.com/cilium/ebpf/rlimit"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/distros"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/odigos-io/odigos/instrumentation"
	"github.com/odigos-io/odigos/odiglet/pkg/detector"

	cilumebpf "github.com/cilium/ebpf"
	processdetector "github.com/odigos-io/runtime-detector"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	numOfPages = 2048

	// JVM metrics eBPF map sizing constants.
	// Uses a hash-of-maps architecture: one outer HashOfMaps keyed by UUID containing
	// per-process inner maps, plus a separate attributes map for resource attributes.
	ProcessKeySize      = 64   // Size of UUID key
	InnerMapIDSize      = 4    // Size of inner map ID (should be 4 bytes hard coded)
	MaxProcessesCount   = 512  // Max number of processes that can have metrics
	AttributesValueSize = 1024 // Size of packed resource attributes value buffer

	// Inner Map configuration
	MetricKeySize    = 4   // uint32 metric_key
	MetricValueSize  = 40  // struct metric_value (40 bytes - size of largest union member: histogram_value)
	MaxMetricsPerMap = 256 // MAX_METRICS per process
)

type InstrumentationManagerOptions struct {
	Factories                  map[string]instrumentation.Factory
	DistributionGetter         *distros.Getter
	OdigletHealthProbeBindPort int
}

// NewManager creates a new instrumentation manager for eBPF which is configured to work with Kubernetes.
// Instrumentation factories must be provided in order to create the instrumentation objects.
// Detector options can be provided to configure the process detector, but if not provided, default options will be used.
func NewManager(
	client client.Client,
	logger logr.Logger,
	opts InstrumentationManagerOptions,
	configUpdates <-chan instrumentation.ConfigUpdate[K8sConfigGroup],
	instrumentationRequests <-chan instrumentation.Request[K8sProcessGroup, K8sConfigGroup, *K8sProcessDetails],
	appendEnvVarNames map[string]struct{},
) (instrumentation.Manager, error) {
	if len(opts.Factories) == 0 {
		return nil, errors.New("instrumentation factories must be provided")
	}

	if opts.DistributionGetter == nil {
		return nil, errors.New("distribution getter must be provided")
	}

	appendEnvVarSlice := make([]string, 0, len(appendEnvVarNames))
	for env := range appendEnvVarNames {
		appendEnvVarSlice = append(appendEnvVarSlice, env)
	}
	appendEnvVarSlice = append(appendEnvVarSlice, k8sconsts.OtelResourceAttributesEnvVar)

	// Create the eBPF maps
	if err := rlimit.RemoveMemlock(); err != nil {
		return nil, fmt.Errorf("failed to remove memlock rlimit: %w", err)
	}

	mapType := cilumebpf.PerfEventArray
	spec := &cilumebpf.MapSpec{
		Type: mapType,
		Name: "traces",
	}

	// Check if the current kernel supports the ring buffer
	ringEn := features.HaveMapType(ebpf.RingBuf) == nil

	if ringEn {
		mapType = cilumebpf.RingBuf
		spec.Type = mapType
		// Set MaxEntries for ring buffer: MaxEntries = numOfPages * os.Getpagesize()
		spec.MaxEntries = uint32(numOfPages * os.Getpagesize())
	}

	tracesMap, err := cilumebpf.NewMap(spec)
	if err != nil {
		return nil, err
	}

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
		InnerMap: &ebpf.MapSpec{
			Name:       "jvm_metrics_inner_map",
			Type:       ebpf.Hash,
			KeySize:    MetricKeySize,
			ValueSize:  MetricValueSize,
			MaxEntries: MaxMetricsPerMap,
		},
	}

	metricsMap, err := cilumebpf.NewMap(metricsSpec)
	if err != nil {
		tracesMap.Close() // Cleanup traces map on error
		return nil, err
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
		tracesMap.Close()
		metricsMap.Close()
		return nil, fmt.Errorf("failed to create metrics attributes eBPF map: %w", err)
	}

	// Create the logs eBPF map - same type as traces (RingBuf or PerfEventArray depending on kernel support)
	logsSpec := &cilumebpf.MapSpec{
		Type: mapType,
		Name: "logs",
	}
	if ringEn {
		logsSpec.MaxEntries = uint32(numOfPages * os.Getpagesize())
	}

	logsMap, err := cilumebpf.NewMap(logsSpec)
	if err != nil {
		tracesMap.Close()
		metricsMap.Close()
		metricsAttributesMap.Close()
		return nil, fmt.Errorf("failed to create logs eBPF map: %w", err)
	}

	managerOpts := instrumentation.ManagerOptions[K8sProcessGroup, K8sConfigGroup, *K8sProcessDetails]{

		Logger:                  logger,
		Factories:               opts.Factories,
		Handler:                 newHandler(logger, client, opts.DistributionGetter),
		DetectorOptions:         detector.DefaultK8sDetectorOptions(logger, appendEnvVarSlice),
		ConfigUpdates:           configUpdates,
		InstrumentationRequests: instrumentationRequests,
		TracesMap:               tracesMap,
		MetricsMap:              metricsMap,
		MetricsAttributesMap:    metricsAttributesMap,
		LogsMap:                 logsMap,
	}

	// Add file open triggers from all distributions.
	// This is required to avoid race conditions in which we would attempt to instrument a process
	// before it load the required native library (e.g. .so file)
	// adding this option to the process detector will add an event to the instrumentation event loop
	fileOpenTriggers := []string{}
	for _, d := range opts.DistributionGetter.GetAllDistros() {
		if d.RuntimeAgent == nil {
			continue
		}
		if d.RuntimeAgent.FileOpenTriggers == nil {
			continue
		}

		// Sanitize the file open triggers
		// TODO: this should not be here but in the distro package - we should have templating resolved in the distro package
		for i, filename := range d.RuntimeAgent.FileOpenTriggers {
			d.RuntimeAgent.FileOpenTriggers[i] = strings.ReplaceAll(filename, distro.AgentPlaceholderDirectory, k8sconsts.OdigosAgentsDirectory)
		}

		fileOpenTriggers = append(fileOpenTriggers, d.RuntimeAgent.FileOpenTriggers...)
	}

	if len(fileOpenTriggers) > 0 {
		managerOpts.DetectorOptions = append(managerOpts.DetectorOptions, processdetector.WithFilesOpenTrigger(fileOpenTriggers...))
		logger.V(0).Info("Added file open triggers to the detector", "triggers", fileOpenTriggers)
	}

	manager, err := instrumentation.NewManager(managerOpts)
	if err != nil {
		return nil, err
	}

	return manager, nil
}

func newHandler(logger logr.Logger, client client.Client, distributionGetter *distros.Getter) *instrumentation.Handler[K8sProcessGroup, K8sConfigGroup, *K8sProcessDetails] {
	reporter := &k8sReporter{
		client: client,
	}
	processDetailsResolver := &k8sDetailsResolver{
		client:             client,
		distributionGetter: distributionGetter,
	}
	settingsGetter := &k8sSettingsGetter{
		client: client,
	}
	return &instrumentation.Handler[K8sProcessGroup, K8sConfigGroup, *K8sProcessDetails]{
		ProcessDetailsResolver: processDetailsResolver,
		Reporter:               reporter,
		SettingsGetter:         settingsGetter,
	}
}
