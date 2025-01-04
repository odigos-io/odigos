package predicates

import (
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

type OtelSdkInstrumentationRulePredicate struct{}


func (o OtelSdkInstrumentationRulePredicate) Create(e event.CreateEvent) bool {
	// check if delete rule is for otel sdk
	instrumentationRule, ok := e.Object.(*odigosv1alpha1.InstrumentationRule)
	if !ok {
		return false
	}

	return instrumentationRule.Spec.OtelSdks != nil
}

func (i OtelSdkInstrumentationRulePredicate) Update(e event.UpdateEvent) bool {
	oldInstrumentationRule, oldOk := e.ObjectOld.(*odigosv1alpha1.InstrumentationRule)
	newInstrumentationRule, newOk := e.ObjectNew.(*odigosv1alpha1.InstrumentationRule)

	if !oldOk || !newOk {
		return false
	}

	// only handle rules for otel sdks
	return oldInstrumentationRule.Spec.OtelSdks != nil || newInstrumentationRule.Spec.OtelSdks != nil
}

func (i OtelSdkInstrumentationRulePredicate) Delete(e event.DeleteEvent) bool {
	// check if delete rule is for otel sdk
	instrumentationRule, ok := e.Object.(*odigosv1alpha1.InstrumentationRule)
	if !ok {
		return false
	}

	return instrumentationRule.Spec.OtelSdks != nil
}

func (i OtelSdkInstrumentationRulePredicate) Generic(e event.GenericEvent) bool {
	return false
}
