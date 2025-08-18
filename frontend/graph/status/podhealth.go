package status

import (
	"github.com/odigos-io/odigos/frontend/graph/computed"
	"github.com/odigos-io/odigos/frontend/graph/model"
)

const (
	PodContainerHealthStatus = "PodCotainerHealth"
	PodHealthStatus          = "PodHealth"
)

type PodContainerHealthReason string

const (
	PodContainerHealthReasonCrashLoopBackOff PodContainerHealthReason = "CrashLoopBackOff"
	PodContainerHealthReasonNotStarted       PodContainerHealthReason = "NotStarted"
	PodContainerHealthReasonNotReady         PodContainerHealthReason = "NotReady"
	PodContainerHealthReasonHealthy          PodContainerHealthReason = "Healthy"
	PodContainerHealthReasonUnknown          PodContainerHealthReason = "Unknown"
)

// pod container is considered healthy if it is started, ready and not in crash loop back off.
func CalculatePodContainerHealthStatus(container *computed.ComputedPodContainer) *model.DesiredConditionStatus {

	if container.IsCrashLoop {
		reasonStr := string(PodContainerHealthReasonCrashLoopBackOff)
		return &model.DesiredConditionStatus{
			Name:       PodContainerHealthStatus,
			Status:     model.DesiredStateProgressFailure,
			ReasonEnum: &reasonStr,
			Message:    "container in crash loop back off: " + *container.WaitingReasonEnum,
		}
	}

	if container.Started == nil || !*container.Started {
		reasonStr := string(PodContainerHealthReasonNotStarted)
		return &model.DesiredConditionStatus{
			Name:       PodContainerHealthStatus,
			Status:     model.DesiredStateProgressWaiting,
			ReasonEnum: &reasonStr,
			Message:    "container has not started yet",
		}

	}

	if !container.IsReady {
		reasonStr := string(PodContainerHealthReasonNotReady)
		return &model.DesiredConditionStatus{
			Name:       PodContainerHealthStatus,
			Status:     model.DesiredStateProgressWaiting,
			ReasonEnum: &reasonStr,
			Message:    "container is not ready yet",
		}
	}

	reasonStr := string(PodContainerHealthReasonHealthy)
	return &model.DesiredConditionStatus{
		Name:       PodContainerHealthStatus,
		Status:     model.DesiredStateProgressSuccess,
		ReasonEnum: &reasonStr,
		Message:    "container is healthy",
	}
}
