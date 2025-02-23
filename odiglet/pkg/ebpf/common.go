package ebpf

import (
	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/instrumentation"
	"github.com/odigos-io/odigos/odiglet/pkg/detector"

	processdetector "github.com/odigos-io/runtime-detector"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type InstrumentationManagerOptions struct {
	Factories       map[instrumentation.OtelDistribution]instrumentation.Factory
	DetectorOptions []processdetector.DetectorOption
}

// NewManager creates a new instrumentation manager for eBPF which is configured to work with Kubernetes.
// Instrumentation factories must be provided in order to create the instrumentation objects.
// Detector options can be provided to configure the process detector, but if not provided, default options will be used.
func NewManager(client client.Client, logger logr.Logger, opts InstrumentationManagerOptions, configUpdates <-chan instrumentation.ConfigUpdate[K8sConfigGroup]) (instrumentation.Manager, error) {
	managerOpts := instrumentation.ManagerOptions[K8sProcessDetails, K8sConfigGroup]{
		Logger:        logger,
		Factories:     opts.Factories,
		Handler:       newHandler(client),
		ConfigUpdates: configUpdates,
	}

	if opts.DetectorOptions != nil {
		managerOpts.DetectorOptions = opts.DetectorOptions
	} else {
		managerOpts.DetectorOptions = detector.DefaultK8sDetectorOptions(logger)
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
