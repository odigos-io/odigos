package startlangdetection

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	odigospredicate "github.com/odigos-io/odigos/k8sutils/pkg/predicate"
)

func SetupWithManager(mgr ctrl.Manager) error {
	err := builder.
		ControllerManagedBy(mgr).
		Named("startlangdetection-deployment").
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
		Named("startlangdetection-daemonset").
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
		Named("startlangdetection-statefulset").
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
		Named("startlangdetection-namespace").
		For(&corev1.Namespace{}).
		Complete(&NamespacesReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("startlangdetection-source").
		For(&v1alpha1.Source{}).
		WithEventFilter(StartLangDetectionSourcePredicate).
		Complete(&SourceReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	return nil
}
