package status

import (
	"github.com/odigos-io/odigos/frontend/graph/computed"
	"github.com/odigos-io/odigos/frontend/graph/model"
)

const (
	PodContainerHealthStatus = "PodCotainerHealthK8s"
	PodHealthStatus          = "PodHealthK8s"
)

type PodContainerK8sHealthReason string

const (
	PodContainerK8sHealthReasonCrashLoopBackOff PodContainerK8sHealthReason = "CrashLoopBackOff"
	PodContainerK8sHealthReasonNotStarted       PodContainerK8sHealthReason = "NotStarted"
	PodContainerK8sHealthReasonNotReady         PodContainerK8sHealthReason = "NotReady"
	PodContainerK8sHealthReasonHealthy          PodContainerK8sHealthReason = "Healthy"
	PodContainerK8sHealthReasonUnknown          PodContainerK8sHealthReason = "Unknown"
)

func createPodHealthK8sStatus(reason PodContainerK8sHealthReason, message string, status model.DesiredStateProgress) *model.DesiredConditionStatus {
	reasonStr := string(reason)
	return &model.DesiredConditionStatus{
		Name:       PodHealthStatus,
		Status:     status,
		ReasonEnum: &reasonStr,
		Message:    message,
	}
}

func createPodContainerHealthK8sStatus(reason PodContainerK8sHealthReason, message string, status model.DesiredStateProgress) *model.DesiredConditionStatus {
	reasonStr := string(reason)
	return &model.DesiredConditionStatus{
		Name:       PodContainerHealthStatus,
		Status:     status,
		ReasonEnum: &reasonStr,
		Message:    message,
	}
}

// pod container is considered healthy if it is started, ready and not in crash loop back off.
func CalculatePodContainerK8sHealthStatus(container *computed.ComputedPodContainer) *model.DesiredConditionStatus {

	if container.IsCrashLoop {
		message := "container in crash loop back off: " + *container.WaitingReasonEnum
		return createPodContainerHealthK8sStatus(PodContainerK8sHealthReasonCrashLoopBackOff, message, model.DesiredStateProgressFailure)
	}

	if container.Started == nil || !*container.Started {
		return createPodContainerHealthK8sStatus(PodContainerK8sHealthReasonNotStarted, "container has not started yet", model.DesiredStateProgressWaiting)
	}

	if !container.IsReady {
		return createPodContainerHealthK8sStatus(PodContainerK8sHealthReasonNotReady, "container is not ready yet", model.DesiredStateProgressWaiting)
	}

	return createPodContainerHealthK8sStatus(PodContainerK8sHealthReasonHealthy, "container is healthy", model.DesiredStateProgressSuccess)
}

func getPodStatusMessageFromReason(reason PodContainerK8sHealthReason) string {
	switch reason {
	case PodContainerK8sHealthReasonHealthy:
		return "all containers in pod are reported healthy in kubernetes"
	case PodContainerK8sHealthReasonNotStarted:
		return "some containers in pod are not started yet"
	case PodContainerK8sHealthReasonNotReady:
		return "some containers in pod are not ready yet"
	case PodContainerK8sHealthReasonCrashLoopBackOff:
		return "some containers in pod are in crash loop back off"
	default:
		return "unknown reason"
	}
}

func CalculatePodHealthK8sStatus(pod *computed.CachedPod, containersK8sHealthConditions []*model.DesiredConditionStatus) *model.DesiredConditionStatus {

	k8sPodHealthStatus := AggregateConditionsBySeverity(containersK8sHealthConditions)
	if k8sPodHealthStatus == nil || k8sPodHealthStatus.ReasonEnum == nil {
		// should not happen, all containers health status should be calculated.
		return createPodHealthK8sStatus(PodContainerK8sHealthReasonUnknown, "not able to determine health status for containers in pod", model.DesiredStateProgressError)
	}

	message := getPodStatusMessageFromReason(PodContainerK8sHealthReason(*k8sPodHealthStatus.ReasonEnum))
	return createPodHealthK8sStatus(PodContainerK8sHealthReason(*k8sPodHealthStatus.ReasonEnum), message, k8sPodHealthStatus.Status)
}
