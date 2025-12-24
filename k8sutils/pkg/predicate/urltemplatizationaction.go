package predicate

import (
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
)

type URLTemplatizationActionPredicate struct {
}

func (u URLTemplatizationActionPredicate) Create(e event.CreateEvent) bool {
	if e.Object == nil {
		return false
	}

	action, ok := e.Object.(*odigosv1.Action)
	if !ok {
		return false
	}

	// Check if this action has URLTemplatization config and is not disabled
	return !action.Spec.Disabled && action.Spec.URLTemplatization != nil
}

func (u URLTemplatizationActionPredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectNew == nil || e.ObjectOld == nil {
		return false
	}

	oldAction, okOld := e.ObjectOld.(*odigosv1.Action)
	newAction, okNew := e.ObjectNew.(*odigosv1.Action)
	if !okOld || !okNew {
		return false
	}

	// Check if URLTemplatization config was added, removed, or changed
	oldHasURLTemplatization := !oldAction.Spec.Disabled && oldAction.Spec.URLTemplatization != nil
	newHasURLTemplatization := !newAction.Spec.Disabled && newAction.Spec.URLTemplatization != nil

	// Trigger if:
	// 1. URLTemplatization was added or removed
	// 2. The disabled status changed for an action with URLTemplatization
	// 3. URLTemplatization config exists and other relevant fields changed
	if oldHasURLTemplatization != newHasURLTemplatization {
		return true
	}

	// If both have URLTemplatization, check if the config changed
	if oldHasURLTemplatization && newHasURLTemplatization {
		// For simplicity, we'll trigger on any spec change when URLTemplatization is present
		return oldAction.Spec.Disabled != newAction.Spec.Disabled ||
			oldAction.Generation != newAction.Generation
	}

	return false
}

func (u URLTemplatizationActionPredicate) Delete(e event.DeleteEvent) bool {
	if e.Object == nil {
		return false
	}

	action, ok := e.Object.(*odigosv1.Action)
	if !ok {
		return false
	}

	// Trigger when a URLTemplatization action is deleted
	return !action.Spec.Disabled && action.Spec.URLTemplatization != nil
}

func (u URLTemplatizationActionPredicate) Generic(e event.GenericEvent) bool {
	return false
}

var _ predicate.Predicate = &URLTemplatizationActionPredicate{}
