package workloadmigrations

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	odigospredicate "github.com/odigos-io/odigos/k8sutils/pkg/predicate"
)

// SetupWithManager sets up the controllers for the workload migrations.
//
// The goal of these controllers is to migrate from features we used to implement by modifying the workloads
// to the new implementations that are based on our custom resources.
func SetupWithManager(mgr ctrl.Manager) error {
	migrationPredicate := predicate.Or(
		&odigospredicate.CreationPredicate{},
		predicate.LabelChangedPredicate{},
		predicate.AnnotationChangedPredicate{},
	)

	err := builder.
		ControllerManagedBy(mgr).
		Named("workloadmigrations-deployment").
		For(&appsv1.Deployment{}).
		WithEventFilter(migrationPredicate).
		Complete(&DeploymentReconciler{
			Client: mgr.GetClient(),
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("workloadmigrations-daemonset").
		For(&appsv1.DaemonSet{}).
		WithEventFilter(migrationPredicate).
		Complete(&DaemonSetReconciler{
			Client: mgr.GetClient(),
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("workloadmigrations-statefulset").
		For(&appsv1.StatefulSet{}).
		WithEventFilter(migrationPredicate).
		Complete(&StatefulSetReconciler{
			Client: mgr.GetClient(),
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("workloadmigrations-namespace").
		For(&corev1.Namespace{}).
		WithEventFilter(migrationPredicate).
		Complete(&NamespacesReconciler{
			Client: mgr.GetClient(),
		})
	if err != nil {
		return err
	}

	return nil
}
