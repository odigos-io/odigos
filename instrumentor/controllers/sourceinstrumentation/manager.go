package sourceinstrumentation

import (
	appsv1 "k8s.io/api/apps/v1"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	odigospredicate "github.com/odigos-io/odigos/k8sutils/pkg/predicate"
)

func SetupWithManager(mgr ctrl.Manager) error {
	err := builder.
		ControllerManagedBy(mgr).
		Named("sourceinstrumentation-source").
		For(&v1alpha1.Source{}).
		Complete(&SourceReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("sourceinstrumentation-deployment").
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
		Named("sourceinstrumentation-daemonset").
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
		Named("sourceinstrumentation-statefulset").
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
		Named("sourceinstrumentation-instrumentationconfig").
		For(&v1alpha1.InstrumentationConfig{}).
		WithEventFilter(&odigospredicate.CreationPredicate{}).
		Complete(&InstrumentationConfigReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("sourceinstrumentation-instrumentedapp-migration").
		For(&v1alpha1.InstrumentedApplication{}).
		WithEventFilter(&odigospredicate.CreationPredicate{}).
		Complete(&InstrumentedApplicationMigrationReconciler{
			Client: mgr.GetClient(),
		})
	if err != nil {
		return err
	}

	return nil
}
