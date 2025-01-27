package odigospro

import (
	odigospredicates "github.com/odigos-io/odigos/k8sutils/pkg/predicate"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

func SetupWithManager(mgr ctrl.Manager) error {

	err := ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Secret{}).
		Named("odigospro-odigospro").
		WithEventFilter(&odigospredicates.OdigosProSecretPredicate).
		Complete(&odigossecretController{
			Client: mgr.GetClient(),
		})
	if err != nil {
		return err
	}

	// it is possbile that the secret was deleted when the controller was down.
	// we want to sync the odigos deployment config map with the secret on startup to reconcile any deleted pro info.
	err = ctrl.NewControllerManagedBy(mgr).
		For(&corev1.ConfigMap{}).
		Named("odigospro-odigosdeployment").
		WithEventFilter(predicate.And(&odigospredicates.OdigosDeploymentConfigMapPredicate, &odigospredicates.CreationPredicate{})).
		Complete(&odigossecretController{
			Client: mgr.GetClient(),
		})
	if err != nil {
		return err
	}

	return nil
}
