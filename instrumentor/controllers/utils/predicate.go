package utils

import "sigs.k8s.io/controller-runtime/pkg/event"

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
