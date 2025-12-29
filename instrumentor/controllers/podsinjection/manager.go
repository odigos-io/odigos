package podsinjection

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/predicate"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

type InstrumentationConfigPodsInjectionPredicate struct{}

func (o InstrumentationConfigPodsInjectionPredicate) Create(e event.CreateEvent) bool {

	// when an instrumentation config is created, we need to sync the pods injection numbers at creation time.
	// this will also help in syncing the ic status when instrumentor was while pods number changed.
	return true
}

func (o InstrumentationConfigPodsInjectionPredicate) Update(e event.UpdateEvent) bool {
	old, oldOk := e.ObjectOld.(*odigosv1.InstrumentationConfig)
	new, newOk := e.ObjectNew.(*odigosv1.InstrumentationConfig)

	if !oldOk || !newOk {
		return false
	}

	// pods relay on the agents meta hash, and when it changes, we need to re-compute the couters based on the new agents meta hash baseline.
	return old.Spec.AgentsMetaHash != new.Spec.AgentsMetaHash
}

func (o InstrumentationConfigPodsInjectionPredicate) Delete(e event.DeleteEvent) bool {
	return false
}

func (o InstrumentationConfigPodsInjectionPredicate) Generic(e event.GenericEvent) bool {
	return true
}

func SetupWithManager(mgr ctrl.Manager) error {

	podsTracker := NewPodsTracker()

	err := builder.
		ControllerManagedBy(mgr).
		Named("podsinjection-pods").
		For(&corev1.Pod{}).
		WithEventFilter(predicate.ExistencePredicate{}).
		Complete(
			&PodsController{
				Client:      mgr.GetClient(),
				PodsTracker: podsTracker,
			},
		)
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("podsinjection-instrumentationconfig").
		For(&odigosv1.InstrumentationConfig{}).
		WithEventFilter(&InstrumentationConfigPodsInjectionPredicate{}).
		Complete(&InstrumentationConfigController{
			Client:      mgr.GetClient(),
			PodsTracker: podsTracker,
		})
	if err != nil {
		return err
	}

	return nil
}
