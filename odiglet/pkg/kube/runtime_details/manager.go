package runtime_details

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func SetupWithManager(mgr ctrl.Manager) error {

	err := builder.
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
		For(&odigosv1.InstrumentationConfig{}).
		Owns(&odigosv1.InstrumentedApplication{}).
		Complete(&InstrumentationConfigReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		For(&odigosv1.OdigosConfiguration{}).
		WithEventFilter(&onlyUpdatesPredicate{}).
		Complete(&OdigosConfigReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	return nil
}

type onlyUpdatesPredicate struct{}

func (o onlyUpdatesPredicate) Create(e event.CreateEvent) bool {
	return false
}

func (i onlyUpdatesPredicate) Update(e event.UpdateEvent) bool {
	return true
}

func (i onlyUpdatesPredicate) Delete(e event.DeleteEvent) bool {
	return false
}

func (i onlyUpdatesPredicate) Generic(e event.GenericEvent) bool {
	return false
}
