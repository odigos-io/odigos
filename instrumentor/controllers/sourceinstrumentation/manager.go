package sourceinstrumentation

import (
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"

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

	// Workload and Namespace reconcilers exist to catch the case where one of these entities is created
	// after the Source that instruments it (because Sources can exist independently of entities).
	// For that reason, we only watch for Create events on these controllers.
	err = builder.
		ControllerManagedBy(mgr).
		Named("sourceinstrumentation-deployment").
		For(&appsv1.Deployment{}).
		WithEventFilter(&odigospredicate.CreationPredicate{}).
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
		WithEventFilter(&odigospredicate.CreationPredicate{}).
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
		WithEventFilter(&odigospredicate.CreationPredicate{}).
		Complete(&StatefulSetReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("sourceinstrumentation-namespace").
		For(&v1.Namespace{}).
		WithEventFilter(&odigospredicate.CreationPredicate{}).
		Complete(&NamespaceReconciler{
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
