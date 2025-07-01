package predicates

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// RuntimeDetailsChangedPredicate is a predicate that checks if the runtime details of an InstrumentationConfig have changed.
//
// For Create events, it returns true if the InstrumentationConfig has any runtime details.
// For Update events, it returns true if the runtime details have changed (currently only checks the length of the runtime details).
// For Delete events, it returns false.
//
// TODO: once we support updating the runtime details more than once, we should improve this predicate to check the actual changes.
type RuntimeDetailsChangedPredicate struct{}

var _ predicate.Predicate = &RuntimeDetailsChangedPredicate{}

var InstrumentationConfigRuntimeDetailsChangedPredicate = RuntimeDetailsChangedPredicate{}

func (o RuntimeDetailsChangedPredicate) Create(e event.CreateEvent) bool {
	if e.Object == nil {
		return false
	}

	ic, ok := e.Object.(*odigosv1.InstrumentationConfig)
	if !ok {
		return false
	}

	return len(ic.Status.RuntimeDetailsByContainer) > 0
}

func (i RuntimeDetailsChangedPredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil || e.ObjectNew == nil {
		return false
	}

	oldIc, oldOk := e.ObjectOld.(*odigosv1.InstrumentationConfig)
	newIc, newOk := e.ObjectNew.(*odigosv1.InstrumentationConfig)

	if !oldOk || !newOk {
		return false
	}

	// currently, we only check the lengths of the runtime details
	// we should improve this once we support updating the runtime details more than once
	if len(oldIc.Status.RuntimeDetailsByContainer) != len(newIc.Status.RuntimeDetailsByContainer) {
		return true
	}

	for i, oldDetails := range oldIc.Status.RuntimeDetailsByContainer {
		if i >= len(newIc.Status.RuntimeDetailsByContainer) {
			return true // container was removed
		}
		newDetails := newIc.Status.RuntimeDetailsByContainer[i]

		if oldDetails.Language != newDetails.Language ||
			oldDetails.RuntimeVersion != newDetails.RuntimeVersion ||
			oldDetails.OtherAgent != newDetails.OtherAgent {
			return true // runtime details have changed
		}
	}

	return false
}

func (i RuntimeDetailsChangedPredicate) Delete(e event.DeleteEvent) bool {
	return false
}

func (i RuntimeDetailsChangedPredicate) Generic(e event.GenericEvent) bool {
	return false
}

type ContainerOverridesChangedPredicate struct{}

var _ predicate.Predicate = &ContainerOverridesChangedPredicate{}

var InstrumentationConfigContainerOverridesChangedPredicate = ContainerOverridesChangedPredicate{}

func (i ContainerOverridesChangedPredicate) Create(e event.CreateEvent) bool {
	return true
}

func (i ContainerOverridesChangedPredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil || e.ObjectNew == nil {
		return false
	}

	oldIc, oldOk := e.ObjectOld.(*odigosv1.InstrumentationConfig)
	newIc, newOk := e.ObjectNew.(*odigosv1.InstrumentationConfig)
	if !oldOk || !newOk {
		return false
	}

	return oldIc.Spec.ContainerOverridesHash != newIc.Spec.ContainerOverridesHash
}

func (i ContainerOverridesChangedPredicate) Delete(e event.DeleteEvent) bool {
	return false
}

func (i ContainerOverridesChangedPredicate) Generic(e event.GenericEvent) bool {
	return false
}
