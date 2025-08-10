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
		return model.DesiredStateProgressError
	case v1alpha1.WorkloadRolloutReasonPreviousRolloutOngoing:
		return model.DesiredStateProgressWaiting
	case v1alpha1.WorkloadRolloutReasonDisabled:
		return model.DesiredStateProgressDisabled
	case v1alpha1.WorkloadRolloutReasonWaitingForRestart:
		return model.DesiredStateProgressPending
	}
	return model.DesiredStateProgressUnknown
}

func GetRolloutStatus(ic *v1alpha1.InstrumentationConfig) *model.DesiredConditionStatus {
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
