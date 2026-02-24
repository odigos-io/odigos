package clustercollectorsgroup

import (
	"context"
	"slices"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/scheduler/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func getOwnMetricsConfig(odigosConfiguration *common.OdigosConfiguration, allDestinations *odigosv1.DestinationList) *odigosv1.CollectorsGroupMetricsCollectionSettings {
	ownMetricsInterval := "10s"
	if odigosConfiguration.MetricsSources != nil &&
		odigosConfiguration.MetricsSources.OdigosOwnMetrics != nil &&
		odigosConfiguration.MetricsSources.OdigosOwnMetrics.Interval != "" {
		ownMetricsInterval = odigosConfiguration.MetricsSources.OdigosOwnMetrics.Interval
	}

	ownMetricsLocalStorageEnabled := false
	if odigosConfiguration.OdigosOwnTelemetryStore == nil &&
		odigosConfiguration.OdigosOwnTelemetryStore.MetricsStoreDisabled == nil &&
		!*odigosConfiguration.OdigosOwnTelemetryStore.MetricsStoreDisabled {
		ownMetricsLocalStorageEnabled = true
	}

	sendToMetricsDestinations := false
	for _, destination := range allDestinations.Items {
		if destination.Spec.Disabled != nil && *destination.Spec.Disabled {
			continue
		}
		if !slices.Contains(destination.Spec.Signals, common.MetricsObservabilitySignal) {
			continue
		}
		if destination.Spec.MetricsSettings != nil &&
			destination.Spec.MetricsSettings.CollectOdigosOwnMetrics != nil &&
			*destination.Spec.MetricsSettings.CollectOdigosOwnMetrics {
			sendToMetricsDestinations = true
			break
		}
	}

	if !ownMetricsLocalStorageEnabled && !sendToMetricsDestinations {
		return nil
	}

	return &odigosv1.CollectorsGroupMetricsCollectionSettings{
		OdigosOwnMetrics: &odigosv1.OdigosOwnMetricsSettings{
			SendToOdigosMetricsStore:  ownMetricsLocalStorageEnabled,
			SendToMetricsDestinations: sendToMetricsDestinations,
			Interval:                  ownMetricsInterval,
		},
	}
}

func newClusterCollectorGroup(namespace string, resourcesSettings *odigosv1.CollectorsGroupResourcesSettings, serviceGraphDisabled *bool, clusterMetricsEnabled *bool,
	httpsProxyAddress *string, nodeSelector *map[string]string, deploymentName string, metricsConfig *odigosv1.CollectorsGroupMetricsCollectionSettings, tailSampling *common.TailSamplingConfiguration) *odigosv1.CollectorsGroup {
	return &odigosv1.CollectorsGroup{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CollectorsGroup",
			APIVersion: "odigos.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigosClusterCollectorCollectorGroupName,
			Namespace: namespace,
		},
		Spec: odigosv1.CollectorsGroupSpec{
			Role:                    odigosv1.CollectorsGroupRoleClusterGateway,
			CollectorOwnMetricsPort: k8sconsts.OdigosClusterCollectorOwnTelemetryPortDefault,
			ResourcesSettings:       *resourcesSettings,
			ServiceGraphDisabled:    serviceGraphDisabled,
			ClusterMetricsEnabled:   clusterMetricsEnabled,
			HttpsProxyAddress:       httpsProxyAddress,
			NodeSelector:            nodeSelector,
			DeploymentName:          deploymentName,
			Metrics:                 metricsConfig,
			TailSampling:            tailSampling,
		},
	}
}

func sync(ctx context.Context, c client.Client, scheme *runtime.Scheme) error {

	namespace := env.GetCurrentNamespace()

	odigosConfiguration, err := k8sutils.GetCurrentOdigosConfiguration(ctx, c)
	if err != nil {
		return err
	}
	resourceSettings := getGatewayResourceSettings(&odigosConfiguration)

	// default servicegraph is enabled (serviceGraphDisabled to false)
	serviceGraphDisabled := odigosConfiguration.CollectorGateway.ServiceGraphDisabled
	if serviceGraphDisabled == nil {
		result := false
		serviceGraphDisabled = &result
	}

	// default cluster metrics is disabled (clusterMetricsEnabled to false)
	clusterMetricsEnabled := odigosConfiguration.CollectorGateway.ClusterMetricsEnabled
	if clusterMetricsEnabled == nil {
		result := false
		clusterMetricsEnabled = &result
	}

	nodeSelector := odigosConfiguration.CollectorGateway.NodeSelector
	deploymentName := odigosConfiguration.CollectorGateway.DeploymentName
	allDestinations := &odigosv1.DestinationList{}
	if err := c.List(ctx, allDestinations); err != nil {
		return err
	}

	ownMetricsConfig := getOwnMetricsConfig(&odigosConfiguration, allDestinations)

	var tailSampling *common.TailSamplingConfiguration
	if odigosConfiguration.Sampling != nil {
		tailSampling = odigosConfiguration.Sampling.TailSampling
	}

	// cluster collector is always set and never deleted at the moment.
	// this is to accelerate spinup time and avoid errors while things are gradually being reconciled
	// and started.
	// in the future we might want to support a deployment of instrumentations only and allow user
	// to setup their own collectors, then we would avoid adding the cluster collector by default.
	clusterCollectorGroup := newClusterCollectorGroup(namespace, resourceSettings, serviceGraphDisabled, clusterMetricsEnabled, odigosConfiguration.CollectorGateway.HttpsProxyAddress, nodeSelector, deploymentName, ownMetricsConfig, tailSampling)
	err = utils.SetOwnerControllerToSchedulerDeployment(ctx, c, clusterCollectorGroup, scheme)
	if err != nil {
		return err
	}

	err = k8sutils.ApplyCollectorGroup(ctx, c, clusterCollectorGroup)
	if err != nil {
		return err
	}

	return nil
}
