package startlangdetection

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
)

func SetupWithManager(mgr ctrl.Manager) error {
	err := builder.
		ControllerManagedBy(mgr).
		Named("startlangdetection-deployment").
		For(&appsv1.Deployment{}).
		WithEventFilter(&WorkloadEnabledPredicate{}).
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
		WithEventFilter(&WorkloadEnabledPredicate{}).
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
		WithEventFilter(&WorkloadEnabledPredicate{}).
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

	//err = builder.
	//	ControllerManagedBy(mgr).
	//	Named("startlangdetection-configmaps").
	//	For(&corev1.ConfigMap{}).
	//	WithEventFilter(&utils.OnlyUpdatesPredicate{}).
	//	Complete(&OdigosConfigReconciler{
	//		Client: mgr.GetClient(),
	//		Scheme: mgr.GetScheme(),
	//	})
	//if err != nil {
	//	return err
	//}

	return nil
}
