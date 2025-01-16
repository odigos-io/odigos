package odigospro

import (
	odigospredicates "github.com/odigos-io/odigos/k8sutils/pkg/predicate"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func SetupWithManager(mgr ctrl.Manager) error {

	err := ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Secret{}).
		Named("odigos-pro").
		WithEventFilter(&odigospredicates.OdigosProSecretPredicate).
		Complete(&odigosConfigController{
			Client: mgr.GetClient(),
		})
	if err != nil {
		return err
	}

	return nil
}
