package status

import (
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/status"
	agentInjectionEnabledStatus "github.com/odigos-io/odigos/status/instrumentationconfig/generated"
)

func CalculateAgentInjectionEnabledStatusForContainer(containerAgentConfig *v1alpha1.ContainerAgentConfig) *model.DesiredConditionStatus {
	if containerAgentConfig == nil {
		return nil
	}

	if containerAgentConfig.AgentEnabledReason == "" {
		if containerAgentConfig.AgentEnabled {
			return agentInjectionEnabledReasonToStatus(agentInjectionEnabledStatus.AgentEnabledEnabledSuccessfully, "agent injection enabled")
		}
		reasonStr := ""
		return &model.DesiredConditionStatus{
			Name:       agentInjectionEnabledStatus.AgentEnabledType,
			Status:     model.DesiredStateProgressError,
			ReasonEnum: &reasonStr,
			Message:    "missing reason why agent is not enabled",
		}
	}

	r, ok := agentInjectionEnabledStatus.AgentEnabledReasonByName(string(containerAgentConfig.AgentEnabledReason))
	if !ok {
		reasonStr := string(containerAgentConfig.AgentEnabledReason)
		return &model.DesiredConditionStatus{
			Name:       agentInjectionEnabledStatus.AgentEnabledType,
			Status:     model.DesiredStateProgressUnknown,
			ReasonEnum: &reasonStr,
			Message:    containerAgentConfig.AgentEnabledMessage,
		}
	}

	return agentInjectionEnabledReasonToStatus(r, containerAgentConfig.AgentEnabledMessage)
}

func CalculateAgentInjectionEnabledStatus(ic *v1alpha1.InstrumentationConfig) *model.DesiredConditionStatus {
	if ic == nil {
		return nil
	}

	for _, c := range ic.Status.Conditions {
		if c.Type != agentInjectionEnabledStatus.AgentEnabledType {
			continue
		}
		r, ok := agentInjectionEnabledStatus.AgentEnabledReasonByName(c.Reason)
		if !ok {
			return &model.DesiredConditionStatus{
				Name:       agentInjectionEnabledStatus.AgentEnabledType,
				Status:     model.DesiredStateProgressUnknown,
				ReasonEnum: &c.Reason,
				Message:    c.Message,
			}
		}

		return agentInjectionEnabledReasonToStatus(r, c.Message)
	}

	return &model.DesiredConditionStatus{
		Name:       agentInjectionEnabledStatus.AgentEnabledType,
		Status:     model.DesiredStateProgressUnknown,
		ReasonEnum: nil,
		Message:    "no status found for agent injection enabled",
	}
}

func agentInjectionEnabledReasonToStatus(r status.Reason, message string) *model.DesiredConditionStatus {
	if message == "" {
		message = r.Message
	}

	actionItems := make([]*model.DesiredConditionActionItem, 0, len(r.ActionItems))
	for _, actionItem := range r.ActionItems {
		actionItems = append(actionItems, &model.DesiredConditionActionItem{
			Type:       model.DesiredConditionActionItemType(actionItem.Type),
			ButtonText: actionItem.ButtonText,
		})
	}

	return &model.DesiredConditionStatus{
		Name:        agentInjectionEnabledStatus.AgentEnabledType,
		Status:      model.DesiredStateProgress(r.OdigosSeverity),
		ReasonEnum:  &r.Title,
		Message:     message,
		ActionItems: actionItems,
	}
}
