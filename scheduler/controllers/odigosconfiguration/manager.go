package odigosconfiguration

import (
	"github.com/odigos-io/odigos/common"
	odigospredicates "github.com/odigos-io/odigos/k8sutils/pkg/predicate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

func SetupWithManager(mgr ctrl.Manager, tier common.OdigosTier, odigosVersion string, dynamicClient *dynamic.DynamicClient) error {

	err := ctrl.NewControllerManagedBy(mgr).
		For(&corev1.ConfigMap{}).
		Named("odigosconfiguration-odigosconfiguration").
		WithEventFilter(predicate.Or(
			&odigospredicates.OdigosConfigMapPredicate,
			&odigospredicates.OdigosRemoteConfigMapPredicate,
			&odigospredicates.OdigosLocalUiConfigMapPredicate)).
		Complete(&odigosConfigurationController{
			Client:        mgr.GetClient(),
			Scheme:        mgr.GetScheme(),
			Tier:          tier,
			OdigosVersion: odigosVersion,
			DynamicClient: dynamicClient,
		})
	if err != nil {
		return err
	}

	err = ctrl.NewControllerManagedBy(mgr).
		For(&corev1.ConfigMap{}).
		Named("odigosconfiguration-odigosdeployment").
		WithEventFilter(&odigospredicates.OdigosDeploymentConfigMapPredicate).
		Complete(&odigosConfigurationController{
			Client:        mgr.GetClient(),
			Scheme:        mgr.GetScheme(),
			Tier:          tier,
			OdigosVersion: odigosVersion,
			DynamicClient: dynamicClient,
		})
	if err != nil {
		return err
	}

	return nil
}
