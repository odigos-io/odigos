package pod

import v1 "k8s.io/api/core/v1"

// isCronJobPod returns true if the Pod is ultimately controlled by a CronJob.
// In practice it’s enough to check for a Job controller: only Jobs/CronJobs
// produce Pods with Started == nil.
func isCronJobPod(pod *v1.Pod) bool {
	for _, ref := range pod.OwnerReferences {
		if ref.Controller != nil && *ref.Controller && (ref.Kind == "Job" || ref.Kind == "CronJob") {
			return true
		}
	}
	return false
}

func AllContainersReady(pod *v1.Pod) bool {
	// If pod has no containers, return false as we can't determine readiness
	if len(pod.Status.ContainerStatuses) == 0 {
		return false
	}
	// Check if pod is in Running phase.
	if pod.Status.Phase != v1.PodRunning {
		return false
	}

	skipStarted := isCronJobPod(pod)

	// Iterate over all containers in the pod
	// Return false if any container is:
	// 1. Not Ready
	// 2. Started is nil or false
	for i := range pod.Status.ContainerStatuses {
		containerStatus := &pod.Status.ContainerStatuses[i]

		if !containerStatus.Ready {
			return false
		}

		// For long-running pods (RestartPolicy=Always) ensure the container
		// has actually entered the running state (`Started == true`).
		if !skipStarted && (containerStatus.Started == nil || !*containerStatus.Started) {
			return false
		}
	}
	return true
}

func IsPodDeleting(pod *v1.Pod) bool {
	return pod.DeletionTimestamp != nil && !pod.DeletionTimestamp.IsZero()
}
