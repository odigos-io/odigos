package predicates

import (
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

type AgentInjectionEnabledActionsPredicate struct{}

func (u AgentInjectionEnabledActionsPredicate) Create(e event.CreateEvent) bool {
	action, ok := e.Object.(*odigosv1alpha1.Action)
	if !ok {
		return false
	}

	return !action.Spec.Disabled &&
		(action.Spec.URLTemplatization != nil || action.Spec.SpanRenamer != nil)
}

func (u AgentInjectionEnabledActionsPredicate) Update(e event.UpdateEvent) bool {
	old, oldOk := e.ObjectOld.(*odigosv1alpha1.Action)
	new, newOk := e.ObjectNew.(*odigosv1alpha1.Action)
	if !oldOk || !newOk {
		return false
	}

	return (!old.Spec.Disabled || !new.Spec.Disabled) &&
		(old.Spec.URLTemplatization != nil || new.Spec.URLTemplatization != nil || old.Spec.SpanRenamer != nil || new.Spec.SpanRenamer != nil)
}

func (u AgentInjectionEnabledActionsPredicate) Delete(e event.DeleteEvent) bool {
	action, ok := e.Object.(*odigosv1alpha1.Action)
	if !ok {
		return false
	}
	return !action.Spec.Disabled &&
		(action.Spec.URLTemplatization != nil || action.Spec.SpanRenamer != nil)
}

func (u AgentInjectionEnabledActionsPredicate) Generic(e event.GenericEvent) bool {
	return true
}
