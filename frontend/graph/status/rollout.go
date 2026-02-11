package status

import (
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/model"
)

func workloadRolloutStatusCondition(reason *string) model.DesiredStateProgress {
	if reason == nil {
		return model.DesiredStateProgressUnknown
	}
	switch v1alpha1.WorkloadRolloutReason(*reason) {

	case v1alpha1.WorkloadRolloutReasonTriggeredSuccessfully:
		return model.DesiredStateProgressSuccess
	case v1alpha1.WorkloadRolloutReasonFailedToPatch:
		return model.DesiredStateProgressFailure
	case v1alpha1.WorkloadRolloutReasonPreviousRolloutOngoing:
		return model.DesiredStateProgressWaiting // short period until the rollout is complete
	case v1alpha1.WorkloadRolloutReasonRolloutFinished:
		return model.DesiredStateProgressSuccess // rollout is finished
	case v1alpha1.WorkloadRolloutReasonDisabled:
		return model.DesiredStateProgressUnknown // rollout is disabled in config, or agent was not enabled etc. need to refine those cases in the future.
	case v1alpha1.WorkloadRolloutReasonNotRequired:
		return model.DesiredStateProgressIrrelevant // rollout is not required
	case v1alpha1.WorkloadRolloutReasonWaitingForRestart:
		return model.DesiredStateProgressIrrelevant // rollout state is irrelevant in this case (no rollout for cronjob for example)
	case v1alpha1.WorkloadRolloutReasonWaitingInQueue:
		return model.DesiredStateProgressWaiting // waiting for other workloads to complete their rollouts
	}

	return model.DesiredStateProgressUnknown
}

func CalculateRolloutStatus(ic *v1alpha1.InstrumentationConfig) *model.DesiredConditionStatus {
	if ic == nil {
		return nil
	}

	for _, c := range ic.Status.Conditions {
		if c.Type == v1alpha1.WorkloadRolloutStatusConditionType {
			conditionStatus := workloadRolloutStatusCondition(&c.Reason)
			return &model.DesiredConditionStatus{
				Name:       c.Type,
				Status:     conditionStatus,
				ReasonEnum: &c.Reason,
				Message:    c.Message,
			}
		}
	}

	return nil
}
