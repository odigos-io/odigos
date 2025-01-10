package odigosconfig

import (
	odigospredicates "github.com/odigos-io/odigos/k8sutils/pkg/predicate"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func SetupWithManager(mgr ctrl.Manager) error {

	err := ctrl.NewControllerManagedBy(mgr).
		For(&corev1.ConfigMap{}).
		Named("odigosconfig-odigosconfig").
		WithEventFilter(&odigospredicates.OdigosConfigMapPredicate).
		Complete(&odigosConfigController{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	return nil
}
