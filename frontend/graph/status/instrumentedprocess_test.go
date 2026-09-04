package status

import (
	"testing"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/model"
)

func instrumentationInstance(healthy *bool, message string) *v1alpha1.InstrumentationInstance {
	return &v1alpha1.InstrumentationInstance{
		Status: v1alpha1.InstrumentationInstanceStatus{
			Healthy: healthy,
			Message: message,
		},
	}
}

func TestCalculateProcessHealthStatus(t *testing.T) {
	healthy, unhealthy := true, false

	tests := []struct {
		name        string
		healthy     *bool
		message     string
		wantStatus  model.DesiredStateProgress
		wantReason  ProcessHealthStatusReason
		wantMessage string
	}{
		{
			// An agent that has not reported yet is "starting", not unhealthy:
			// reporting it as a failure would flag every freshly started pod.
			name:        "not reported yet",
			healthy:     nil,
			message:     "agent is loading",
			wantStatus:  model.DesiredStateProgressUnknown,
			wantReason:  ProcessHealthStatusReasonStarting,
			wantMessage: "agent is loading",
		},
		{
			name:        "healthy",
			healthy:     &healthy,
			message:     "instrumentation is running",
			wantStatus:  model.DesiredStateProgressSuccess,
			wantReason:  ProcessHealthStatusReasonHealthy,
			wantMessage: "instrumentation is running",
		},
		{
			name:        "unhealthy",
			healthy:     &unhealthy,
			message:     "failed to load the agent",
			wantStatus:  model.DesiredStateProgressFailure,
			wantReason:  ProcessHealthStatusReasonUnhealthy,
			wantMessage: "failed to load the agent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateProcessHealthStatus(instrumentationInstance(tt.healthy, tt.message))
			if got == nil {
				t.Fatal("expected a status, got nil")
			}
			if got.Name != ProcessHealthStatusName {
				t.Fatalf("expected name %q, got %q", ProcessHealthStatusName, got.Name)
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

func TestCalculateProcessHealthStatusNilInstance(t *testing.T) {
	if got := CalculateProcessHealthStatus(nil); got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}
}
