package gateway

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	KubeObjectName = "odigos-gateway"
	CollectorLabel = "odigos.io/gateway"
)

var (
	commonLabels = map[string]string{
		CollectorLabel: "true",
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

	odigosSystemNamespaceName := env.GetCurrentNamespace()
	var odigosConfig odigosv1.OdigosConfiguration
	if err := client.Get(ctx, types.NamespacedName{Namespace: odigosSystemNamespaceName, Name: consts.DefaultOdigosConfigurationName}, &odigosConfig); err != nil {
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

	memConfig := GetMemoryConfigurations(odigosConfig)

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
