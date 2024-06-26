package runtime_details

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
)

func SetupWithManager(mgr ctrl.Manager) error {

	err := builder.
		ControllerManagedBy(mgr).
		For(&appsv1.Deployment{}).
		Owns(&odigosv1.InstrumentedApplication{}).
		WithEventFilter(&WorkloadEnabledPredicate{}).
		Complete(&DeploymentsReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		For(&appsv1.StatefulSet{}).
		Owns(&odigosv1.InstrumentedApplication{}).
		WithEventFilter(&WorkloadEnabledPredicate{}).
		Complete(&StatefulSetsReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		For(&appsv1.DaemonSet{}).
		Owns(&odigosv1.InstrumentedApplication{}).
		WithEventFilter(&WorkloadEnabledPredicate{}).
		Complete(&DaemonSetsReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		For(&corev1.Namespace{}).
		WithEventFilter(&nameSpaceEnabledPredicate{}).
		Owns(&odigosv1.InstrumentedApplication{}).
		Complete(&NamespacesReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		For(&odigosv1.OdigosConfiguration{}).
		Complete(&OdigosConfigReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	return nil
}
