package instrumentednodes

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	odigospredicate "github.com/odigos-io/odigos/k8sutils/pkg/predicate"
)

func SetupWithManager(ctx context.Context, mgr ctrl.Manager, nodeLabelRetention time.Duration) error {
	// Index pods by Spec.NodeName so sync can list only pods on a given node
	// (MatchingFields) instead of scanning the full pod cache.
	err := mgr.GetFieldIndexer().IndexField(
		ctx,
		&corev1.Pod{},
		podNodeNameIndex,
		func(obj client.Object) []string {
			pod, ok := obj.(*corev1.Pod)
			if !ok || pod.Spec.NodeName == "" {
				return nil
			}
			return []string{pod.Spec.NodeName}
		},
	)
	if err != nil {
		return err
	}

	// check if label should be set when pods are created/scheduled on a node
	err = builder.
		ControllerManagedBy(mgr).
		Named("instrumentednodes-pods").
		For(&corev1.Pod{}).
		WithEventFilter(&podNodeNamePredicate{}).
		Complete(&PodsReconciler{
			Client:             mgr.GetClient(),
			NodeLabelRetention: nodeLabelRetention,
		})
	if err != nil {
		return err
	}

	// Reconcile on node create (pod may already be in cache before the node is)
	// and when FirstInstrumentedPodAtNodeLabel changes. Label-change reconciles are
	// usually a no-op, but cover cache update races so the label value stays correct.
	err = builder.
		ControllerManagedBy(mgr).
		Named("instrumentednodes-nodes").
		For(&corev1.Node{}).
		WithEventFilter(&nodeInstrumentedPodsLabelPredicate{}).
		Complete(&NodesReconciler{
			Client:             mgr.GetClient(),
			NodeLabelRetention: nodeLabelRetention,
		})
	if err != nil {
		return err
	}

	// we need to update the label when something becomes "marked for instrumentation",
	// to allow odiglet to do runtime detection prior to pods with agents injectedm,
	// and also need to react to instrumented pods with no label ("optional pod manifest injection" a.k.a "no restart")
	err = builder.
		ControllerManagedBy(mgr).
		Named("instrumentednodes-instrumentationconfig").
		For(&odigosv1.InstrumentationConfig{}).
		WithEventFilter(odigospredicate.ExistencePredicate{}).
		Complete(&InstrumentationConfigReconciler{
			Client:             mgr.GetClient(),
			NodeLabelRetention: nodeLabelRetention,
		})
	if err != nil {
		return err
	}

	return nil
}
