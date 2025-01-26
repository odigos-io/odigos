package predicate

import (
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// DeletionPredicate only allows delete events.
type DeletionPredicate struct{}

func (o DeletionPredicate) Create(e event.CreateEvent) bool {
	return false
}

func (i DeletionPredicate) Update(e event.UpdateEvent) bool {
	return false
}

func (i DeletionPredicate) Delete(e event.DeleteEvent) bool {
	return true
}

func (i DeletionPredicate) Generic(e event.GenericEvent) bool {
	return false
}

var _ predicate.Predicate = &DeletionPredicate{}
