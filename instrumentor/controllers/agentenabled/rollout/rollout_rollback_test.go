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
