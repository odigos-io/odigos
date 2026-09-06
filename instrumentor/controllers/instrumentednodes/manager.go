package instrumentednodes

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	odigospredicate "github.com/odigos-io/odigos/k8sutils/pkg/predicate"
)

func mapPodToNode(ctx context.Context, obj client.Object) []reconcile.Request {
	pod, ok := obj.(*corev1.Pod)
	if !ok || pod.Spec.NodeName == "" {
		return nil
	}
	return []reconcile.Request{{
		NamespacedName: client.ObjectKey{Name: pod.Spec.NodeName},
	}}
}

func SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
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

	err = builder.
		ControllerManagedBy(mgr).
		Named("instrumentednodes-nodes").
		For(&corev1.Node{}).
		Watches(
			&corev1.Pod{},
			handler.EnqueueRequestsFromMapFunc(mapPodToNode),
			builder.WithPredicates(odigospredicate.ExistencePredicate{}),
		).
		Complete(&NodesReconciler{
			Client: mgr.GetClient(),
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("instrumentednodes-instrumentationconfig").
		For(&odigosv1.InstrumentationConfig{}).
		WithEventFilter(odigospredicate.ExistencePredicate{}).
		Complete(&InstrumentationConfigReconciler{
			Client: mgr.GetClient(),
		})
	if err != nil {
		return err
	}

	return nil
}

func SetupLabelCleanupWithManager(mgr ctrl.Manager) error {
	return mgr.Add(&removeInstrumentedPodsNodeLabelsRunnable{
		Client: mgr.GetClient(),
	})
}
