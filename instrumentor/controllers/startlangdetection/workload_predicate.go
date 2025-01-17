package startlangdetection

import (
	"context"

	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	sourceutils "github.com/odigos-io/odigos/k8sutils/pkg/source"
)

// this predicate is used for workload reconciler to only pass events
// where the workload has replicas available to instrument.
type WorkloadAvailablePredicate struct {
	predicate.Funcs
	Client client.Client
}

func (i *WorkloadAvailablePredicate) Create(e event.CreateEvent) bool {
	w, err := workload.ObjectToWorkload(e.Object)
	if err != nil {
		return false
	}

	instrumented, _ := sourceutils.IsObjectInstrumentedBySource(context.Background(), i.Client, e.Object)
	if !instrumented || !workload.IsObjectLabeledForInstrumentation(e.Object) {
		return false
	}

	return w.AvailableReplicas() > 0
}

func (i *WorkloadAvailablePredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil {
		return false
	}
	if e.ObjectNew == nil {
		return false
	}
	// filter our own namespace
	if e.ObjectNew.GetNamespace() == env.GetCurrentNamespace() {
		return false
	}

	wOld, err := workload.ObjectToWorkload(e.ObjectOld)
	if err != nil {
		return false
	}

	wNew, err := workload.ObjectToWorkload(e.ObjectNew)
	if err != nil {
		return false
	}

	instrumented, _ := sourceutils.IsObjectInstrumentedBySource(context.Background(), i.Client, e.ObjectNew)
	if !instrumented || !workload.IsObjectLabeledForInstrumentation(e.ObjectNew) {
		return false
	}

	newReplicas := wNew.AvailableReplicas()
	oldReplicas := wOld.AvailableReplicas()

	// 1. workload has available (running) replicas
	if newReplicas > 0 {
		return true
	}

	// 2. replicas became available
	replicasBecameAvailable := (oldReplicas == 0) && (newReplicas > 0)
	return replicasBecameAvailable
}

func (i *WorkloadAvailablePredicate) Delete(e event.DeleteEvent) bool {
	// no need to calculate runtime details for deleted workloads
	return false
}

func (i *WorkloadAvailablePredicate) Generic(e event.GenericEvent) bool {
	// not sure when exactly this would be called, but we don't need to handle it
	return false
}
