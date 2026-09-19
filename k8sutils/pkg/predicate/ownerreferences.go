package predicate

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	cr_predicate "sigs.k8s.io/controller-runtime/pkg/predicate"
)

// OwnerReferencesChangedPredicate allows update events when OwnerReferences change.
type OwnerReferencesChangedPredicate struct{}

func (p OwnerReferencesChangedPredicate) Create(e event.CreateEvent) bool {
	return false
}

func (p OwnerReferencesChangedPredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil || e.ObjectNew == nil {
		return false
	}
	return !ownerReferencesEqual(e.ObjectOld.GetOwnerReferences(), e.ObjectNew.GetOwnerReferences())
}

func (p OwnerReferencesChangedPredicate) Delete(e event.DeleteEvent) bool {
	return false
}

func (p OwnerReferencesChangedPredicate) Generic(e event.GenericEvent) bool {
	return false
}

func ownerReferencesEqual(a, b []metav1.OwnerReference) bool {
	return reflect.DeepEqual(a, b)
}

var _ cr_predicate.Predicate = &OwnerReferencesChangedPredicate{}
