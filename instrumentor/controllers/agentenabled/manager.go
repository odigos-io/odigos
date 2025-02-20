package agentenabled

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/distros"
	instrumentorpredicate "github.com/odigos-io/odigos/instrumentor/controllers/utils/predicates"
	odigospredicate "github.com/odigos-io/odigos/k8sutils/pkg/predicate"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

func SetupWithManager(mgr ctrl.Manager, dp *distros.Provider) error {
	err := builder.
		ControllerManagedBy(mgr).
		Named("agentenabled-collectorsgroup").
		For(&odigosv1.CollectorsGroup{}).
		WithEventFilter(predicate.And(&odigospredicate.OdigosCollectorsGroupNodePredicate, &odigospredicate.CgBecomesReadyPredicate{})).
		Complete(&CollectorsGroupReconciler{
			Client: mgr.GetClient(),
			DistrosProvider: dp,
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("agentenabled-instrumentationconfig").
		For(&odigosv1.InstrumentationConfig{}).
		// When the runtime details change we need to potentially update the instrumentation config and roll out the workload.
		// When the instrumentation config is deleted, we need to roll out the workload to un-instrument it.
		WithEventFilter(predicate.Or(&instrumentorpredicate.RuntimeDetailsChangedPredicate{}, odigospredicate.DeletionPredicate{})).
		Complete(&InstrumentationConfigReconciler{
			Client: mgr.GetClient(),
			DistrosProvider: dp,
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
			DistrosProvider: dp,
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
			DistrosProvider: dp,
		})
	if err != nil {
		return err
	}

	err = builder.
		WebhookManagedBy(mgr).
		For(&corev1.Pod{}).
		WithDefaulter(&PodsWebhook{
			Client: mgr.GetClient(),
			DistrosGetter: dp.Getter,
		}).
		Complete()
	if err != nil {
		return err
	}

	return nil
}
