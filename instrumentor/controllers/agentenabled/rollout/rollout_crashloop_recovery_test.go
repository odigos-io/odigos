package rollout_test

import (
	"testing"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/instrumentor/controllers/agentenabled/rollout"
	"github.com/odigos-io/odigos/instrumentor/internal/testutil"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func Test_Do_Recovery_RecoveryNeeded_RequeuesAfterPersisting(t *testing.T) {
	// Arrange: Rollback occurred, recovery requested, annotation empty.
	s := newTestSetup()
	deployment := testutil.NewMockTestDeployment(s.ns, "test-deployment")
	ic := mockICMidRollout(testutil.NewMockInstrumentationConfig(deployment))
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	now := time.Now().Format(time.RFC3339)
	ic.Status.RollbackOccurred = true
	ic.Annotations = map[string]string{
		k8sconsts.RollbackRecoveryAtAnnotation: now,
	}

	fakeClient := s.newFakeClientWithStatus([]client.Object{deployment, ic}, ic)
	rateLimiter := newRolloutConcurrencyLimiterNoLimit()

	// Act
	rolloutResult, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider, rateLimiter)

	// Assert: Recovery persisted, requeued for re-reconcile.
	assert.NoError(t, err)
	assert.False(t, ic.Status.RollbackOccurred, "expected RollbackOccurred to be cleared after recovery")
	assert.Equal(t, now, ic.Annotations[k8sconsts.RollbackRecoveryProcessedAtAnnotation])
	assert.Equal(t, reconcile.Result{Requeue: true}, rolloutResult.Result, "expected requeue after recovery")
}

func Test_Do_Recovery_NoRecoveryNeeded_ProceedsNormally(t *testing.T) {
	// Arrange: Annotation already matches spec — Do() should skip recovery and continue.
	s := newTestSetup()
	deployment := testutil.NewMockTestDeployment(s.ns, "test-deployment")
	ic := mockICMidRollout(testutil.NewMockInstrumentationConfig(deployment))
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	now := time.Now().Format(time.RFC3339)
	ic.Status.RollbackOccurred = true
	ic.Annotations = map[string]string{
		k8sconsts.RollbackRecoveryAtAnnotation:          now,
		k8sconsts.RollbackRecoveryProcessedAtAnnotation: now,
	}

	fakeClient := s.newFakeClient(deployment, ic)
	rateLimiter := newRolloutConcurrencyLimiterNoLimit()

	// Act
	rolloutResult, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider, rateLimiter)

	// Assert: No recovery, proceeds with normal rollout logic.
	assert.NoError(t, err)
	assert.True(t, ic.Status.RollbackOccurred, "expected RollbackOccurred to remain true (no recovery)")
	assert.NotEqual(t, reconcile.Result{Requeue: true}, rolloutResult.Result)
}
