package status

import (
	"testing"
	"time"

	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/computed"
	"github.com/odigos-io/odigos/frontend/graph/model"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func TestCalculateAutoRollbackStatus_NilInstrumentationConfig(t *testing.T) {
	if got := CalculateAutoRollbackStatus(nil, enabledConfig()); got != nil {
		t.Fatalf("expected nil for a workload without an instrumentation config, got %+v", got)
	}
}

// Only RollbackOccurred is a user visible notice; every other outcome is either
// informational or a transient wait. Anything else here would make the whole
// workload's health drop for a source that is behaving normally.
func TestCalculateAutoRollbackStatus(t *testing.T) {
	instrumentedAt := func(ago time.Duration) *metav1.Time {
		t := metav1.NewTime(time.Now().Add(-ago))
		return &t
	}

	tests := []struct {
		name        string
		ic          func() *odigosv1alpha1.InstrumentationConfig
		config      *computed.AutoRollbackConfig
		wantStatus  model.DesiredStateProgress
		wantReason  AutoRollbackReason
		wantMessage string
	}{
		{
			name: "disabled in the odigos configuration",
			ic: func() *odigosv1alpha1.InstrumentationConfig {
				ic := &odigosv1alpha1.InstrumentationConfig{}
				ic.Spec.AgentInjectionEnabled = true
				ic.Status.InstrumentationTime = instrumentedAt(2 * time.Hour)
				return ic
			},
			config:      &computed.AutoRollbackConfig{Enabled: false},
			wantStatus:  model.DesiredStateProgressIrrelevant,
			wantReason:  AutoRollbackReasonDisabled,
			wantMessage: "Auto rollback is disabled in the odigos configuration",
		},
		{
			// Config takes precedence over the agent state: with auto-rollback
			// switched off there is nothing to evaluate either way.
			name: "disabled in config while the agent is also not enabled",
			ic: func() *odigosv1alpha1.InstrumentationConfig {
				return &odigosv1alpha1.InstrumentationConfig{}
			},
			config:      &computed.AutoRollbackConfig{Enabled: false},
			wantStatus:  model.DesiredStateProgressIrrelevant,
			wantReason:  AutoRollbackReasonDisabled,
			wantMessage: "Auto rollback is disabled in the odigos configuration",
		},
		{
			name: "agent injection not enabled",
			ic: func() *odigosv1alpha1.InstrumentationConfig {
				ic := &odigosv1alpha1.InstrumentationConfig{}
				ic.Spec.AgentInjectionEnabled = false
				ic.Spec.PodManifestInjectionOptional = true
				return ic
			},
			config:      enabledConfig(),
			wantStatus:  model.DesiredStateProgressIrrelevant,
			wantReason:  AutoRollbackReasonAgentNotEnabled,
			wantMessage: "odigos agent is not set to run with this source, auto rollback is not applicable",
		},
		{
			// A "no restart required" agent never restarts the app, so there is
			// no crash window to watch.
			name: "optional pod manifest injection",
			ic: func() *odigosv1alpha1.InstrumentationConfig {
				ic := &odigosv1alpha1.InstrumentationConfig{}
				ic.Spec.AgentInjectionEnabled = true
				ic.Spec.PodManifestInjectionOptional = true
				ic.Status.InstrumentationTime = instrumentedAt(2 * time.Hour)
				return ic
			},
			config:      enabledConfig(),
			wantStatus:  model.DesiredStateProgressIrrelevant,
			wantReason:  AutoRollbackReasonNotApplicable,
			wantMessage: "no stability check for source that odigos agent is not restarting",
		},
		{
			name: "not rolled out yet",
			ic: func() *odigosv1alpha1.InstrumentationConfig {
				ic := &odigosv1alpha1.InstrumentationConfig{}
				ic.Spec.AgentInjectionEnabled = true
				return ic
			},
			config:      enabledConfig(),
			wantStatus:  model.DesiredStateProgressWaiting,
			wantReason:  AutoRollbackReasonWaitingForRollout,
			wantMessage: "source stability will be checked after the source is rolled out by odigos",
		},
		{
			name: "within the stability window",
			ic: func() *odigosv1alpha1.InstrumentationConfig {
				ic := &odigosv1alpha1.InstrumentationConfig{}
				ic.Spec.AgentInjectionEnabled = true
				ic.Status.InstrumentationTime = instrumentedAt(time.Minute)
				return ic
			},
			config:      enabledConfig(),
			wantStatus:  model.DesiredStateProgressSuccess,
			wantReason:  AutoRollbackReasonEvaluating,
			wantMessage: "evaluating pods stability for 1h0m0s",
		},
		{
			name: "past the stability window",
			ic: func() *odigosv1alpha1.InstrumentationConfig {
				ic := &odigosv1alpha1.InstrumentationConfig{}
				ic.Spec.AgentInjectionEnabled = true
				ic.Status.InstrumentationTime = instrumentedAt(2 * time.Hour)
				return ic
			},
			config:      enabledConfig(),
			wantStatus:  model.DesiredStateProgressSuccess,
			wantReason:  AutoRollbackReasonStable,
			wantMessage: "pods are stable after instrumentation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateAutoRollbackStatus(tt.ic(), tt.config)
			if got == nil {
				t.Fatal("expected a status, got nil")
			}
			if got.Name != RollbackStatus {
				t.Fatalf("expected name %q, got %q", RollbackStatus, got.Name)
			}
			if got.Status != tt.wantStatus {
				t.Fatalf("expected status %q, got %q", tt.wantStatus, got.Status)
			}
			if reasonOf(t, got) != string(tt.wantReason) {
				t.Fatalf("expected reason %q, got %q", tt.wantReason, reasonOf(t, got))
			}
			if got.Message != tt.wantMessage {
				t.Fatalf("expected message %q, got %q", tt.wantMessage, got.Message)
			}
		})
	}
}

// The stability window comes from the odigos configuration, so the reported
// window has to follow it rather than a hardcoded default.
func TestCalculateAutoRollbackStatus_StabilityWindowFromConfig(t *testing.T) {
	ic := &odigosv1alpha1.InstrumentationConfig{}
	ic.Spec.AgentInjectionEnabled = true
	instrumentationTime := metav1.NewTime(time.Now().Add(-10 * time.Minute))
	ic.Status.InstrumentationTime = &instrumentationTime

	config := &computed.AutoRollbackConfig{Enabled: true, GraceTime: 5 * time.Minute, StabilityWindow: 30 * time.Minute}

	got := CalculateAutoRollbackStatus(ic, config)
	if reasonOf(t, got) != string(AutoRollbackReasonEvaluating) {
		t.Fatalf("expected reason %q, got %q", AutoRollbackReasonEvaluating, reasonOf(t, got))
	}
	if got.Message != "evaluating pods stability for 30m0s" {
		t.Fatalf("unexpected message %q", got.Message)
	}

	// the same workload is stable once the shorter window has elapsed.
	config.StabilityWindow = time.Minute
	if got := CalculateAutoRollbackStatus(ic, config); reasonOf(t, got) != string(AutoRollbackReasonStable) {
		t.Fatalf("expected reason %q, got %q", AutoRollbackReasonStable, reasonOf(t, got))
	}
}
