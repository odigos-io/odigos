package deleteinstrumentedapplication

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/client"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

func SetupWithManager(mgr ctrl.Manager) error {
	// Create a new client with fallback to API server
	// We are doing this because client-go cache is not supporting dynamic cache rules
	// Sometimes we will need to get/list objects that are out of the cache (e.g. when namespace is labeled)
	clientWithFallback := k8sutils.NewKubernetesClientFromCacheWithAPIFallback(mgr.GetClient(), mgr.GetAPIReader())

	err := builder.
		ControllerManagedBy(mgr).
		For(&appsv1.Deployment{}).
		WithEventFilter(predicate.LabelChangedPredicate{}).
		Complete(&DeploymentReconciler{
			Client: clientWithFallback,
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		For(&appsv1.StatefulSet{}).
		WithEventFilter(predicate.LabelChangedPredicate{}).
		Complete(&StatefulSetReconciler{
			Client: clientWithFallback,
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		For(&appsv1.DaemonSet{}).
		WithEventFilter(predicate.LabelChangedPredicate{}).
		Complete(&DaemonSetReconciler{
			Client: clientWithFallback,
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		For(&corev1.Namespace{}).
		WithEventFilter(predicate.LabelChangedPredicate{}).
		Complete(&NamespaceReconciler{
			Client: clientWithFallback,
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		For(&odigosv1.InstrumentedApplication{}).
		Complete(&InstrumentedApplicationReconciler{
			Client: clientWithFallback,
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	return nil

}
