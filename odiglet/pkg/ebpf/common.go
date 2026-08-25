package ebpf

import (
	"errors"
	"fmt"
	"strings"

	cilumebpf "github.com/cilium/ebpf"
	"github.com/cilium/ebpf/rlimit"

	"github.com/odigos-io/odigos/api/k8sconsts"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/distros"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/odigos-io/odigos/instrumentation"
	"github.com/odigos-io/odigos/odiglet/pkg/detector"
	"github.com/odigos-io/odigos/odiglet/pkg/metrics"

	processdetector "github.com/odigos-io/runtime-detector"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type InstrumentationManagerOptions struct {
	Factories map[string]instrumentation.Factory
	// GenericFactories are factories run for every process regardless of distro, in addition to
	// the factory selected by the process's distribution (e.g. OBI network metrics, eBPF log
	// capture). They are handled off the main path and never report their status.
	// See instrumentation.ManagerOptions.GenericFactories.
	GenericFactories           map[string]instrumentation.Factory
	DistributionGetter         *distros.Getter
	OdigletHealthProbeBindPort int
	// TracesMap, MetricsMap, MetricsAttributesMap and LogsMap are shared eBPF maps owned by the
	// caller and handed to the instrumentation factories, which then run as external readers.
	// Creating them and serving their file descriptors is enterprise-only, so OSS leaves these
	// nil and every factory falls back to the map it creates itself.
	TracesMap            *cilumebpf.Map
	MetricsMap           *cilumebpf.Map
	MetricsAttributesMap *cilumebpf.Map
	LogsMap              *cilumebpf.Map

	// MapMemoryReporter supplies eBPF map memory accounting recorded when
	// each map was loaded, so the metrics collector can report it without
	// rediscovering it from /proc on every scrape. The accounting lives in
	// a module OSS cannot import, so like the shared maps above this is
	// enterprise-only: OSS leaves it nil and the collector falls back to
	// walking /proc, which is the only source it has for maps loaded
	// outside that accounting regardless.
	MapMemoryReporter metrics.MapMemoryReporter
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

	// The instrumentations load eBPF programs of their own, so the rlimit still has to be raised
	// here even though odiglet itself no longer creates any map.
	if err := rlimit.RemoveMemlock(); err != nil {
		return nil, fmt.Errorf("failed to remove memlock rlimit: %w", err)
	}

	managerOpts := instrumentation.ManagerOptions[K8sProcessGroup, K8sConfigGroup, *K8sProcessDetails]{

		Logger:                  logger,
		Factories:               opts.Factories,
		GenericFactories:        opts.GenericFactories,
		Handler:                 newHandler(client, opts.DistributionGetter),
		DetectorOptions:         detector.DefaultK8sDetectorOptions(appendEnvVarSlice),
		ConfigUpdates:           configUpdates,
		InstrumentationRequests: instrumentationRequests,
		TracesMap:               opts.TracesMap,
		MetricsMap:              opts.MetricsMap,
		MetricsAttributesMap:    opts.MetricsAttributesMap,
		LogsMap:                 opts.LogsMap,
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
