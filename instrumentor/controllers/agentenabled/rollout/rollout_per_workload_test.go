package rollout_test

import (
	"testing"

	argorolloutsv1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/instrumentor/controllers/agentenabled/rollout"
	"github.com/odigos-io/odigos/instrumentor/internal/testutil"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func Test_NoRollout_StaticPodNoIC(t *testing.T) {
	// Arrange: StaticPod workload exists but has no InstrumentationConfig (IC is nil)
	s := newTestSetup()
	staticPod := testutil.NewMockTestStaticPod(s.ns, "test-staticpod")
	staticPodPw := k8sconsts.PodWorkload{Name: staticPod.Name, Namespace: staticPod.Namespace, Kind: k8sconsts.WorkloadKindStaticPod}

	fakeClient := s.newFakeClient(staticPod)
	var ic *odigosv1alpha1.InstrumentationConfig
	rateLimiter := newRateLimiterNoLimit()

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, staticPodPw, s.conf, s.distroProvider, rateLimiter)

	// Assert: No status change - StaticPod without IC don't need rollout
	assertNoStatusChange(t, statusChanged, result, err)
}

func Test_NoRollout_StaticPodWithIC_NotSupportingRestart(t *testing.T) {
	// Arrange: StaticPod workload exists, has an IC
	s := newTestSetup()
	staticPod := testutil.NewMockTestStaticPod(s.ns, "test-staticpod")
	staticPodPw := k8sconsts.PodWorkload{Name: staticPod.Name, Namespace: staticPod.Namespace, Kind: k8sconsts.WorkloadKindStaticPod}
	ic := testutil.NewMockInstrumentationConfig(staticPod)
	fakeClient := s.newFakeClient(staticPod, ic)
	rateLimiter := newRateLimiterNoLimit()

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, staticPodPw, s.conf, s.distroProvider, rateLimiter)

	// Assert: No rollout - StaticPod with IC don't support restart
	// NOTE: despite that the status is changed, the rollout is not triggered because the workload doesn't support restart
	// this is a hack to appease the UI.
	assert.Equal(t, true, statusChanged)
	assert.Equal(t, reconcile.Result{}, result)
	assert.NoError(t, err)
	assert.Equal(t, string(odigosv1alpha1.WorkloadRolloutReasonWorkloadNotSupporting), ic.Status.Conditions[0].Reason)
	assert.Equal(t, "static pods don't support restart", ic.Status.Conditions[0].Message)
}

func Test_NoRollout_JobOrCronjobNoIC(t *testing.T) {
	// Arrange: Job and CronJob workloads exist but have no InstrumentationConfig (IC is nil)
	s := newTestSetup()
	job := testutil.NewMockTestJob(s.ns, "test-job")
	cronjob := testutil.NewMockTestCronJob(s.ns, "test-cronjob")
	jobPw := k8sconsts.PodWorkload{Name: job.Name, Namespace: job.Namespace, Kind: k8sconsts.WorkloadKindJob}
	cronJobPw := k8sconsts.PodWorkload{Name: cronjob.Name, Namespace: cronjob.Namespace, Kind: k8sconsts.WorkloadKindCronJob}

	fakeClient := s.newFakeClient(job, cronjob)
	var ic *odigosv1alpha1.InstrumentationConfig
	rateLimiter := newRateLimiterNoLimit()

	// Act
	jobStatusChanged, jobResult, jobErr := rollout.Do(s.ctx, fakeClient, ic, jobPw, s.conf, s.distroProvider, rateLimiter)
	cronJobStatusChanged, cronJobResult, cronJobErr := rollout.Do(s.ctx, fakeClient, ic, cronJobPw, s.conf, s.distroProvider, rateLimiter)

	// Assert: No status change - Jobs/CronJobs without IC don't need rollout
	assertNoStatusChange(t, jobStatusChanged, jobResult, jobErr)
	assertNoStatusChange(t, cronJobStatusChanged, cronJobResult, cronJobErr)
}

func Test_NoRollout_JobOrCronjobWaitingForRestart(t *testing.T) {
	// Arrange: Job and CronJob with InstrumentationConfig - these can't be force-restarted like Deployments
	s := newTestSetup()
	job := testutil.NewMockTestJob(s.ns, "test-job")
	cronjob := testutil.NewMockTestCronJob(s.ns, "test-cronjob")
	jobPw := k8sconsts.PodWorkload{Name: job.Name, Namespace: job.Namespace, Kind: k8sconsts.WorkloadKindJob}
	cronJobPw := k8sconsts.PodWorkload{Name: cronjob.Name, Namespace: cronjob.Namespace, Kind: k8sconsts.WorkloadKindCronJob}
	jobIc := testutil.NewMockInstrumentationConfig(job)
	cronJobIc := testutil.NewMockInstrumentationConfig(cronjob)

	fakeClient := s.newFakeClient(job, cronjob)
	rateLimiter := newRateLimiterNoLimit()

	// Act
	jobStatusChanged, jobResult, jobErr := rollout.Do(s.ctx, fakeClient, jobIc, jobPw, s.conf, s.distroProvider, rateLimiter)
	cronJobStatusChanged, cronJobResult, cronJobErr := rollout.Do(s.ctx, fakeClient, cronJobIc, cronJobPw, s.conf, s.distroProvider, rateLimiter)

	// Assert: Status updated to "WaitingForRestart" - Jobs must trigger themselves, no requeue needed
	assertTriggeredRolloutNoRequeue(t, jobStatusChanged, jobResult, jobErr)
	assert.Equal(t, string(odigosv1alpha1.WorkloadRolloutReasonWaitingForRestart), jobIc.Status.Conditions[0].Reason)
	assert.Equal(t, "Waiting for job to trigger by itself", jobIc.Status.Conditions[0].Message)
	assertTriggeredRolloutNoRequeue(t, cronJobStatusChanged, cronJobResult, cronJobErr)
	assert.Equal(t, string(odigosv1alpha1.WorkloadRolloutReasonWaitingForRestart), cronJobIc.Status.Conditions[0].Reason)
	assert.Equal(t, "Waiting for job to trigger by itself", cronJobIc.Status.Conditions[0].Message)
}

func Test_Rollout_ICNil_HasAgents_RestartsUsing_rolloutRestartWorkload(t *testing.T) {
	// Arrange: Pod has odigos agent label, IC is nil, automatic rollout is enabled, so we should rollout the workload
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

func Test_Rollout_ICNil_HasAgents_RestartsStatefulSet(t *testing.T) {
	// Arrange: StatefulSet with instrumented pods (has odigos agent label), IC is nil
	s := newTestSetup()
	statefulSet := testutil.NewMockTestStatefulSet(s.ns)
	pw := k8sconsts.PodWorkload{Name: statefulSet.Name, Namespace: statefulSet.Namespace, Kind: k8sconsts.WorkloadKindStatefulSet}

	instrumentedPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: s.ns.Name,
			Labels: map[string]string{
				"app.kubernetes.io/name":            "test-ss",
				k8sconsts.OdigosAgentsMetaHashLabel: "abc123",
			},
		},
	}

	fakeClient := s.newFakeClient(statefulSet, instrumentedPod)
	var ic *odigosv1alpha1.InstrumentationConfig
	rateLimiter := newRateLimiterNoLimit()

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider, rateLimiter)

	// Assert: Workload IS restarted (to remove odigos agents), but no IC status change (IC is nil)
	assertNoStatusChange(t, statusChanged, result, err)
	var updatedStatefulSet appsv1.StatefulSet
	err = fakeClient.Get(s.ctx, client.ObjectKey{Name: statefulSet.Name, Namespace: statefulSet.Namespace}, &updatedStatefulSet)
	assert.NoError(t, err)
	assert.Contains(t, updatedStatefulSet.Spec.Template.Annotations, "kubectl.kubernetes.io/restartedAt")
}

func Test_Rollout_ICNil_HasAgents_RestartsDaemonSet(t *testing.T) {
	// Arrange: DaemonSet with instrumented pods (has odigos agent label), IC is nil
	s := newTestSetup()
	daemonSet := testutil.NewMockTestDaemonSet(s.ns)
	pw := k8sconsts.PodWorkload{Name: daemonSet.Name, Namespace: daemonSet.Namespace, Kind: k8sconsts.WorkloadKindDaemonSet}

	instrumentedPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: s.ns.Name,
			Labels: map[string]string{
				"app.kubernetes.io/name":            "test-ds",
				k8sconsts.OdigosAgentsMetaHashLabel: "abc123",
			},
		},
	}

	fakeClient := s.newFakeClient(daemonSet, instrumentedPod)
	var ic *odigosv1alpha1.InstrumentationConfig
	rateLimiter := newRateLimiterNoLimit()

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider, rateLimiter)

	// Assert: Workload IS restarted (to remove odigos agents), but no IC status change (IC is nil)
	assertNoStatusChange(t, statusChanged, result, err)
	var updatedDaemonSet appsv1.DaemonSet
	err = fakeClient.Get(s.ctx, client.ObjectKey{Name: daemonSet.Name, Namespace: daemonSet.Namespace}, &updatedDaemonSet)
	assert.NoError(t, err)
	assert.Contains(t, updatedDaemonSet.Spec.Template.Annotations, "kubectl.kubernetes.io/restartedAt")
}

func Test_Rollout_ICNil_HasAgents_RestartsArgoRollout(t *testing.T) {
	// Arrange: Argo Rollout with instrumented pods (has odigos agent label), IC is nil
	s := newTestSetup()
	argoRollout := newMockArgoRollout(s.ns, "test-rollout")
	pw := k8sconsts.PodWorkload{Name: argoRollout.Name, Namespace: argoRollout.Namespace, Kind: k8sconsts.WorkloadKindArgoRollout}

	instrumentedPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: s.ns.Name,
			Labels: map[string]string{
				"app.kubernetes.io/name":            argoRollout.Name,
				k8sconsts.OdigosAgentsMetaHashLabel: "abc123",
			},
		},
	}

	fakeClient := s.newFakeClient(argoRollout, instrumentedPod)
	var ic *odigosv1alpha1.InstrumentationConfig
	rateLimiter := newRateLimiterNoLimit()

	// Act
	statusChanged, result, err := rollout.Do(s.ctx, fakeClient, ic, pw, s.conf, s.distroProvider, rateLimiter)

	// Assert: Workload IS restarted (to remove odigos agents), but no IC status change (IC is nil)
	assertNoStatusChange(t, statusChanged, result, err)
	// Verify Argo Rollout was restarted with spec.restartAt field (different from other workloads)
	var updatedRollout argorolloutsv1alpha1.Rollout
	err = fakeClient.Get(s.ctx, client.ObjectKey{Name: argoRollout.Name, Namespace: argoRollout.Namespace}, &updatedRollout)
	assert.NoError(t, err)
	assert.NotNil(t, updatedRollout.Spec.RestartAt, "expected restartAt to be set for Argo Rollout")
}
