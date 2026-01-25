package rollout_test

// ****************
// Do() tests
// ****************

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/instrumentor/controllers/agentenabled/rollout"
	"github.com/odigos-io/odigos/instrumentor/internal/testutil"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_NoRollout_ObjectKeyIsMissing(t *testing.T) {
	// Arrange: Deployment exists in IC but NOT in the cluster (client has no deployment object)
	s := newTestSetup()
	deployment := testutil.NewMockTestDeployment(s.ns, "test-deployment")
	ic := testutil.NewMockInstrumentationConfig(deployment)
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}
	fakeClient := s.newFakeClientWithStatus(nil, ic)
	rateLimiter := newRateLimiterNoLimit()

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider, rateLimiter)

	// Assert: No status change - workload doesn't exist in cluster (nothing to restart)
	assertNoStatusChange(t, statusChanged, result, err)
}

func Test_NoRolloutWithError_ICNil_WorkloadHasOdigosAgents(t *testing.T) {
	// Arrange: Deployment with nil selector (invalid) and no IC - checking for instrumented pods will fail
	s := newTestSetup()
	deployment := testutil.NewMockTestDeployment(s.ns, "test-deployment")
	deployment.Spec.Selector = nil // nil selector causes instrumentedPodsSelector to fail
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	fakeClient := s.newFakeClient(deployment)
	var ic *odigosv1alpha1.InstrumentationConfig
	rateLimiter := newRateLimiterNoLimit()

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider, rateLimiter)

	// Assert: Error returned - cannot determine if pods have odigos agents due to nil selector
	assertErrorNoStatusChange(t, statusChanged, result, err)
}

func Test_NoRollout_ICNil_NoOdigosAgents(t *testing.T) {
	// Arrange: Deployment with pod that has NO odigos agent label, and IC is nil (uninstrumented workload)
	s := newTestSetup()
	deployment := testutil.NewMockTestDeployment(s.ns, "test-deployment")
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}
	// Create a pod WITHOUT the odigos agent label
	uninstrumentedPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: s.ns.Name,
			Labels: map[string]string{
				"app.kubernetes.io/name": deployment.Name,
			},
		},
	}

	fakeClient := s.newFakeClient(deployment, uninstrumentedPod)
	var ic *odigosv1alpha1.InstrumentationConfig
	rateLimiter := newRateLimiterNoLimit()

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider, rateLimiter)

	// Assert: No status change and no restart - pods don't have odigos agents, nothing to uninstrument
	assertNoStatusChange(t, statusChanged, result, err)
	assertWorkloadNotRestarted(t, s.ctx, fakeClient, pw)
}

func Test_TriggeredRollout_WorkloadWithInstrumentedPods(t *testing.T) {
	// Arrange: Pod has odigos agent label but IC is nil - need to uninstrument by restarting
	s := newTestSetup()
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
	rateLimiter := newRateLimiterNoLimit()

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider, rateLimiter)

	// Assert: Workload IS restarted (to remove odigos agents), but no IC status change (IC is nil)
	assertNoStatusChange(t, statusChanged, result, err)
	assertWorkloadRestarted(t, s.ctx, fakeClient, pw)
}

func Test_TriggeredRollout_DistributionRequiresRollout(t *testing.T) {
	// Arrange: IC with distribution that requires rollout (e.g. eBPF-based instrumentation)
	s := newTestSetup()
	deployment := testutil.NewMockTestDeployment(s.ns, "test-deployment")
	ic := mockICRolloutRequiredDistro(testutil.NewMockInstrumentationConfig(deployment))
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	fakeClient := s.newFakeClient(deployment)
	rateLimiter := newRateLimiterNoLimit()

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider, rateLimiter)

	// Assert: Rollout triggered successfully, requeue to monitor rollout progress
	assertTriggeredRolloutWithRequeue(t, statusChanged, result, err)
	assert.Equal(t, string(odigosv1alpha1.WorkloadRolloutReasonTriggeredSuccessfully), ic.Status.Conditions[0].Reason)
	assert.Equal(t, "workload rollout triggered successfully", ic.Status.Conditions[0].Message)
}

func Test_NoRollout_DistributionDoesntRequireRollout(t *testing.T) {
	// Arrange: IC with default distribution that doesn't require rollout (e.g., native instrumentation)
	s := newTestSetup()
	deployment := testutil.NewMockTestDeployment(s.ns, "test-deployment")
	ic := testutil.NewMockInstrumentationConfig(deployment)
	// Set PodManifestInjectionOptional to true to indicate distribution doesn't require rollout
	ic.Spec.PodManifestInjectionOptional = true
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	fakeClient := s.newFakeClient(deployment)
	rateLimiter := newRateLimiterNoLimit()

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider, rateLimiter)

	// Assert: Status updated to "NotRequired" - distribution doesn't need app restart, no requeue
	assertTriggeredRolloutNoRequeue(t, statusChanged, result, err)
	assert.NotEmpty(t, ic.Status.Conditions, "expected conditions to be set")
	assert.Equal(t, string(odigosv1alpha1.WorkloadRolloutReasonNotRequired), ic.Status.Conditions[0].Reason)
	assert.Equal(t, "The selected instrumentation distributions do not require application restart", ic.Status.Conditions[0].Message)
}
