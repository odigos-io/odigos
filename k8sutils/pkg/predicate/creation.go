package predicate

import (
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// CreationPredicate only allows create events.
type CreationPredicate struct{}

func (i CreationPredicate) Create(e event.CreateEvent) bool {
	return true
}

func (i CreationPredicate) Update(e event.UpdateEvent) bool {
	return false
}

func (i CreationPredicate) Delete(e event.DeleteEvent) bool {
	return false
}

func (i CreationPredicate) Generic(e event.GenericEvent) bool {
	return false
}

var _ predicate.Predicate = &DeletionPredicate{}