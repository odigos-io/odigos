package predicate

import (
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"

	k8spod "github.com/odigos-io/odigos/k8sutils/pkg/pod"
)

// AllContainersBecomeReadyPredicate is a predicate that checks if all containers in a pod are becoming ready.
//
// For Create events, it returns false
// For Update events, it returns true if the new pod has all containers ready and started,
// and the old pod had at least one container not ready or not started.
// For Delete events, it returns false.
type AllContainersBecomeReadyPredicate struct{}

func (p *AllContainersBecomeReadyPredicate) Create(e event.CreateEvent) bool {
	return false
}

func (p *AllContainersBecomeReadyPredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil || e.ObjectNew == nil {
		return false
	}

	oldPod, oldOk := e.ObjectOld.(*corev1.Pod)
	newPod, newOk := e.ObjectNew.(*corev1.Pod)

	if !oldOk || !newOk {
		return false
	}

	// First check if all containers in newPod are ready and started
	allNewContainersReady := k8spod.AllContainersReady(newPod)
	isDeleting := k8spod.IsPodDeleting(newPod)

	// If new containers aren't all ready, return false
	if !allNewContainersReady || isDeleting {
		return false
	}

	// Now check if any container in oldPod was not ready or not started
	allOldContainersReady := k8spod.AllContainersReady(oldPod)

	// Return true only if old pods had at least one container not ready/not started
	// and new pod has all containers ready/started
	return !allOldContainersReady && allNewContainersReady
}

func (p *AllContainersBecomeReadyPredicate) Delete(e event.DeleteEvent) bool {
	return false
}

func (p *AllContainersBecomeReadyPredicate) Generic(e event.GenericEvent) bool {
	return false
}
