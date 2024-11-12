package predicate

import (
	"sigs.k8s.io/controller-runtime/pkg/event"
	cr_predicate "sigs.k8s.io/controller-runtime/pkg/predicate"
)

// This predicate only allows update events.
// It is useful if you handle the initial state when the controller starts, and only need to apply changes
// for example - when monitoring odigos config changes.
type OnlyUpdatesPredicate struct{}

func (o OnlyUpdatesPredicate) Create(e event.CreateEvent) bool {
	return false
}

func (i OnlyUpdatesPredicate) Update(e event.UpdateEvent) bool {
	return true
}

func (i OnlyUpdatesPredicate) Delete(e event.DeleteEvent) bool {
	return false
}

func (i OnlyUpdatesPredicate) Generic(e event.GenericEvent) bool {
	return false
}

var _ cr_predicate.Predicate = &OnlyUpdatesPredicate{}
