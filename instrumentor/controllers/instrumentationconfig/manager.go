package instrumentationconfig

import (
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"

	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	instrumentorpredicate "github.com/odigos-io/odigos/instrumentor/controllers/utils/predicates"
)

func SetupWithManager(mgr ctrl.Manager) error {
	// Watch InstrumentationRule
	err := builder.
		ControllerManagedBy(mgr).
		Named("instrumentor-instrumentationconfig-instrumentationrule").
		For(&odigosv1alpha1.InstrumentationRule{}).
		Complete(&InstrumentationRuleReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("instrumentor-instrumentationconfig-instrumentationconfig").
		For(&odigosv1alpha1.InstrumentationConfig{}).
		WithEventFilter(&instrumentorpredicate.RuntimeDetailsChangedPredicate{}).
		Complete(&InstrumentationConfigReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	// Watch for Source changes to update InstrumentationConfig
	if err := builder.
		ControllerManagedBy(mgr).
		Named("instrumentor-instrumentationconfig-source").
		For(&odigosv1alpha1.Source{}).
		Complete(&SourceReconciler{
			Client: mgr.GetClient(),
		}); err != nil {
		return err
	}

	return nil
}
