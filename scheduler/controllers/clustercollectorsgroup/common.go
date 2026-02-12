package clustercollectorsgroup

import (
	"context"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/scheduler/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func newClusterCollectorGroup(namespace string, resourcesSettings *odigosv1.CollectorsGroupResourcesSettings, serviceGraphDisabled *bool, clusterMetricsEnabled *bool,
	httpsProxyAddress *string, nodeSelector *map[string]string, deploymentName string) *odigosv1.CollectorsGroup {
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

	// cluster collector is always set and never deleted at the moment.
	// this is to accelerate spinup time and avoid errors while things are gradually being reconciled
	// and started.
	// in the future we might want to support a deployment of instrumentations only and allow user
	// to setup their own collectors, then we would avoid adding the cluster collector by default.
	clusterCollectorGroup := newClusterCollectorGroup(namespace, resourceSettings, serviceGraphDisabled, clusterMetricsEnabled, odigosConfiguration.CollectorGateway.HttpsProxyAddress, nodeSelector, deploymentName)
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
