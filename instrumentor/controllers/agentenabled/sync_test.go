package agentenabled

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/instrumentor/internal/testutil"
	agentInjectionEnabled "github.com/odigos-io/odigos/status/instrumentationconfig/generated"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	assert.Equal(t, odigosv1alpha1.AgentEnabledReason(agentInjectionEnabled.AgentEnabledReasonCrashLoopBackOff), condition.Reason)
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

// ****************
// isReadyForInstrumentation() tests
// ****************

func newReadyNodeCollectorsGroup() *odigosv1alpha1.CollectorsGroup {
	return &odigosv1alpha1.CollectorsGroup{
		Status: odigosv1alpha1.CollectorsGroupStatus{
			Ready:           true,
			ReceiverSignals: []common.ObservabilitySignal{common.TracesObservabilitySignal},
		},
	}
}

// Injecting an agent before the pipeline can accept its telemetry, or before runtime inspection
// completed, produces pods that either crash or export nowhere. Every negative case must therefore
// report a transient reason, so the workload is instrumented once the prerequisite is met.
func TestIsReadyForInstrumentation(t *testing.T) {
	runtimeInfo := &odigosv1alpha1.RuntimeDetailsByContainer{
		ContainerName: "app",
		Language:      common.JavascriptProgrammingLanguage,
	}

	for _, tc := range []struct {
		name            string
		collectorsGroup *odigosv1alpha1.CollectorsGroup
		ic              *odigosv1alpha1.InstrumentationConfig
		expectedReady   bool
		expectedReason  agentInjectionEnabled.AgentEnabledReason
	}{
		{
			name:            "no node collectors group",
			collectorsGroup: nil,
			ic:              &odigosv1alpha1.InstrumentationConfig{},
			expectedReady:   false,
			expectedReason:  agentInjectionEnabled.AgentEnabledReasonWaitingForNodeCollector,
		},
		{
			name:            "node collectors group not ready",
			collectorsGroup: &odigosv1alpha1.CollectorsGroup{},
			ic:              &odigosv1alpha1.InstrumentationConfig{},
			expectedReady:   false,
			expectedReason:  agentInjectionEnabled.AgentEnabledReasonWaitingForNodeCollector,
		},
		{
			// a ready collector with no receiver means every signal is disabled.
			name: "node collector ready with no receiver signals",
			collectorsGroup: &odigosv1alpha1.CollectorsGroup{
				Status: odigosv1alpha1.CollectorsGroupStatus{Ready: true},
			},
			ic:             &odigosv1alpha1.InstrumentationConfig{},
			expectedReady:  false,
			expectedReason: agentInjectionEnabled.AgentEnabledReasonNoCollectedSignals,
		},
		{
			name:            "runtime inspection did not report yet",
			collectorsGroup: newReadyNodeCollectorsGroup(),
			ic:              &odigosv1alpha1.InstrumentationConfig{},
			expectedReady:   false,
			expectedReason:  agentInjectionEnabled.AgentEnabledReasonWaitingForRuntimeInspection,
		},
		{
			// no pod to inspect is not a transient wait for odiglet, and the UI shows it
			// differently, so it must not be reported as waiting for inspection.
			name:            "runtime inspection has no running pods to inspect",
			collectorsGroup: newReadyNodeCollectorsGroup(),
			ic: &odigosv1alpha1.InstrumentationConfig{
				Status: odigosv1alpha1.InstrumentationConfigStatus{
					Conditions: []metav1.Condition{{
						Type:   odigosv1alpha1.RuntimeDetectionStatusConditionType,
						Status: metav1.ConditionFalse,
						Reason: string(odigosv1alpha1.RuntimeDetectionReasonNoRunningPods),
					}},
				},
			},
			expectedReady:  false,
			expectedReason: agentInjectionEnabled.AgentEnabledReasonRuntimeDetailsUnavailable,
		},
		{
			name:            "runtime details are available from automatic detection",
			collectorsGroup: newReadyNodeCollectorsGroup(),
			ic: &odigosv1alpha1.InstrumentationConfig{
				Status: odigosv1alpha1.InstrumentationConfigStatus{
					RuntimeDetailsByContainer: []odigosv1alpha1.RuntimeDetailsByContainer{*runtimeInfo},
				},
			},
			expectedReady:  true,
			expectedReason: agentInjectionEnabled.AgentEnabledReasonEnabledSuccessfully,
		},
		{
			// a manual runtime info override replaces detection, so odigos must not keep waiting
			// for odiglet to report - it never will for a workload with no running pods.
			name:            "runtime info override without automatic detection",
			collectorsGroup: newReadyNodeCollectorsGroup(),
			ic: &odigosv1alpha1.InstrumentationConfig{
				Spec: odigosv1alpha1.InstrumentationConfigSpec{
					ContainersOverrides: []odigosv1alpha1.ContainerOverride{{
						ContainerName: "app",
						RuntimeInfo:   runtimeInfo,
					}},
				},
			},
			expectedReady:  true,
			expectedReason: agentInjectionEnabled.AgentEnabledReasonEnabledSuccessfully,
		},
		{
			name:            "container override without runtime info does not replace detection",
			collectorsGroup: newReadyNodeCollectorsGroup(),
			ic: &odigosv1alpha1.InstrumentationConfig{
				Spec: odigosv1alpha1.InstrumentationConfigSpec{
					ContainersOverrides: []odigosv1alpha1.ContainerOverride{{ContainerName: "app"}},
				},
			},
			expectedReady:  false,
			expectedReason: agentInjectionEnabled.AgentEnabledReasonWaitingForRuntimeInspection,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ready, reason, message := isReadyForInstrumentation(tc.collectorsGroup, tc.ic)

			assert.Equal(t, tc.expectedReady, ready)
			assert.Equal(t, odigosv1alpha1.AgentEnabledReason(tc.expectedReason), reason)
			if tc.expectedReady {
				assert.Empty(t, message)
			} else {
				assert.NotEmpty(t, message, "the user needs a message explaining what odigos waits for")
			}
		})
	}
}

// ****************
// containerConfigToStatusCondition() tests
// ****************

// The k8s condition status is looked up per reason, and it is what tells apart a workload that will
// never be instrumented (False) from one that is still settling (Unknown). A lookup that falls back
// to the default reports transient states as permanent failures.
func TestContainerConfigToStatusCondition(t *testing.T) {
	for _, tc := range []struct {
		name            string
		containerConfig odigosv1alpha1.ContainerAgentConfig
		expectedStatus  metav1.ConditionStatus
	}{
		{
			name: "enabled agent",
			containerConfig: odigosv1alpha1.ContainerAgentConfig{
				ContainerName: "app",
				AgentEnabled:  true,
			},
			expectedStatus: metav1.ConditionTrue,
		},
		{
			name: "transient reason keeps an unknown status",
			containerConfig: odigosv1alpha1.ContainerAgentConfig{
				ContainerName:      "app",
				AgentEnabledReason: odigosv1alpha1.AgentEnabledReason(agentInjectionEnabled.AgentEnabledReasonRuntimeDetailsUnavailable),
			},
			expectedStatus: metav1.ConditionUnknown,
		},
		{
			name: "permanent reason reports a false status",
			containerConfig: odigosv1alpha1.ContainerAgentConfig{
				ContainerName:      "app",
				AgentEnabledReason: odigosv1alpha1.AgentEnabledReason(agentInjectionEnabled.AgentEnabledReasonUnsupportedProgrammingLanguage),
			},
			expectedStatus: metav1.ConditionFalse,
		},
		{
			name:            "unrecognized reason reports a false status",
			containerConfig: odigosv1alpha1.ContainerAgentConfig{ContainerName: "app"},
			expectedStatus:  metav1.ConditionFalse,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			condition := containerConfigToStatusCondition(tc.containerConfig)

			require.NotNil(t, condition)
			assert.Equal(t, tc.expectedStatus, condition.Status)
			if tc.containerConfig.AgentEnabled {
				assert.Equal(t, odigosv1alpha1.AgentEnabledReason(agentInjectionEnabled.AgentEnabledReasonEnabledSuccessfully), condition.Reason)
			} else {
				assert.Equal(t, tc.containerConfig.AgentEnabledReason, condition.Reason)
			}
		})
	}
}
