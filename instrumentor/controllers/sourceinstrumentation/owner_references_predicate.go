package sourceinstrumentation

import (
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// ownerReferencesCountChangedPredicate allows update events when the number of OwnerReferences changes.
// InstrumentationConfigs are owned by enabling Sources; when a Source is deleted, k8s removes that
// owner ref and we need to re-check whether the workload is still instrumented by any remaining Source.
type ownerReferencesCountChangedPredicate struct{}

func (p ownerReferencesCountChangedPredicate) Create(e event.CreateEvent) bool {
	return false
}

func (p ownerReferencesCountChangedPredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil || e.ObjectNew == nil {
		return false
	}
	return len(e.ObjectOld.GetOwnerReferences()) != len(e.ObjectNew.GetOwnerReferences())
}

func (p ownerReferencesCountChangedPredicate) Delete(e event.DeleteEvent) bool {
	return false
}

func (p ownerReferencesCountChangedPredicate) Generic(e event.GenericEvent) bool {
	return false
}

var _ predicate.Predicate = &ownerReferencesCountChangedPredicate{}
