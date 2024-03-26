package gateway

import (
	"context"
	"reflect"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	"github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/autoscaler/controllers/gateway/config"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	configKey = "collector-conf"
)

func syncConfigMap(dests *odigosv1.DestinationList, processors *odigosv1.ProcessorList, gateway *odigosv1.CollectorsGroup, ctx context.Context, c client.Client, scheme *runtime.Scheme, memConfig *memoryConfigurations) (string, error) {
	logger := log.FromContext(ctx)

	memoryLimiterConfiguration := common.GenericMap{
		"check_interval":  "1s",
		"limit_mib":       memConfig.memoryLimiterLimitMiB,
		"spike_limit_mib": memConfig.memoryLimiterSpikeLimitMiB,
	}

	desiredData, err := config.Calculate(dests, processors, memoryLimiterConfiguration)
	if err != nil {
		logger.Error(err, "Failed to calculate config")
		return "", err
	}

	desired := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      gateway.Name,
			Namespace: gateway.Namespace,
		},
		Data: map[string]string{
			configKey: desiredData,
		},
	}

	if err := ctrl.SetControllerReference(gateway, desired, scheme); err != nil {
		logger.Error(err, "Failed to set controller reference")
		return "", err
	}

	existing := &v1.ConfigMap{}
	if err := c.Get(ctx, client.ObjectKey{Namespace: gateway.Namespace, Name: kubeObjectName}, existing); err != nil {
		if apierrors.IsNotFound(err) {
			logger.V(0).Info("Creating gateway config map")
			_, err := createConfigMap(desired, ctx, c)
			if err != nil {
				logger.Error(err, "Failed to create gateway config map")
				return "", err
			}
			return desiredData, nil
		} else {
			logger.Error(err, "Failed to get gateway config map")
			return "", err
		}
	}

	logger.V(0).Info("Patching gateway config map")
	_, err = patchConfigMap(existing, desired, ctx, c)
	if err != nil {
		logger.Error(err, "Failed to patch gateway config map")
		return "", err
	}

	return desiredData, nil
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
