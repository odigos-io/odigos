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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
