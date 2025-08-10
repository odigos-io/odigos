package status

import (
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/model"
)

func agentEnabledStatusCondition(reason *string) model.DesiredStateProgress {
	if reason == nil {
		return model.DesiredStateProgressUnknown
	}
	switch v1alpha1.AgentEnabledReason(*reason) {
	case v1alpha1.AgentEnabledReasonEnabledSuccessfully:
		return model.DesiredStateProgressSuccess
	case v1alpha1.AgentEnabledReasonWaitingForRuntimeInspection:
		return model.DesiredStateProgressWaiting
	case v1alpha1.AgentEnabledReasonWaitingForNodeCollector:
		return model.DesiredStateProgressWaiting
	case v1alpha1.AgentEnabledReasonIgnoredContainer:
		return model.DesiredStateProgressDisabled
	case v1alpha1.AgentEnabledReasonNoCollectedSignals:
		return model.DesiredStateProgressNotice
	case v1alpha1.AgentEnabledReasonUnsupportedProgrammingLanguage:
		return model.DesiredStateProgressDisabled
	case v1alpha1.AgentEnabledReasonNoAvailableAgent:
		return model.DesiredStateProgressDisabled
	case v1alpha1.AgentEnabledReasonInjectionConflict:
		return model.DesiredStateProgressDisabled
	case v1alpha1.AgentEnabledReasonUnsupportedRuntimeVersion:
		return model.DesiredStateProgressDisabled
	case v1alpha1.AgentEnabledReasonMissingDistroParameter:
		return model.DesiredStateProgressError
	case v1alpha1.AgentEnabledReasonOtherAgentDetected:
		return model.DesiredStateProgressNotice
	case v1alpha1.AgentEnabledReasonRuntimeDetailsUnavailable:
		return model.DesiredStateProgressPending
	case v1alpha1.AgentEnabledReasonCrashLoopBackOff:
		return model.DesiredStateProgressError
	}
	return model.DesiredStateProgressUnknown
}

func GetAgentInjectionEnabledStatusForContainer(containerAgentConfig *v1alpha1.ContainerAgentConfig) *model.DesiredConditionStatus {
	if containerAgentConfig == nil {
		return nil
	}
	reasonStr := string(containerAgentConfig.AgentEnabledReason)
	return &model.DesiredConditionStatus{
		Name:       v1alpha1.AgentEnabledStatusConditionType,
		Status:     agentEnabledStatusCondition(&reasonStr),
		ReasonEnum: &reasonStr,
		Message:    containerAgentConfig.AgentEnabledMessage,
	}
}

func GetAgentInjectionEnabledStatus(ic *v1alpha1.InstrumentationConfig) *model.DesiredConditionStatus {

	if ic == nil {
		reasonStr := string(v1alpha1.RuntimeDetectionReasonNotMakredForInstrumentation)
		return &model.DesiredConditionStatus{
			Name:       v1alpha1.RuntimeDetectionStatusConditionType,
			Status:     model.DesiredStateProgressDisabled,
			ReasonEnum: &reasonStr,
			Message:    "workload is not marked for instrumentation, agent is not enabled for this workload",
		}
	}

	for _, c := range ic.Status.Conditions {
		if c.Type == v1alpha1.AgentEnabledStatusConditionType {
			conditionStatus := agentEnabledStatusCondition(&c.Reason)
			return &model.DesiredConditionStatus{
				Name:       c.Type,
				Status:     conditionStatus,
				ReasonEnum: &c.Reason,
				Message:    c.Message,
			}
		}
	}

	return &model.DesiredConditionStatus{
		Name:       v1alpha1.AgentEnabledStatusConditionType,
		Status:     model.DesiredStateProgressUnknown,
		ReasonEnum: nil,
		Message:    "no status found for agent injection enabled",
	}
}
