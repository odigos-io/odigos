package predicates

import (
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

type AgentInjectionRelevantRulesPredicate struct{}

func (o AgentInjectionRelevantRulesPredicate) Create(e event.CreateEvent) bool {
	// check if delete rule is relevant for agent enabling controllers
	instrumentationRule, ok := e.Object.(*odigosv1alpha1.InstrumentationRule)
	if !ok {
		return false
	}

	return instrumentationRule.Spec.OtelSdks != nil || instrumentationRule.Spec.OtelDistros != nil
}

func (i AgentInjectionRelevantRulesPredicate) Update(e event.UpdateEvent) bool {
	oldInstrumentationRule, oldOk := e.ObjectOld.(*odigosv1alpha1.InstrumentationRule)
	newInstrumentationRule, newOk := e.ObjectNew.(*odigosv1alpha1.InstrumentationRule)

	if !oldOk || !newOk {
		return false
	}

	// only handle rules for otel sdks or distros configuration
	return oldInstrumentationRule.Spec.OtelSdks != nil || newInstrumentationRule.Spec.OtelSdks != nil ||
		oldInstrumentationRule.Spec.OtelDistros != nil || newInstrumentationRule.Spec.OtelDistros != nil
}

func (i AgentInjectionRelevantRulesPredicate) Delete(e event.DeleteEvent) bool {
	// check if delete rule is for otel sdk
	instrumentationRule, ok := e.Object.(*odigosv1alpha1.InstrumentationRule)
	if !ok {
		return false
	}

	return instrumentationRule.Spec.OtelSdks != nil || instrumentationRule.Spec.OtelDistros != nil
}

func (i AgentInjectionRelevantRulesPredicate) Generic(e event.GenericEvent) bool {
	return false
}
