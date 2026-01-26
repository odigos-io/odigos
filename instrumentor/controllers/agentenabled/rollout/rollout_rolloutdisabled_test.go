package rollout_test

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/instrumentor/controllers/agentenabled/rollout"
	"github.com/odigos-io/odigos/instrumentor/internal/testutil"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_NoRollout_ICNil_AutomaticRolloutDisabled(t *testing.T) {
	// Arrange: Pod has odigos agent label but IC is nil AND automatic rollout is disabled in config
	s := newTestSetup()
	automaticRolloutDisabled := true
	s.conf.Rollout = &common.RolloutConfiguration{
		AutomaticRolloutDisabled: &automaticRolloutDisabled,
	}
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
	rateLimiter := newRolloutConcurrencyLimiterNoLimit()

	// Act
	rolloutResult, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider, rateLimiter)

	// Assert: No status change and workload NOT restarted - automatic rollout disabled
	assertNoStatusChange(t, rolloutResult, err)
	assertWorkloadNotRestarted(t, s.ctx, fakeClient, pw)
}
func Test_NoRollout_InvalidRollbackGraceTime(t *testing.T) {
	// Arrange: Config has invalid RollbackGraceTime string that can't be parsed as duration
	s := newTestSetup()
	s.conf.RollbackGraceTime = "invalid"
	deployment := testutil.NewMockTestDeployment(s.ns, "test-deployment")
	ic := mockICRolloutRequiredDistro(testutil.NewMockInstrumentationConfig(deployment))
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	fakeClient := s.newFakeClient(deployment)
	rateLimiter := newRolloutConcurrencyLimiterNoLimit()

	// Act
	rolloutResult, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider, rateLimiter)

	// Assert: Error returned - invalid config prevents rollout
	assertErrorNoStatusChange(t, rolloutResult, err)
}

func Test_NoRollout_InvalidRollbackStabilityWindow(t *testing.T) {
	// Arrange: Config has invalid RollbackStabilityWindow string that can't be parsed as duration
	s := newTestSetup()
	s.conf.RollbackStabilityWindow = "invalid"
	deployment := testutil.NewMockTestDeployment(s.ns, "test-deployment")
	ic := mockICRolloutRequiredDistro(testutil.NewMockInstrumentationConfig(deployment))
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	fakeClient := s.newFakeClient(deployment)
	rateLimiter := newRolloutConcurrencyLimiterNoLimit()

	// Act
	rolloutResult, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider, rateLimiter)

	// Assert: Error returned - invalid config prevents rollout
	assertErrorNoStatusChange(t, rolloutResult, err)
}

func Test_TriggeredRollout_ConfigNil(t *testing.T) {
	// Arrange: IC requires rollout and s.conf.Rollout is nil (default config allows automatic rollout)
	s := newTestSetup()
	deployment := testutil.NewMockTestDeployment(s.ns, "test-deployment")
	ic := mockICRolloutRequiredDistro(testutil.NewMockInstrumentationConfig(deployment))
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	fakeClient := s.newFakeClient(deployment)
	rateLimiter := newRolloutConcurrencyLimiterNoLimit()

	// Act
	rolloutResult, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider, rateLimiter)

	// Assert: Rollout triggered - nil config defaults to enabled, requeue to monitor
	assertTriggeredRolloutWithRequeue(t, rolloutResult, err)
	assert.Equal(t, string(odigosv1alpha1.WorkloadRolloutReasonTriggeredSuccessfully), ic.Status.Conditions[0].Reason)
}

func Test_TriggeredRollout_AutomaticRolloutDisabledNil(t *testing.T) {
	// Arrange: RolloutConfiguration exists but AutomaticRolloutDisabled is nil (defaults to enabled)
	s := newTestSetup()
	s.conf.Rollout = &common.RolloutConfiguration{}
	deployment := testutil.NewMockTestDeployment(s.ns, "test-deployment")
	ic := mockICRolloutRequiredDistro(testutil.NewMockInstrumentationConfig(deployment))
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	fakeClient := s.newFakeClient(deployment)
	rateLimiter := newRolloutConcurrencyLimiterNoLimit()

	// Act
	rolloutResult, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider, rateLimiter)

	// Assert: Rollout triggered - nil pointer defaults to enabled, requeue to monitor
	assertTriggeredRolloutWithRequeue(t, rolloutResult, err)
	assert.Equal(t, string(odigosv1alpha1.WorkloadRolloutReasonTriggeredSuccessfully), ic.Status.Conditions[0].Reason)
}

func Test_TriggeredRollout_AutomaticRolloutDisabledFalse(t *testing.T) {
	// Arrange: AutomaticRolloutDisabled explicitly set to false (rollout enabled)
	s := newTestSetup()
	automaticRolloutDisabled := false
	s.conf.Rollout = &common.RolloutConfiguration{
		AutomaticRolloutDisabled: &automaticRolloutDisabled,
	}
	deployment := testutil.NewMockTestDeployment(s.ns, "test-deployment")
	ic := mockICRolloutRequiredDistro(testutil.NewMockInstrumentationConfig(deployment))
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	fakeClient := s.newFakeClient(deployment)
	rateLimiter := newRolloutConcurrencyLimiterNoLimit()

	// Act
	rolloutResult, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider, rateLimiter)

	// Assert: Rollout triggered successfully, requeue to monitor
	assertTriggeredRolloutWithRequeue(t, rolloutResult, err)
	assert.Equal(t, string(odigosv1alpha1.WorkloadRolloutReasonTriggeredSuccessfully), ic.Status.Conditions[0].Reason)
}

func Test_NoRollout_AutomaticRolloutDisabledTrue(t *testing.T) {
	// Arrange: AutomaticRolloutDisabled explicitly set to true (rollout disabled by user)
	s := newTestSetup()
	automaticRolloutDisabled := true
	s.conf.Rollout = &common.RolloutConfiguration{
		AutomaticRolloutDisabled: &automaticRolloutDisabled,
	}
	deployment := testutil.NewMockTestDeployment(s.ns, "test-deployment")
	ic := mockICRolloutRequiredDistro(testutil.NewMockInstrumentationConfig(deployment))
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	fakeClient := s.newFakeClient(deployment)
	rateLimiter := newRolloutConcurrencyLimiterNoLimit()

	// Act
	rolloutResult, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider, rateLimiter)

	// Assert: Status updated to "Disabled" - no rollout triggered, no requeue needed
	assertTriggeredRolloutNoRequeue(t, rolloutResult, err)
	assert.Equal(t, string(odigosv1alpha1.WorkloadRolloutReasonDisabled), ic.Status.Conditions[0].Reason)
	assert.Equal(t, "odigos automatic rollout is disabled", ic.Status.Conditions[0].Message)
}
