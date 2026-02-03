package predicate

import (
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"

	k8scontainer "github.com/odigos-io/odigos/k8sutils/pkg/container"
)

// AllContainersReadyPredicate is a predicate that checks if all containers in a pod are ready or becoming ready.
//
// For Create events, it returns true if the pod is in Running phase and all containers are ready.
// For Update events, it returns true if the new pod has all containers ready and started,
// and the old pod had at least one container not ready or not started.
// For Delete events, it returns false.
type AllContainersReadyPredicate struct{}

func (p *AllContainersReadyPredicate) Create(e event.CreateEvent) bool {
	if e.Object == nil {
		return false
	}

	pod, ok := e.Object.(*corev1.Pod)
	if !ok {
		return false
	}

	allContainersReady := k8scontainer.AllContainersReady(pod)
	isDeleting := pod.DeletionTimestamp != nil && !pod.DeletionTimestamp.IsZero()
	// If all containers are not ready, return false.
	// Otherwise, return true
	return allContainersReady && !isDeleting
}

func (p *AllContainersReadyPredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil || e.ObjectNew == nil {
		return false
	}

	oldPod, oldOk := e.ObjectOld.(*corev1.Pod)
	newPod, newOk := e.ObjectNew.(*corev1.Pod)

	if !oldOk || !newOk {
		return false
	}

	// First check if all containers in newPod are ready and started
	allNewContainersReady := k8scontainer.AllContainersReady(newPod)
	isDeleting := newPod.DeletionTimestamp != nil && !newPod.DeletionTimestamp.IsZero()

	// If new containers aren't all ready, return false
	if !allNewContainersReady || isDeleting {
		return false
	}

	// Now check if any container in oldPod was not ready or not started
	allOldContainersReady := k8scontainer.AllContainersReady(oldPod)

	// Return true only if old pods had at least one container not ready/not started
	// and new pod has all containers ready/started
	return !allOldContainersReady && allNewContainersReady
}

func (p *AllContainersReadyPredicate) Delete(e event.DeleteEvent) bool {
	return false
}

func (p *AllContainersReadyPredicate) Generic(e event.GenericEvent) bool {
	return false
}
