package gateway

import (
	"context"
	odigosv1 "github.com/keyval-dev/odigos/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	kubeObjectName = "odigos-gateway"
)

func Sync(ctx context.Context, client client.Client, scheme *runtime.Scheme) error {
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
		logger.V(3).Info("gateway collector group not exists, nothing to sync")
		return nil
	}

	var dests odigosv1.DestinationList
	if err := client.List(ctx, &dests); err != nil {
		logger.Error(err, "failed to list destinations")
		return err
	}

	return syncGateway(&dests, gatewayCollectorGroup, ctx, client, scheme)
}

func syncGateway(dests *odigosv1.DestinationList, gateway *odigosv1.CollectorsGroup, ctx context.Context, c client.Client, scheme *runtime.Scheme) error {
	logger := log.FromContext(ctx)
	_, err := syncConfigMap(dests, gateway, ctx, c, scheme)
	if err != nil {
		logger.Error(err, "failed to sync config map")
		return err
	}

	gateway.Status.Ready = true
	return c.Status().Update(ctx, gateway)
}
