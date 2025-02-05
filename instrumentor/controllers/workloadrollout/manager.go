package workloadrollout

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

func SetupWithManager(mgr ctrl.Manager) error {
	return builder.
		ControllerManagedBy(mgr).
		Named("workloadrollout-instrumentationconfig").
		For(&odigosv1.InstrumentationConfig{}).
		WithEventFilter(&predicate.GenerationChangedPredicate{}).
		Complete(&instrumentationConfigReconciler{
			Client: mgr.GetClient(),
		})
}
