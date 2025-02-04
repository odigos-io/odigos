package workloadrollout

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
)

func SetupWithManager(mgr ctrl.Manager) error {
	return builder.
		ControllerManagedBy(mgr).
		Named("workloadrollout-instrumentationconfig").
		For(&odigosv1.InstrumentationConfig{}).
		Complete(&instrumentationConfigReconciler{
			Client: mgr.GetClient(),
		})
}
