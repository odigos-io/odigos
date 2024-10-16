package instrumentationconfig

import (
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

func SetupWithManager(mgr ctrl.Manager) error {
	err := builder.
		ControllerManagedBy(mgr).
		Named("instrumentor-instrumentationconfig-instrumentationrule").
		For(&odigosv1alpha1.InstrumentationRule{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(&InstrumentationRuleReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("instrumentor-instrumentationconfig-instrumentedapplication").
		For(&odigosv1alpha1.InstrumentedApplication{}).
		Complete(&InstrumentedApplicationReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	return nil
}
