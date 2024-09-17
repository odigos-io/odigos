package runtime_details

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
)

func SetupWithManager(mgr ctrl.Manager, clientset *kubernetes.Clientset) error {
	err := builder.
		ControllerManagedBy(mgr).
		For(&odigosv1.InstrumentationConfig{}).
		Owns(&odigosv1.InstrumentedApplication{}).
		Complete(&DeprecatedInstrumentationConfigReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("Odiglet-RuntimeDetails-Pods").
		For(&corev1.Pod{}).
		WithEventFilter(&podPredicate{}).
		Complete(&PodsReconciler{
			Client:    mgr.GetClient(),
			Scheme:    mgr.GetScheme(),
			Clientset: clientset,
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("Odiglet-RuntimeDetails-InstrumentationConfig").
		For(&odigosv1.InstrumentationConfig{}).
		WithEventFilter(&instrumentationConfigPredicate{}).
		Complete(&InstrumentationConfigReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	return nil
}
