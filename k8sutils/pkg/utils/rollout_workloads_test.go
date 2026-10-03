package utils

import (
	"testing"

	argorolloutsv1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	openshiftappsv1 "github.com/openshift/api/apps/v1"
	"github.com/tj/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/odigos-io/odigos/k8sutils/pkg/conditions"
)

const rolloutTestNamespace = "checkout"

// Every fixture below is a workload whose rollout has finished, with deliberately non-default
// values so that a guard reading the wrong field is visible. Each test case then changes exactly
// one field, which is what makes it possible to attribute a "not done" answer to a single guard.

func doneDeployment() *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "frontend",
			Namespace:  rolloutTestNamespace,
			Generation: 7,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(4),
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "frontend"}},
		},
		Status: appsv1.DeploymentStatus{
			ObservedGeneration: 7,
			Replicas:           4,
			UpdatedReplicas:    4,
			AvailableReplicas:  4,
		},
	}
}

func doneStatefulSet() *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "cart",
			Namespace:  rolloutTestNamespace,
			Generation: 7,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:       int32Ptr(4),
			Selector:       &metav1.LabelSelector{MatchLabels: map[string]string{"app": "cart"}},
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{Type: appsv1.RollingUpdateStatefulSetStrategyType},
		},
		Status: appsv1.StatefulSetStatus{
			ObservedGeneration: 7,
			Replicas:           4,
			ReadyReplicas:      4,
			UpdatedReplicas:    4,
			CurrentRevision:    "cart-7c9f",
			UpdateRevision:     "cart-7c9f",
		},
	}
}

func doneDaemonSet() *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "log-shipper",
			Namespace:  rolloutTestNamespace,
			Generation: 7,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector:       &metav1.LabelSelector{MatchLabels: map[string]string{"app": "log-shipper"}},
			UpdateStrategy: appsv1.DaemonSetUpdateStrategy{Type: appsv1.RollingUpdateDaemonSetStrategyType},
		},
		Status: appsv1.DaemonSetStatus{
			ObservedGeneration:     7,
			DesiredNumberScheduled: 5,
			UpdatedNumberScheduled: 5,
			NumberAvailable:        5,
		},
	}
}

func doneDeploymentConfig() *openshiftappsv1.DeploymentConfig {
	return &openshiftappsv1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "billing",
			Namespace:  rolloutTestNamespace,
			Generation: 7,
		},
		Spec: openshiftappsv1.DeploymentConfigSpec{
			Replicas: 4,
			Selector: map[string]string{"app": "billing"},
		},
		Status: openshiftappsv1.DeploymentConfigStatus{
			ObservedGeneration:  7,
			Replicas:            4,
			UpdatedReplicas:     4,
			AvailableReplicas:   4,
			UnavailableReplicas: 0,
		},
	}
}

func doneArgoRollout() *argorolloutsv1alpha1.Rollout {
	return &argorolloutsv1alpha1.Rollout{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "search",
			Namespace:  rolloutTestNamespace,
			Generation: 7,
		},
		Spec: argorolloutsv1alpha1.RolloutSpec{
			Replicas: int32Ptr(4),
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "search"}},
		},
		Status: argorolloutsv1alpha1.RolloutStatus{
			ObservedGeneration: "7",
			Replicas:           4,
			UpdatedReplicas:    4,
			AvailableReplicas:  4,
			Phase:              argorolloutsv1alpha1.RolloutPhaseHealthy,
		},
	}
}

func progressingCondition(reason string) appsv1.DeploymentCondition {
	return appsv1.DeploymentCondition{
		Type:   appsv1.DeploymentProgressing,
		Status: corev1.ConditionFalse,
		Reason: reason,
	}
}

func TestIsDeploymentRolloutDone(t *testing.T) {
	tests := []struct {
		name     string
		mutate   func(d *appsv1.Deployment)
		expected bool
	}{
		{
			name:     "fully rolled out",
			mutate:   func(d *appsv1.Deployment) {},
			expected: true,
		},
		{
			name:     "spec change not observed yet",
			mutate:   func(d *appsv1.Deployment) { d.Status.ObservedGeneration = 6 },
			expected: false,
		},
		{
			name:     "controller observed a newer generation than the spec",
			mutate:   func(d *appsv1.Deployment) { d.Status.ObservedGeneration = 8 },
			expected: true,
		},
		{
			name: "progress deadline exceeded",
			mutate: func(d *appsv1.Deployment) {
				d.Status.Conditions = []appsv1.DeploymentCondition{progressingCondition(conditions.TimedOutReason)}
			},
			expected: false,
		},
		{
			name: "progressing for any other reason is not a timeout",
			mutate: func(d *appsv1.Deployment) {
				d.Status.Conditions = []appsv1.DeploymentCondition{progressingCondition("NewReplicaSetAvailable")}
			},
			expected: true,
		},
		{
			name: "a timed out reason on a different condition type is ignored",
			mutate: func(d *appsv1.Deployment) {
				d.Status.Conditions = []appsv1.DeploymentCondition{{
					Type:   appsv1.DeploymentAvailable,
					Reason: conditions.TimedOutReason,
				}}
			},
			expected: true,
		},
		{
			name:     "not every replica has been updated",
			mutate:   func(d *appsv1.Deployment) { d.Status.UpdatedReplicas = 3 },
			expected: false,
		},
		{
			// Scaling up: every pod that exists is updated and available, but the deployment has
			// not reached the desired replica count yet. Only the desired count check sees this.
			name: "the deployment has not scaled up to the desired replica count",
			mutate: func(d *appsv1.Deployment) {
				d.Status.Replicas = 3
				d.Status.UpdatedReplicas = 3
				d.Status.AvailableReplicas = 3
			},
			expected: false,
		},
		{
			name:     "old replicas are still terminating",
			mutate:   func(d *appsv1.Deployment) { d.Status.Replicas = 5 },
			expected: false,
		},
		{
			name:     "updated replicas are not all available",
			mutate:   func(d *appsv1.Deployment) { d.Status.AvailableReplicas = 3 },
			expected: false,
		},
		{
			name: "unset spec replicas skips the desired replica check",
			mutate: func(d *appsv1.Deployment) {
				d.Spec.Replicas = nil
				d.Status.UpdatedReplicas = 3
				d.Status.Replicas = 3
				d.Status.AvailableReplicas = 3
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deployment := doneDeployment()
			tt.mutate(deployment)
			assert.Equal(t, tt.expected, isDeploymentRolloutDone(deployment))
		})
	}
}

func TestIsStatefulSetRolloutDone(t *testing.T) {
	tests := []struct {
		name     string
		mutate   func(s *appsv1.StatefulSet)
		expected bool
	}{
		{
			name:     "fully rolled out",
			mutate:   func(s *appsv1.StatefulSet) {},
			expected: true,
		},
		{
			name: "OnDelete statefulsets have no rollout status and are always done",
			mutate: func(s *appsv1.StatefulSet) {
				s.Spec.UpdateStrategy.Type = appsv1.OnDeleteStatefulSetStrategyType
				s.Status.ObservedGeneration = 0
				s.Status.ReadyReplicas = 0
				s.Status.UpdateRevision = "cart-newer"
			},
			expected: true,
		},
		{
			name:     "spec update not observed at all",
			mutate:   func(s *appsv1.StatefulSet) { s.Status.ObservedGeneration = 0 },
			expected: false,
		},
		{
			// An unobserved generation of zero is not "the spec matches"; it means the controller
			// has not written a status yet, which the generation comparison alone cannot tell.
			name: "a statefulset whose status has never been written",
			mutate: func(s *appsv1.StatefulSet) {
				s.Generation = 0
				s.Status.ObservedGeneration = 0
			},
			expected: false,
		},
		{
			name:     "spec change not observed yet",
			mutate:   func(s *appsv1.StatefulSet) { s.Status.ObservedGeneration = 6 },
			expected: false,
		},
		{
			name:     "not all pods are ready",
			mutate:   func(s *appsv1.StatefulSet) { s.Status.ReadyReplicas = 3 },
			expected: false,
		},
		{
			name: "unset spec replicas skips the ready replica check",
			mutate: func(s *appsv1.StatefulSet) {
				s.Spec.Replicas = nil
				s.Status.ReadyReplicas = 0
			},
			expected: true,
		},
		{
			name:     "rolling update still in progress",
			mutate:   func(s *appsv1.StatefulSet) { s.Status.UpdateRevision = "cart-9d1a" },
			expected: false,
		},
		{
			name: "a partitioned rollout is done once the pods above the partition are updated",
			mutate: func(s *appsv1.StatefulSet) {
				s.Spec.UpdateStrategy.RollingUpdate = &appsv1.RollingUpdateStatefulSetStrategy{
					Partition: int32Ptr(1),
				}
				s.Status.UpdatedReplicas = 3
			},
			expected: true,
		},
		{
			name: "a partitioned rollout is not done below the partition boundary",
			mutate: func(s *appsv1.StatefulSet) {
				s.Spec.UpdateStrategy.RollingUpdate = &appsv1.RollingUpdateStatefulSetStrategy{
					Partition: int32Ptr(1),
				}
				s.Status.UpdatedReplicas = 2
			},
			expected: false,
		},
		{
			name: "a partitioned rollout with unset spec replicas skips the partition check",
			mutate: func(s *appsv1.StatefulSet) {
				s.Spec.UpdateStrategy.RollingUpdate = &appsv1.RollingUpdateStatefulSetStrategy{
					Partition: int32Ptr(1),
				}
				s.Spec.Replicas = nil
				s.Status.UpdatedReplicas = 0
			},
			expected: true,
		},
		{
			name: "an explicit rolling update strategy short circuits the revision comparison",
			mutate: func(s *appsv1.StatefulSet) {
				s.Spec.UpdateStrategy.RollingUpdate = &appsv1.RollingUpdateStatefulSetStrategy{}
				s.Status.UpdateRevision = "cart-9d1a"
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			statefulSet := doneStatefulSet()
			tt.mutate(statefulSet)
			assert.Equal(t, tt.expected, isStatefulSetRolloutDone(statefulSet))
		})
	}
}

func TestIsDaemonSetRolloutDone(t *testing.T) {
	tests := []struct {
		name     string
		mutate   func(d *appsv1.DaemonSet)
		expected bool
	}{
		{
			name:     "fully rolled out",
			mutate:   func(d *appsv1.DaemonSet) {},
			expected: true,
		},
		{
			name: "OnDelete daemonsets have no rollout status and are always done",
			mutate: func(d *appsv1.DaemonSet) {
				d.Spec.UpdateStrategy.Type = appsv1.OnDeleteDaemonSetStrategyType
				d.Status.ObservedGeneration = 3
				d.Status.UpdatedNumberScheduled = 0
				d.Status.NumberAvailable = 0
			},
			expected: true,
		},
		{
			name:     "spec change not observed yet",
			mutate:   func(d *appsv1.DaemonSet) { d.Status.ObservedGeneration = 6 },
			expected: false,
		},
		{
			name:     "not every node runs the updated pod",
			mutate:   func(d *appsv1.DaemonSet) { d.Status.UpdatedNumberScheduled = 4 },
			expected: false,
		},
		{
			name:     "updated pods are not all available",
			mutate:   func(d *appsv1.DaemonSet) { d.Status.NumberAvailable = 4 },
			expected: false,
		},
		{
			// Availability is measured against the nodes the daemonset wants to run on, not
			// against the pods it has already updated, which can transiently be more.
			name: "more updated pods than desired nodes does not hold the rollout back",
			mutate: func(d *appsv1.DaemonSet) {
				d.Status.UpdatedNumberScheduled = 6
				d.Status.NumberAvailable = 5
			},
			expected: true,
		},
		{
			name: "a daemonset scheduled on no node is done",
			mutate: func(d *appsv1.DaemonSet) {
				d.Status.DesiredNumberScheduled = 0
				d.Status.UpdatedNumberScheduled = 0
				d.Status.NumberAvailable = 0
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			daemonSet := doneDaemonSet()
			tt.mutate(daemonSet)
			assert.Equal(t, tt.expected, isDaemonSetRolloutDone(daemonSet))
		})
	}
}

func TestIsDeploymentConfigRolloutDone(t *testing.T) {
	tests := []struct {
		name     string
		mutate   func(dc *openshiftappsv1.DeploymentConfig)
		expected bool
	}{
		{
			name:     "fully rolled out",
			mutate:   func(dc *openshiftappsv1.DeploymentConfig) {},
			expected: true,
		},
		{
			name:     "spec change not observed yet",
			mutate:   func(dc *openshiftappsv1.DeploymentConfig) { dc.Status.ObservedGeneration = 6 },
			expected: false,
		},
		{
			name:     "old replicas are still terminating",
			mutate:   func(dc *openshiftappsv1.DeploymentConfig) { dc.Status.Replicas = 5 },
			expected: false,
		},
		{
			name:     "updated replicas are not all available",
			mutate:   func(dc *openshiftappsv1.DeploymentConfig) { dc.Status.AvailableReplicas = 3 },
			expected: false,
		},
		{
			name:     "some replicas are reported unavailable",
			mutate:   func(dc *openshiftappsv1.DeploymentConfig) { dc.Status.UnavailableReplicas = 1 },
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deploymentConfig := doneDeploymentConfig()
			tt.mutate(deploymentConfig)
			assert.Equal(t, tt.expected, isDeploymentConfigRolloutDone(deploymentConfig))
		})
	}
}

// IsWorkloadRolloutDone is the entry point every caller uses. Each supported kind must reach its
// own implementation, so the "not done" fixture for each kind is broken in a way that only that
// kind's implementation looks at.
func TestIsWorkloadRolloutDoneDispatchesPerWorkloadKind(t *testing.T) {
	tests := []struct {
		name     string
		obj      metav1.Object
		expected bool
	}{
		{name: "deployment done", obj: doneDeployment(), expected: true},
		{name: "statefulset done", obj: doneStatefulSet(), expected: true},
		{name: "daemonset done", obj: doneDaemonSet(), expected: true},
		{name: "deploymentconfig done", obj: doneDeploymentConfig(), expected: true},
		{name: "argo rollout done", obj: doneArgoRollout(), expected: true},
		{
			name: "deployment waiting on available replicas",
			obj: func() metav1.Object {
				d := doneDeployment()
				d.Status.AvailableReplicas = 3
				return d
			}(),
			expected: false,
		},
		{
			name: "statefulset waiting on a revision roll",
			obj: func() metav1.Object {
				s := doneStatefulSet()
				s.Status.UpdateRevision = "cart-9d1a"
				return s
			}(),
			expected: false,
		},
		{
			name: "daemonset waiting on nodes to pick up the update",
			obj: func() metav1.Object {
				d := doneDaemonSet()
				d.Status.UpdatedNumberScheduled = 4
				return d
			}(),
			expected: false,
		},
		{
			name: "deploymentconfig with unavailable replicas",
			obj: func() metav1.Object {
				dc := doneDeploymentConfig()
				dc.Status.UnavailableReplicas = 2
				return dc
			}(),
			expected: false,
		},
		{
			name: "argo rollout degraded",
			obj: func() metav1.Object {
				r := doneArgoRollout()
				r.Status.Phase = argorolloutsv1alpha1.RolloutPhaseDegraded
				return r
			}(),
			expected: false,
		},
		{
			name:     "an unsupported kind is never considered rolled out",
			obj:      &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "frontend-abc", Namespace: rolloutTestNamespace}},
			expected: false,
		},
		{
			name:     "a replicaset is not a workload odigos rolls out",
			obj:      &appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{Name: "frontend-7c9f", Namespace: rolloutTestNamespace}},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsWorkloadRolloutDone(tt.obj))
		})
	}
}

// The selector is what turns a workload into the set of pods to inspect. Each kind carries a
// distinct selector value so that reading it off the wrong workload kind cannot pass.
func TestGetMatchLabels(t *testing.T) {
	tests := []struct {
		name     string
		obj      metav1.Object
		expected map[string]string
	}{
		{name: "deployment", obj: doneDeployment(), expected: map[string]string{"app": "frontend"}},
		{name: "statefulset", obj: doneStatefulSet(), expected: map[string]string{"app": "cart"}},
		{name: "daemonset", obj: doneDaemonSet(), expected: map[string]string{"app": "log-shipper"}},
		{
			name:     "deploymentconfig uses a plain selector map rather than a label selector",
			obj:      doneDeploymentConfig(),
			expected: map[string]string{"app": "billing"},
		},
		{name: "argo rollout", obj: doneArgoRollout(), expected: map[string]string{"app": "search"}},
		{
			name:     "unsupported kind",
			obj:      &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "frontend-abc"}},
			expected: nil,
		},
		{
			name: "a selector with no match labels",
			obj: func() metav1.Object {
				d := doneDeployment()
				d.Spec.Selector = &metav1.LabelSelector{}
				return d
			}(),
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, GetMatchLabels(tt.obj))
		})
	}
}
