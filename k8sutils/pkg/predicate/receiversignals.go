package predicate

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type ReceiverSignalsChangedPredicate struct {
}

func (o ReceiverSignalsChangedPredicate) Create(e event.CreateEvent) bool {
	if e.Object == nil {
		return false
	}

	return true
}

func (i ReceiverSignalsChangedPredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectNew == nil || e.ObjectOld == nil {
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

	// check if the receiver signals array has changed (len or content)
	if len(oldCollectorGroup.Status.ReceiverSignals) != len(newCollectorGroup.Status.ReceiverSignals) {
		return true
	}
	for i := 0; i < len(oldCollectorGroup.Status.ReceiverSignals); i++ {
		if oldCollectorGroup.Status.ReceiverSignals[i] != newCollectorGroup.Status.ReceiverSignals[i] {
			return true
		}
	}

	return false
}

func (i ReceiverSignalsChangedPredicate) Delete(e event.DeleteEvent) bool {
	return false
}

func (i ReceiverSignalsChangedPredicate) Generic(e event.GenericEvent) bool {
	return false
}

var _ predicate.Predicate = &ReceiverSignalsChangedPredicate{}
