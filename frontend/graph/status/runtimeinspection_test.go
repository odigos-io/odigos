package status

import (
	"testing"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/model"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func runtimeDetectionCondition(reason v1alpha1.RuntimeDetectionReason, message string) metav1.Condition {
	return metav1.Condition{
		Type:    v1alpha1.RuntimeDetectionStatusConditionType,
		Reason:  string(reason),
		Message: message,
	}
}

func icWithConditions(conditions ...metav1.Condition) *v1alpha1.InstrumentationConfig {
	return &v1alpha1.InstrumentationConfig{
		Status: v1alpha1.InstrumentationConfigStatus{Conditions: conditions},
	}
}

// Each reason is written by the odiglet runtime inspection and drives a
// different visual in the UI, so a wrong mapping either hides a detection
// failure or reports a healthy workload as broken.
func TestRuntimeDetectionStatusCondition(t *testing.T) {
	tests := []struct {
		reason v1alpha1.RuntimeDetectionReason
		want   model.DesiredStateProgress
	}{
		{reason: v1alpha1.RuntimeDetectionReasonDetectedSuccessfully, want: model.DesiredStateProgressSuccess},
		{reason: v1alpha1.RuntimeDetectionReasonResolvedFromMultipleLanguages, want: model.DesiredStateProgressSuccess},
		{reason: v1alpha1.RuntimeDetectionReasonUnresolvedMultipleLanguages, want: model.DesiredStateProgressNotice},
		{reason: v1alpha1.RuntimeDetectionReasonWaitingForDetection, want: model.DesiredStateProgressWaiting},
		{reason: v1alpha1.RuntimeDetectionReasonNoRunningPods, want: model.DesiredStateProgressPending},
		{reason: v1alpha1.RuntimeDetectionReasonError, want: model.DesiredStateProgressFailure},
	}

	for _, tt := range tests {
		t.Run(string(tt.reason), func(t *testing.T) {
			reason := string(tt.reason)
			if got := runtimeDetectionStatusCondition(&reason); got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestRuntimeDetectionStatusConditionUnmappedReasons(t *testing.T) {
	if got := runtimeDetectionStatusCondition(nil); got != model.DesiredStateProgressUnknown {
		t.Fatalf("nil reason: expected %q, got %q", model.DesiredStateProgressUnknown, got)
	}

	unrecognized := "SomeReasonAddedLater"
	if got := runtimeDetectionStatusCondition(&unrecognized); got != model.DesiredStateProgressUnknown {
		t.Fatalf("unrecognized reason: expected %q, got %q", model.DesiredStateProgressUnknown, got)
	}
}

func TestCalculateRuntimeInspectionStatusNilInstrumentationConfig(t *testing.T) {
	if got := CalculateRuntimeInspectionStatus(nil); got != nil {
		t.Fatalf("expected nil for a workload without an instrumentation config, got %+v", got)
	}
}

// Unlike the other Calculate* helpers this one always returns a condition, so
// a workload whose instrumentation config has not been reconciled yet still
// reports a runtime detection condition instead of an empty one.
func TestCalculateRuntimeInspectionStatusNoConditionYet(t *testing.T) {
	got := CalculateRuntimeInspectionStatus(icWithConditions())
	if got == nil {
		t.Fatal("expected a condition, got nil")
	}
	if got.Name != v1alpha1.RuntimeDetectionStatusConditionType {
		t.Fatalf("expected name %q, got %q", v1alpha1.RuntimeDetectionStatusConditionType, got.Name)
	}
	if got.Status != model.DesiredStateProgressUnknown {
		t.Fatalf("expected %q, got %q", model.DesiredStateProgressUnknown, got.Status)
	}
	if got.ReasonEnum != nil {
		t.Fatalf("expected no reason, got %q", *got.ReasonEnum)
	}
	if got.Message != "runtime detection status not yet available" {
		t.Fatalf("unexpected message %q", got.Message)
	}
}

func TestCalculateRuntimeInspectionStatusReadsTheRuntimeDetectionCondition(t *testing.T) {
	ic := icWithConditions(
		metav1.Condition{Type: v1alpha1.WorkloadRolloutStatusConditionType, Reason: string(v1alpha1.WorkloadRolloutReasonFailedToPatch), Message: "rollout message"},
		runtimeDetectionCondition(v1alpha1.RuntimeDetectionReasonError, "failed to inspect the container"),
	)

	got := CalculateRuntimeInspectionStatus(ic)
	if got == nil {
		t.Fatal("expected a condition, got nil")
	}
	if got.Status != model.DesiredStateProgressFailure {
		t.Fatalf("expected %q, got %q", model.DesiredStateProgressFailure, got.Status)
	}
	if reasonOf(t, got) != string(v1alpha1.RuntimeDetectionReasonError) {
		t.Fatalf("expected reason %q, got %q", v1alpha1.RuntimeDetectionReasonError, reasonOf(t, got))
	}
	if got.Message != "failed to inspect the container" {
		t.Fatalf("unexpected message %q", got.Message)
	}
}

// The loop does not stop at the first match, so the last runtime detection
// condition in the list is the one reported. Odigos writes a single condition
// per type, but the resolution order matters if that ever stops being true.
func TestCalculateRuntimeInspectionStatusUsesTheLastMatchingCondition(t *testing.T) {
	ic := icWithConditions(
		runtimeDetectionCondition(v1alpha1.RuntimeDetectionReasonWaitingForDetection, "waiting"),
		runtimeDetectionCondition(v1alpha1.RuntimeDetectionReasonDetectedSuccessfully, "detected"),
	)

	got := CalculateRuntimeInspectionStatus(ic)
	if reasonOf(t, got) != string(v1alpha1.RuntimeDetectionReasonDetectedSuccessfully) {
		t.Fatalf("expected reason %q, got %q", v1alpha1.RuntimeDetectionReasonDetectedSuccessfully, reasonOf(t, got))
	}
	if got.Message != "detected" {
		t.Fatalf("unexpected message %q", got.Message)
	}
}
