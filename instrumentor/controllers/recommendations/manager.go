package recommendations

import (
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	odigospredicate "github.com/odigos-io/odigos/k8sutils/pkg/predicate"
)

func SetupWithManager(mgr ctrl.Manager) error {
	err := ctrl.NewControllerManagedBy(mgr).
		Named("recommendations-sync").
		Watches(&corev1.ConfigMap{}, &handler.EnqueueRequestForObject{}, builder.WithPredicates(&odigospredicate.OdigosEffectiveConfigMapPredicate)).
		Watches(&odigosv1.Action{}, &handler.EnqueueRequestForObject{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Watches(&odigosv1.InstrumentationConfig{}, &handler.EnqueueRequestForObject{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Complete(&RecommendationsSyncReconciler{Client: mgr.GetClient()})
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named("recommendations-recommendation").
		For(&odigosv1.Recommendation{}).
		WithEventFilter(predicate.Or(
			predicate.GenerationChangedPredicate{},
			&odigospredicate.DeletionPredicate{},
		)).
		Complete(&RecommendationReconciler{Client: mgr.GetClient()})
}
