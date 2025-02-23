package ebpf

import (
	"errors"
	"strings"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/distros"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/odigos-io/odigos/instrumentation"
	"github.com/odigos-io/odigos/odiglet/pkg/detector"

	processdetector "github.com/odigos-io/runtime-detector"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type InstrumentationManagerOptions struct {
	Factories          map[instrumentation.OtelDistribution]instrumentation.Factory
	DistributionGetter *distros.Getter
}

// NewManager creates a new instrumentation manager for eBPF which is configured to work with Kubernetes.
// Instrumentation factories must be provided in order to create the instrumentation objects.
// Detector options can be provided to configure the process detector, but if not provided, default options will be used.
func NewManager(client client.Client, logger logr.Logger, opts InstrumentationManagerOptions, configUpdates <-chan instrumentation.ConfigUpdate[K8sConfigGroup]) (instrumentation.Manager, error) {
	if len(opts.Factories) == 0 {
		return nil, errors.New("instrumentation factories must be provided")
	}

	if opts.DistributionGetter == nil {
		return nil, errors.New("distribution getter must be provided")
	}

	managerOpts := instrumentation.ManagerOptions[K8sProcessDetails, K8sConfigGroup]{
		Logger:          logger,
		Factories:       opts.Factories,
		Handler:         newHandler(client),
		DetectorOptions: detector.DefaultK8sDetectorOptions(logger),
		ConfigUpdates:   configUpdates,
	}

	// Add file open triggers from all distributions.
	// This is required to avoid race conditions in which we would attempt to instrument a process
	// before it load the required native library (e.g. .so file)
	// adding this option to the process detector will add an event to the instrumentation event loop
	fileOpenTriggers := []string{}
	for _, d := range(opts.DistributionGetter.GetAllDistros()) {
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

func newHandler(client client.Client) *instrumentation.Handler[K8sProcessDetails, K8sConfigGroup] {
	reporter := &k8sReporter{
		client: client,
	}
	processDetailsResolver := &k8sDetailsResolver{
		client: client,
	}
	configGroupResolver := &k8sConfigGroupResolver{}
	settingsGetter := &k8sSettingsGetter{
		client: client,
	}
	distributionMatcher := &podDeviceDistributionMatcher{}
	return &instrumentation.Handler[K8sProcessDetails, K8sConfigGroup]{
		ProcessDetailsResolver: processDetailsResolver,
		ConfigGroupResolver:    configGroupResolver,
		Reporter:               reporter,
		DistributionMatcher:    distributionMatcher,
		SettingsGetter:         settingsGetter,
	}
}
