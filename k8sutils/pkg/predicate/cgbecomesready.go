package predicate

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	cr_predicate "sigs.k8s.io/controller-runtime/pkg/predicate"
)

// this event filter will only trigger reconciliation when the collectors group was not ready and now it is ready.
// some controllers in odigos reacts to this specific event, and should not be triggered by other events such as spec updates or status conditions changes.
// for create events, it will only trigger reconciliation if the collectors group is ready.
type CgBecomesReadyPredicate struct{}

func (i *CgBecomesReadyPredicate) Create(e event.CreateEvent) bool {
	if e.Object == nil {
		return false
	}

	cg, ok := e.Object.(*odigosv1.CollectorsGroup)
	if !ok {
		return false
	}
	return cg.Status.Ready
}

func (i *CgBecomesReadyPredicate) Update(e event.UpdateEvent) bool {

	if e.ObjectOld == nil || e.ObjectNew == nil {
		return false
	}

	oldCollectorGroup, ok := e.ObjectOld.(*odigosv1.CollectorsGroup)
	if !ok {
		return false
	}
	newCollectorGroup, ok := e.ObjectNew.(*odigosv1.CollectorsGroup)
	if !ok {
		return false
	}

	wasReady := oldCollectorGroup.Status.Ready
	nowReady := newCollectorGroup.Status.Ready
	becameReady := !wasReady && nowReady

	return becameReady
}

func (i *CgBecomesReadyPredicate) Delete(e event.DeleteEvent) bool {
	return false
}

func (i *CgBecomesReadyPredicate) Generic(e event.GenericEvent) bool {
	return false
}

var _ cr_predicate.Predicate = &CgBecomesReadyPredicate{}
