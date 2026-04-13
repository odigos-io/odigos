package ebpf

import (
	"errors"
	"fmt"
	"strings"

	cilumebpf "github.com/cilium/ebpf"
	"github.com/cilium/ebpf/rlimit"

	"github.com/odigos-io/odigos/api/k8sconsts"
	ebpfcommon "github.com/odigos-io/odigos/common/ebpf"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/distros"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/odigos-io/odigos/instrumentation"
	"github.com/odigos-io/odigos/odiglet/pkg/detector"

	processdetector "github.com/odigos-io/runtime-detector"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type InstrumentationManagerOptions struct {
	Factories                  map[string]instrumentation.Factory
	DistributionGetter         *distros.Getter
	OdigletHealthProbeBindPort int
	// OnLogsMapCreated is an optional callback invoked after the logs eBPF map is created.
	// It allows callers (e.g. enterprise odiglet) to receive the map for use with
	// external reader mode in the log capture BPF programs.
	OnLogsMapCreated func(*cilumebpf.Map)
	// LogsAttrSubscribe streams per-process resource attributes to the collector.
	LogsAttrSubscribe func() (updates <-chan string, snapshot []string)
}

// NewManager creates a new instrumentation manager for eBPF which is configured to work with Kubernetes.
// Instrumentation factories must be provided in order to create the instrumentation objects.
// Detector options can be provided to configure the process detector, but if not provided, default options will be used.
// logger is optional; when provided it is used by the instrumentation manager for logging.
func NewManager(
	client client.Client,
	logger *commonlogger.OdigosLogger,
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

	tracesMap, err := ebpfcommon.CreateTracesMap()
	if err != nil {
		return nil, fmt.Errorf("failed to create traces map: %w", err)
	}

	metricsMap, metricsAttributesMap, err := ebpfcommon.CreateMetricsMaps()
	if err != nil {
		tracesMap.Close()
		return nil, fmt.Errorf("failed to create metrics attributes eBPF maps: %w", err)
	}

	logsMap, err := ebpfcommon.CreateLogsMap()
	if err != nil {
		tracesMap.Close()
		metricsMap.Close()
		metricsAttributesMap.Close()
		return nil, fmt.Errorf("failed to create logs eBPF map: %w", err)
	}

	if logsMap != nil && opts.OnLogsMapCreated != nil {
		opts.OnLogsMapCreated(logsMap)
	}

	managerOpts := instrumentation.ManagerOptions[K8sProcessGroup, K8sConfigGroup, *K8sProcessDetails]{

		Logger:                  logger,
		Factories:               opts.Factories,
		Handler:                 newHandler(client, opts.DistributionGetter),
		DetectorOptions:         detector.DefaultK8sDetectorOptions(appendEnvVarSlice),
		ConfigUpdates:           configUpdates,
		InstrumentationRequests: instrumentationRequests,
		TracesMap:               tracesMap,
		MetricsMap:              metricsMap,
		MetricsAttributesMap:    metricsAttributesMap,
		LogsMap:                 logsMap,
		LogsAttrSubscribe:      opts.LogsAttrSubscribe,
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
		commonlogger.LoggerCompat().With("subsystem", "ebpfcommon").Info("Added file open triggers to the detector", "triggers", fileOpenTriggers)
	}

	manager, err := instrumentation.NewManager(managerOpts)
	if err != nil {
		return nil, err
	}

	return manager, nil
}

func newHandler(client client.Client, distributionGetter *distros.Getter) *instrumentation.Handler[K8sProcessGroup, K8sConfigGroup, *K8sProcessDetails] {
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
