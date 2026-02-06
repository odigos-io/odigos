package rollout_test

import (
	"testing"
	"time"

	argorolloutsv1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/instrumentor/controllers/agentenabled/rollout"
	"github.com/odigos-io/odigos/instrumentor/internal/testutil"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func Test_NoRollout_PodInMidRollout_WithRollbackDisabled(t *testing.T) {
	// Arrange: Mid-rollout with crashlooping pod, but RollbackDisabled=true prevents automatic rollback
	s := newTestSetup()
	rollbackDisabled := true
	s.conf.RollbackDisabled = &rollbackDisabled
	deployment := newMockDeploymentMidRollout(s.ns, "test-deployment")
	ic := mockICMidRollout(testutil.NewMockInstrumentationConfig(deployment))
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}
	// Add params for checking backoff
	now := metav1.Now()
	ic.Status.InstrumentationTime = &now
	ic.Spec.AgentInjectionEnabled = true

	// Crashlooping pod that would normally trigger rollback
	podStartTime := metav1.NewTime(time.Now().Add(-6 * time.Minute))
	crashingPod := newMockCrashingPod(s.ns, deployment.Name, ic.Spec.AgentsMetaHash, podStartTime)

	fakeClient := s.newFakeClient(deployment, crashingPod)
	rateLimiter := newRolloutConcurrencyLimiterNoLimit()

	// Act
	rolloutResult, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider, rateLimiter)

	// Assert: No status change - rollbackDisabled config prevents rollback despite crashlooping pod
	assertNoStatusChange(t, rolloutResult, err)
}

func Test_NoRollout_PodInMidRollout_FailedToGetBackoffInfo(t *testing.T) {
	// Arrange: Mid-rollout deployment with nil selector - cannot query pods to check for backoff
	s := newTestSetup()
	deployment := newMockDeploymentMidRollout(s.ns, "test-deployment")
	deployment.Spec.Selector = nil // nil selector causes instrumentedPodsSelector to fail
	ic := mockICMidRollout(testutil.NewMockInstrumentationConfig(deployment))
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	fakeClient := s.newFakeClient(deployment)
	rateLimiter := newRolloutConcurrencyLimiterNoLimit()

	// Act
	rolloutResult, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider, rateLimiter)

	// Assert: Error returned - cannot determine backoff status due to nil selector
	assertErrorNoStatusChange(t, rolloutResult, err)
}

func TestNoRolloutMidRolloutInstrumentationTimeIsNil(t *testing.T) {
	// Arrange: Mid-rollout deployment with IC that has nil InstrumentationTime (can't calculate backoff)
	s := newTestSetup()
	deployment := newMockDeploymentMidRollout(s.ns, "test-deployment")
	ic := mockICMidRollout(testutil.NewMockInstrumentationConfig(deployment))
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	fakeClient := s.newFakeClient(deployment)
	rateLimiter := newRolloutConcurrencyLimiterNoLimit()

	// Act
	rolloutResult, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider, rateLimiter)

	// Assert: No status change, requeue after 10s - waiting for rollout without backoff detection
	assert.NoError(t, err)
	assert.False(t, rolloutResult.StatusChanged, "expected no status change")
	// requeueWaitingForWorkloadRollout is 10 seconds but not publicly exported
	assert.Equal(t, reconcile.Result{RequeueAfter: rollout.RequeueWaitingForWorkloadRollout}, rolloutResult.Result, "expected requeue after rollout")
}

func Test_NoRollout_PodInMidRollout_BackoffDurationLessThanGraceTime(t *testing.T) {
	// Arrange: Crashlooping pod started 5s ago - within rollback grace time (default 5m), not yet triggering rollback
	s := newTestSetup()
	deployment := newMockDeploymentMidRollout(s.ns, "test-deployment")
	ic := mockICMidRollout(testutil.NewMockInstrumentationConfig(deployment))
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}
	// InstrumentationTime is set to a recent time (within rollbackStabilityWindow)
	now := metav1.Now()
	ic.Status.InstrumentationTime = &now
	ic.Spec.AgentInjectionEnabled = true

	// Crashlooping pod (started 5s ago, within grace time)
	podStartTime := metav1.NewTime(time.Now().Add(-5 * time.Second))
	crashingPod := newMockCrashingPod(s.ns, deployment.Name, ic.Spec.AgentsMetaHash, podStartTime)

	fakeClient := s.newFakeClient(deployment, crashingPod)
	rateLimiter := newRolloutConcurrencyLimiterNoLimit()

	// Act
	rolloutResult, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider, rateLimiter)

	// Assert: No rollback yet - still in grace period, requeue to check again after remaining grace time
	assert.NoError(t, err)
	assert.False(t, rolloutResult.StatusChanged, "expected no status change during grace period")
	assert.True(t, rolloutResult.Result.RequeueAfter > 0, "expected requeue with remaining grace time")
}

func Test_NoRollout_PodInMidRollout_ClientUpdateError(t *testing.T) {
	// Arrange: Crashlooping pod past grace time, but client.Update() fails when trying to update IC
	s := newTestSetup()
	deployment := newMockDeploymentMidRollout(s.ns, "test-deployment")
	ic := mockICMidRollout(testutil.NewMockInstrumentationConfig(deployment))
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}
	// InstrumentationTime is set to a recent time (within rollbackStabilityWindow)
	now := metav1.Now()
	ic.Status.InstrumentationTime = &now
	ic.Spec.AgentInjectionEnabled = true

	// Crashlooping pod with more than grace time
	podStartTime := metav1.NewTime(time.Now().Add(-6 * time.Minute))
	crashingPod := newMockCrashingPod(s.ns, deployment.Name, ic.Spec.AgentsMetaHash, podStartTime)

	// Use interceptor to make c.Update(ic) fail
	fakeClient := s.newFakeClientWithICUpdateError(deployment, crashingPod)
	rateLimiter := newRolloutConcurrencyLimiterNoLimit()

	// Act
	rolloutResult, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider, rateLimiter)

	// Assert: Error returned - rollback should happen but IC update failed
	assertErrorNoStatusChange(t, rolloutResult, err)
}

func Test_TriggeredRollout_PodInMidRollout_RollbackRestartAnnotation(t *testing.T) {
	// Arrange: Crashlooping pod for 6min (past 5min grace time) - should trigger automatic rollback
	s := newTestSetup()
	deployment := newMockDeploymentMidRollout(s.ns, "test-deployment")
	ic := mockICMidRollout(testutil.NewMockInstrumentationConfig(deployment))
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}
	// InstrumentationTime is set to a recent time (within rollbackStabilityWindow)
	now := metav1.Now()
	ic.Status.InstrumentationTime = &now
	ic.Spec.AgentInjectionEnabled = true

	// Crashlooping pod with more than grace time
	podStartTime := metav1.NewTime(time.Now().Add(-6 * time.Minute))
	crashingPod := newMockCrashingPod(s.ns, deployment.Name, ic.Spec.AgentsMetaHash, podStartTime)

	fakeClient := s.newFakeClient(deployment, crashingPod, ic)
	rateLimiter := newRolloutConcurrencyLimiterNoLimit()

	// Act
	rolloutResult, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider, rateLimiter)

	// Assert: Rollback triggered - IC updated, deployment gets restartedAt annotation to force restart
	assertTriggeredRollback(t, rolloutResult, err, ic)

	// Assert deployment has restart annotation (kubectl.kubernetes.io/restartedAt)
	var updatedDeployment appsv1.Deployment
	err = fakeClient.Get(s.ctx, client.ObjectKey{Name: deployment.Name, Namespace: deployment.Namespace}, &updatedDeployment)
	assert.NoError(t, err)
	assert.Contains(t, updatedDeployment.Spec.Template.Annotations, "kubectl.kubernetes.io/restartedAt")
}

func Test_TriggeredRollout_PodInMidRollout_RollbackRestartAtArgoRollout(t *testing.T) {
	// Arrange: Argo Rollout with crashlooping pod for 6min - should trigger rollback via spec.restartAt
	s := newTestSetup()
	argoRollout := newMockArgoRolloutMidRollout(s.ns, "test-rollout")
	ic := mockICMidRollout(testutil.NewMockInstrumentationConfig(argoRollout))
	pw := k8sconsts.PodWorkload{Name: argoRollout.Name, Namespace: argoRollout.Namespace, Kind: k8sconsts.WorkloadKindArgoRollout}
	now := metav1.Now()
	ic.Status.InstrumentationTime = &now
	ic.Spec.AgentInjectionEnabled = true

	// Crashlooping pod with backoff duration >= rollbackGraceTime (5m) to trigger rollback
	podStartTime := metav1.NewTime(time.Now().Add(-6 * time.Minute))
	crashingPod := newMockCrashingPod(s.ns, argoRollout.Name, ic.Spec.AgentsMetaHash, podStartTime)

	fakeClient := s.newFakeClient(argoRollout, crashingPod, ic)
	rateLimiter := newRolloutConcurrencyLimiterNoLimit()

	// Act
	rolloutResult, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider, rateLimiter)

	// Assert: Rollback triggered - Argo Rollout uses spec.restartAt instead of annotation
	assertTriggeredRollback(t, rolloutResult, err, ic)

	// Assert Argo Rollout has spec.restartAt set
	var updatedRollout argorolloutsv1alpha1.Rollout
	err = fakeClient.Get(s.ctx, client.ObjectKey{Name: argoRollout.Name, Namespace: argoRollout.Namespace}, &updatedRollout)
	assert.NoError(t, err)
	assert.NotNil(t, updatedRollout.Spec.RestartAt, "expected spec.restartAt to be set for Argo Rollout")
}

func Test_Rollback_WebhookInstrumentedPodCrashloops_WhileWorkloadWaitingInQueue(t *testing.T) {
	// Scenario: Webhook-instrumented pod crashlooping before workload rollout: AgentsMetaHash is populated but WorkloadRolloutHash is empty.
	//
	// 1. Workload is waiting in rate limiter queue to be rolled out
	// 2. ic.Status.WorkloadRolloutHash is EMPTY (no rollout happened yet)
	// 3. A new pod is added to the workload
	// 4. The webhook (pods_webhook.go) instruments this new pod immediately
	// 5. The instrumented pod starts crashlooping
	// 6. Reconciliation detects crashloop via pods triggered by webhook handling
	// 7. Rollback is triggered, bypassing the rate limiter
	s := newTestSetup()

	// Deployment that hasn't been rolled out yet (waiting in queue)
	// Using a stable deployment (not mid-rollout) because the rollout hasn't started
	deployment := testutil.NewMockTestDeployment(s.ns, "waiting-deployment")

	// IC with rollout required, but WorkloadRolloutHash is EMPTY (no rollout happened yet)
	ic := mockICRolloutRequiredDistro(testutil.NewMockInstrumentationConfig(deployment))
	// Key: WorkloadRolloutHash is empty/unset - this is the bug trigger
	ic.Status.WorkloadRolloutHash = "" // Explicitly empty - no rollout has happened
	now := metav1.Now()
	// For this scenario, use AgentsMetaHashChangedTime (set when webhook instruments, before rollout)
	// InstrumentationTime is only set AFTER rollout happens
	ic.Spec.AgentsMetaHashChangedTime = &now
	ic.Spec.AgentInjectionEnabled = true

	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	// Crashlooping pod that was instrumented by the webhook (not via rollout)
	// This pod has the AgentsMetaHash label because the webhook added instrumentation
	podStartTime := metav1.NewTime(time.Now().Add(-6 * time.Minute))
	crashingPod := newMockCrashingPod(s.ns, deployment.Name, ic.Spec.AgentsMetaHash, podStartTime)

	fakeClient := s.newFakeClient(deployment, crashingPod, ic)

	// Rate limiter is exhausted - workload is waiting in queue
	rateLimiter := newRolloutConcurrencyLimiterExhausted()

	// Act: Reconcile the workload
	rolloutResult, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider, rateLimiter)

	// Assert: Rollback SHOULD be triggered for the crashlooping pod
	assert.NoError(t, err)
	assert.True(t, rolloutResult.StatusChanged, "expected status change after rollback")
	assert.True(t, ic.Status.RollbackOccurred, "expected RollbackOccurred to be true - crashlooping pod should trigger rollback")
	assert.False(t, ic.Spec.AgentInjectionEnabled, "agent injection should be disabled after rollback")

	// Verify the deployment was restarted via rollback (uninstrument the crashlooping pod)
	var updatedDeployment appsv1.Deployment
	err = fakeClient.Get(s.ctx, client.ObjectKey{Name: deployment.Name, Namespace: deployment.Namespace}, &updatedDeployment)
	assert.NoError(t, err)
	assert.Contains(t, updatedDeployment.Spec.Template.Annotations, "kubectl.kubernetes.io/restartedAt",
		"crashlooping webhook-instrumented pod should trigger rollback restart")
}

func Test_Rollback_BypassesRateLimiter_WhenOtherDeploymentsWaitingInQueue(t *testing.T) {
	// Scenario: Multiple deployments are being instrumented with rate limiting enabled (limit=1).
	// One deployment is mid-rollout and enters CrashLoopBackOff.
	// Other deployments are waiting in queue (rate limited).
	// The crashlooping deployment should be able to rollback (auto-heal) immediately,
	// bypassing the rate limiter to prevent prolonged downtime.
	s := newTestSetup()

	// Create 3 deployments:
	// - crashloopDeployment: mid-rollout, will crashloop and need rollback
	// - waitingDeployment2, waitingDeployment3: waiting to be instrumented (rate limited)
	crashloopDeployment := newMockDeploymentMidRollout(s.ns, "crashloop-deployment")
	waitingDeployment2 := testutil.NewMockTestDeployment(s.ns, "waiting-deployment-2")
	waitingDeployment3 := testutil.NewMockTestDeployment(s.ns, "waiting-deployment-3")

	// IC for crashlooping deployment: mid-rollout state (hash matches, instrumentation triggered)
	crashloopIC := mockICMidRollout(testutil.NewMockInstrumentationConfig(crashloopDeployment))
	now := metav1.Now()
	crashloopIC.Status.InstrumentationTime = &now
	crashloopIC.Spec.AgentInjectionEnabled = true

	// ICs for waiting deployments: need rollout but haven't started yet
	waitingIC2 := mockICRolloutRequiredDistro(testutil.NewMockInstrumentationConfig(waitingDeployment2))
	waitingIC3 := mockICRolloutRequiredDistro(testutil.NewMockInstrumentationConfig(waitingDeployment3))

	crashloopPW := k8sconsts.PodWorkload{Name: crashloopDeployment.Name, Namespace: crashloopDeployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}
	waitingPW2 := k8sconsts.PodWorkload{Name: waitingDeployment2.Name, Namespace: waitingDeployment2.Namespace, Kind: k8sconsts.WorkloadKindDeployment}
	waitingPW3 := k8sconsts.PodWorkload{Name: waitingDeployment3.Name, Namespace: waitingDeployment3.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	// Create crashlooping pod (6 minutes - past grace time)
	podStartTime := metav1.NewTime(time.Now().Add(-6 * time.Minute))
	crashingPod := newMockCrashingPod(s.ns, crashloopDeployment.Name, crashloopIC.Spec.AgentsMetaHash, podStartTime)

	fakeClient := s.newFakeClient(crashloopDeployment, waitingDeployment2, waitingDeployment3, crashloopIC, crashingPod)

	// Rate limiter with limit of 1 - only one rollout at a time
	// Pre-acquire a slot for the crashlooping deployment (simulating it already being in-flight)
	setConfigConcurrentRolloutLimit(s.conf, 1)
	rateLimiter := newRolloutConcurrencyLimiterWithLimit(1)
	crashloopWorkloadKey := rollout.WorkloadKey(crashloopPW)
	rateLimiter.TryAcquire(crashloopWorkloadKey, 1)
	assert.Equal(t, 1, rateLimiter.InFlightCount(), "rate limiter should have 1 in-flight rollout (crashloop deployment)")

	// Step 1: Waiting deployments try to rollout but are rate limited
	rolloutResult2, err := rollout.Do(s.ctx, fakeClient, waitingIC2, waitingPW2, s.conf, s.distroProvider, rateLimiter)
	assert.NoError(t, err)
	assert.True(t, rolloutResult2.StatusChanged)
	assert.Equal(t, "WaitingInQueue", waitingIC2.Status.Conditions[0].Reason,
		"waiting deployment 2 should be in queue")

	rolloutResult3, err := rollout.Do(s.ctx, fakeClient, waitingIC3, waitingPW3, s.conf, s.distroProvider, rateLimiter)
	assert.NoError(t, err)
	assert.True(t, rolloutResult3.StatusChanged)
	assert.Equal(t, "WaitingInQueue", waitingIC3.Status.Conditions[0].Reason,
		"waiting deployment 3 should be in queue")

	// Verify rate limiter is exhausted (slot held by crashloop deployment)
	assert.Equal(t, 1, rateLimiter.InFlightCount(), "rate limiter should still have 1 in-flight rollout")

	// Step 2: Crashlooping deployment reconciles - should detect crashloop and trigger rollback
	// Key assertion: rollback should succeed even though rate limiter slot is "held"
	rolloutResult, err := rollout.Do(s.ctx, fakeClient, crashloopIC, crashloopPW, s.conf, s.distroProvider, rateLimiter)

	// Assert: Rollback triggered successfully
	assert.NoError(t, err)
	assert.True(t, rolloutResult.StatusChanged, "expected status change after rollback")
	assert.True(t, crashloopIC.Status.RollbackOccurred, "expected RollbackOccurred to be true")
	assert.False(t, crashloopIC.Spec.AgentInjectionEnabled, "agent injection should be disabled after rollback")

	// Verify the deployment was restarted via rollback
	var updatedDeployment appsv1.Deployment
	err = fakeClient.Get(s.ctx, client.ObjectKey{Name: crashloopDeployment.Name, Namespace: crashloopDeployment.Namespace}, &updatedDeployment)
	assert.NoError(t, err)
	assert.Contains(t, updatedDeployment.Spec.Template.Annotations, "kubectl.kubernetes.io/restartedAt",
		"crashlooping deployment should be restarted via rollback")

	// Verify the rate limiter slot was released after rollback
	// This is critical: auto-heal releases the slot so other workloads can proceed
	assert.Equal(t, 0, rateLimiter.InFlightCount(),
		"rate limiter slot should be released after rollback - auto-heal releases its slot")

	// Step 3: Verify waiting deployments can now proceed (rate limiter has capacity)
	// Re-reconcile waiting deployment 2 - should now acquire the slot and proceed
	rolloutResult2After, err := rollout.Do(s.ctx, fakeClient, waitingIC2, waitingPW2, s.conf, s.distroProvider, rateLimiter)
	assert.NoError(t, err)
	assertTriggeredRolloutWithRequeue(t, rolloutResult2After, err)
	assert.Equal(t, "RolloutTriggeredSuccessfully", waitingIC2.Status.Conditions[0].Reason,
		"waiting deployment 2 should now be able to rollout after crashloop deployment released its slot")
}

func Test_Rollback_WebhookInstrumentedPodCrashloops_WhileWorkloadRolloutNotStarted(t *testing.T) {
	// Scenario: Webhook-instrumented pod crashlooping before workload rollout.
	//
	// 1. Workload is waiting in rate limiter queue to be rolled out
	// 2. ic.Status.WorkloadRolloutHash is EMPTY (no rollout happened yet)
	// 3. A new pod is added to the workload
	// 4. The webhook (pods_webhook.go) instruments this new pod immediately
	// 5. The instrumented pod starts crashlooping
	// 6. Reconciliation detects crashloop via pods triggered by webhook handling
	// 7. Rollback is triggered, bypassing the rate limiter
	s := newTestSetup()

	// Deployment that hasn't been rolled out yet (waiting in queue)
	// Using a stable deployment (not mid-rollout) because the rollout hasn't started
	deployment := testutil.NewMockTestDeployment(s.ns, "waiting-deployment")

	// IC with rollout required, but WorkloadRolloutHash is EMPTY (no rollout happened yet)
	ic := mockICRolloutRequiredDistro(testutil.NewMockInstrumentationConfig(deployment))
	// Key: WorkloadRolloutHash is empty/unset - this is the bug trigger
	ic.Status.WorkloadRolloutHash = "" // Explicitly empty - no rollout has happened
	now := metav1.Now()
	// For scenario, use AgentsMetaHashChangedTime (set when webhook instruments, before rollout)
	// InstrumentationTime is only set AFTER rollout happens
	ic.Spec.AgentsMetaHashChangedTime = &now
	ic.Spec.AgentInjectionEnabled = true

	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	// Crashlooping pod that was instrumented by the webhook (not via rollout)
	// This pod has the AgentsMetaHash label because the webhook added instrumentation
	podStartTime := metav1.NewTime(time.Now().Add(-6 * time.Minute))
	crashingPod := newMockCrashingPod(s.ns, deployment.Name, ic.Spec.AgentsMetaHash, podStartTime)

	fakeClient := s.newFakeClient(deployment, crashingPod, ic)

	// Rate limiter is exhausted - workload is waiting in queue
	rateLimiter := newRolloutConcurrencyLimiterExhausted()

	// Act: Reconcile the workload
	rolloutResult, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider, rateLimiter)

	// Assert: Rollback SHOULD be triggered for the crashlooping pod
	assert.NoError(t, err)
	assert.True(t, rolloutResult.StatusChanged, "expected status change after rollback")
	assert.True(t, ic.Status.RollbackOccurred, "expected RollbackOccurred to be true - crashlooping pod should trigger rollback")
	assert.False(t, ic.Spec.AgentInjectionEnabled, "agent injection should be disabled after rollback")

	// Verify the deployment was restarted via rollback (uninstrument the crashlooping pod)
	var updatedDeployment appsv1.Deployment
	err = fakeClient.Get(s.ctx, client.ObjectKey{Name: deployment.Name, Namespace: deployment.Namespace}, &updatedDeployment)
	assert.NoError(t, err)
	assert.Contains(t, updatedDeployment.Spec.Template.Annotations, "kubectl.kubernetes.io/restartedAt",
		"crashlooping webhook-instrumented pod should trigger rollback restart")
}
