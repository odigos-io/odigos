package clustercollectorsgroup

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	odigospredicates "github.com/odigos-io/odigos/k8sutils/pkg/predicate"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func SetupWithManager(mgr ctrl.Manager) error {

	err := ctrl.NewControllerManagedBy(mgr).
		For(&odigosv1.Destination{}).
		Named("clustercollectorgroup-destinations").
		WithEventFilter(&odigospredicates.ExistencePredicate{}).
		Complete(&clusterCollectorsGroupController{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = ctrl.NewControllerManagedBy(mgr).
		For(&corev1.ConfigMap{}).
		Named("clustercollectorgroup-odigosconfig").
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
