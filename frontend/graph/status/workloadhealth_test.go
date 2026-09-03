package status

import (
	"testing"
	"time"

	argorolloutsv1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/model"
	openshiftappsv1 "github.com/openshift/api/apps/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// assertWorkloadHealth checks the whole condition, because the replica counts
// are formatted into the message and a mixed-up counter is only visible there.
func assertWorkloadHealth(t *testing.T, got *model.DesiredConditionStatus, wantStatus model.DesiredStateProgress, wantReason WorkloadHealthStatusReason, wantMessage string) {
	t.Helper()
	if got == nil {
		t.Fatal("expected a status, got nil")
	}
	if got.Name != WorkloadHealthStatus {
		t.Fatalf("expected name %q, got %q", WorkloadHealthStatus, got.Name)
	}
	if got.Status != wantStatus {
		t.Fatalf("expected status %q, got %q", wantStatus, got.Status)
	}
	if reasonOf(t, got) != string(wantReason) {
		t.Fatalf("expected reason %q, got %q", wantReason, reasonOf(t, got))
	}
	if got.Message != wantMessage {
		t.Fatalf("expected message %q, got %q", wantMessage, got.Message)
	}
}

func TestCalculateDeploymentHealthStatus(t *testing.T) {
	tests := []struct {
		name        string
		status      appsv1.DeploymentStatus
		wantStatus  model.DesiredStateProgress
		wantReason  WorkloadHealthStatusReason
		wantMessage string
	}{
		{
			name: "available condition is false",
			status: appsv1.DeploymentStatus{
				Conditions: []appsv1.DeploymentCondition{
					{Type: appsv1.DeploymentAvailable, Status: corev1.ConditionFalse},
				},
			},
			wantStatus:  model.DesiredStateProgressFailure,
			wantReason:  WorkloadHealthStatusReasonNoAvailableReplicas,
			wantMessage: "Deployment does not have at least the minimum number of available replicas required",
		},
		{
			name: "progress deadline exceeded",
			status: appsv1.DeploymentStatus{
				Conditions: []appsv1.DeploymentCondition{
					{Type: appsv1.DeploymentProgressing, Status: corev1.ConditionFalse, Reason: "ProgressDeadlineExceeded"},
				},
			},
			wantStatus:  model.DesiredStateProgressFailure,
			wantReason:  WorkloadHealthStatusReasonProgressingDeadlineExceeded,
			wantMessage: "Deployment failed to start new pods after the deadline",
		},
		{
			// A rollout in flight is expected while odigos restarts the
			// workload to apply the agent, so it must not read as a failure.
			name: "replica set updated",
			status: appsv1.DeploymentStatus{
				Conditions: []appsv1.DeploymentCondition{
					{Type: appsv1.DeploymentProgressing, Status: corev1.ConditionTrue, Reason: "ReplicaSetUpdated"},
				},
			},
			wantStatus:  model.DesiredStateProgressWaiting,
			wantReason:  WorkloadHealthStatusReasonProgressing,
			wantMessage: "Deployment is progressing after rollout or revision update, new pods are not yet available",
		},
		{
			name: "progressing is not true for another reason",
			status: appsv1.DeploymentStatus{
				Conditions: []appsv1.DeploymentCondition{
					{Type: appsv1.DeploymentProgressing, Status: corev1.ConditionUnknown, Reason: "NewReplicaSetCreated"},
				},
			},
			wantStatus:  model.DesiredStateProgressFailure,
			wantReason:  WorkloadHealthStatusReasonProgressingError,
			wantMessage: "Deployment progressing is unhealthy",
		},
		{
			name: "replica failure",
			status: appsv1.DeploymentStatus{
				Conditions: []appsv1.DeploymentCondition{
					{Type: appsv1.DeploymentReplicaFailure, Status: corev1.ConditionTrue},
				},
			},
			wantStatus:  model.DesiredStateProgressFailure,
			wantReason:  WorkloadHealthStatusReasonReplicaFailure,
			wantMessage: "Deployment has pods which failed to be created or deleted",
		},
		{
			// A healthy progressing condition falls through to the replica
			// counters rather than short-circuiting as healthy.
			name: "progressing is true and replicas are fine",
			status: appsv1.DeploymentStatus{
				Conditions: []appsv1.DeploymentCondition{
					{Type: appsv1.DeploymentAvailable, Status: corev1.ConditionTrue},
					{Type: appsv1.DeploymentProgressing, Status: corev1.ConditionTrue, Reason: "NewReplicaSetAvailable"},
				},
				Replicas: 3, UpdatedReplicas: 3, ReadyReplicas: 3, AvailableReplicas: 3,
			},
			wantStatus:  model.DesiredStateProgressSuccess,
			wantReason:  WorkloadHealthStatusReasonHealthy,
			wantMessage: "All deployment replicase are available and ready",
		},
		{
			name:        "unavailable replicas",
			status:      appsv1.DeploymentStatus{Replicas: 5, UnavailableReplicas: 2, UpdatedReplicas: 5, ReadyReplicas: 3, AvailableReplicas: 3},
			wantStatus:  model.DesiredStateProgressWaiting,
			wantReason:  WorkloadHealthStatusReasonNoAvailableReplicas,
			wantMessage: "Deployment has 2/5 unavailable replicas",
		},
		{
			name:        "not all replicas updated",
			status:      appsv1.DeploymentStatus{Replicas: 4, UpdatedReplicas: 3, ReadyReplicas: 4, AvailableReplicas: 4},
			wantStatus:  model.DesiredStateProgressWaiting,
			wantReason:  WorkloadHealthStatusReasonProgressing,
			wantMessage: "not all deployment replicase are avaiable and ready",
		},
		{
			name:        "not all replicas ready",
			status:      appsv1.DeploymentStatus{Replicas: 4, UpdatedReplicas: 4, ReadyReplicas: 3, AvailableReplicas: 4},
			wantStatus:  model.DesiredStateProgressWaiting,
			wantReason:  WorkloadHealthStatusReasonProgressing,
			wantMessage: "not all deployment replicase are avaiable and ready",
		},
		{
			name:        "not all replicas available",
			status:      appsv1.DeploymentStatus{Replicas: 4, UpdatedReplicas: 4, ReadyReplicas: 4, AvailableReplicas: 3},
			wantStatus:  model.DesiredStateProgressWaiting,
			wantReason:  WorkloadHealthStatusReasonProgressing,
			wantMessage: "not all deployment replicase are avaiable and ready",
		},
		{
			name:        "scaled to zero",
			status:      appsv1.DeploymentStatus{},
			wantStatus:  model.DesiredStateProgressSuccess,
			wantReason:  WorkloadHealthStatusReasonHealthy,
			wantMessage: "All deployment replicase are available and ready",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertWorkloadHealth(t, CalculateDeploymentHealthStatus(tt.status), tt.wantStatus, tt.wantReason, tt.wantMessage)
		})
	}
}

// Conditions are evaluated in list order and the first one that resolves wins,
// so an available deployment that also reports a replica failure is a failure.
func TestCalculateDeploymentHealthStatusFirstResolvingConditionWins(t *testing.T) {
	got := CalculateDeploymentHealthStatus(appsv1.DeploymentStatus{
		Conditions: []appsv1.DeploymentCondition{
			{Type: appsv1.DeploymentAvailable, Status: corev1.ConditionTrue},
			{Type: appsv1.DeploymentReplicaFailure, Status: corev1.ConditionTrue},
			{Type: appsv1.DeploymentProgressing, Status: corev1.ConditionFalse},
		},
		Replicas: 3, UpdatedReplicas: 3, ReadyReplicas: 3, AvailableReplicas: 3,
	})

	assertWorkloadHealth(t, got, model.DesiredStateProgressFailure, WorkloadHealthStatusReasonReplicaFailure,
		"Deployment has pods which failed to be created or deleted")
}

func TestCalculateDaemonSetHealthStatus(t *testing.T) {
	tests := []struct {
		name        string
		status      appsv1.DaemonSetStatus
		wantStatus  model.DesiredStateProgress
		wantReason  WorkloadHealthStatusReason
		wantMessage string
	}{
		{
			// No node matches the daemon set, so there is nothing to instrument
			// and nothing to warn about.
			name:        "nothing scheduled",
			status:      appsv1.DaemonSetStatus{DesiredNumberScheduled: 0},
			wantStatus:  model.DesiredStateProgressIrrelevant,
			wantReason:  WorkloadHealthStatusReasonNoAvailableReplicas,
			wantMessage: "DaemonSet has no desired replicas scheduled",
		},
		{
			// A daemon set that is not fully scheduled cannot have more updated
			// than current pods, so the counters trail the current one; the
			// message identifies which counter the check actually read.
			name:        "not all scheduled",
			status:      appsv1.DaemonSetStatus{DesiredNumberScheduled: 6, CurrentNumberScheduled: 4, UpdatedNumberScheduled: 4, NumberAvailable: 4, NumberReady: 4},
			wantStatus:  model.DesiredStateProgressWaiting,
			wantReason:  WorkloadHealthStatusReasonProgressing,
			wantMessage: "DaemonSet has 4/6 current replicas scheduled",
		},
		{
			name:        "not all updated",
			status:      appsv1.DaemonSetStatus{DesiredNumberScheduled: 6, CurrentNumberScheduled: 6, UpdatedNumberScheduled: 3, NumberAvailable: 6, NumberReady: 6},
			wantStatus:  model.DesiredStateProgressWaiting,
			wantReason:  WorkloadHealthStatusReasonProgressing,
			wantMessage: "DaemonSet has 3/6 updated replicas",
		},
		{
			name:        "not all available",
			status:      appsv1.DaemonSetStatus{DesiredNumberScheduled: 6, CurrentNumberScheduled: 6, UpdatedNumberScheduled: 6, NumberAvailable: 2, NumberReady: 6},
			wantStatus:  model.DesiredStateProgressWaiting,
			wantReason:  WorkloadHealthStatusReasonProgressing,
			wantMessage: "DaemonSet has 2/6 available replicas",
		},
		{
			name:        "not all ready",
			status:      appsv1.DaemonSetStatus{DesiredNumberScheduled: 6, CurrentNumberScheduled: 6, UpdatedNumberScheduled: 6, NumberAvailable: 6, NumberReady: 1},
			wantStatus:  model.DesiredStateProgressWaiting,
			wantReason:  WorkloadHealthStatusReasonProgressing,
			wantMessage: "DaemonSet has 1/6 ready replicas",
		},
		{
			name:        "healthy",
			status:      appsv1.DaemonSetStatus{DesiredNumberScheduled: 6, CurrentNumberScheduled: 6, UpdatedNumberScheduled: 6, NumberAvailable: 6, NumberReady: 6},
			wantStatus:  model.DesiredStateProgressSuccess,
			wantReason:  WorkloadHealthStatusReasonHealthy,
			wantMessage: "DaemonSet replicas are reported healthy in kubernetes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertWorkloadHealth(t, CalculateDaemonSetHealthStatus(tt.status), tt.wantStatus, tt.wantReason, tt.wantMessage)
		})
	}
}

func TestCalculateStatefulSetHealthStatus(t *testing.T) {
	tests := []struct {
		name        string
		status      appsv1.StatefulSetStatus
		wantStatus  model.DesiredStateProgress
		wantReason  WorkloadHealthStatusReason
		wantMessage string
	}{
		{
			name:        "no replicas",
			status:      appsv1.StatefulSetStatus{Replicas: 0},
			wantStatus:  model.DesiredStateProgressIrrelevant,
			wantReason:  WorkloadHealthStatusReasonNoAvailableReplicas,
			wantMessage: "StatefulSet has no replicas",
		},
		{
			name:        "not all ready",
			status:      appsv1.StatefulSetStatus{Replicas: 5, ReadyReplicas: 2, AvailableReplicas: 5, UpdatedReplicas: 5},
			wantStatus:  model.DesiredStateProgressWaiting,
			wantReason:  WorkloadHealthStatusReasonProgressing,
			wantMessage: "StatefulSet has 2/5 ready replicas",
		},
		{
			name:        "not all available",
			status:      appsv1.StatefulSetStatus{Replicas: 5, ReadyReplicas: 5, AvailableReplicas: 3, UpdatedReplicas: 5},
			wantStatus:  model.DesiredStateProgressWaiting,
			wantReason:  WorkloadHealthStatusReasonProgressing,
			wantMessage: "StatefulSet has 3/5 available replicas",
		},
		{
			name:        "not all updated",
			status:      appsv1.StatefulSetStatus{Replicas: 5, ReadyReplicas: 5, AvailableReplicas: 5, UpdatedReplicas: 1},
			wantStatus:  model.DesiredStateProgressWaiting,
			wantReason:  WorkloadHealthStatusReasonProgressing,
			wantMessage: "StatefulSet has 1/5 updated replicas",
		},
		{
			// All counters look healthy but the revisions differ, which is the
			// only signal that a rolling update is still in flight.
			name: "rolling update in progress",
			status: appsv1.StatefulSetStatus{
				Replicas: 5, ReadyReplicas: 5, AvailableReplicas: 5, UpdatedReplicas: 5,
				CurrentRevision: "cart-6cf9d", UpdateRevision: "cart-84bbf",
			},
			wantStatus:  model.DesiredStateProgressWaiting,
			wantReason:  WorkloadHealthStatusReasonProgressing,
			wantMessage: "StatefulSet is in the middle of a rolling update",
		},
		{
			name: "healthy",
			status: appsv1.StatefulSetStatus{
				Replicas: 5, ReadyReplicas: 5, AvailableReplicas: 5, UpdatedReplicas: 5,
				CurrentRevision: "cart-84bbf", UpdateRevision: "cart-84bbf",
			},
			wantStatus:  model.DesiredStateProgressSuccess,
			wantReason:  WorkloadHealthStatusReasonHealthy,
			wantMessage: "StatefulSet replicas are reported healthy in kubernetes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertWorkloadHealth(t, CalculateStatefulSetHealthStatus(tt.status), tt.wantStatus, tt.wantReason, tt.wantMessage)
		})
	}
}

func TestCalculateCronJobHealthStatus(t *testing.T) {
	scheduledAt := metav1.NewTime(time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC))
	succeededAt := metav1.NewTime(scheduledAt.Add(time.Minute))

	tests := []struct {
		name        string
		status      batchv1.CronJobStatus
		wantStatus  model.DesiredStateProgress
		wantReason  WorkloadHealthStatusReason
		wantMessage string
	}{
		{
			// A running job is healthy even before the first job completed, and
			// even before a schedule time was recorded.
			name:        "active jobs",
			status:      batchv1.CronJobStatus{Active: []corev1.ObjectReference{{Name: "backup-1"}, {Name: "backup-2"}}},
			wantStatus:  model.DesiredStateProgressSuccess,
			wantReason:  WorkloadHealthStatusReasonHealthy,
			wantMessage: "CronJob has 2 active jobs running",
		},
		{
			name:        "never scheduled",
			status:      batchv1.CronJobStatus{},
			wantStatus:  model.DesiredStateProgressPending,
			wantReason:  WorkloadHealthStatusReasonProgressing,
			wantMessage: "CronJob has never been scheduled",
		},
		{
			name:        "completed successfully in the past",
			status:      batchv1.CronJobStatus{LastScheduleTime: &scheduledAt, LastSuccessfulTime: &succeededAt},
			wantStatus:  model.DesiredStateProgressSuccess,
			wantReason:  WorkloadHealthStatusReasonHealthy,
			wantMessage: "CronJob is healthy and has completed successfully",
		},
		{
			name:        "scheduled but never succeeded",
			status:      batchv1.CronJobStatus{LastScheduleTime: &scheduledAt},
			wantStatus:  model.DesiredStateProgressPending,
			wantReason:  WorkloadHealthStatusReasonProgressing,
			wantMessage: "CronJob is waiting for next scheduled run",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertWorkloadHealth(t, CalculateCronJobHealthStatus(tt.status), tt.wantStatus, tt.wantReason, tt.wantMessage)
		})
	}
}

func TestCalculateDeploymentConfigHealthStatus(t *testing.T) {
	tests := []struct {
		name        string
		status      openshiftappsv1.DeploymentConfigStatus
		wantStatus  model.DesiredStateProgress
		wantReason  WorkloadHealthStatusReason
		wantMessage string
	}{
		{
			name: "available condition is false",
			status: openshiftappsv1.DeploymentConfigStatus{
				Conditions: []openshiftappsv1.DeploymentCondition{
					{Type: openshiftappsv1.DeploymentAvailable, Status: corev1.ConditionFalse},
				},
			},
			wantStatus:  model.DesiredStateProgressFailure,
			wantReason:  WorkloadHealthStatusReasonNoAvailableReplicas,
			wantMessage: "DeploymentConfig does not have at least the minimum number of available replicas required",
		},
		{
			name: "progressing is not true",
			status: openshiftappsv1.DeploymentConfigStatus{
				Conditions: []openshiftappsv1.DeploymentCondition{
					{Type: openshiftappsv1.DeploymentProgressing, Status: corev1.ConditionFalse},
				},
			},
			wantStatus:  model.DesiredStateProgressFailure,
			wantReason:  WorkloadHealthStatusReasonProgressingError,
			wantMessage: "DeploymentConfig progressing is unhealthy",
		},
		{
			name: "replica failure",
			status: openshiftappsv1.DeploymentConfigStatus{
				Conditions: []openshiftappsv1.DeploymentCondition{
					{Type: openshiftappsv1.DeploymentReplicaFailure, Status: corev1.ConditionTrue},
				},
			},
			wantStatus:  model.DesiredStateProgressFailure,
			wantReason:  WorkloadHealthStatusReasonReplicaFailure,
			wantMessage: "DeploymentConfig has pods which failed to be created or deleted",
		},
		{
			name:        "unavailable replicas",
			status:      openshiftappsv1.DeploymentConfigStatus{Replicas: 7, UnavailableReplicas: 3, UpdatedReplicas: 7, ReadyReplicas: 4, AvailableReplicas: 4},
			wantStatus:  model.DesiredStateProgressWaiting,
			wantReason:  WorkloadHealthStatusReasonNoAvailableReplicas,
			wantMessage: "DeploymentConfig has 3/7 unavailable replicas",
		},
		{
			name:        "not all replicas updated",
			status:      openshiftappsv1.DeploymentConfigStatus{Replicas: 4, UpdatedReplicas: 2, ReadyReplicas: 4, AvailableReplicas: 4},
			wantStatus:  model.DesiredStateProgressWaiting,
			wantReason:  WorkloadHealthStatusReasonProgressing,
			wantMessage: "not all deploymentconfig replicas are available and ready",
		},
		{
			name:        "not all replicas ready",
			status:      openshiftappsv1.DeploymentConfigStatus{Replicas: 4, UpdatedReplicas: 4, ReadyReplicas: 2, AvailableReplicas: 4},
			wantStatus:  model.DesiredStateProgressWaiting,
			wantReason:  WorkloadHealthStatusReasonProgressing,
			wantMessage: "not all deploymentconfig replicas are available and ready",
		},
		{
			name:        "not all replicas available",
			status:      openshiftappsv1.DeploymentConfigStatus{Replicas: 4, UpdatedReplicas: 4, ReadyReplicas: 4, AvailableReplicas: 2},
			wantStatus:  model.DesiredStateProgressWaiting,
			wantReason:  WorkloadHealthStatusReasonProgressing,
			wantMessage: "not all deploymentconfig replicas are available and ready",
		},
		{
			name: "healthy",
			status: openshiftappsv1.DeploymentConfigStatus{
				Conditions: []openshiftappsv1.DeploymentCondition{
					{Type: openshiftappsv1.DeploymentAvailable, Status: corev1.ConditionTrue},
					{Type: openshiftappsv1.DeploymentProgressing, Status: corev1.ConditionTrue},
				},
				Replicas: 4, UpdatedReplicas: 4, ReadyReplicas: 4, AvailableReplicas: 4,
			},
			wantStatus:  model.DesiredStateProgressSuccess,
			wantReason:  WorkloadHealthStatusReasonHealthy,
			wantMessage: "All deploymentconfig replicas are available and ready",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertWorkloadHealth(t, CalculateDeploymentConfigHealthStatus(tt.status), tt.wantStatus, tt.wantReason, tt.wantMessage)
		})
	}
}

func TestCalculateRolloutHealthStatus(t *testing.T) {
	tests := []struct {
		name        string
		status      argorolloutsv1alpha1.RolloutStatus
		wantStatus  model.DesiredStateProgress
		wantReason  WorkloadHealthStatusReason
		wantMessage string
	}{
		{
			name:        "no replicas",
			status:      argorolloutsv1alpha1.RolloutStatus{Replicas: 0},
			wantStatus:  model.DesiredStateProgressIrrelevant,
			wantReason:  WorkloadHealthStatusReasonNoAvailableReplicas,
			wantMessage: "Argo Rollout has no replicas",
		},
		{
			name:        "not all available",
			status:      argorolloutsv1alpha1.RolloutStatus{Replicas: 8, AvailableReplicas: 5, UpdatedReplicas: 8, ReadyReplicas: 8},
			wantStatus:  model.DesiredStateProgressWaiting,
			wantReason:  WorkloadHealthStatusReasonProgressing,
			wantMessage: "Argo Rollout has 5/8 available replicas",
		},
		{
			name:        "not all updated",
			status:      argorolloutsv1alpha1.RolloutStatus{Replicas: 8, AvailableReplicas: 8, UpdatedReplicas: 6, ReadyReplicas: 8},
			wantStatus:  model.DesiredStateProgressWaiting,
			wantReason:  WorkloadHealthStatusReasonProgressing,
			wantMessage: "Argo Rollout has 6/8 updated replicas",
		},
		{
			name:        "not all ready",
			status:      argorolloutsv1alpha1.RolloutStatus{Replicas: 8, AvailableReplicas: 8, UpdatedReplicas: 8, ReadyReplicas: 2},
			wantStatus:  model.DesiredStateProgressWaiting,
			wantReason:  WorkloadHealthStatusReasonProgressing,
			wantMessage: "Rollout has 2/8 ready replicas",
		},
		{
			name:        "healthy",
			status:      argorolloutsv1alpha1.RolloutStatus{Replicas: 8, AvailableReplicas: 8, UpdatedReplicas: 8, ReadyReplicas: 8},
			wantStatus:  model.DesiredStateProgressSuccess,
			wantReason:  WorkloadHealthStatusReasonHealthy,
			wantMessage: "All Argo Rollout replicas are available and ready",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertWorkloadHealth(t, CalculateRolloutHealthStatus(tt.status), tt.wantStatus, tt.wantReason, tt.wantMessage)
		})
	}
}

func TestCalculateStaticPodHealthStatus(t *testing.T) {
	phases := []corev1.PodPhase{
		corev1.PodPending,
		corev1.PodSucceeded,
		corev1.PodFailed,
		corev1.PodUnknown,
		corev1.PodPhase(""),
	}

	for _, phase := range phases {
		t.Run("phase "+string(phase), func(t *testing.T) {
			got := CalculateStaticPodHealthStatus(corev1.PodStatus{Phase: phase})
			assertWorkloadHealth(t, got, model.DesiredStateProgressWaiting, WorkloadHealthStatusReasonProgressing, "StaticPod is not running")
		})
	}

	t.Run("phase Running", func(t *testing.T) {
		got := CalculateStaticPodHealthStatus(corev1.PodStatus{Phase: corev1.PodRunning})
		assertWorkloadHealth(t, got, model.DesiredStateProgressSuccess, WorkloadHealthStatusReasonHealthy, "StaticPod is running")
	})
}
