package runtime_details

import (
	"fmt"

	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type nameSpaceEnabledPredicate struct {
	predicate.Funcs
}

func (i *nameSpaceEnabledPredicate) Create(e event.CreateEvent) bool {
	return false
}

func (i *nameSpaceEnabledPredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil {
		return false
	}
	if e.ObjectNew == nil {
		return false
	}

	oldEnabled := workload.IsObjectLabeledForInstrumentation(e.ObjectOld)
	newEnabled := workload.IsObjectLabeledForInstrumentation(e.ObjectNew)
	becameEnabled := !oldEnabled && newEnabled

	fmt.Printf("namespace becameEnabled: %v\n", becameEnabled)

	return becameEnabled
}

func (i *nameSpaceEnabledPredicate) Delete(e event.DeleteEvent) bool {
	// no need to calculate runtime details for deleted workloads
	return false
}

func (i *nameSpaceEnabledPredicate) Generic(e event.GenericEvent) bool {
	// not sure when exactly this would be called, but we don't need to handle it
	return false
}