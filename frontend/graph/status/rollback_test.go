package status

import (
	"testing"
	"time"

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
