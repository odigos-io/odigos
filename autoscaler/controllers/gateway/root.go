package gateway

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	kubeObjectName = "odigos-gateway"
	collectorLabel = "odigos.io/collector"
)

var (
	CommonLabels = map[string]string{
		collectorLabel: "true",
	}
)

func Sync(ctx context.Context, client client.Client, scheme *runtime.Scheme, imagePullSecrets []string, odigosVersion string) error {
	logger := log.FromContext(ctx)
	var collectorGroups odigosv1.CollectorsGroupList
	if err := client.List(ctx, &collectorGroups); err != nil {
		logger.Error(err, "Failed to list collectors groups")
		return err
	}

	var gatewayCollectorGroup *odigosv1.CollectorsGroup
	for _, collectorGroup := range collectorGroups.Items {
		if collectorGroup.Spec.Role == odigosv1.CollectorsGroupRoleClusterGateway {
			gatewayCollectorGroup = &collectorGroup
			break
		}
	}

	if gatewayCollectorGroup == nil {
		logger.V(3).Info("Gateway collector group doesn't exist, nothing to sync")
		return nil
	}

	var dests odigosv1.DestinationList
	if err := client.List(ctx, &dests); err != nil {
		logger.Error(err, "Failed to list destinations")
		return err
	}

	var processors odigosv1.ProcessorList
	if err := client.List(ctx, &processors); err != nil {
		logger.Error(err, "Failed to list processors")
		return err
	}
	// Add the generic batch processor to the list of processors
	processors.Items = append(processors.Items, commonconf.GetGenericBatchProcessor())

	odigosSystemNamespaceName := env.GetCurrentNamespace()
	var odigosConfig odigosv1.OdigosConfiguration
	if err := client.Get(ctx, types.NamespacedName{Namespace: odigosSystemNamespaceName, Name: consts.OdigosConfigurationName}, &odigosConfig); err != nil {
		logger.Error(err, "failed to get odigos config")
		return err
	}

	return syncGateway(&dests, &processors, gatewayCollectorGroup, ctx, client, scheme, imagePullSecrets, odigosVersion, &odigosConfig)
}

func syncGateway(dests *odigosv1.DestinationList, processors *odigosv1.ProcessorList,
	gateway *odigosv1.CollectorsGroup, ctx context.Context,
	c client.Client, scheme *runtime.Scheme, imagePullSecrets []string, odigosVersion string, odigosConfig *odigosv1.OdigosConfiguration) error {
	logger := log.FromContext(ctx)
	logger.V(0).Info("Syncing gateway")

	memConfig := getMemoryConfigurations(odigosConfig)

	configData, err := syncConfigMap(dests, processors, gateway, ctx, c, scheme, memConfig)
	if err != nil {
		logger.Error(err, "Failed to sync config map")
		return err
	}

	err = deletePreviousServices(ctx, c, gateway.Namespace)
	if err != nil {
		logger.Error(err, "Failed to delete previous services")
		return err
	}

	_, err = syncService(gateway, ctx, c, scheme)
	if err != nil {
		logger.Error(err, "Failed to sync service")
		return err
	}

	dep, err := syncDeployment(dests, gateway, configData, ctx, c, scheme, imagePullSecrets, odigosVersion, memConfig)
	if err != nil {
		logger.Error(err, "Failed to sync deployment")
		return err
	}

	if isMetricsServerInstalled(ctx, c) {
		err = syncHPA(gateway, ctx, c, scheme, memConfig)
		if err != nil {
			logger.Error(err, "Failed to sync HPA")
		}
	}

	isReady := dep.Status.ReadyReplicas > 0
	if !gateway.Status.Ready && isReady {
		err := c.Status().Patch(ctx, gateway, client.RawPatch(
			types.MergePatchType,
			[]byte(`{"status": { "ready": true }}`),
		))
		if err != nil {
			return err
		}
	}

	return nil
}

func isMetricsServerInstalled(ctx context.Context, c client.Client) bool {
	// Check if Kubernetes metrics server is installed by checking if the metrics-server deployment exists
	logger := log.FromContext(ctx)
	var metricsServerDeployment appsv1.Deployment
	err := c.Get(ctx, types.NamespacedName{Name: "metrics-server", Namespace: "kube-system"}, &metricsServerDeployment)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			logger.Error(err, "Failed to get metrics-server deployment")
			return false
		}

		logger.V(0).Info("Metrics server not found, skipping HPA creation")
		return false
	}

	logger.V(0).Info("Metrics server found, creating HPA for Gateway")
	return true
}
