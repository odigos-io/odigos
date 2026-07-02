package status

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/computed"
	"github.com/odigos-io/odigos/frontend/graph/model"
)

func enabledConfig() *computed.AutoRollbackConfig {
	return &computed.AutoRollbackConfig{
		Enabled:         true,
		GraceTime:       5 * time.Minute,
		StabilityWindow: time.Hour,
	}
}

func reasonOf(t *testing.T, s *model.DesiredConditionStatus) string {
	t.Helper()
	if s == nil || s.ReasonEnum == nil {
		return ""
	}
	return *s.ReasonEnum
}

func TestCalculateAutoRollbackStatus_NilConfig_ReturnsNil(t *testing.T) {
	if got := CalculateAutoRollbackStatus(nil, enabledConfig()); got != nil {
		t.Fatalf("expected nil for nil InstrumentationConfig, got %+v", got)
	}
}

// A rolled-back workload always has AgentInjectionEnabled=false (the rollout
// controller disables injection when it sets RollbackOccurred=true). The
// RollbackOccurred status must win so the UI surfaces the recovery action.
func TestCalculateAutoRollbackStatus_RollbackOccurred_WhileAgentInjectionDisabled(t *testing.T) {
	ic := &odigosv1alpha1.InstrumentationConfig{}
	ic.Spec.AgentInjectionEnabled = false
	ic.Status.RollbackOccurred = true

	got := CalculateAutoRollbackStatus(ic, enabledConfig())

	if got == nil {
		t.Fatal("expected a status, got nil")
	}
	if reasonOf(t, got) != string(AutoRollbackReasonRollbackOccurred) {
		t.Fatalf("expected reason %q, got %q", AutoRollbackReasonRollbackOccurred, reasonOf(t, got))
	}
	// Notice (not Irrelevant) is what the UI maps to a visible condition + recovery button.
	if got.Status != model.DesiredStateProgressNotice {
		t.Fatalf("expected status %q, got %q", model.DesiredStateProgressNotice, got.Status)
	}
}

// Even if auto-rollback was later disabled in config, a workload that already
// rolled back still needs the recovery action surfaced.
func TestCalculateAutoRollbackStatus_RollbackOccurred_WhileConfigDisabled(t *testing.T) {
	ic := &odigosv1alpha1.InstrumentationConfig{}
	ic.Spec.AgentInjectionEnabled = false
	ic.Status.RollbackOccurred = true

	got := CalculateAutoRollbackStatus(ic, &computed.AutoRollbackConfig{Enabled: false})

	if reasonOf(t, got) != string(AutoRollbackReasonRollbackOccurred) {
		t.Fatalf("expected reason %q, got %q", AutoRollbackReasonRollbackOccurred, reasonOf(t, got))
	}
	if got.Status != model.DesiredStateProgressNotice {
		t.Fatalf("expected status %q, got %q", model.DesiredStateProgressNotice, got.Status)
	}
}

func TestCalculateAutoRollbackStatus_ConfigDisabled_NoRollback(t *testing.T) {
	ic := &odigosv1alpha1.InstrumentationConfig{}
	ic.Spec.AgentInjectionEnabled = true

	got := CalculateAutoRollbackStatus(ic, &computed.AutoRollbackConfig{Enabled: false})

	if reasonOf(t, got) != string(AutoRollbackReasonDisabled) {
		t.Fatalf("expected reason %q, got %q", AutoRollbackReasonDisabled, reasonOf(t, got))
	}
}

func TestCalculateAutoRollbackStatus_AgentNotEnabled_NoRollback(t *testing.T) {
	ic := &odigosv1alpha1.InstrumentationConfig{}
	ic.Spec.AgentInjectionEnabled = false

	got := CalculateAutoRollbackStatus(ic, enabledConfig())

	if reasonOf(t, got) != string(AutoRollbackReasonAgentNotEnabled) {
		t.Fatalf("expected reason %q, got %q", AutoRollbackReasonAgentNotEnabled, reasonOf(t, got))
	}
}

func TestCalculateAutoRollbackStatus_WaitingForRollout(t *testing.T) {
	ic := &odigosv1alpha1.InstrumentationConfig{}
	ic.Spec.AgentInjectionEnabled = true
	ic.Status.InstrumentationTime = nil

	got := CalculateAutoRollbackStatus(ic, enabledConfig())

	if reasonOf(t, got) != string(AutoRollbackReasonWaitingForRollout) {
		t.Fatalf("expected reason %q, got %q", AutoRollbackReasonWaitingForRollout, reasonOf(t, got))
	}
}

func TestCalculateAutoRollbackStatus_Evaluating_WithinStabilityWindow(t *testing.T) {
	ic := &odigosv1alpha1.InstrumentationConfig{}
	ic.Spec.AgentInjectionEnabled = true
	now := metav1.Now()
	ic.Status.InstrumentationTime = &now

	got := CalculateAutoRollbackStatus(ic, enabledConfig())

	if reasonOf(t, got) != string(AutoRollbackReasonEvaluating) {
		t.Fatalf("expected reason %q, got %q", AutoRollbackReasonEvaluating, reasonOf(t, got))
	}
}

func TestCalculateAutoRollbackStatus_Stable_AfterStabilityWindow(t *testing.T) {
	ic := &odigosv1alpha1.InstrumentationConfig{}
	ic.Spec.AgentInjectionEnabled = true
	old := metav1.NewTime(time.Now().Add(-2 * time.Hour))
	ic.Status.InstrumentationTime = &old

	got := CalculateAutoRollbackStatus(ic, enabledConfig())

	if reasonOf(t, got) != string(AutoRollbackReasonStable) {
		t.Fatalf("expected reason %q, got %q", AutoRollbackReasonStable, reasonOf(t, got))
	}
}
