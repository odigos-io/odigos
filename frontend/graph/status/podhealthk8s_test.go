package status

import (
	"testing"

	"github.com/odigos-io/odigos/frontend/graph/computed"
	"github.com/odigos-io/odigos/frontend/graph/model"
)

func k8sHealthContainer(started *bool, isReady, isCrashLoop bool, waitingReason string) *computed.ComputedPodContainer {
	container := &computed.ComputedPodContainer{
		ContainerName: "app",
		Started:       started,
		IsReady:       isReady,
		IsCrashLoop:   isCrashLoop,
	}
	if waitingReason != "" {
		container.WaitingReasonEnum = &waitingReason
	}
	return container
}

// The four container states are checked in a fixed order, and each one maps to
// a different severity. Getting the order wrong makes a crash-looping
// container report as merely "not started" (Waiting instead of Failure), which
// drops it out of the workload's error aggregation entirely.
func TestCalculatePodContainerK8sHealthStatus(t *testing.T) {
	started, notStarted := true, false

	tests := []struct {
		name        string
		container   *computed.ComputedPodContainer
		wantStatus  model.DesiredStateProgress
		wantReason  PodContainerK8sHealthReason
		wantMessage string
	}{
		{
			name:        "crash loop back off",
			container:   k8sHealthContainer(&started, true, true, "CrashLoopBackOff"),
			wantStatus:  model.DesiredStateProgressFailure,
			wantReason:  PodContainerK8sHealthReasonCrashLoopBackOff,
			wantMessage: "container in crash loop back off: CrashLoopBackOff",
		},
		{
			// Crash loop is checked first, so it wins over both other problems.
			name:        "crash loop wins over not started and not ready",
			container:   k8sHealthContainer(&notStarted, false, true, "CrashLoopBackOff"),
			wantStatus:  model.DesiredStateProgressFailure,
			wantReason:  PodContainerK8sHealthReasonCrashLoopBackOff,
			wantMessage: "container in crash loop back off: CrashLoopBackOff",
		},
		{
			name:        "no started value in the container status",
			container:   k8sHealthContainer(nil, true, false, ""),
			wantStatus:  model.DesiredStateProgressWaiting,
			wantReason:  PodContainerK8sHealthReasonNotStarted,
			wantMessage: "container has not started yet",
		},
		{
			name:        "not started",
			container:   k8sHealthContainer(&notStarted, true, false, ""),
			wantStatus:  model.DesiredStateProgressWaiting,
			wantReason:  PodContainerK8sHealthReasonNotStarted,
			wantMessage: "container has not started yet",
		},
		{
			name:        "started but not ready",
			container:   k8sHealthContainer(&started, false, false, ""),
			wantStatus:  model.DesiredStateProgressWaiting,
			wantReason:  PodContainerK8sHealthReasonNotReady,
			wantMessage: "container is not ready yet",
		},
		{
			name:        "healthy",
			container:   k8sHealthContainer(&started, true, false, ""),
			wantStatus:  model.DesiredStateProgressSuccess,
			wantReason:  PodContainerK8sHealthReasonHealthy,
			wantMessage: "container is healthy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculatePodContainerK8sHealthStatus(tt.container)
			if got == nil {
				t.Fatal("expected a status, got nil")
			}
			if got.Name != PodContainerHealthStatus {
				t.Fatalf("expected name %q, got %q", PodContainerHealthStatus, got.Name)
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

// The crash loop message dereferences WaitingReasonEnum. The loader only sets
// IsCrashLoop from a waiting container state, which also sets the waiting
// reason, so the two always travel together; this pins that the reason the
// kubelet reported is the one surfaced.
func TestCalculatePodContainerK8sHealthStatusCarriesTheWaitingReason(t *testing.T) {
	started := true
	got := CalculatePodContainerK8sHealthStatus(k8sHealthContainer(&started, true, true, "ImagePullBackOff"))
	if got.Message != "container in crash loop back off: ImagePullBackOff" {
		t.Fatalf("unexpected message %q", got.Message)
	}
}

// The pod condition is a rewrite of the most severe container condition: it
// keeps the container's severity but replaces the message with a pod-level one.
func TestCalculatePodHealthK8sStatus(t *testing.T) {
	tests := []struct {
		name        string
		conditions  []*model.DesiredConditionStatus
		wantStatus  model.DesiredStateProgress
		wantReason  PodContainerK8sHealthReason
		wantMessage string
	}{
		{
			name: "all containers healthy",
			conditions: []*model.DesiredConditionStatus{
				createPodContainerHealthK8sStatus(PodContainerK8sHealthReasonHealthy, "container is healthy", model.DesiredStateProgressSuccess),
				createPodContainerHealthK8sStatus(PodContainerK8sHealthReasonHealthy, "container is healthy", model.DesiredStateProgressSuccess),
			},
			wantStatus:  model.DesiredStateProgressSuccess,
			wantReason:  PodContainerK8sHealthReasonHealthy,
			wantMessage: "all containers in pod are reported healthy in kubernetes",
		},
		{
			name: "one container not started",
			conditions: []*model.DesiredConditionStatus{
				createPodContainerHealthK8sStatus(PodContainerK8sHealthReasonHealthy, "container is healthy", model.DesiredStateProgressSuccess),
				createPodContainerHealthK8sStatus(PodContainerK8sHealthReasonNotStarted, "container has not started yet", model.DesiredStateProgressWaiting),
			},
			wantStatus:  model.DesiredStateProgressWaiting,
			wantReason:  PodContainerK8sHealthReasonNotStarted,
			wantMessage: "some containers in pod are not started yet",
		},
		{
			name: "one container not ready",
			conditions: []*model.DesiredConditionStatus{
				createPodContainerHealthK8sStatus(PodContainerK8sHealthReasonNotReady, "container is not ready yet", model.DesiredStateProgressWaiting),
			},
			wantStatus:  model.DesiredStateProgressWaiting,
			wantReason:  PodContainerK8sHealthReasonNotReady,
			wantMessage: "some containers in pod are not ready yet",
		},
		{
			// The crash looping container is the most severe, so it is the one
			// the pod reports even though another container is only waiting.
			name: "crash loop wins over not ready",
			conditions: []*model.DesiredConditionStatus{
				createPodContainerHealthK8sStatus(PodContainerK8sHealthReasonNotReady, "container is not ready yet", model.DesiredStateProgressWaiting),
				createPodContainerHealthK8sStatus(PodContainerK8sHealthReasonCrashLoopBackOff, "container in crash loop back off: CrashLoopBackOff", model.DesiredStateProgressFailure),
			},
			wantStatus:  model.DesiredStateProgressFailure,
			wantReason:  PodContainerK8sHealthReasonCrashLoopBackOff,
			wantMessage: "some containers in pod are in crash loop back off",
		},
		{
			name: "reason with no pod level message",
			conditions: []*model.DesiredConditionStatus{
				createPodContainerHealthK8sStatus(PodContainerK8sHealthReasonUnknown, "container state is unknown", model.DesiredStateProgressWaiting),
			},
			wantStatus:  model.DesiredStateProgressWaiting,
			wantReason:  PodContainerK8sHealthReasonUnknown,
			wantMessage: "unknown reason",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculatePodHealthK8sStatus(&computed.CachedPod{PodName: "pod-1"}, tt.conditions)
			if got == nil {
				t.Fatal("expected a status, got nil")
			}
			if got.Name != PodHealthStatus {
				t.Fatalf("expected name %q, got %q", PodHealthStatus, got.Name)
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

// A pod with no container conditions, or with conditions carrying no reason,
// cannot be judged. It is reported as an error rather than silently as healthy.
func TestCalculatePodHealthK8sStatusWithoutAnUsableCondition(t *testing.T) {
	tests := []struct {
		name       string
		conditions []*model.DesiredConditionStatus
	}{
		{name: "no conditions", conditions: nil},
		{name: "only nil conditions", conditions: []*model.DesiredConditionStatus{nil}},
		{
			name:       "condition without a reason",
			conditions: []*model.DesiredConditionStatus{{Name: PodContainerHealthStatus, Status: model.DesiredStateProgressSuccess}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculatePodHealthK8sStatus(&computed.CachedPod{PodName: "pod-1"}, tt.conditions)
			if got == nil {
				t.Fatal("expected a status, got nil")
			}
			if got.Status != model.DesiredStateProgressError {
				t.Fatalf("expected status %q, got %q", model.DesiredStateProgressError, got.Status)
			}
			if reasonOf(t, got) != string(PodContainerK8sHealthReasonUnknown) {
				t.Fatalf("expected reason %q, got %q", PodContainerK8sHealthReasonUnknown, reasonOf(t, got))
			}
			if got.Message != "not able to determine health status for containers in pod" {
				t.Fatalf("unexpected message %q", got.Message)
			}
		})
	}
}
