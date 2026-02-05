package agentenabled

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/instrumentor/internal/testutil"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ****************
// hasUninstrumentedPodsWithBackoff() tests
// ****************

func TestHasUninstrumentedPodsWithBackoff_NoPods(t *testing.T) {
	// Arrange: Deployment exists but has no pods
	setup := newSyncTestSetup()
	deployment := testutil.NewMockTestDeployment(setup.ns, "test-deployment")
	ic := testutil.NewMockInstrumentationConfig(deployment)
	client := setup.newFakeClient(setup.ns, deployment, ic)
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	// Act
	condition, err := hasUninstrumentedPodsWithBackoff(setup.ctx, client, pw, ic, setup.logger)

	// Assert: No backoff detected when no pods exist
	assert.NoError(t, err)
	assert.Nil(t, condition, "expected no condition when no pods exist")
}

func TestHasUninstrumentedPodsWithBackoff_HealthyPods(t *testing.T) {
	// Arrange: Deployment with healthy running pods
	setup := newSyncTestSetup()
	deployment := testutil.NewMockTestDeployment(setup.ns, "test-deployment")
	ic := testutil.NewMockInstrumentationConfig(deployment)
	healthyPod := newHealthyPod(setup.ns, "test-deployment", "healthy-pod-1")
	client := setup.newFakeClient(setup.ns, deployment, ic, healthyPod)
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	// Act
	condition, err := hasUninstrumentedPodsWithBackoff(setup.ctx, client, pw, ic, setup.logger)

	// Assert: No backoff detected with healthy pods
	assert.NoError(t, err)
	assert.Nil(t, condition, "expected no condition with healthy pods")
}

func TestHasUninstrumentedPodsWithBackoff_CrashLoopBackOff(t *testing.T) {
	// Arrange: Deployment with pod in CrashLoopBackOff WITHOUT odigos label (pre-existing crash)
	setup := newSyncTestSetup()
	deployment := testutil.NewMockTestDeployment(setup.ns, "test-deployment")
	ic := testutil.NewMockInstrumentationConfig(deployment)
	crashingPod := newCrashLoopBackOffPodWithoutOdigosLabel(setup.ns, "test-deployment", "crashing-pod")
	client := setup.newFakeClient(setup.ns, deployment, ic, crashingPod)
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	// Act
	condition, err := hasUninstrumentedPodsWithBackoff(setup.ctx, client, pw, ic, setup.logger)

	// Assert: Backoff detected - condition returned
	assert.NoError(t, err)
	assert.NotNil(t, condition, "expected condition when backoff detected")
	assert.Equal(t, metav1.ConditionFalse, condition.Status)
	assert.Equal(t, odigosv1alpha1.AgentEnabledReasonCrashLoopBackOff, condition.Reason)
}

func TestHasUninstrumentedPodsWithBackoff_CrashLoopBackOff_WithOdigosLabel(t *testing.T) {
	// Arrange: Deployment with crashing pod that HAS odigos label (already instrumented by Odigos)
	setup := newSyncTestSetup()
	deployment := testutil.NewMockTestDeployment(setup.ns, "test-deployment")
	ic := testutil.NewMockInstrumentationConfig(deployment)
	crashingPod := newCrashLoopBackOffPodWithOdigosLabel(setup.ns, "test-deployment", "crashing-pod")
	client := setup.newFakeClient(setup.ns, deployment, ic, crashingPod)
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	// Act
	condition, err := hasUninstrumentedPodsWithBackoff(setup.ctx, client, pw, ic, setup.logger)

	// Assert: No backoff detected - pods with odigos label are handled by rollback logic
	assert.NoError(t, err)
	assert.Nil(t, condition, "expected no condition for pods with odigos label")
}

func TestHasUninstrumentedPodsWithBackoff_WorkloadNotFound(t *testing.T) {
	// Arrange: Workload does not exist
	setup := newSyncTestSetup()
	ic := &odigosv1alpha1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment-nonexistent",
			Namespace: setup.ns.Name,
		},
	}
	client := setup.newFakeClient(setup.ns, ic)
	pw := k8sconsts.PodWorkload{Name: "nonexistent", Namespace: setup.ns.Name, Kind: k8sconsts.WorkloadKindDeployment}

	// Act
	condition, err := hasUninstrumentedPodsWithBackoff(setup.ctx, client, pw, ic, setup.logger)

	// Assert: No error, no condition - workload not found is handled gracefully
	assert.NoError(t, err)
	assert.Nil(t, condition, "expected no condition when workload not found")
}

func TestHasUninstrumentedPodsWithBackoff_CronJob_Skipped(t *testing.T) {
	// Arrange: CronJob workload - should skip backoff check entirely
	setup := newSyncTestSetup()
	cronJob := newMockCronJob(setup.ns, "test-cronjob")
	ic := testutil.NewMockInstrumentationConfig(cronJob)
	client := setup.newFakeClient(setup.ns, cronJob, ic)
	pw := k8sconsts.PodWorkload{Name: cronJob.Name, Namespace: cronJob.Namespace, Kind: k8sconsts.WorkloadKindCronJob}

	// Act
	condition, err := hasUninstrumentedPodsWithBackoff(setup.ctx, client, pw, ic, setup.logger)

	// Assert: No error, no condition - CronJob workloads skip the backoff check
	assert.NoError(t, err)
	assert.Nil(t, condition, "expected no condition for CronJob workloads")
}

func TestHasUninstrumentedPodsWithBackoff_Job_Skipped(t *testing.T) {
	// Arrange: Job workload - should skip backoff check entirely
	setup := newSyncTestSetup()
	job := newMockJob(setup.ns, "test-job")
	ic := testutil.NewMockInstrumentationConfig(job)
	client := setup.newFakeClient(setup.ns, job, ic)
	pw := k8sconsts.PodWorkload{Name: job.Name, Namespace: job.Namespace, Kind: k8sconsts.WorkloadKindJob}

	// Act
	condition, err := hasUninstrumentedPodsWithBackoff(setup.ctx, client, pw, ic, setup.logger)

	// Assert: No error, no condition - Job workloads skip the backoff check
	assert.NoError(t, err)
	assert.Nil(t, condition, "expected no condition for Job workloads")
}
