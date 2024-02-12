package gateway

import (
	"context"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	kubeObjectName = "odigos-gateway"
	collectorLabel = "odigos.io/collector"
)

var (
	commonLabels = map[string]string{
		collectorLabel: "true",
	}
)

func Sync(ctx context.Context, client client.Client, scheme *runtime.Scheme, imagePullSecrets []string) error {
	logger := log.FromContext(ctx)
	var collectorGroups odigosv1.CollectorsGroupList
	if err := client.List(ctx, &collectorGroups); err != nil {
		logger.Error(err, "failed to list collectors groups")
		return err
	}

	var gatewayCollectorGroup *odigosv1.CollectorsGroup
	for _, collectorGroup := range collectorGroups.Items {
		if collectorGroup.Spec.Role == odigosv1.CollectorsGroupRoleGateway {
			gatewayCollectorGroup = &collectorGroup
			break
		}
	}

	if gatewayCollectorGroup == nil {
		logger.V(3).Info("gateway collector group doesn't exist, nothing to sync")
		return nil
	}

	var dests odigosv1.DestinationList
	if err := client.List(ctx, &dests); err != nil {
		logger.Error(err, "failed to list destinations")
		return err
	}

	var processors odigosv1.ProcessorList
	if err := client.List(ctx, &processors); err != nil {
		logger.Error(err, "failed to list processors")
		return err
	}

	return syncGateway(&dests, &processors, gatewayCollectorGroup, ctx, client, scheme, imagePullSecrets)
}

func syncGateway(dests *odigosv1.DestinationList, processors *odigosv1.ProcessorList,
	gateway *odigosv1.CollectorsGroup, ctx context.Context,
	c client.Client, scheme *runtime.Scheme, imagePullSecrets []string) error {
	logger := log.FromContext(ctx)
	logger.V(0).Info("syncing gateway")

	configData, err := syncConfigMap(dests, processors, gateway, ctx, c, scheme)
	if err != nil {
		logger.Error(err, "failed to sync config map")
		return err
	}

	_, err = syncService(gateway, ctx, c, scheme)
	if err != nil {
		logger.Error(err, "failed to sync service")
		return err
	}

	dep, err := syncDeployment(dests, gateway, configData, ctx, c, scheme, imagePullSecrets)
	if err != nil {
		logger.Error(err, "failed to sync deployment")
		return err
	}

	gateway.Status.Ready = dep.Status.ReadyReplicas > 0
	return c.Status().Update(ctx, gateway)
}
