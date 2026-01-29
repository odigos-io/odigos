package rollout_test

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/instrumentor/controllers/agentenabled/rollout"
	"github.com/odigos-io/odigos/instrumentor/internal/testutil"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ****************
// WorkloadHasPodInBackoff() tests
// ****************

func TestWorkloadHasPodInBackoff_NoPods(t *testing.T) {
	// Arrange: Deployment exists but has no pods
	setup := newTestSetup()
	deployment := testutil.NewMockTestDeployment(setup.ns, "test-deployment")
	c := setup.newFakeClient(setup.ns, deployment)

	// Act
	hasBackoff, err := rollout.WorkloadHasPodInBackoff(setup.ctx, c, deployment)

	// Assert: No backoff detected when no pods exist
	assert.NoError(t, err)
	assert.False(t, hasBackoff)
}

func TestWorkloadHasPodInBackoff_HealthyPods(t *testing.T) {
	// Arrange: Deployment with healthy running pods
	setup := newTestSetup()
	deployment := testutil.NewMockTestDeployment(setup.ns, "test-deployment")
	healthyPod := newHealthyPod(setup.ns, "test-deployment", "healthy-pod-1")
	c := setup.newFakeClient(setup.ns, deployment, healthyPod)

	// Act
	hasBackoff, err := rollout.WorkloadHasPodInBackoff(setup.ctx, c, deployment)

	// Assert: No backoff detected with healthy pods
	assert.NoError(t, err)
	assert.False(t, hasBackoff)
}

func TestWorkloadHasPodInBackoff_CrashLoopBackOff(t *testing.T) {
	// Arrange: Deployment with pod in CrashLoopBackOff WITHOUT odigos label (pre-existing crash)
	setup := newTestSetup()
	deployment := testutil.NewMockTestDeployment(setup.ns, "test-deployment")
	crashingPod := newCrashLoopBackOffPodWithoutOdigosLabel(setup.ns, "test-deployment", "crashing-pod")
	c := setup.newFakeClient(setup.ns, deployment, crashingPod)

	// Act
	hasBackoff, err := rollout.WorkloadHasPodInBackoff(setup.ctx, c, deployment)

	// Assert: Backoff detected - this is a pre-existing crash, not caused by Odigos
	assert.NoError(t, err)
	assert.True(t, hasBackoff)
}

func TestWorkloadHasPodInBackoff_CrashLoopBackOff_WithOdigosLabel(t *testing.T) {
	// Arrange: Deployment with crashing pod that HAS odigos label (instrumented by Odigos)
	setup := newTestSetup()
	deployment := testutil.NewMockTestDeployment(setup.ns, "test-deployment")
	crashingPod := newCrashLoopBackOffPod(setup.ns, "test-deployment", "crashing-pod")
	c := setup.newFakeClient(setup.ns, deployment, crashingPod)

	// Act
	hasBackoff, err := rollout.WorkloadHasPodInBackoff(setup.ctx, c, deployment)

	// Assert: No backoff detected - pods with odigos label are handled by rollback logic instead
	assert.NoError(t, err)
	assert.False(t, hasBackoff)
}

func TestWorkloadHasPodInBackoff_ImagePullBackOff(t *testing.T) {
	// Arrange: Deployment with pod in ImagePullBackOff (no odigos label)
	setup := newTestSetup()
	deployment := testutil.NewMockTestDeployment(setup.ns, "test-deployment")
	imagePullPod := newImagePullBackOffPod(setup.ns, "test-deployment", "image-pull-pod")
	c := setup.newFakeClient(setup.ns, deployment, imagePullPod)

	// Act
	hasBackoff, err := rollout.WorkloadHasPodInBackoff(setup.ctx, c, deployment)

	// Assert: Backoff detected for ImagePullBackOff - we don't want to instrument the workload
	assert.NoError(t, err)
	assert.True(t, hasBackoff)
}
func TestWorkloadHasPodInBackoff_MixedPods(t *testing.T) {
	// Arrange: Deployment with mix of healthy pods and one crashing pod (no odigos label)
	setup := newTestSetup()
	deployment := testutil.NewMockTestDeployment(setup.ns, "test-deployment")
	healthyPod1 := newHealthyPod(setup.ns, "test-deployment", "healthy-pod-1")
	healthyPod2 := newHealthyPod(setup.ns, "test-deployment", "healthy-pod-2")
	crashingPod := newCrashLoopBackOffPodWithoutOdigosLabel(setup.ns, "test-deployment", "crashing-pod")
	c := setup.newFakeClient(setup.ns, deployment, healthyPod1, healthyPod2, crashingPod)

	// Act
	hasBackoff, err := rollout.WorkloadHasPodInBackoff(setup.ctx, c, deployment)

	// Assert: Backoff detected even when some pods are healthy - we don't want to instrument the workload
	assert.NoError(t, err)
	assert.True(t, hasBackoff)
}

func TestWorkloadHasPodInBackoff_InitContainerBackoff(t *testing.T) {
	// Arrange: Deployment with pod whose init container is in CrashLoopBackOff
	setup := newTestSetup()
	deployment := testutil.NewMockTestDeployment(setup.ns, "test-deployment")
	podWithInitBackoff := newInitContainerBackOffPod(setup.ns, "test-deployment", "init-crash-pod")
	c := setup.newFakeClient(setup.ns, deployment, podWithInitBackoff)

	// Act
	hasBackoff, err := rollout.WorkloadHasPodInBackoff(setup.ctx, c, deployment)

	// Assert: Backoff detected for init container in CrashLoopBackOff - we don't want to instrument the workload
	assert.NoError(t, err)
	assert.True(t, hasBackoff)
}

// ****************
// Do() tests
// ****************

func TestDo_WorkloadHasPodInBackoff_UpdateError(t *testing.T) {
	// Arrange: Deployment with pod in backoff state, but IC update will fail
	setup := newTestSetup()
	deployment := testutil.NewMockTestDeployment(setup.ns, "test-deployment")
	podInBackoff := newCrashLoopBackOffPodWithoutOdigosLabel(setup.ns, "test-deployment", "pod-in-backoff")
	ic := testutil.NewMockInstrumentationConfig(deployment)
	c := setup.newFakeClientWithICUpdateError(setup.ns, deployment, podInBackoff, ic)
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	// Act
	statusChanged, _, err := rollout.Do(setup.ctx, c, ic, pw, setup.conf, setup.distroProvider)

	// Assert: Error returned, no status change
	assert.Error(t, err)
	assert.False(t, statusChanged, "expected no status change when update fails")
}

func TestDo_WorkloadHasPodInBackoff(t *testing.T) {
	// Arrange: Deployment with pod in backoff state (without odigos label - pre-existing crash)
	setup := newTestSetup()
	deployment := testutil.NewMockTestDeployment(setup.ns, "test-deployment")
	podInBackoff := newCrashLoopBackOffPodWithoutOdigosLabel(setup.ns, "test-deployment", "pod-in-backoff")
	ic := testutil.NewMockInstrumentationConfig(deployment)
	c := setup.newFakeClient(setup.ns, deployment, podInBackoff, ic)
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	// Act
	statusChanged, result, err := rollout.Do(setup.ctx, c, ic, pw, setup.conf, setup.distroProvider)

	// Assert: Status changed (instrumentation disabled), no error, no requeue
	assert.NoError(t, err)
	assert.True(t, statusChanged, "expected status change when preventing instrumentation")
	assert.Empty(t, result, "expected no requeue when preventing instrumentation")

	// Assert: Instrumentation config has AgentInjectionEnabled as false
	assert.False(t, ic.Spec.AgentInjectionEnabled)

	// Assert: all containers have AgentEnabled as false
	for _, container := range ic.Spec.Containers {
		assert.False(t, container.AgentEnabled)
	}

	// Assert: we add conditions to the instrumentation config
	assert.Equal(t, odigosv1alpha1.WorkloadRolloutStatusConditionType, ic.Status.Conditions[0].Type)
	assert.Equal(t, metav1.ConditionFalse, ic.Status.Conditions[0].Status)
	assert.Equal(t, odigosv1alpha1.AgentEnabledStatusConditionType, ic.Status.Conditions[1].Type)
	assert.Equal(t, metav1.ConditionFalse, ic.Status.Conditions[1].Status)
}
