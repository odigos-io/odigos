package status

import (
	"testing"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/model"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func rolloutCondition(reason v1alpha1.WorkloadRolloutReason, message string) metav1.Condition {
	return metav1.Condition{
		Type:    v1alpha1.WorkloadRolloutStatusConditionType,
		Reason:  string(reason),
		Message: message,
	}
}

// A rollout reason mapped to the wrong progress is what makes the UI either
// nag for a manual rollout that odigos is already handling, or stay quiet
// about a rollout that will never happen.
func TestWorkloadRolloutStatusCondition(t *testing.T) {
	tests := []struct {
		reason v1alpha1.WorkloadRolloutReason
		want   model.DesiredStateProgress
	}{
		{reason: v1alpha1.WorkloadRolloutReasonTriggeredSuccessfully, want: model.DesiredStateProgressSuccess},
		{reason: v1alpha1.WorkloadRolloutReasonRolloutFinished, want: model.DesiredStateProgressSuccess},
		{reason: v1alpha1.WorkloadRolloutReasonFailedToPatch, want: model.DesiredStateProgressFailure},
		{reason: v1alpha1.WorkloadRolloutReasonPreviousRolloutOngoing, want: model.DesiredStateProgressWaiting},
		{reason: v1alpha1.WorkloadRolloutReasonWaitingInQueue, want: model.DesiredStateProgressWaiting},
		{reason: v1alpha1.WorkloadRolloutReasonDisabled, want: model.DesiredStateProgressUnknown},
		{reason: v1alpha1.WorkloadRolloutReasonNotRequired, want: model.DesiredStateProgressIrrelevant},
		{reason: v1alpha1.WorkloadRolloutReasonWaitingForRestart, want: model.DesiredStateProgressIrrelevant},
		{reason: v1alpha1.WorkloadRolloutReasonWorkloadNotSupporting, want: model.DesiredStateProgressUnsupported},
	}

	for _, tt := range tests {
		t.Run(string(tt.reason), func(t *testing.T) {
			reason := string(tt.reason)
			if got := workloadRolloutStatusCondition(&reason); got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestWorkloadRolloutStatusConditionUnmappedReasons(t *testing.T) {
	if got := workloadRolloutStatusCondition(nil); got != model.DesiredStateProgressUnknown {
		t.Fatalf("nil reason: expected %q, got %q", model.DesiredStateProgressUnknown, got)
	}

	unrecognized := "SomeReasonAddedLater"
	if got := workloadRolloutStatusCondition(&unrecognized); got != model.DesiredStateProgressUnknown {
		t.Fatalf("unrecognized reason: expected %q, got %q", model.DesiredStateProgressUnknown, got)
	}
}

func TestCalculateRolloutStatusNoCondition(t *testing.T) {
	tests := []struct {
		name string
		ic   *v1alpha1.InstrumentationConfig
	}{
		{name: "nil instrumentation config", ic: nil},
		{name: "no conditions at all", ic: icWithConditions()},
		{
			name: "only conditions of other types",
			ic:   icWithConditions(runtimeDetectionCondition(v1alpha1.RuntimeDetectionReasonDetectedSuccessfully, "detected")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CalculateRolloutStatus(tt.ic); got != nil {
				t.Fatalf("expected nil, got %+v", got)
			}
		})
	}
}

func TestCalculateRolloutStatusReadsTheRolloutCondition(t *testing.T) {
	ic := icWithConditions(
		runtimeDetectionCondition(v1alpha1.RuntimeDetectionReasonDetectedSuccessfully, "detected"),
		rolloutCondition(v1alpha1.WorkloadRolloutReasonFailedToPatch, "failed to patch the deployment"),
	)

	got := CalculateRolloutStatus(ic)
	if got == nil {
		t.Fatal("expected a condition, got nil")
	}
	if got.Name != v1alpha1.WorkloadRolloutStatusConditionType {
		t.Fatalf("expected name %q, got %q", v1alpha1.WorkloadRolloutStatusConditionType, got.Name)
	}
	if got.Status != model.DesiredStateProgressFailure {
		t.Fatalf("expected %q, got %q", model.DesiredStateProgressFailure, got.Status)
	}
	if reasonOf(t, got) != string(v1alpha1.WorkloadRolloutReasonFailedToPatch) {
		t.Fatalf("expected reason %q, got %q", v1alpha1.WorkloadRolloutReasonFailedToPatch, reasonOf(t, got))
	}
	if got.Message != "failed to patch the deployment" {
		t.Fatalf("unexpected message %q", got.Message)
	}
}

// Opposite of CalculateRuntimeInspectionStatus, which keeps scanning: this one
// returns at the first rollout condition it finds.
func TestCalculateRolloutStatusUsesTheFirstMatchingCondition(t *testing.T) {
	ic := icWithConditions(
		rolloutCondition(v1alpha1.WorkloadRolloutReasonPreviousRolloutOngoing, "ongoing"),
		rolloutCondition(v1alpha1.WorkloadRolloutReasonRolloutFinished, "finished"),
	)

	got := CalculateRolloutStatus(ic)
	if reasonOf(t, got) != string(v1alpha1.WorkloadRolloutReasonPreviousRolloutOngoing) {
		t.Fatalf("expected reason %q, got %q", v1alpha1.WorkloadRolloutReasonPreviousRolloutOngoing, reasonOf(t, got))
	}
	if got.Message != "ongoing" {
		t.Fatalf("unexpected message %q", got.Message)
	}
}
