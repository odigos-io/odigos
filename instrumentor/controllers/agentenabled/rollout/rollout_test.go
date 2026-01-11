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
	// Arrange
	s := newTestSetup()
	deployment := testutil.NewMockTestDeployment(s.ns, "test-deployment")
	ic := testutil.NewMockInstrumentationConfig(deployment)
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}
	fakeClient := s.newFakeClientWithStatus(nil, ic)

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider)

	// Assert
	assertNoRollout(t, statusChanged, result, err)
}

func TestNoRolloutJobOrCronjobNoIC(t *testing.T) {
	// Arrange
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

	// Assert
	assertNoRollout(t, jobStatusChanged, jobResult, jobErr)
	assertNoRollout(t, cronJobStatusChanged, cronJobResult, cronJobErr)
}

func TestNoRolloutJobOrCronjobWaitingForRestart(t *testing.T) {
	// Arrange
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

	// Assert
	assertTriggeredRolloutNoRequeue(t, jobStatusChanged, jobResult, jobErr)
	assert.Equal(t, string(odigosv1alpha1.WorkloadRolloutReasonWaitingForRestart), jobIc.Status.Conditions[0].Reason)
	assert.Equal(t, "Waiting for job to trigger by itself", jobIc.Status.Conditions[0].Message)
	assertTriggeredRolloutNoRequeue(t, cronJobStatusChanged, cronJobResult, cronJobErr)
	assert.Equal(t, string(odigosv1alpha1.WorkloadRolloutReasonWaitingForRestart), cronJobIc.Status.Conditions[0].Reason)
	assert.Equal(t, "Waiting for job to trigger by itself", cronJobIc.Status.Conditions[0].Message)
}

func TestNoRolloutMidRolloutAlreadyComplete(t *testing.T) {
	// Arrange
	s := newTestSetup()
	deployment := testutil.NewMockTestDeployment(s.ns, "test-deployment")
	ic := mockICMidRollout(testutil.NewMockInstrumentationConfig(deployment))
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	fakeClient := s.newFakeClient(deployment)

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider)

	// Assert
	assertNoRollout(t, statusChanged, result, err)
}

func TestNoRolloutICNilWorkloadHasOdigosAgentsError(t *testing.T) {
	// Arrange
	s := newTestSetup()
	deployment := testutil.NewMockTestDeployment(s.ns, "test-deployment")
	deployment.Spec.Selector = nil // nil selector causes instrumentedPodsSelector to fail
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	fakeClient := s.newFakeClient(deployment)
	var ic *odigosv1alpha1.InstrumentationConfig

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider)

	// Assert
	assertErrorNoRollout(t, statusChanged, result, err)
}

func TestNoRolloutICNilNoOdigosAgents(t *testing.T) {
	// Arrange
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

	// Assert
	assertNoRollout(t, statusChanged, result, err)
}

func TestNoRolloutICNilAutomaticRolloutDisabled(t *testing.T) {
	// Arrange
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

	// Assert
	assertNoRollout(t, statusChanged, result, err)
	// Verify deployment was NOT restarted
	var updatedDeployment appsv1.Deployment
	err = fakeClient.Get(s.ctx, client.ObjectKey{Name: deployment.Name, Namespace: deployment.Namespace}, &updatedDeployment)
	assert.NoError(t, err)
	assert.NotContains(t, updatedDeployment.Spec.Template.Annotations, "kubectl.kubernetes.io/restartedAt")
}

func TestTriggeredRolloutWorkloadWithInstrumentedPods(t *testing.T) {
	// Arrange
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

	// Assert
	assertNoRollout(t, statusChanged, result, err)
	var updatedDeployment appsv1.Deployment
	err = fakeClient.Get(s.ctx, client.ObjectKey{Name: deployment.Name, Namespace: deployment.Namespace}, &updatedDeployment)
	assert.NoError(t, err)
	assert.Contains(t, updatedDeployment.Spec.Template.Annotations, "kubectl.kubernetes.io/restartedAt")
}

func TestTriggeredRolloutDistributionRequiresRollout(t *testing.T) {
	// Arrange
	s := newTestSetup()
	deployment := testutil.NewMockTestDeployment(s.ns, "test-deployment")
	ic := mockICRolloutRequiredDistro(testutil.NewMockInstrumentationConfig(deployment))
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	fakeClient := s.newFakeClient(deployment)

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider)

	// Assert
	assertTriggeredRolloutWithRequeue(t, statusChanged, result, err)
	assert.Equal(t, string(odigosv1alpha1.WorkloadRolloutReasonTriggeredSuccessfully), ic.Status.Conditions[0].Reason)
	assert.Equal(t, "workload rollout triggered successfully", ic.Status.Conditions[0].Message)
}

func TestNoRolloutDistributionDoesntRequire(t *testing.T) {
	// Arrange
	s := newTestSetup()
	deployment := testutil.NewMockTestDeployment(s.ns, "test-deployment")
	ic := testutil.NewMockInstrumentationConfig(deployment)
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	fakeClient := s.newFakeClient(deployment)

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider)

	// Assert
	assertTriggeredRolloutNoRequeue(t, statusChanged, result, err)
	assert.Equal(t, string(odigosv1alpha1.WorkloadRolloutReasonNotRequired), ic.Status.Conditions[0].Reason)
	assert.Equal(t, "The selected instrumentation distributions do not require application restart", ic.Status.Conditions[0].Message)
}

func TestNoRolloutInvalidRollbackGraceTime(t *testing.T) {
	// Arrange
	s := newTestSetup()
	s.conf.RollbackGraceTime = "invalid"
	deployment := testutil.NewMockTestDeployment(s.ns, "test-deployment")
	ic := mockICRolloutRequiredDistro(testutil.NewMockInstrumentationConfig(deployment))
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	fakeClient := s.newFakeClient(deployment)

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider)

	// Assert
	assertErrorNoRollout(t, statusChanged, result, err)
}

func TestNoRolloutInvalidRollbackStabilityWindow(t *testing.T) {
	// Arrange
	s := newTestSetup()
	s.conf.RollbackStabilityWindow = "invalid"
	deployment := testutil.NewMockTestDeployment(s.ns, "test-deployment")
	ic := mockICRolloutRequiredDistro(testutil.NewMockInstrumentationConfig(deployment))
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	fakeClient := s.newFakeClient(deployment)

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider)

	// Assert
	assertErrorNoRollout(t, statusChanged, result, err)
}

func TestTriggeredRolloutConfigNil(t *testing.T) {
	// Arrange
	s := newTestSetup()
	deployment := testutil.NewMockTestDeployment(s.ns, "test-deployment")
	ic := mockICRolloutRequiredDistro(testutil.NewMockInstrumentationConfig(deployment))
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	fakeClient := s.newFakeClient(deployment)

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider)

	// Assert
	assertTriggeredRolloutWithRequeue(t, statusChanged, result, err)
	assert.Equal(t, string(odigosv1alpha1.WorkloadRolloutReasonTriggeredSuccessfully), ic.Status.Conditions[0].Reason)
}

func TestTriggeredRolloutAutomaticRolloutDisabledNil(t *testing.T) {
	// Arrange
	s := newTestSetup()
	s.conf.Rollout = &common.RolloutConfiguration{}
	deployment := testutil.NewMockTestDeployment(s.ns, "test-deployment")
	ic := mockICRolloutRequiredDistro(testutil.NewMockInstrumentationConfig(deployment))
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	fakeClient := s.newFakeClient(deployment)

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider)

	// Assert
	assertTriggeredRolloutWithRequeue(t, statusChanged, result, err)
	assert.Equal(t, string(odigosv1alpha1.WorkloadRolloutReasonTriggeredSuccessfully), ic.Status.Conditions[0].Reason)
}

func TestTriggeredRolloutAutomaticRolloutDisabledFalse(t *testing.T) {
	// Arrange
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

	// Assert
	assertTriggeredRolloutWithRequeue(t, statusChanged, result, err)
	assert.Equal(t, string(odigosv1alpha1.WorkloadRolloutReasonTriggeredSuccessfully), ic.Status.Conditions[0].Reason)
}

func TestNoRolloutAutomaticRolloutDisabledTrue(t *testing.T) {
	// Arrange
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

	// Assert
	assertTriggeredRolloutNoRequeue(t, statusChanged, result, err)
	assert.Equal(t, string(odigosv1alpha1.WorkloadRolloutReasonDisabled), ic.Status.Conditions[0].Reason)
	assert.Equal(t, "odigos automatic rollout is disabled", ic.Status.Conditions[0].Message)
}

func TestNoRolloutMidRolloutRollbackDisabled(t *testing.T) {
	// Arrange
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

	// Assert - should NOT trigger rollback because rollbackDisabled is true
	assertNoRollout(t, statusChanged, result, err)
}

func TestNoRolloutMidRolloutWaitingNoBackoff(t *testing.T) {
	// Arrange
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

	// Assert - requeue waiting for workload rollout to complete
	assert.NoError(t, err)
	assert.False(t, statusChanged, "expected no status change")
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, result)
}

func TestTriggeredRolloutPreviousRolloutOngoing(t *testing.T) {
	// Arrange
	s := newTestSetup()
	deployment := newMockDeploymentMidRollout(s.ns, "test-deployment")
	ic := mockICRolloutRequiredDistro(testutil.NewMockInstrumentationConfig(deployment))
	// Set a DIFFERENT hash to trigger new rollout path (not mid-rollout)
	ic.Status.WorkloadRolloutHash = "old-hash"
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	fakeClient := s.newFakeClient(deployment)

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider)

	// Assert - status changed to PreviousRolloutOngoing, requeue
	assert.NoError(t, err)
	assert.True(t, statusChanged, "expected status change")
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, result)
	assert.Equal(t, string(odigosv1alpha1.WorkloadRolloutReasonPreviousRolloutOngoing), ic.Status.Conditions[0].Reason)
}

func TestNoRolloutMidRolloutFailedToGetBackoffInfo(t *testing.T) {
	// Arrange
	s := newTestSetup()
	deployment := newMockDeploymentMidRollout(s.ns, "test-deployment")
	deployment.Spec.Selector = nil // nil selector causes instrumentedPodsSelector to fail
	ic := mockICMidRollout(testutil.NewMockInstrumentationConfig(deployment))
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	fakeClient := s.newFakeClient(deployment)

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider)

	// Assert
	assertErrorNoRollout(t, statusChanged, result, err)
}

func TestNoRolloutMidRolloutInstrumentationTimeIsNil(t *testing.T) {
	// Arrange
	s := newTestSetup()
	deployment := newMockDeploymentMidRollout(s.ns, "test-deployment")
	ic := mockICMidRollout(testutil.NewMockInstrumentationConfig(deployment))
	pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

	fakeClient := s.newFakeClient(deployment)

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider)

	// Assert
	assert.NoError(t, err)
	assert.False(t, statusChanged, "expected no status change")
	// requeueWaitingForWorkloadRollout is 10 seconds but not publicly exported
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, result, "expected requeue after rollout")
}

func TestNoRolloutMidRolloutBackoffDurationLessThanGraceTime(t *testing.T) {
	// Arrange
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

	// Assert
	assert.NoError(t, err)
	assert.False(t, statusChanged, "expected no status change during grace period")
	assert.True(t, result.RequeueAfter > 0, "expected requeue with remaining grace time")
}

func TestNoRolloutMidRolloutClientUpdateError(t *testing.T) {
	// Arrange
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

	// Assert
	assertErrorNoRollout(t, statusChanged, result, err)
}

func TestTriggeredRolloutMidRolloutRollbackRestartAnnotation(t *testing.T) {
	// Arrange
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

	// Assert
	assertTriggeredRollback(t, statusChanged, result, err, ic)

	// Assert deployment has restart annotation (kubectl.kubernetes.io/restartedAt)
	var updatedDeployment appsv1.Deployment
	err = fakeClient.Get(s.ctx, client.ObjectKey{Name: deployment.Name, Namespace: deployment.Namespace}, &updatedDeployment)
	assert.NoError(t, err)
	assert.Contains(t, updatedDeployment.Spec.Template.Annotations, "kubectl.kubernetes.io/restartedAt")
}

func TestTriggeredRolloutMidRolloutRollbackRestartAtArgoRollout(t *testing.T) {
	// Arrange
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

	// Assert
	assertTriggeredRollback(t, statusChanged, result, err, ic)

	// Assert Argo Rollout has spec.restartAt set
	var updatedRollout argorolloutsv1alpha1.Rollout
	err = fakeClient.Get(s.ctx, client.ObjectKey{Name: argoRollout.Name, Namespace: argoRollout.Namespace}, &updatedRollout)
	assert.NoError(t, err)
	assert.NotNil(t, updatedRollout.Spec.RestartAt, "expected spec.restartAt to be set for Argo Rollout")
}
