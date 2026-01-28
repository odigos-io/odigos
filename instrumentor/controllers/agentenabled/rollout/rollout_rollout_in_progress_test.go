package rollout_test

import (
	"testing"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/instrumentor/controllers/agentenabled/rollout"
	"github.com/odigos-io/odigos/instrumentor/internal/testutil"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func Test_NoRollout_PodInMidRollout_AlreadyComplete(t *testing.T) {
	// Arrange: IC shows mid-rollout state but deployment has already completed rolling out (no pending replicas)
	s := newTestSetup()
	deployment := testutil.NewMockTestDeployment(s.ns, "test-deployment")
	ic := mockICMidRollout(testutil.NewMockInstrumentationConfig(deployment))
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	fakeClient := s.newFakeClient(deployment)

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider)

	// Assert: Status changed - workload rollout already complete
	assert.NoError(t, err)
	assert.True(t, statusChanged, "expected status change")
	assert.Equal(t, reconcile.Result{}, result)
}

func Test_NoRollout_PodInMidRollout_WaitingNoBackoff(t *testing.T) {
	// Arrange: Mid-rollout deployment with healthy running pod - no crashloop, just waiting for rollout
	s := newTestSetup()
	deployment := newMockDeploymentMidRollout(s.ns, "test-deployment")
	ic := mockICMidRollout(testutil.NewMockInstrumentationConfig(deployment))
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}
	now := metav1.Now()
	ic.Status.InstrumentationTime = &now
	ic.Spec.AgentInjectionEnabled = true

	// Create a healthy pod
	healthyPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "healthy-pod",
			Namespace: s.ns.Name,
			Labels: map[string]string{
				"app.kubernetes.io/name":            deployment.Name,
				k8sconsts.OdigosAgentsMetaHashLabel: ic.Spec.AgentsMetaHash,
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  "test",
					Ready: true,
					State: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{},
					},
				},
			},
		},
	}

	fakeClient := s.newFakeClient(deployment, healthyPod)

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider)

	// Assert: No status change, requeue after 10s to continue monitoring rollout progress
	assert.NoError(t, err)
	assert.False(t, statusChanged, "expected no status change")
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, result)
}

func Test_TriggeredRollout_PodInMidRollout_PreviousRolloutOngoing(t *testing.T) {
	// Arrange: Deployment mid-rollout with different hash in IC status - a previous rollout is still in progress
	s := newTestSetup()
	deployment := newMockDeploymentMidRollout(s.ns, "test-deployment")
	ic := mockICRolloutRequiredDistro(testutil.NewMockInstrumentationConfig(deployment))
	// Set a DIFFERENT hash to trigger new rollout path (not mid-rollout)
	ic.Status.WorkloadRolloutHash = "old-hash"
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	fakeClient := s.newFakeClient(deployment)

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider)

	// Assert: Status updated to "PreviousRolloutOngoing", requeue to wait for previous rollout
	assert.NoError(t, err)
	assert.True(t, statusChanged, "expected status change")
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, result)
	assert.Equal(t, string(odigosv1alpha1.WorkloadRolloutReasonPreviousRolloutOngoing), ic.Status.Conditions[0].Reason)
}

func Test_TriggeredRolloutRequeue_RolloutInProgress_TriggeredExternally(t *testing.T) {
	// Arrange: Deployment looks complete by status fields, simulating a rollout triggered externally (e.g., kubectl rollout restart)
	s := newTestSetup()
	deployment := testutil.NewMockTestDeployment(s.ns, "test-deployment")
	// Simulate an external rollout restart: status *looks* healthy (replicas ready),
	// but the new spec hasn't been observed yet (Generation > ObservedGeneration).
	replicas := int32(1)
	deployment.Spec.Replicas = &replicas
	deployment.Generation = 2
	deployment.Status.ObservedGeneration = 1
	deployment.Status.Replicas = replicas
	deployment.Status.UpdatedReplicas = replicas
	deployment.Status.AvailableReplicas = replicas
	ic := mockICRolloutRequiredDistro(testutil.NewMockInstrumentationConfig(deployment))
	// Set a DIFFERENT hash to trigger new rollout path
	ic.Status.WorkloadRolloutHash = "old-hash"
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	fakeClient := s.newFakeClient(deployment)

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider)

	// Assert: Status updated to "PreviousRolloutOngoing", requeue to wait for previous rollout
	assert.NoError(t, err)
	assert.True(t, statusChanged, "expected status change")
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, result)
	assert.Equal(t, string(odigosv1alpha1.WorkloadRolloutReasonPreviousRolloutOngoing), ic.Status.Conditions[0].Reason)
}
