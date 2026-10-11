package status

import (
	"testing"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/model"
	agentInjectionEnabledStatus "github.com/odigos-io/odigos/status/instrumentationconfig/generated"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func agentEnabledCondition(reason string, message string) metav1.Condition {
	return metav1.Condition{
		Type:    agentInjectionEnabledStatus.AgentEnabledType,
		Reason:  reason,
		Message: message,
	}
}

// This condition is one of the signals the overall workload Odigos health
// aggregates, so its severity decides whether a workload that odigos decided
// not to instrument reads as a problem or as an intentional state.
func TestCalculateAgentInjectionEnabledStatusSeverityPerReason(t *testing.T) {
	tests := []struct {
		reason     agentInjectionEnabledStatus.AgentEnabledReason
		wantStatus model.DesiredStateProgress
	}{
		{reason: agentInjectionEnabledStatus.AgentEnabledReasonEnabledSuccessfully, wantStatus: model.DesiredStateProgressSuccess},
		{reason: agentInjectionEnabledStatus.AgentEnabledReasonEnabledWithOtherAgents, wantStatus: model.DesiredStateProgressNotice},
		{reason: agentInjectionEnabledStatus.AgentEnabledReasonWaitingForNodeCollector, wantStatus: model.DesiredStateProgressWaiting},
		{reason: agentInjectionEnabledStatus.AgentEnabledReasonWaitingForRuntimeInspection, wantStatus: model.DesiredStateProgressIrrelevant},
		{reason: agentInjectionEnabledStatus.AgentEnabledReasonIgnoredContainer, wantStatus: model.DesiredStateProgressDisabled},
		{reason: agentInjectionEnabledStatus.AgentEnabledReasonNoAvailableAgent, wantStatus: model.DesiredStateProgressUnsupported},
		{reason: agentInjectionEnabledStatus.AgentEnabledReasonNoCollectedSignals, wantStatus: model.DesiredStateProgressNotice},
	}

	for _, tt := range tests {
		t.Run(string(tt.reason), func(t *testing.T) {
			ic := &v1alpha1.InstrumentationConfig{}
			ic.Status.Conditions = []metav1.Condition{agentEnabledCondition(string(tt.reason), "condition message")}

			got := CalculateAgentInjectionEnabledStatus(ic)
			if got == nil {
				t.Fatal("expected a status, got nil")
			}
			if got.Name != agentInjectionEnabledStatus.AgentEnabledType {
				t.Fatalf("expected name %q, got %q", agentInjectionEnabledStatus.AgentEnabledType, got.Name)
			}
			if got.Status != tt.wantStatus {
				t.Fatalf("expected status %q, got %q", tt.wantStatus, got.Status)
			}
			// The UI keys off the catalog title, not the raw condition reason.
			catalogReason, ok := agentInjectionEnabledStatus.AgentEnabledReasonByName(string(tt.reason))
			if !ok {
				t.Fatalf("reason %q is missing from the status catalog", tt.reason)
			}
			if reasonOf(t, got) != catalogReason.Title {
				t.Fatalf("expected reason %q, got %q", catalogReason.Title, reasonOf(t, got))
			}
			if got.Message != "condition message" {
				t.Fatalf("expected the condition message, got %q", got.Message)
			}
		})
	}
}

func TestCalculateAgentInjectionEnabledStatusNoCondition(t *testing.T) {
	tests := []struct {
		name string
		ic   *v1alpha1.InstrumentationConfig
	}{
		{name: "nil instrumentation config", ic: nil},
		{name: "no conditions at all", ic: &v1alpha1.InstrumentationConfig{}},
		{
			name: "only conditions of other types",
			ic: &v1alpha1.InstrumentationConfig{
				Status: v1alpha1.InstrumentationConfigStatus{
					Conditions: []metav1.Condition{{Type: "RuntimeDetection", Reason: "DetectedSuccessfully", Message: "detected"}},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateAgentInjectionEnabledStatus(tt.ic)

			if tt.ic == nil {
				if got != nil {
					t.Fatalf("expected nil for a workload without an instrumentation config, got %+v", got)
				}
				return
			}

			if got == nil {
				t.Fatal("expected a status, got nil")
			}
			if got.Status != model.DesiredStateProgressUnknown {
				t.Fatalf("expected status %q, got %q", model.DesiredStateProgressUnknown, got.Status)
			}
			if got.ReasonEnum != nil {
				t.Fatalf("expected no reason, got %q", *got.ReasonEnum)
			}
			if got.Message != "no status found for agent injection enabled" {
				t.Fatalf("unexpected message %q", got.Message)
			}
		})
	}
}

// A reason written by a newer instrumentor than this UI is reported verbatim
// with an unknown severity instead of being mapped to a wrong one.
func TestCalculateAgentInjectionEnabledStatusUnknownReason(t *testing.T) {
	ic := &v1alpha1.InstrumentationConfig{}
	ic.Status.Conditions = []metav1.Condition{agentEnabledCondition("SomeReasonAddedLater", "a message from the future")}

	got := CalculateAgentInjectionEnabledStatus(ic)
	if got == nil {
		t.Fatal("expected a status, got nil")
	}
	if got.Status != model.DesiredStateProgressUnknown {
		t.Fatalf("expected status %q, got %q", model.DesiredStateProgressUnknown, got.Status)
	}
	if reasonOf(t, got) != "SomeReasonAddedLater" {
		t.Fatalf("expected the raw reason, got %q", reasonOf(t, got))
	}
	if got.Message != "a message from the future" {
		t.Fatalf("unexpected message %q", got.Message)
	}
}

// The instrumentor may leave the condition message empty, in which case the
// catalog's default message is what the user sees.
func TestCalculateAgentInjectionEnabledStatusFallsBackToTheCatalogMessage(t *testing.T) {
	catalogReason, ok := agentInjectionEnabledStatus.AgentEnabledReasonByName(string(agentInjectionEnabledStatus.AgentEnabledReasonNoAvailableAgent))
	if !ok {
		t.Fatal("expected the NoAvailableAgent reason in the status catalog")
	}

	ic := &v1alpha1.InstrumentationConfig{}
	ic.Status.Conditions = []metav1.Condition{
		agentEnabledCondition(string(agentInjectionEnabledStatus.AgentEnabledReasonNoAvailableAgent), ""),
	}

	got := CalculateAgentInjectionEnabledStatus(ic)
	if got.Message != catalogReason.Message {
		t.Fatalf("expected the catalog message %q, got %q", catalogReason.Message, got.Message)
	}
	if got.Message == "" {
		t.Fatal("expected a non empty message")
	}
}

// Action items are what render the recovery buttons, so they have to survive
// the conversion at the workload level too, not only per container.
func TestCalculateAgentInjectionEnabledStatusCarriesActionItems(t *testing.T) {
	ic := &v1alpha1.InstrumentationConfig{}
	ic.Status.Conditions = []metav1.Condition{
		agentEnabledCondition(string(agentInjectionEnabledStatus.AgentEnabledReasonOtherAgentDetected), "other agent detected"),
	}

	got := CalculateAgentInjectionEnabledStatus(ic)
	if len(got.ActionItems) != 1 {
		t.Fatalf("expected one action item, got %d", len(got.ActionItems))
	}
	if got.ActionItems[0].Type != model.DesiredConditionActionItemTypeAllowConcurrentAgentsForContainer {
		t.Fatalf("unexpected action item type %q", got.ActionItems[0].Type)
	}
}

func TestCalculateAgentInjectionEnabledStatusForContainerWithoutAReason(t *testing.T) {
	t.Run("agent enabled", func(t *testing.T) {
		got := CalculateAgentInjectionEnabledStatusForContainer(&v1alpha1.ContainerAgentConfig{
			ContainerName: "app",
			AgentEnabled:  true,
		})
		if got == nil {
			t.Fatal("expected a status, got nil")
		}
		if got.Status != model.DesiredStateProgressSuccess {
			t.Fatalf("expected status %q, got %q", model.DesiredStateProgressSuccess, got.Status)
		}
		if reasonOf(t, got) != agentInjectionEnabledStatus.AgentEnabledEnabledSuccessfully.Title {
			t.Fatalf("expected reason %q, got %q", agentInjectionEnabledStatus.AgentEnabledEnabledSuccessfully.Title, reasonOf(t, got))
		}
		if got.Message != "agent injection enabled" {
			t.Fatalf("unexpected message %q", got.Message)
		}
	})

	t.Run("agent not enabled", func(t *testing.T) {
		// An unexplained "not enabled" is an odigos bug, not a user facing
		// state, so it is reported at the most severe level.
		got := CalculateAgentInjectionEnabledStatusForContainer(&v1alpha1.ContainerAgentConfig{
			ContainerName: "app",
			AgentEnabled:  false,
		})
		if got == nil {
			t.Fatal("expected a status, got nil")
		}
		if got.Status != model.DesiredStateProgressError {
			t.Fatalf("expected status %q, got %q", model.DesiredStateProgressError, got.Status)
		}
		if reasonOf(t, got) != "" {
			t.Fatalf("expected an empty reason, got %q", reasonOf(t, got))
		}
		if got.Message != "missing reason why agent is not enabled" {
			t.Fatalf("unexpected message %q", got.Message)
		}
	})
}

func TestCalculateAgentInjectionEnabledStatusForContainerUnknownReason(t *testing.T) {
	got := CalculateAgentInjectionEnabledStatusForContainer(&v1alpha1.ContainerAgentConfig{
		ContainerName:       "app",
		AgentEnabledReason:  v1alpha1.AgentEnabledReason("SomeReasonAddedLater"),
		AgentEnabledMessage: "a message from the future",
	})
	if got == nil {
		t.Fatal("expected a status, got nil")
	}
	if got.Status != model.DesiredStateProgressUnknown {
		t.Fatalf("expected status %q, got %q", model.DesiredStateProgressUnknown, got.Status)
	}
	if reasonOf(t, got) != "SomeReasonAddedLater" {
		t.Fatalf("expected the raw reason, got %q", reasonOf(t, got))
	}
	if got.Message != "a message from the future" {
		t.Fatalf("unexpected message %q", got.Message)
	}
}

func TestCalculateAgentInjectionEnabledStatusForContainerNilConfig(t *testing.T) {
	if got := CalculateAgentInjectionEnabledStatusForContainer(nil); got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}
}
