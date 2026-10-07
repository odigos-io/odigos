package nodecollectorsgroup

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	odigospredicates "github.com/odigos-io/odigos/k8sutils/pkg/predicate"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

func SetupWithManager(mgr ctrl.Manager) error {

	err := ctrl.NewControllerManagedBy(mgr).
		For(&odigosv1.InstrumentationConfig{}).
		Named("nodecollectorgroup-instrumentationconfig").
		WithEventFilter(&odigospredicates.ExistencePredicate{}).
		Complete(&instrumentationConfigController{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = ctrl.NewControllerManagedBy(mgr).
		For(&corev1.ConfigMap{}).
		Named("nodecollectorgroup-odigosconfiguration").
		WithEventFilter(&odigospredicates.OdigosEffectiveConfigMapPredicate).
		Complete(&odigosConfigurationController{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = ctrl.NewControllerManagedBy(mgr).
		For(&odigosv1.CollectorsGroup{}).
		Named("nodecollectorgroup-clustercollectorsgroup").
		WithEventFilter(predicate.And(&odigospredicates.OdigosCollectorsGroupClusterPredicate, &odigospredicates.CgBecomesReadyPredicate{})).
		Complete(&clusterCollectorsGroupController{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = ctrl.NewControllerManagedBy(mgr).
		For(&odigosv1.Destination{}).
		Named("nodecollectorgroup-destinations").
		// trace surge rules have the agents they cover record span metrics, which the node collector routes.
		Watches(&odigosv1.Sampling{}, &handler.EnqueueRequestForObject{}).
		// we care if destinations are created or deleted, or enabled/disabled, and of changed sampling rules.
		WithEventFilter(&predicate.GenerationChangedPredicate{}).
		Complete(&DestinationsReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	return nil
}
