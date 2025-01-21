package deleteinstrumentationconfig

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	odigospredicate "github.com/odigos-io/odigos/k8sutils/pkg/predicate"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// The startlangdetection controller handles uninstrumenting workloads.
// It provides a Source controller, which handles most events where uninstrumentation will occur:
// either by creating a new disabled Source object, or deleting an old Source object that enabled instrumentation.
// However, we also have controllers to monitor workload objects and namespaces.
// Because Sources are decoupled from these resources, a Source event might not immediately trigger
// uninstrumentation (for example, if a Deployment is deleted before a Source, then the Deployment
// event will trigger instrumentation). This design ensures 2-way reconciliation between
// Source CRD and workloads.
// Uninstrumentation itself is handled by the InstrumentationConfig controller, and these objects
// represent whether a workload is actively instrumented in the backend.
func SetupWithManager(mgr ctrl.Manager) error {
	err := builder.
		ControllerManagedBy(mgr).
		Named("deleteinstrumentationconfig-deployment").
		For(&appsv1.Deployment{}).
		WithEventFilter(predicate.LabelChangedPredicate{}).
		Complete(&DeploymentReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("deleteinstrumentationconfig-statefulset").
		For(&appsv1.StatefulSet{}).
		WithEventFilter(predicate.LabelChangedPredicate{}).
		Complete(&StatefulSetReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("deleteinstrumentationconfig-daemonset").
		For(&appsv1.DaemonSet{}).
		WithEventFilter(predicate.LabelChangedPredicate{}).
		Complete(&DaemonSetReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("deleteinstrumentationconfig-namespace").
		For(&corev1.Namespace{}).
		Complete(&NamespaceReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("deleteinstrumentationconfig-instrumentationconfig").
		For(&odigosv1.InstrumentationConfig{}).
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
		Named("deleteinstrumentationconfig-source").
		WithEventFilter(DeleteInstrumentationSourcePredicate).
		For(&odigosv1.Source{}).
		Complete(&SourceReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("deleteinstrumentationconfig-instrumentedapp-migration").
		For(&odigosv1.InstrumentedApplication{}).
		WithEventFilter(&odigospredicate.CreationPredicate{}).
		Complete(&InstrumentedApplicationMigrationReconciler{
			Client: mgr.GetClient(),
		})
	if err != nil {
		return err
	}

	return nil

}
