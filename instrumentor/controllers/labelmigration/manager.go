package labelmigration

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	odigospredicate "github.com/odigos-io/odigos/k8sutils/pkg/predicate"
)

func SetupWithManager(mgr ctrl.Manager) error {
	err := builder.
		ControllerManagedBy(mgr).
		Named("labelmigration-deployment").
		For(&appsv1.Deployment{}).
		WithEventFilter(predicate.Or(&odigospredicate.CreationPredicate{}, predicate.LabelChangedPredicate{})).
		Complete(&DeploymentReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("labelmigration-daemonset").
		For(&appsv1.DaemonSet{}).
		WithEventFilter(predicate.Or(&odigospredicate.CreationPredicate{}, predicate.LabelChangedPredicate{})).
		Complete(&DaemonSetReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("labelmigration-statefulset").
		For(&appsv1.StatefulSet{}).
		WithEventFilter(predicate.Or(&odigospredicate.CreationPredicate{}, predicate.LabelChangedPredicate{})).
		Complete(&StatefulSetReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("labelmigration-namespace").
		For(&corev1.Namespace{}).
		WithEventFilter(predicate.Or(&odigospredicate.CreationPredicate{}, predicate.LabelChangedPredicate{})).
		Complete(&NamespacesReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	return nil
}
