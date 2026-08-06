package status

import (
	"testing"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/model"
	agentInjectionEnabledStatus "github.com/odigos-io/odigos/status/instrumentationconfig/generated"
)

func TestCalculateAgentInjectionEnabledStatusForContainerConcurrentAgentActions(t *testing.T) {
	tests := []struct {
		name               string
		reason             agentInjectionEnabledStatus.AgentEnabledReason
		expectedActionType model.DesiredConditionActionItemType
		expectedButtonText string
	}{
		{
			name:               "other agent detected",
			reason:             agentInjectionEnabledStatus.AgentEnabledReasonOtherAgentDetected,
			expectedActionType: model.DesiredConditionActionItemTypeAllowConcurrentAgentsForContainer,
			expectedButtonText: "Run Odigos With Other Agents",
		},
		{
			name:               "enabled with other agents",
			reason:             agentInjectionEnabledStatus.AgentEnabledReasonEnabledWithOtherAgents,
			expectedActionType: model.DesiredConditionActionItemTypeDisallowConcurrentAgentsForContainer,
			expectedButtonText: "Don't Run With Other Agents",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateAgentInjectionEnabledStatusForContainer(&v1alpha1.ContainerAgentConfig{
				AgentEnabledReason: v1alpha1.AgentEnabledReason(tt.reason),
			})
			if got == nil {
				t.Fatal("expected a status, got nil")
			}
			if len(got.ActionItems) != 1 {
				t.Fatalf("expected one action item, got %d", len(got.ActionItems))
			}
			if got.ActionItems[0].Type != tt.expectedActionType {
				t.Fatalf("expected action type %q, got %q", tt.expectedActionType, got.ActionItems[0].Type)
			}
			if got.ActionItems[0].ButtonText != tt.expectedButtonText {
				t.Fatalf("expected button text %q, got %q", tt.expectedButtonText, got.ActionItems[0].ButtonText)
			}
		})
	}
}
