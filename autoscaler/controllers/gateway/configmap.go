package gateway

import (
	"context"
	"reflect"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/autoscaler/controllers/common"
	odigoscommon "github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	odgiosK8s "github.com/odigos-io/odigos/k8sutils/pkg/conditions"
	"github.com/odigos-io/odigos/k8sutils/pkg/consts"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	destinationConfiguredType = "DestinationConfigured"
)

func syncConfigMap(dests *odigosv1.DestinationList, allProcessors *odigosv1.ProcessorList, gateway *odigosv1.CollectorsGroup, ctx context.Context, c client.Client, scheme *runtime.Scheme, memConfig *memoryConfigurations) (string, []odigoscommon.ObservabilitySignal, error) {
	logger := log.FromContext(ctx)

	memoryLimiterConfiguration := config.GenericMap{
		"check_interval":  "1s",
		"limit_mib":       memConfig.memoryLimiterLimitMiB,
		"spike_limit_mib": memConfig.memoryLimiterSpikeLimitMiB,
	}

	processors := common.FilterAndSortProcessorsByOrderHint(allProcessors, odigosv1.CollectorsGroupRoleClusterGateway)

	desiredData, err, status, signals := config.Calculate(
		common.ToExporterConfigurerArray(dests),
		common.ToProcessorConfigurerArray(processors),
		memoryLimiterConfiguration,
	)
	if err != nil {
		logger.Error(err, "Failed to calculate config")
		return "", nil, err
	}

	for destName, destErr := range status.Destination {
		if destErr != nil {
			logger.Error(destErr, "Failed to calculate config for destination", "destination", destName)
		}
	}
	for name, err := range status.Processor {
		if err != nil {
			logger.Info(err.Error(), "processor", name)
		}
	}

	// Update destination status conditions in k8s
	for _, dest := range dests.Items {
		if destErr, found := status.Destination[dest.ObjectMeta.Name]; found {
			if destErr != nil {
				err := odgiosK8s.UpdateStatusConditions(ctx, c, &dest, &dest.Status.Conditions, metav1.ConditionFalse, destinationConfiguredType, "ErrConfigDestination", destErr.Error())
				if err != nil {
					logger.Error(err, "Failed to update destination error status conditions")
				}
			} else {
				err := odgiosK8s.UpdateStatusConditions(ctx, c, &dest, &dest.Status.Conditions, metav1.ConditionTrue, destinationConfiguredType, "TransformedToOtelcolConfig", "destination successfully transformed to otelcol configuration")
				if err != nil {
					logger.Error(err, "Failed to update destination success status conditions")
				}
			}
		}
	}

	desiredCM := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.OdigosClusterCollectorConfigMapName,
			Namespace: gateway.Namespace,
		},
		Data: map[string]string{
			consts.OdigosClusterCollectorConfigMapKey: desiredData,
		},
	}

	if err := ctrl.SetControllerReference(gateway, desiredCM, scheme); err != nil {
		logger.Error(err, "Failed to set controller reference")
		return "", nil, err
	}

	existing := &v1.ConfigMap{}
	if err := c.Get(ctx, client.ObjectKey{Namespace: gateway.Namespace, Name: consts.OdigosClusterCollectorConfigMapName}, existing); err != nil {
		if apierrors.IsNotFound(err) {
			logger.V(0).Info("Creating gateway config map")
			_, err := createConfigMap(desiredCM, ctx, c)
			if err != nil {
				logger.Error(err, "Failed to create gateway config map")
				return "", nil, err
			}
			return desiredData, signals, nil
		} else {
			logger.Error(err, "Failed to get gateway config map")
			return "", nil, err
		}
	}

	logger.V(0).Info("Patching gateway config map")
	_, err = patchConfigMap(existing, desiredCM, ctx, c)
	if err != nil {
		logger.Error(err, "Failed to patch gateway config map")
		return "", nil, err
	}

	return desiredData, signals, nil
}

func createConfigMap(desired *v1.ConfigMap, ctx context.Context, c client.Client) (*v1.ConfigMap, error) {
	if err := c.Create(ctx, desired); err != nil {
		return nil, err
	}

	return desired, nil
}

func patchConfigMap(existing *v1.ConfigMap, desired *v1.ConfigMap, ctx context.Context, c client.Client) (*v1.ConfigMap, error) {
	if reflect.DeepEqual(existing.Data, desired.Data) &&
		reflect.DeepEqual(existing.ObjectMeta.OwnerReferences, desired.ObjectMeta.OwnerReferences) {
		log.FromContext(ctx).V(0).Info("Gateway config maps already match")
		return existing, nil
	}
	updated := existing.DeepCopy()
	updated.Data = desired.Data
	updated.ObjectMeta.OwnerReferences = desired.ObjectMeta.OwnerReferences
	patch := client.MergeFrom(existing)
	if err := c.Patch(ctx, updated, patch); err != nil {
		return nil, err
	}

	return updated, nil
}
