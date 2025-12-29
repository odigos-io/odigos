package utils

import (
	"testing"

	argorolloutsv1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	"github.com/tj/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func int32Ptr(i int32) *int32 {
	return &i
}

func TestIsArgoRolloutRolloutDone(t *testing.T) {
	tests := []struct {
		name     string
		rollout  *argorolloutsv1alpha1.Rollout
		expected bool
	}{
		{
			name: "fully rolled out - all replicas updated and available",
			rollout: &argorolloutsv1alpha1.Rollout{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
				Spec: argorolloutsv1alpha1.RolloutSpec{
					Replicas: int32Ptr(10),
				},
				Status: argorolloutsv1alpha1.RolloutStatus{
					ObservedGeneration: "1",
					Replicas:           10,
					UpdatedReplicas:    10,
					AvailableReplicas:  10,
					Phase:              argorolloutsv1alpha1.RolloutPhaseHealthy,
				},
			},
			expected: true,
		},
		{
			name: "paused during canary - should be considered done for instrumentation",
			rollout: &argorolloutsv1alpha1.Rollout{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 2,
				},
				Spec: argorolloutsv1alpha1.RolloutSpec{
					Replicas: int32Ptr(10),
				},
				Status: argorolloutsv1alpha1.RolloutStatus{
					ObservedGeneration: "2",
					Replicas:           10,
					UpdatedReplicas:    2, // Only 20% updated (canary)
					AvailableReplicas:  10,
					Phase:              argorolloutsv1alpha1.RolloutPhasePaused,
				},
			},
			expected: true, // Paused state should allow instrumentation
		},
		{
			name: "progressing - not done",
			rollout: &argorolloutsv1alpha1.Rollout{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 2,
				},
				Spec: argorolloutsv1alpha1.RolloutSpec{
					Replicas: int32Ptr(10),
				},
				Status: argorolloutsv1alpha1.RolloutStatus{
					ObservedGeneration: "2",
					Replicas:           10,
					UpdatedReplicas:    5,
					AvailableReplicas:  8,
					Phase:              argorolloutsv1alpha1.RolloutPhaseProgressing,
				},
			},
			expected: false,
		},
		{
			name: "generation not yet observed",
			rollout: &argorolloutsv1alpha1.Rollout{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 3,
				},
				Spec: argorolloutsv1alpha1.RolloutSpec{
					Replicas: int32Ptr(10),
				},
				Status: argorolloutsv1alpha1.RolloutStatus{
					ObservedGeneration: "2", // Older generation
					Replicas:           10,
					UpdatedReplicas:    10,
					AvailableReplicas:  10,
					Phase:              argorolloutsv1alpha1.RolloutPhaseHealthy,
				},
			},
			expected: false,
		},
		{
			name: "old replicas pending termination",
			rollout: &argorolloutsv1alpha1.Rollout{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
				Spec: argorolloutsv1alpha1.RolloutSpec{
					Replicas: int32Ptr(10),
				},
				Status: argorolloutsv1alpha1.RolloutStatus{
					ObservedGeneration: "1",
					Replicas:           12, // More replicas than updated (old ones still running)
					UpdatedReplicas:    10,
					AvailableReplicas:  12,
					Phase:              argorolloutsv1alpha1.RolloutPhaseProgressing,
				},
			},
			expected: false,
		},
		{
			name: "not all updated replicas are available",
			rollout: &argorolloutsv1alpha1.Rollout{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
				Spec: argorolloutsv1alpha1.RolloutSpec{
					Replicas: int32Ptr(10),
				},
				Status: argorolloutsv1alpha1.RolloutStatus{
					ObservedGeneration: "1",
					Replicas:           10,
					UpdatedReplicas:    10,
					AvailableReplicas:  8, // Some updated replicas not yet available
					Phase:              argorolloutsv1alpha1.RolloutPhaseProgressing,
				},
			},
			expected: false,
		},
		{
			name: "invalid observed generation string",
			rollout: &argorolloutsv1alpha1.Rollout{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
				Spec: argorolloutsv1alpha1.RolloutSpec{
					Replicas: int32Ptr(10),
				},
				Status: argorolloutsv1alpha1.RolloutStatus{
					ObservedGeneration: "invalid",
					Replicas:           10,
					UpdatedReplicas:    10,
					AvailableReplicas:  10,
				},
			},
			expected: false,
		},
		{
			name: "degraded state - pods unhealthy, rollout not done",
			rollout: &argorolloutsv1alpha1.Rollout{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
				Spec: argorolloutsv1alpha1.RolloutSpec{
					Replicas: int32Ptr(10),
				},
				Status: argorolloutsv1alpha1.RolloutStatus{
					ObservedGeneration: "1",
					Replicas:           10,
					UpdatedReplicas:    10,
					AvailableReplicas:  10,
					Phase:              argorolloutsv1alpha1.RolloutPhaseDegraded,
				},
			},
			expected: false, // Degraded means pods are unhealthy, rollout not complete
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isArgoRolloutRolloutDone(tt.rollout)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsWorkloadRolloutDone(t *testing.T) {
	t.Run("deployment", func(t *testing.T) {
		dep := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Generation: 1,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: int32Ptr(3),
			},
			Status: appsv1.DeploymentStatus{
				ObservedGeneration: 1,
				Replicas:           3,
				UpdatedReplicas:    3,
				AvailableReplicas:  3,
			},
		}
		assert.True(t, IsWorkloadRolloutDone(dep))
	})

	t.Run("argo rollout paused", func(t *testing.T) {
		rollout := &argorolloutsv1alpha1.Rollout{
			ObjectMeta: metav1.ObjectMeta{
				Generation: 1,
			},
			Spec: argorolloutsv1alpha1.RolloutSpec{
				Replicas: int32Ptr(10),
			},
			Status: argorolloutsv1alpha1.RolloutStatus{
				ObservedGeneration: "1",
				Replicas:           10,
				UpdatedReplicas:    2, // Mid-canary
				AvailableReplicas:  10,
				Phase:              argorolloutsv1alpha1.RolloutPhasePaused,
			},
		}
		// A paused rollout should be considered "done" for instrumentation purposes
		assert.True(t, IsWorkloadRolloutDone(rollout))
	})
}
