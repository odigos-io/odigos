package predicates

import (
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

type AgentInjectionRelevantRulesPredicate struct{}

func isRuleRelevantForAgentInjection(spec *odigosv1alpha1.InstrumentationRuleSpec) bool {
	return spec.OtelDistros != nil ||
		spec.HeadersCollection != nil ||
		spec.TraceConfig != nil ||
		spec.PayloadCollection != nil ||
		spec.TraceVerbosity != nil ||
		spec.CustomInstrumentations != nil ||
		spec.CodeAttributes != nil ||
		spec.EbpfLogCapture != nil
}

func (o AgentInjectionRelevantRulesPredicate) Create(e event.CreateEvent) bool {
	instrumentationRule, ok := e.Object.(*odigosv1alpha1.InstrumentationRule)
	if !ok {
		return false
	}

	return isRuleRelevantForAgentInjection(&instrumentationRule.Spec)
}

func (i AgentInjectionRelevantRulesPredicate) Update(e event.UpdateEvent) bool {
	old, oldOk := e.ObjectOld.(*odigosv1alpha1.InstrumentationRule)
	new, newOk := e.ObjectNew.(*odigosv1alpha1.InstrumentationRule)

	if !oldOk || !newOk {
		return false
	}

	return isRuleRelevantForAgentInjection(&old.Spec) || isRuleRelevantForAgentInjection(&new.Spec)
}

func (i AgentInjectionRelevantRulesPredicate) Delete(e event.DeleteEvent) bool {
	instrumentationRule, ok := e.Object.(*odigosv1alpha1.InstrumentationRule)
	if !ok {
		return false
	}

	return isRuleRelevantForAgentInjection(&instrumentationRule.Spec)
}

func (i AgentInjectionRelevantRulesPredicate) Generic(e event.GenericEvent) bool {
	return false
}
