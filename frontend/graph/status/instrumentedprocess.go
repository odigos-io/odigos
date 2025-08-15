package status

import (
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/model"
)

const (
	// for a single process
	ProcessHealthStatusName = "ProcessHealthStatus"

	// for all processes in a container / pod / workload
	ProcessesHealthStatusName = "ProcessesHealthStatus"
)

type ProcessHealthStatusReason string

const (
	ProcessHealthStatusReasonHealthy   ProcessHealthStatusReason = "Healthy"
	ProcessHealthStatusReasonUnhealthy ProcessHealthStatusReason = "Unhealthy"
	ProcessHealthStatusReasonStarting  ProcessHealthStatusReason = "Starting"
)

type ProcessesHealthStatusReason string

const (
	ProcessesHealthStatusReasonAllHealthy    ProcessesHealthStatusReason = "AllHealthy"
	ProcessesHealthStatusReasonSomeUnhealthy ProcessesHealthStatusReason = "SomeUnhealthy"
	ProcessesHealthStatusReasonStarting      ProcessesHealthStatusReason = "Starting"    // some processes are in starting state
	ProcessesHealthStatusReasonUnsupported   ProcessesHealthStatusReason = "Unsupported" // when the distro does not record health status
	ProcessesHealthStatusReasonNoProcesses   ProcessesHealthStatusReason = "NoProcesses" // no instrumented processes when expected
)

func CalculateProcessHealthStatus(instrumentationInstance *v1alpha1.InstrumentationInstance) *model.DesiredConditionStatus {

	if instrumentationInstance == nil {
		return nil
	}

	if instrumentationInstance.Status.Healthy == nil {
		reasonStr := string(ProcessHealthStatusReasonStarting)
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
