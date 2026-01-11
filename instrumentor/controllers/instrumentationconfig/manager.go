package instrumentationconfig

import (
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	instrumentorpredicate "github.com/odigos-io/odigos/instrumentor/controllers/utils/predicates"
	utilpredicate "github.com/odigos-io/odigos/k8sutils/pkg/predicate"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

func SetupWithManager(mgr ctrl.Manager) error {
	// Watch InstrumentationRule
	err := builder.
		ControllerManagedBy(mgr).
		Named("instrumentor-instrumentationconfig-instrumentationrule").
		For(&odigosv1alpha1.InstrumentationRule{}).
		// We filter for only created or updated events, and ignore events of status updates (aka no Spec change).
		WithEventFilter(predicate.Or(predicate.GenerationChangedPredicate{}, utilpredicate.CreationPredicate{})).
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
		// The SDK config might need to get updated if either:
		// - runtime details (auto detection) is updated.
		// - runtime overrides is updated by the user.
		WithEventFilter(predicate.Or(
			&instrumentorpredicate.RuntimeDetailsChangedPredicate{},
			&instrumentorpredicate.ContainerOverridesChangedPredicate{},
		)).
		Complete(&InstrumentationConfigReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	// watch effective config for any changes OR specific runtime metrics changes
	err = builder.
		ControllerManagedBy(mgr).
		Named("instrumentor-effectiveconfig-instrumentationconfig").
		For(&corev1.ConfigMap{}).
		WithEventFilter(
			utilpredicate.OdigosEffectiveConfigMapPredicate).
		Complete(&EffectiveConfigReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}
	return nil
}
