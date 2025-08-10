package status

import (
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/model"
)

func runtimeDetectionStatusCondition(reason *string) model.DesiredStateProgress {
	if reason == nil {
		return model.DesiredStateProgressUnknown
	}
	switch v1alpha1.RuntimeDetectionReason(*reason) {
	case v1alpha1.RuntimeDetectionReasonDetectedSuccessfully:
		return model.DesiredStateProgressSuccess
	case v1alpha1.RuntimeDetectionReasonWaitingForDetection:
		return model.DesiredStateProgressWaiting
	case v1alpha1.RuntimeDetectionReasonNoRunningPods:
		return model.DesiredStateProgressPending
	case v1alpha1.RuntimeDetectionReasonError:
		return model.DesiredStateProgressError
	}
	return model.DesiredStateProgressUnknown
}

func GetRuntimeInspectionStatus(ic *v1alpha1.InstrumentationConfig) *model.DesiredConditionStatus {

	if ic == nil {
		return nil
	}

	var runtimeInfoReason *string
	var runtimeInfoMessage string = "runtime detection status not yet available"
	for _, c := range ic.Status.Conditions {
		if c.Type == v1alpha1.RuntimeDetectionStatusConditionType {
			runtimeInfoReason = &c.Reason
			runtimeInfoMessage = c.Message
		}
	}

	return &model.DesiredConditionStatus{
		Name:       v1alpha1.RuntimeDetectionStatusConditionType,
		Status:     runtimeDetectionStatusCondition(runtimeInfoReason),
		ReasonEnum: runtimeInfoReason,
		Message:    runtimeInfoMessage,
	}
}
