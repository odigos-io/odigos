package status

import (
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/model"
)

const (
	ProcessHealthStatusName = "ProcessHealthStatus"
)

type ProcessHealthStatusReason string

const (
	ProcessHealthStatusReasonHealthy   ProcessHealthStatusReason = "Healthy"
	ProcessHealthStatusReasonUnhealthy ProcessHealthStatusReason = "Unhealthy"
	ProcessHealthStatusReasonUnknown   ProcessHealthStatusReason = "Unknown"
)

func CalculateProcessHealthStatus(instrumentationInstance *v1alpha1.InstrumentationInstance) *model.DesiredConditionStatus {

	if instrumentationInstance == nil {
		return nil
	}

	if instrumentationInstance.Status.Healthy == nil {
		reasonStr := string(ProcessHealthStatusReasonUnknown)
		return &model.DesiredConditionStatus{
			Name:       ProcessHealthStatusName,
			Status:     model.DesiredStateProgressUnknown,
			ReasonEnum: &reasonStr,
			Message:    instrumentationInstance.Status.Message,
		}
	}

	healthy := *instrumentationInstance.Status.Healthy
	var reasonStr string
	var state model.DesiredStateProgress
	if healthy {
		reasonStr = string(ProcessHealthStatusReasonHealthy)
		state = model.DesiredStateProgressSuccess
	} else {
		reasonStr = string(ProcessHealthStatusReasonUnhealthy)
		state = model.DesiredStateProgressFailure
	}

	return &model.DesiredConditionStatus{
		Name:       ProcessHealthStatusName,
		Status:     state,
		ReasonEnum: &reasonStr,
		Message:    instrumentationInstance.Status.Message,
	}
}
