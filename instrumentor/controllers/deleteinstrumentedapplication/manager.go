package deleteinstrumentedapplication

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/predicate"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

func SetupWithManager(mgr ctrl.Manager) error {
	err := builder.
		ControllerManagedBy(mgr).
		Named("deleteinstrumentedapplication-deployment").
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
		Named("deleteinstrumentedapplication-statefulset").
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
		Named("deleteinstrumentedapplication-daemonset").
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
		Named("deleteinstrumentedapplication-namespace").
		For(&corev1.Namespace{}).
		WithEventFilter(&NsLabelBecameDisabledPredicate{}).
		Complete(&NamespaceReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("deleteinstrumentedapplication-instrumentedapplication").
		For(&odigosv1.InstrumentedApplication{}).
		Complete(&InstrumentedApplicationReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = builder.
	ControllerManagedBy(mgr).
	Named("deleteinstrumentedapplication-source").
	WithEventFilter(&k8sutils.OnlyUpdatesPredicate{}).
	For(&odigosv1.Source{}).
	Complete(&SourceReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	})
if err != nil {
	return err
}

	return nil

}
