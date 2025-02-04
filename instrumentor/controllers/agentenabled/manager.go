package agentenabled

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	instrumentorpredicate "github.com/odigos-io/odigos/instrumentor/controllers/utils/predicates"
	odigospredicate "github.com/odigos-io/odigos/k8sutils/pkg/predicate"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

func SetupWithManager(mgr ctrl.Manager) error {
	err := builder.
		ControllerManagedBy(mgr).
		Named("agentenabled-collectorsgroup").
		For(&odigosv1.CollectorsGroup{}).
		WithEventFilter(predicate.And(&odigospredicate.OdigosCollectorsGroupNodePredicate, &odigospredicate.CgBecomesReadyPredicate{})).
		Complete(&CollectorsGroupReconciler{
			Client: mgr.GetClient(),
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("agentenabled-instrumentationconfig").
		For(&odigosv1.InstrumentationConfig{}).
		WithEventFilter(&instrumentorpredicate.RuntimeDetailsChangedPredicate{}).
		Complete(&InstrumentationConfigReconciler{
			Client: mgr.GetClient(),
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("agentenabled-instrumentationrules").
		For(&odigosv1.InstrumentationRule{}).
		WithEventFilter(&instrumentorpredicate.OtelSdkInstrumentationRulePredicate{}).
		Complete(&InstrumentationRuleReconciler{
			Client: mgr.GetClient(),
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("agentenabled-effectiveconfig").
		For(&corev1.ConfigMap{}).
		WithEventFilter(odigospredicate.OdigosEffectiveConfigMapPredicate).
		Complete(&EffectiveConfigReconciler{
			Client: mgr.GetClient(),
		})
	if err != nil {
		return err
	}

	err = builder.
		WebhookManagedBy(mgr).
		For(&corev1.Pod{}).
		WithDefaulter(&PodsWebhook{
			Client: mgr.GetClient(),
		}).
		Complete()
	if err != nil {
		return err
	}

	return nil
}
