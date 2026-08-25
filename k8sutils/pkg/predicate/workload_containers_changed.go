package predicate

import (
	"slices"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	cr_predicate "sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
)

// WorkloadContainersChangedPredicate only allows update events in which the containers
// of the workload pod template changed (a container was added, removed or renamed).
//
// The instrumentation config of a workload holds one container override entry per container
// in the workload pod template, and consumers rely on that list to enumerate the containers
// of the workload, so it has to be recalculated when the pod template containers change.
//
// Only the container names are compared, since any other change to a container (image, env, etc)
// does not affect the set of containers odigos needs to track.
type WorkloadContainersChangedPredicate struct{}

func (i WorkloadContainersChangedPredicate) Create(e event.CreateEvent) bool {
	return false
}

func (i WorkloadContainersChangedPredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil || e.ObjectNew == nil {
		return false
	}

	oldContainerNames, err := workloadContainerNames(e.ObjectOld)
	if err != nil {
		return false
	}

	newContainerNames, err := workloadContainerNames(e.ObjectNew)
	if err != nil {
		return false
	}

	return !slices.Equal(oldContainerNames, newContainerNames)
}

func (i WorkloadContainersChangedPredicate) Delete(e event.DeleteEvent) bool {
	return false
}

func (i WorkloadContainersChangedPredicate) Generic(e event.GenericEvent) bool {
	return false
}

func workloadContainerNames(obj client.Object) ([]string, error) {
	workloadObj, err := workload.ObjectToWorkload(obj)
	if err != nil {
		return nil, err
	}

	containers := workloadObj.PodSpec().Containers
	names := make([]string, 0, len(containers))
	for i := range containers {
		names = append(names, containers[i].Name)
	}

	return names, nil
}

var _ cr_predicate.Predicate = &WorkloadContainersChangedPredicate{}
