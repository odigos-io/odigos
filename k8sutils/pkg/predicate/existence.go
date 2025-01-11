package predicate

import (
	"sigs.k8s.io/controller-runtime/pkg/event"
	cr_predicate "sigs.k8s.io/controller-runtime/pkg/predicate"
)

// existence predicate only allows create and delete events.
// it is useful when the controller only reacts to the presence of an object (e.g. if it exists or not),
// and not to it's content or status changes.
//
// Notice these 2 things when using this predicate:
//  1. The event filter will allow events for each object on startup, as all the objects are "created" in the cache.
//  2. If you have important task to do on delete events, make sure it is applied if
//     the event is missed and the controller restarts, since the delete event will not be triggered on
//     controller restart as the object is no longer in k8s.
type ExistencePredicate struct{}

func (o ExistencePredicate) Create(e event.CreateEvent) bool {
	return true
}

func (i ExistencePredicate) Update(e event.UpdateEvent) bool {
	return false
}

func (i ExistencePredicate) Delete(e event.DeleteEvent) bool {
	return true
}

func (i ExistencePredicate) Generic(e event.GenericEvent) bool {
	return false
}

var _ cr_predicate.Predicate = &ExistencePredicate{}
