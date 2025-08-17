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
		return model.DesiredStateProgressIrrelevant // disregard this state if we are blocked on another one
	case v1alpha1.AgentEnabledReasonWaitingForNodeCollector:
		return model.DesiredStateProgressWaiting
	case v1alpha1.AgentEnabledReasonIgnoredContainer:
		return model.DesiredStateProgressDisabled // ignored per user configuration
	case v1alpha1.AgentEnabledReasonNoCollectedSignals:
		return model.DesiredStateProgressNotice // the setting make it so no signals are collected
	case v1alpha1.AgentEnabledReasonUnsupportedProgrammingLanguage:
		return model.DesiredStateProgressUnsupported
	case v1alpha1.AgentEnabledReasonNoAvailableAgent:
		return model.DesiredStateProgressUnsupported
	case v1alpha1.AgentEnabledReasonInjectionConflict:
		return model.DesiredStateProgressUnsupported // cannot inject agent under currnet conditions
	case v1alpha1.AgentEnabledReasonUnsupportedRuntimeVersion:
		return model.DesiredStateProgressUnsupported
	case v1alpha1.AgentEnabledReasonMissingDistroParameter:
		return model.DesiredStateProgressUnsupported
	case v1alpha1.AgentEnabledReasonOtherAgentDetected:
		return model.DesiredStateProgressNotice // other agent detected, need use action
	case v1alpha1.AgentEnabledReasonRuntimeDetailsUnavailable:
		return model.DesiredStateProgressIrrelevant // we should refactor this out and merge with AgentEnabledReasonWaitingForRuntimeInspection in future PR
	case v1alpha1.AgentEnabledReasonCrashLoopBackOff:
		return model.DesiredStateProgressNotice // crash loop back off detected, rollback applied and source is uninstrumented
	}
	return model.DesiredStateProgressUnknown
}

func CalculateAgentInjectionEnabledStatusForContainer(containerAgentConfig *v1alpha1.ContainerAgentConfig) *model.DesiredConditionStatus {
	if containerAgentConfig == nil {
		return nil
	}

	if containerAgentConfig.AgentEnabledReason == "" {
		if containerAgentConfig.AgentEnabled {
			reasonStr := string(v1alpha1.AgentEnabledReasonEnabledSuccessfully)
			return &model.DesiredConditionStatus{
				Name:       v1alpha1.AgentEnabledStatusConditionType,
				Status:     model.DesiredStateProgressSuccess,
				ReasonEnum: &reasonStr,
				Message:    "agent injection enabled",
			}
		} else {
			reasonStr := ""
			return &model.DesiredConditionStatus{
				Name:       v1alpha1.AgentEnabledStatusConditionType,
				Status:     model.DesiredStateProgressError,
				ReasonEnum: &reasonStr,
				Message:    "missing reason why agent is not enabled",
			}
		}
	}

	reasonStr := string(containerAgentConfig.AgentEnabledReason)
	return &model.DesiredConditionStatus{
		Name:       v1alpha1.AgentEnabledStatusConditionType,
		Status:     agentEnabledStatusCondition(&reasonStr),
		ReasonEnum: &reasonStr,
		Message:    containerAgentConfig.AgentEnabledMessage,
	}
}

func CalculateAgentInjectionEnabledStatus(ic *v1alpha1.InstrumentationConfig) *model.DesiredConditionStatus {

	if ic == nil {
		return nil
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
