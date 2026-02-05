package rollout_test

import (
	"testing"

	"github.com/odigos-io/odigos/instrumentor/controllers/agentenabled/rollout"
	"github.com/odigos-io/odigos/instrumentor/internal/testutil"
	"github.com/stretchr/testify/assert"
)

// ****************
// WorkloadHasNonInstrumentedPodInBackoff() tests
// ****************

func TestWorkloadHasPodInBackoff_NoPods(t *testing.T) {
	// Arrange: Deployment exists but has no pods
	setup := newTestSetup()
	deployment := testutil.NewMockTestDeployment(setup.ns, "test-deployment")
	c := setup.newFakeClient(setup.ns, deployment)

	// Act
	hasBackoff, err := rollout.WorkloadHasNonInstrumentedPodInBackoff(setup.ctx, c, deployment)

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
	hasBackoff, err := rollout.WorkloadHasNonInstrumentedPodInBackoff(setup.ctx, c, deployment)

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
	hasBackoff, err := rollout.WorkloadHasNonInstrumentedPodInBackoff(setup.ctx, c, deployment)

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
	hasBackoff, err := rollout.WorkloadHasNonInstrumentedPodInBackoff(setup.ctx, c, deployment)

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
	hasBackoff, err := rollout.WorkloadHasNonInstrumentedPodInBackoff(setup.ctx, c, deployment)

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
	hasBackoff, err := rollout.WorkloadHasNonInstrumentedPodInBackoff(setup.ctx, c, deployment)

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
	hasBackoff, err := rollout.WorkloadHasNonInstrumentedPodInBackoff(setup.ctx, c, deployment)

	// Assert: Backoff detected for init container in CrashLoopBackOff - we don't want to instrument the workload
	assert.NoError(t, err)
	assert.True(t, hasBackoff)
}

func TestWorkloadHasPodInBackoff_StaticPod(t *testing.T) {
	// Arrange: StaticPod in CrashLoopBackOff - the workload object IS the pod, so it checks itself directly
	setup := newTestSetup()
	crashingStaticPod := newCrashLoopBackOffStaticPod(setup.ns, "test-staticpod")
	c := setup.newFakeClient(setup.ns, crashingStaticPod)

	// Act
	hasBackoff, err := rollout.WorkloadHasNonInstrumentedPodInBackoff(setup.ctx, c, crashingStaticPod)

	// Assert: Backoff detected for StaticPod - we don't want to instrument the workload
	assert.NoError(t, err)
	assert.True(t, hasBackoff)
}
