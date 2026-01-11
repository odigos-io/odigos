package rollout_test

// ****************
// Do() tests
// ****************

import (
	"testing"
	"time"

	argorolloutsv1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/instrumentor/controllers/agentenabled/rollout"
	"github.com/odigos-io/odigos/instrumentor/internal/testutil"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestNoRolloutObjectKeyIsMissing(t *testing.T) {
	// Arrange: Deployment exists in IC but NOT in the cluster (client has no deployment object)
	s := newTestSetup()
	deployment := testutil.NewMockTestDeployment(s.ns, "test-deployment")
	ic := testutil.NewMockInstrumentationConfig(deployment)
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}
	fakeClient := s.newFakeClientWithStatus(nil, ic)

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider)

	// Assert: No rollout triggered - workload doesn't exist in cluster
	assertNoRollout(t, statusChanged, result, err)
}

func TestNoRolloutJobOrCronjobNoIC(t *testing.T) {
	// Arrange: Job and CronJob workloads exist but have no InstrumentationConfig (IC is nil)
	s := newTestSetup()
	job := testutil.NewMockTestJob(s.ns, "test-job")
	cronjob := testutil.NewMockTestCronJob(s.ns, "test-cronjob")
	jobPw := k8sconsts.PodWorkload{Name: job.Name, Namespace: job.Namespace, Kind: k8sconsts.WorkloadKindJob}
	cronJobPw := k8sconsts.PodWorkload{Name: cronjob.Name, Namespace: cronjob.Namespace, Kind: k8sconsts.WorkloadKindCronJob}

	fakeClient := s.newFakeClient(job, cronjob)
	var ic *odigosv1alpha1.InstrumentationConfig

	// Act
	jobStatusChanged, jobResult, jobErr := rollout.Do(s.ctx, fakeClient, ic, jobPw, s.conf, s.distroProvider)
	cronJobStatusChanged, cronJobResult, cronJobErr := rollout.Do(s.ctx, fakeClient, ic, cronJobPw, s.conf, s.distroProvider)

	// Assert: No rollout - Jobs/CronJobs without IC don't need rollout
	assertNoRollout(t, jobStatusChanged, jobResult, jobErr)
	assertNoRollout(t, cronJobStatusChanged, cronJobResult, cronJobErr)
}

func TestNoRolloutJobOrCronjobWaitingForRestart(t *testing.T) {
	// Arrange: Job and CronJob with InstrumentationConfig - these can't be force-restarted like Deployments
	s := newTestSetup()
	job := testutil.NewMockTestJob(s.ns, "test-job")
	cronjob := testutil.NewMockTestCronJob(s.ns, "test-cronjob")
	jobPw := k8sconsts.PodWorkload{Name: job.Name, Namespace: job.Namespace, Kind: k8sconsts.WorkloadKindJob}
	cronJobPw := k8sconsts.PodWorkload{Name: cronjob.Name, Namespace: cronjob.Namespace, Kind: k8sconsts.WorkloadKindCronJob}
	jobIc := testutil.NewMockInstrumentationConfig(job)
	cronJobIc := testutil.NewMockInstrumentationConfig(cronjob)

	fakeClient := s.newFakeClient(job, cronjob)

	// Act
	jobStatusChanged, jobResult, jobErr := rollout.Do(s.ctx, fakeClient, jobIc, jobPw, s.conf, s.distroProvider)
	cronJobStatusChanged, cronJobResult, cronJobErr := rollout.Do(s.ctx, fakeClient, cronJobIc, cronJobPw, s.conf, s.distroProvider)

	// Assert: Status updated to "WaitingForRestart" - Jobs must trigger themselves, no requeue needed
	assertTriggeredRolloutNoRequeue(t, jobStatusChanged, jobResult, jobErr)
	assert.Equal(t, string(odigosv1alpha1.WorkloadRolloutReasonWaitingForRestart), jobIc.Status.Conditions[0].Reason)
	assert.Equal(t, "Waiting for job to trigger by itself", jobIc.Status.Conditions[0].Message)
	assertTriggeredRolloutNoRequeue(t, cronJobStatusChanged, cronJobResult, cronJobErr)
	assert.Equal(t, string(odigosv1alpha1.WorkloadRolloutReasonWaitingForRestart), cronJobIc.Status.Conditions[0].Reason)
	assert.Equal(t, "Waiting for job to trigger by itself", cronJobIc.Status.Conditions[0].Message)
}

func TestNoRolloutMidRolloutAlreadyComplete(t *testing.T) {
	// Arrange: IC shows mid-rollout state but deployment has already completed rolling out (no pending replicas)
	s := newTestSetup()
	deployment := testutil.NewMockTestDeployment(s.ns, "test-deployment")
	ic := mockICMidRollout(testutil.NewMockInstrumentationConfig(deployment))
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	fakeClient := s.newFakeClient(deployment)

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider)

	// Assert: No rollout needed - workload rollout already complete
	assertNoRollout(t, statusChanged, result, err)
}

func TestNoRolloutICNilWorkloadHasOdigosAgentsError(t *testing.T) {
	// Arrange: Deployment with nil selector (invalid) and no IC - checking for instrumented pods will fail
	s := newTestSetup()
	deployment := testutil.NewMockTestDeployment(s.ns, "test-deployment")
	deployment.Spec.Selector = nil // nil selector causes instrumentedPodsSelector to fail
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	fakeClient := s.newFakeClient(deployment)
	var ic *odigosv1alpha1.InstrumentationConfig

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider)

	// Assert: Error returned - cannot determine if pods have odigos agents due to nil selector
	assertErrorNoRollout(t, statusChanged, result, err)
}

func TestNoRolloutICNilNoOdigosAgents(t *testing.T) {
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

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider)

	// Assert: No rollout - pods don't have odigos agents, nothing to uninstrument
	assertNoRollout(t, statusChanged, result, err)
}

func TestNoRolloutICNilAutomaticRolloutDisabled(t *testing.T) {
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

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider)

	// Assert: No rollout - automatic rollout disabled, deployment NOT restarted even though it has agents
	assertNoRollout(t, statusChanged, result, err)
	// Verify deployment was NOT restarted
	var updatedDeployment appsv1.Deployment
	err = fakeClient.Get(s.ctx, client.ObjectKey{Name: deployment.Name, Namespace: deployment.Namespace}, &updatedDeployment)
	assert.NoError(t, err)
	assert.NotContains(t, updatedDeployment.Spec.Template.Annotations, "kubectl.kubernetes.io/restartedAt")
}

func TestTriggeredRolloutWorkloadWithInstrumentedPods(t *testing.T) {
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

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider)

	// Assert: Rollout triggered - deployment restarted to remove odigos agents (restartedAt annotation added)
	assertNoRollout(t, statusChanged, result, err)
	var updatedDeployment appsv1.Deployment
	err = fakeClient.Get(s.ctx, client.ObjectKey{Name: deployment.Name, Namespace: deployment.Namespace}, &updatedDeployment)
	assert.NoError(t, err)
	assert.Contains(t, updatedDeployment.Spec.Template.Annotations, "kubectl.kubernetes.io/restartedAt")
}

func TestTriggeredRolloutDistributionRequiresRollout(t *testing.T) {
	// Arrange: IC with distribution that requires rollout (e.g. eBPF-based instrumentation)
	s := newTestSetup()
	deployment := testutil.NewMockTestDeployment(s.ns, "test-deployment")
	ic := mockICRolloutRequiredDistro(testutil.NewMockInstrumentationConfig(deployment))
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	fakeClient := s.newFakeClient(deployment)

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider)

	// Assert: Rollout triggered successfully, requeue to monitor rollout progress
	assertTriggeredRolloutWithRequeue(t, statusChanged, result, err)
	assert.Equal(t, string(odigosv1alpha1.WorkloadRolloutReasonTriggeredSuccessfully), ic.Status.Conditions[0].Reason)
	assert.Equal(t, "workload rollout triggered successfully", ic.Status.Conditions[0].Message)
}

func TestNoRolloutDistributionDoesntRequire(t *testing.T) {
	// Arrange: IC with default distribution that doesn't require rollout (e.g., native instrumentation)
	s := newTestSetup()
	deployment := testutil.NewMockTestDeployment(s.ns, "test-deployment")
	ic := testutil.NewMockInstrumentationConfig(deployment)
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	fakeClient := s.newFakeClient(deployment)

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider)

	// Assert: Status updated to "NotRequired" - distribution doesn't need app restart, no requeue
	assertTriggeredRolloutNoRequeue(t, statusChanged, result, err)
	assert.Equal(t, string(odigosv1alpha1.WorkloadRolloutReasonNotRequired), ic.Status.Conditions[0].Reason)
	assert.Equal(t, "The selected instrumentation distributions do not require application restart", ic.Status.Conditions[0].Message)
}

func TestNoRolloutInvalidRollbackGraceTime(t *testing.T) {
	// Arrange: Config has invalid RollbackGraceTime string that can't be parsed as duration
	s := newTestSetup()
	s.conf.RollbackGraceTime = "invalid"
	deployment := testutil.NewMockTestDeployment(s.ns, "test-deployment")
	ic := mockICRolloutRequiredDistro(testutil.NewMockInstrumentationConfig(deployment))
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	fakeClient := s.newFakeClient(deployment)

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider)

	// Assert: Error returned - invalid config prevents rollout
	assertErrorNoRollout(t, statusChanged, result, err)
}

func TestNoRolloutInvalidRollbackStabilityWindow(t *testing.T) {
	// Arrange: Config has invalid RollbackStabilityWindow string that can't be parsed as duration
	s := newTestSetup()
	s.conf.RollbackStabilityWindow = "invalid"
	deployment := testutil.NewMockTestDeployment(s.ns, "test-deployment")
	ic := mockICRolloutRequiredDistro(testutil.NewMockInstrumentationConfig(deployment))
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	fakeClient := s.newFakeClient(deployment)

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider)

	// Assert: Error returned - invalid config prevents rollout
	assertErrorNoRollout(t, statusChanged, result, err)
}

func TestTriggeredRolloutConfigNil(t *testing.T) {
	// Arrange: IC requires rollout and s.conf.Rollout is nil (default config allows automatic rollout)
	s := newTestSetup()
	deployment := testutil.NewMockTestDeployment(s.ns, "test-deployment")
	ic := mockICRolloutRequiredDistro(testutil.NewMockInstrumentationConfig(deployment))
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	fakeClient := s.newFakeClient(deployment)

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider)

	// Assert: Rollout triggered - nil config defaults to enabled, requeue to monitor
	assertTriggeredRolloutWithRequeue(t, statusChanged, result, err)
	assert.Equal(t, string(odigosv1alpha1.WorkloadRolloutReasonTriggeredSuccessfully), ic.Status.Conditions[0].Reason)
}

func TestTriggeredRolloutAutomaticRolloutDisabledNil(t *testing.T) {
	// Arrange: RolloutConfiguration exists but AutomaticRolloutDisabled is nil (defaults to enabled)
	s := newTestSetup()
	s.conf.Rollout = &common.RolloutConfiguration{}
	deployment := testutil.NewMockTestDeployment(s.ns, "test-deployment")
	ic := mockICRolloutRequiredDistro(testutil.NewMockInstrumentationConfig(deployment))
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	fakeClient := s.newFakeClient(deployment)

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider)

	// Assert: Rollout triggered - nil pointer defaults to enabled, requeue to monitor
	assertTriggeredRolloutWithRequeue(t, statusChanged, result, err)
	assert.Equal(t, string(odigosv1alpha1.WorkloadRolloutReasonTriggeredSuccessfully), ic.Status.Conditions[0].Reason)
}

func TestTriggeredRolloutAutomaticRolloutDisabledFalse(t *testing.T) {
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

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider)

	// Assert: Rollout triggered successfully, requeue to monitor
	assertTriggeredRolloutWithRequeue(t, statusChanged, result, err)
	assert.Equal(t, string(odigosv1alpha1.WorkloadRolloutReasonTriggeredSuccessfully), ic.Status.Conditions[0].Reason)
}

func TestNoRolloutAutomaticRolloutDisabledTrue(t *testing.T) {
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

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider)

	// Assert: Status updated to "Disabled" - no rollout triggered, no requeue needed
	assertTriggeredRolloutNoRequeue(t, statusChanged, result, err)
	assert.Equal(t, string(odigosv1alpha1.WorkloadRolloutReasonDisabled), ic.Status.Conditions[0].Reason)
	assert.Equal(t, "odigos automatic rollout is disabled", ic.Status.Conditions[0].Message)
}

func TestNoRolloutMidRolloutRollbackDisabled(t *testing.T) {
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

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider)

	// Assert: No rollback triggered - rollbackDisabled config prevents it despite crashlooping pod
	assertNoRollout(t, statusChanged, result, err)
}

func TestNoRolloutMidRolloutWaitingNoBackoff(t *testing.T) {
	// Arrange: Mid-rollout deployment with healthy running pod - no crashloop, just waiting for rollout
	s := newTestSetup()
	deployment := newMockDeploymentMidRollout(s.ns, "test-deployment")
	ic := mockICMidRollout(testutil.NewMockInstrumentationConfig(deployment))
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}
	now := metav1.Now()
	ic.Status.InstrumentationTime = &now
	ic.Spec.AgentInjectionEnabled = true

	// Create a healthy pod
	healthyPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "healthy-pod",
			Namespace: s.ns.Name,
			Labels: map[string]string{
				"app.kubernetes.io/name":            deployment.Name,
				k8sconsts.OdigosAgentsMetaHashLabel: ic.Spec.AgentsMetaHash,
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  "test",
					Ready: true,
					State: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{},
					},
				},
			},
		},
	}

	fakeClient := s.newFakeClient(deployment, healthyPod)

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider)

	// Assert: No status change, requeue after 10s to continue monitoring rollout progress
	assert.NoError(t, err)
	assert.False(t, statusChanged, "expected no status change")
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, result)
}

func TestTriggeredRolloutPreviousRolloutOngoing(t *testing.T) {
	// Arrange: Deployment mid-rollout with different hash in IC status - a previous rollout is still in progress
	s := newTestSetup()
	deployment := newMockDeploymentMidRollout(s.ns, "test-deployment")
	ic := mockICRolloutRequiredDistro(testutil.NewMockInstrumentationConfig(deployment))
	// Set a DIFFERENT hash to trigger new rollout path (not mid-rollout)
	ic.Status.WorkloadRolloutHash = "old-hash"
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	fakeClient := s.newFakeClient(deployment)

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider)

	// Assert: Status updated to "PreviousRolloutOngoing", requeue to wait for previous rollout
	assert.NoError(t, err)
	assert.True(t, statusChanged, "expected status change")
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, result)
	assert.Equal(t, string(odigosv1alpha1.WorkloadRolloutReasonPreviousRolloutOngoing), ic.Status.Conditions[0].Reason)
}

func TestNoRolloutMidRolloutFailedToGetBackoffInfo(t *testing.T) {
	// Arrange: Mid-rollout deployment with nil selector - cannot query pods to check for backoff
	s := newTestSetup()
	deployment := newMockDeploymentMidRollout(s.ns, "test-deployment")
	deployment.Spec.Selector = nil // nil selector causes instrumentedPodsSelector to fail
	ic := mockICMidRollout(testutil.NewMockInstrumentationConfig(deployment))
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	fakeClient := s.newFakeClient(deployment)

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider)

	// Assert: Error returned - cannot determine backoff status due to nil selector
	assertErrorNoRollout(t, statusChanged, result, err)
}

func TestNoRolloutMidRolloutInstrumentationTimeIsNil(t *testing.T) {
	// Arrange: Mid-rollout deployment with IC that has nil InstrumentationTime (can't calculate backoff)
	s := newTestSetup()
	deployment := newMockDeploymentMidRollout(s.ns, "test-deployment")
	ic := mockICMidRollout(testutil.NewMockInstrumentationConfig(deployment))
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	fakeClient := s.newFakeClient(deployment)

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider)

	// Assert: No status change, requeue after 10s - waiting for rollout without backoff detection
	assert.NoError(t, err)
	assert.False(t, statusChanged, "expected no status change")
	// requeueWaitingForWorkloadRollout is 10 seconds but not publicly exported
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, result, "expected requeue after rollout")
}

func TestNoRolloutMidRolloutBackoffDurationLessThanGraceTime(t *testing.T) {
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

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider)

	// Assert: No rollback yet - still in grace period, requeue to check again after remaining grace time
	assert.NoError(t, err)
	assert.False(t, statusChanged, "expected no status change during grace period")
	assert.True(t, result.RequeueAfter > 0, "expected requeue with remaining grace time")
}

func TestNoRolloutMidRolloutClientUpdateError(t *testing.T) {
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

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider)

	// Assert: Error returned - rollback should happen but IC update failed
	assertErrorNoRollout(t, statusChanged, result, err)
}

func TestTriggeredRolloutMidRolloutRollbackRestartAnnotation(t *testing.T) {
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

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider)

	// Assert: Rollback triggered - IC updated, deployment gets restartedAt annotation to force restart
	assertTriggeredRollback(t, statusChanged, result, err, ic)

	// Assert deployment has restart annotation (kubectl.kubernetes.io/restartedAt)
	var updatedDeployment appsv1.Deployment
	err = fakeClient.Get(s.ctx, client.ObjectKey{Name: deployment.Name, Namespace: deployment.Namespace}, &updatedDeployment)
	assert.NoError(t, err)
	assert.Contains(t, updatedDeployment.Spec.Template.Annotations, "kubectl.kubernetes.io/restartedAt")
}

func TestTriggeredRolloutMidRolloutRollbackRestartAtArgoRollout(t *testing.T) {
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

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider)

	// Assert: Rollback triggered - Argo Rollout uses spec.restartAt instead of annotation
	assertTriggeredRollback(t, statusChanged, result, err, ic)

	// Assert Argo Rollout has spec.restartAt set
	var updatedRollout argorolloutsv1alpha1.Rollout
	err = fakeClient.Get(s.ctx, client.ObjectKey{Name: argoRollout.Name, Namespace: argoRollout.Namespace}, &updatedRollout)
	assert.NoError(t, err)
	assert.NotNil(t, updatedRollout.Spec.RestartAt, "expected spec.restartAt to be set for Argo Rollout")
}
