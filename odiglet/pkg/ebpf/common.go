package ebpf

import (
	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/common/instrumentation"
	"github.com/odigos-io/odigos/odiglet/pkg/detector"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewManager creates a new instrumentation manager for eBPF which is configured to work with Kubernetes.
func NewManager(client client.Client, logger logr.Logger, factories map[instrumentation.OtelDistribution]instrumentation.Factory, configUpdates <-chan instrumentation.ConfigUpdate[K8sConfigGroup]) (instrumentation.Manager, error) {
	managerOpts := instrumentation.ManagerOptions[K8sDetails, K8sConfigGroup]{
		Logger:          logger,
		Factories:       factories,
		Handler:         newHandler(client),
		DetectorOptions: detector.K8sDetectorOptions(logger),
		ConfigUpdates:   configUpdates,
	}

	manager, err := instrumentation.NewManager(managerOpts)
	if err != nil {
		return nil, err
	}

	return manager, nil
}

func newHandler(client client.Client) *instrumentation.Handler[K8sDetails, K8sConfigGroup] {
	reporter := &k8sReporter{
		client: client,
	}
	detailsResolver := &k8sDetailsResolver{
		client: client,
	}
	configGroupResolver := &k8sConfigGroupResolver{}
	settingsGetter := &k8sSettingsGetter{
		client: client,
	}
	distributionMatcher := &podDeviceDistributionMatcher{}
	return &instrumentation.Handler[K8sDetails, K8sConfigGroup]{
		DetailsResolver:     detailsResolver,
		ConfigGroupResolver: configGroupResolver,
		Reporter:            reporter,
		DistributionMatcher: distributionMatcher,
		SettingsGetter:      settingsGetter,
	}
}
