package rollout_test

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/instrumentor/controllers/agentenabled/rollout"
	"github.com/odigos-io/odigos/instrumentor/internal/testutil"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ****************
// Rollout Concurrency Limiting - Instrumentation Tests
// ****************

func Test_Instrumentation_RateLimited_WaitingInQueue(t *testing.T) {
	// Arrange: IC requires rollout but rate limiter is exhausted
	s := newTestSetup()
	setConfigConcurrentRolloutLimit(s.conf, 1) // Set limit to 1 in configuration
	deployment := testutil.NewMockTestDeployment(s.ns, "test-deployment")
	ic := mockICRolloutRequiredDistro(testutil.NewMockInstrumentationConfig(deployment))
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	fakeClient := s.newFakeClient(deployment)
	limiter := newRolloutConcurrencyLimiterExhausted() // limiter with one slot already taken

	// Act
	rolloutResult, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider, limiter)

	// Assert: Status set to WaitingInQueue, requeued for later retry
	assert.NoError(t, err)
	assert.True(t, rolloutResult.StatusChanged, "expected status change to WaitingInQueue")
	assert.Equal(t, reconcile.Result{RequeueAfter: rollout.RequeueWaitingForWorkloadRollout}, rolloutResult.Result)

	// Verify the condition is set to WaitingInQueue
	assert.NotEmpty(t, ic.Status.Conditions, "expected conditions to be set")
	assert.Equal(t, string(odigosv1alpha1.WorkloadRolloutReasonWaitingInQueue), ic.Status.Conditions[0].Reason)
	assert.Equal(t, "Waiting for other workload rollouts to complete", ic.Status.Conditions[0].Message)

	// Verify workload was NOT restarted (rate limited)
	assertWorkloadNotRestarted(t, s.ctx, fakeClient, pw)
}

func Test_Instrumentation_FirstWorkload_AllowedByRateLimiter(t *testing.T) {
	// Arrange: IC requires rollout, rate limiter has available tokens
	s := newTestSetup()
	setConfigConcurrentRolloutLimit(s.conf, 1) // Set limit to 1 in configuration
	deployment := testutil.NewMockTestDeployment(s.ns, "test-deployment")
	ic := mockICRolloutRequiredDistro(testutil.NewMockInstrumentationConfig(deployment))
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	fakeClient := s.newFakeClient(deployment)
	limiter := newRolloutConcurrencyLimiterActive() // Fresh limiter with no slots taken

	// Act
	rolloutResult, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider, limiter)

	// Assert: Rollout triggered successfully
	assertTriggeredRolloutWithRequeue(t, rolloutResult, err)
	assert.Equal(t, string(odigosv1alpha1.WorkloadRolloutReasonTriggeredSuccessfully), ic.Status.Conditions[0].Reason)
	assertWorkloadRestarted(t, s.ctx, fakeClient, pw)
}

func Test_Instrumentation_SecondWorkload_RateLimited(t *testing.T) {
	// Arrange: Two workloads need instrumentation, rate limiter allows only 1
	s := newTestSetup()
	setConfigConcurrentRolloutLimit(s.conf, 1) // Set limit to 1 in configuration

	// First deployment
	deployment1 := testutil.NewMockTestDeployment(s.ns, "deployment-1")
	ic1 := mockICRolloutRequiredDistro(testutil.NewMockInstrumentationConfig(deployment1))
	pw1 := k8sconsts.PodWorkload{Name: deployment1.Name, Namespace: deployment1.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	// Second deployment
	deployment2 := testutil.NewMockTestDeployment(s.ns, "deployment-2")
	ic2 := mockICRolloutRequiredDistro(testutil.NewMockInstrumentationConfig(deployment2))
	pw2 := k8sconsts.PodWorkload{Name: deployment2.Name, Namespace: deployment2.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	fakeClient := s.newFakeClient(deployment1, deployment2)
	limiter := newRolloutConcurrencyLimiterActive() // Fresh limiter

	// Act: First workload
	rolloutResult1, err1 := rollout.Do(s.ctx, fakeClient, ic1, pw1, s.conf, s.distroProvider, limiter)

	// Assert: First workload succeeds
	assertTriggeredRolloutWithRequeue(t, rolloutResult1, err1)
	assert.Equal(t, string(odigosv1alpha1.WorkloadRolloutReasonTriggeredSuccessfully), ic1.Status.Conditions[0].Reason)

	// Act: Second workload (should be rate limited)
	rolloutResult2, err2 := rollout.Do(s.ctx, fakeClient, ic2, pw2, s.conf, s.distroProvider, limiter)

	// Assert: Second workload is rate limited
	assert.NoError(t, err2)
	assert.True(t, rolloutResult2.StatusChanged, "expected status change to WaitingInQueue")
	assert.Equal(t, reconcile.Result{RequeueAfter: rollout.RequeueWaitingForWorkloadRollout}, rolloutResult2.Result)
	assert.Equal(t, string(odigosv1alpha1.WorkloadRolloutReasonWaitingInQueue), ic2.Status.Conditions[0].Reason)
}

func Test_Instrumentation_MultipleWorkloads_HigherLimitAllowsMore(t *testing.T) {
	// Arrange: Three workloads need instrumentation, rate limiter allows 2
	s := newTestSetup()
	setConfigConcurrentRolloutLimit(s.conf, 2) // Set limit to 2 in configuration

	deployments := make([]*appsv1.Deployment, 3)
	ics := make([]*odigosv1alpha1.InstrumentationConfig, 3)
	pws := make([]k8sconsts.PodWorkload, 3)

	for i := 0; i < 3; i++ {
		deployments[i] = testutil.NewMockTestDeployment(s.ns, "deployment-"+string(rune('a'+i)))
		ics[i] = mockICRolloutRequiredDistro(testutil.NewMockInstrumentationConfig(deployments[i]))
		pws[i] = k8sconsts.PodWorkload{Name: deployments[i].Name, Namespace: deployments[i].Namespace, Kind: k8sconsts.WorkloadKindDeployment}
	}

	fakeClient := s.newFakeClient(deployments[0], deployments[1], deployments[2])
	limiter := newRolloutConcurrencyLimiterActive() // Fresh limiter

	// Act & Assert: First two workloads succeed
	for i := 0; i < 2; i++ {
		rolloutResult, err := rollout.Do(s.ctx, fakeClient, ics[i], pws[i], s.conf, s.distroProvider, limiter)
		assertTriggeredRolloutWithRequeue(t, rolloutResult, err)
		assert.Equal(t, string(odigosv1alpha1.WorkloadRolloutReasonTriggeredSuccessfully), ics[i].Status.Conditions[0].Reason,
			"workload %d should have succeeded", i)
	}

	// Act & Assert: Third workload is rate limited
	rolloutResult, err := rollout.Do(s.ctx, fakeClient, ics[2], pws[2], s.conf, s.distroProvider, limiter)
	assert.NoError(t, err)
	assert.True(t, rolloutResult.StatusChanged)
	assert.Equal(t, reconcile.Result{RequeueAfter: rollout.RequeueWaitingForWorkloadRollout}, rolloutResult.Result)
	assert.Equal(t, string(odigosv1alpha1.WorkloadRolloutReasonWaitingInQueue), ics[2].Status.Conditions[0].Reason,
		"workload 2 should be rate limited")
}

// ****************
// Rate Limiting - De-instrumentation Tests
// ****************

func Test_Deinstrumentation_RateLimited_NoRequeue(t *testing.T) {
	// Arrange: Instrumented pod needs de-instrumentation, but rate limiter is exhausted
	s := newTestSetup()
	setConfigConcurrentRolloutLimit(s.conf, 1) // Set limit to 1 in configuration
	deployment := testutil.NewMockTestDeployment(s.ns, "test-deployment")
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	// Pod with odigos agent label (needs de-instrumentation)
	instrumentedPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: s.ns.Name,
			Labels: map[string]string{
				"app.kubernetes.io/name":            deployment.Name,
				k8sconsts.OdigosAgentsMetaHashLabel: "abc123",
			},
		},
	}

	fakeClient := s.newFakeClient(deployment, instrumentedPod)
	var ic *odigosv1alpha1.InstrumentationConfig // nil IC = de-instrumentation
	limiter := newRolloutConcurrencyLimiterExhausted() // Limiter with one slot already taken

	// Act
	rolloutResult, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider, limiter)

	// Assert: De-instrumentation is not rate limited, no requeue
	assertNoStatusChange(t, rolloutResult, err)

	// Verify workload was restarted - rate limited de-instrumentation is not supported
	assertWorkloadRestarted(t, s.ctx, fakeClient, pw)
}

func Test_Deinstrumentation_AllowedByRateLimiter(t *testing.T) {
	// Arrange: Instrumented pod needs de-instrumentation, rate limiter has tokens
	s := newTestSetup()
	setConfigConcurrentRolloutLimit(s.conf, 1) // Set limit to 1 in configuration
	deployment := testutil.NewMockTestDeployment(s.ns, "test-deployment")
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	instrumentedPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: s.ns.Name,
			Labels: map[string]string{
				"app.kubernetes.io/name":            deployment.Name,
				k8sconsts.OdigosAgentsMetaHashLabel: "abc123",
			},
		},
	}

	fakeClient := s.newFakeClient(deployment, instrumentedPod)
	var ic *odigosv1alpha1.InstrumentationConfig
	limiter := newRolloutConcurrencyLimiterActive() // Fresh limiter with no slots taken

	// Act
	rolloutResult, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider, limiter)

	// Assert: De-instrumentation proceeds - workload is restarted
	assertNoStatusChange(t, rolloutResult, err) // No IC status change (IC is nil)
	assertWorkloadRestarted(t, s.ctx, fakeClient, pw)
}

// ****************
// Rate Limiting - No Rate Limiting (Infinite Limit)
// ****************

func Test_NoRateLimit_AllWorkloadsProcessedImmediately(t *testing.T) {
	// Arrange: Multiple workloads with no rate limiting (infinite limit)
	s := newTestSetup()
	setConfigConcurrentRolloutLimit(s.conf, 0) // Set limit to 0 = no rate limiting

	deployments := make([]*appsv1.Deployment, 5)
	ics := make([]*odigosv1alpha1.InstrumentationConfig, 5)
	pws := make([]k8sconsts.PodWorkload, 5)

	for i := 0; i < 5; i++ {
		deployments[i] = testutil.NewMockTestDeployment(s.ns, "deployment-"+string(rune('a'+i)))
		ics[i] = mockICRolloutRequiredDistro(testutil.NewMockInstrumentationConfig(deployments[i]))
		pws[i] = k8sconsts.PodWorkload{Name: deployments[i].Name, Namespace: deployments[i].Namespace, Kind: k8sconsts.WorkloadKindDeployment}
	}

	fakeClient := s.newFakeClient(deployments[0], deployments[1], deployments[2], deployments[3], deployments[4])
	limiter := newRolloutConcurrencyLimiterActive() // Limiter (limit comes from config)

	// Act & Assert: All workloads succeed
	for i := 0; i < 5; i++ {
		rolloutResult, err := rollout.Do(s.ctx, fakeClient, ics[i], pws[i], s.conf, s.distroProvider, limiter)
		assertTriggeredRolloutWithRequeue(t, rolloutResult, err)
		assert.Equal(t, string(odigosv1alpha1.WorkloadRolloutReasonTriggeredSuccessfully), ics[i].Status.Conditions[0].Reason,
			"workload %d should have succeeded with no rate limit", i)
	}
}

// ****************
// Rate Limiting - Edge Cases
// ****************

func Test_RateLimiting_NilRateLimiter_FailsOpen(t *testing.T) {
	// Arrange: nil rate limiter should allow the rollout (fail-open behavior)
	s := newTestSetup()
	setConfigConcurrentRolloutLimit(s.conf, 1) // Even with limit set, nil limiter fails open
	deployment := testutil.NewMockTestDeployment(s.ns, "test-deployment")
	ic := mockICRolloutRequiredDistro(testutil.NewMockInstrumentationConfig(deployment))
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	fakeClient := s.newFakeClient(deployment)

	// Act: Pass nil rate limiter
	rolloutResult, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider, nil)

	// Assert: Rollout proceeds (nil limiter fails open)
	assertTriggeredRolloutWithRequeue(t, rolloutResult, err)
	assert.Equal(t, string(odigosv1alpha1.WorkloadRolloutReasonTriggeredSuccessfully), ic.Status.Conditions[0].Reason)
}

func Test_RateLimiting_WorkloadNotRequiringRollout_NotAffected(t *testing.T) {
	// Arrange: Workload that doesn't require rollout (e.g., native instrumentation)
	// should NOT consume rate limiter tokens
	s := newTestSetup()
	setConfigConcurrentRolloutLimit(s.conf, 1) // Set limit to 1 in configuration
	deployment := testutil.NewMockTestDeployment(s.ns, "test-deployment")
	ic := testutil.NewMockInstrumentationConfig(deployment)
	ic.Spec.PodManifestInjectionOptional = true // Distribution doesn't require rollout
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	fakeClient := s.newFakeClient(deployment)
	limiter := newRolloutConcurrencyLimiterExhausted() // Even with exhausted limiter

	// Act
	rolloutResult, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider, limiter)

	// Assert: Status is NotRequired (rate limiter not involved)
	assertTriggeredRolloutNoRequeue(t, rolloutResult, err)
	assert.Equal(t, string(odigosv1alpha1.WorkloadRolloutReasonNotRequired), ic.Status.Conditions[0].Reason)
}

func Test_RateLimiting_JobsAndCronjobs_NotAffected(t *testing.T) {
	// Arrange: Jobs don't use rate limiting (they trigger themselves)
	s := newTestSetup()
	setConfigConcurrentRolloutLimit(s.conf, 1) // Set limit to 1 in configuration
	job := testutil.NewMockTestJob(s.ns, "test-job")
	jobPw := k8sconsts.PodWorkload{Name: job.Name, Namespace: job.Namespace, Kind: k8sconsts.WorkloadKindJob}
	jobIc := testutil.NewMockInstrumentationConfig(job)

	fakeClient := s.newFakeClient(job)
	limiter := newRolloutConcurrencyLimiterExhausted() // Even with exhausted limiter

	// Act
	rolloutResult, err := rollout.Do(s.ctx, fakeClient, jobIc, jobPw, s.conf, s.distroProvider, limiter)

	// Assert: Job gets WaitingForRestart status, not affected by rate limiter
	assertTriggeredRolloutNoRequeue(t, rolloutResult, err)
	assert.Equal(t, string(odigosv1alpha1.WorkloadRolloutReasonWaitingForRestart), jobIc.Status.Conditions[0].Reason)
}

func Test_RateLimiting_StaticPods_NotAffected(t *testing.T) {
	// Arrange: Static pods don't use rate limiting (they don't support restart)
	s := newTestSetup()
	setConfigConcurrentRolloutLimit(s.conf, 1) // Set limit to 1 in configuration
	staticPod := testutil.NewMockTestStaticPod(s.ns, "test-staticpod")
	staticPodPw := k8sconsts.PodWorkload{Name: staticPod.Name, Namespace: staticPod.Namespace, Kind: k8sconsts.WorkloadKindStaticPod}
	ic := testutil.NewMockInstrumentationConfig(staticPod)

	fakeClient := s.newFakeClient(staticPod, ic)
	limiter := newRolloutConcurrencyLimiterExhausted() // Even with exhausted limiter

	// Act
	rolloutResult, err := rollout.Do(s.ctx, fakeClient, ic, staticPodPw, s.conf, s.distroProvider, limiter)

	// Assert: Static pod gets WorkloadNotSupporting status, not affected by rate limiter
	assert.Equal(t, true, rolloutResult.StatusChanged)
	assert.Equal(t, reconcile.Result{}, rolloutResult.Result)
	assert.NoError(t, err)
	assert.Equal(t, string(odigosv1alpha1.WorkloadRolloutReasonWorkloadNotSupporting), ic.Status.Conditions[0].Reason)
}

func Test_RateLimiting_PreviousRolloutOngoing_RateLimiterNotConsumed(t *testing.T) {
	// Arrange: If a previous rollout is ongoing, rate limiter should NOT be consumed
	s := newTestSetup()
	setConfigConcurrentRolloutLimit(s.conf, 1) // Set limit to 1 in configuration
	deployment := newMockDeploymentMidRollout(s.ns, "test-deployment")
	ic := mockICRolloutRequiredDistro(testutil.NewMockInstrumentationConfig(deployment))
	ic.Status.WorkloadRolloutHash = "old-hash" // Different hash triggers new rollout check
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	fakeClient := s.newFakeClient(deployment)
	limiter := newRolloutConcurrencyLimiterActive() // Fresh limiter with no slots taken

	// Act
	rolloutResult, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider, limiter)

	// Assert: Previous rollout ongoing status, rate limiter NOT consumed
	assert.NoError(t, err)
	assert.True(t, rolloutResult.StatusChanged)
	assert.Equal(t, reconcile.Result{RequeueAfter: rollout.RequeueWaitingForWorkloadRollout}, rolloutResult.Result)
	assert.Equal(t, string(odigosv1alpha1.WorkloadRolloutReasonPreviousRolloutOngoing), ic.Status.Conditions[0].Reason)

	// Verify rate limiter still has capacity (slot wasn't acquired for this workload)
	assert.Equal(t, 0, limiter.InFlightCount(), "rate limiter should have no in-flight rollouts - slot wasn't acquired")
}
