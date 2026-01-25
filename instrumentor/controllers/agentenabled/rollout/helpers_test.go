package rollout_test

import (
	"context"
	"errors"
	"testing"
	"time"

	argorolloutsv1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/distros"
	"github.com/odigos-io/odigos/instrumentor/controllers/agentenabled/rollout"
	"github.com/odigos-io/odigos/instrumentor/internal/testutil"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// testSetup holds common test dependencies to reduce repetition across tests.
type testSetup struct {
	ctx            context.Context
	scheme         *runtime.Scheme
	ns             *corev1.Namespace
	conf           *common.OdigosConfiguration
	distroProvider *distros.Provider
}

// ****************
// Setup helpers
// ****************
func newTestSetup() *testSetup {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = odigosv1alpha1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = batchv1.AddToScheme(scheme)
	_ = batchv1beta1.AddToScheme(scheme)
	_ = argorolloutsv1alpha1.AddToScheme(scheme)

	getter, _ := distros.NewCommunityGetter()
	provider, _ := distros.NewProvider(distros.NewCommunityDefaulter(), getter)

	return &testSetup{
		ctx:            context.Background(),
		scheme:         scheme,
		ns:             testutil.NewMockNamespace(),
		conf:           &common.OdigosConfiguration{},
		distroProvider: provider,
	}
}

func (s *testSetup) newFakeClient(objects ...client.Object) client.WithWatch {
	return fake.NewClientBuilder().
		WithScheme(s.scheme).
		WithObjects(objects...).
		Build()
}

func (s *testSetup) newFakeClientWithStatus(objects []client.Object, statusSubresources ...client.Object) client.WithWatch {
	return fake.NewClientBuilder().
		WithScheme(s.scheme).
		WithObjects(objects...).
		WithStatusSubresource(statusSubresources...).
		Build()
}

func (s *testSetup) newFakeClientWithICUpdateError(objects ...client.Object) client.WithWatch {
	return fake.NewClientBuilder().
		WithScheme(s.scheme).
		WithObjects(objects...).
		WithInterceptorFuncs(interceptor.Funcs{
			Update: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
				// Only fail updates to InstrumentationConfig
				if _, ok := obj.(*odigosv1alpha1.InstrumentationConfig); ok {
					return errors.New("simulated update error")
				}
				return c.Update(ctx, obj, opts...)
			},
			Patch: func(ctx context.Context, c client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
				// Only fail patches to InstrumentationConfig
				if _, ok := obj.(*odigosv1alpha1.InstrumentationConfig); ok {
					return errors.New("simulated patch error")
				}
				return c.Patch(ctx, obj, patch, opts...)
			},
		}).
		Build()
}

// ****************
// Assert helpers
// ****************

// assertNoStatusChange verifies Do() returned without changing IC status.
// Note: This does NOT check whether the workload was restarted - use assertWorkloadRestarted/assertWorkloadNotRestarted for that.
func assertNoStatusChange(t *testing.T, statusChanged bool, result reconcile.Result, err error) {
	t.Helper()
	assert.NoError(t, err)
	assert.False(t, statusChanged, "expected no status change")
	assert.Equal(t, reconcile.Result{}, result)
}

func assertTriggeredRolloutNoRequeue(t *testing.T, statusChanged bool, result reconcile.Result, err error) {
	t.Helper()
	assert.NoError(t, err)
	assert.True(t, statusChanged, "expected status change")
	assert.Equal(t, reconcile.Result{}, result)
}

// assertErrorNoStatusChange verifies Do() returned an error without changing IC status.
func assertErrorNoStatusChange(t *testing.T, statusChanged bool, result reconcile.Result, err error) {
	t.Helper()
	assert.Error(t, err)
	assert.False(t, statusChanged, "expected no status change on error")
	assert.Equal(t, reconcile.Result{}, result)
}

// assertWorkloadRestarted verifies the workload was restarted by checking for the restartedAt annotation.
func assertWorkloadRestarted(t *testing.T, ctx context.Context, c client.Client, pw k8sconsts.PodWorkload) {
	t.Helper()
	var dep appsv1.Deployment
	err := c.Get(ctx, client.ObjectKey{Name: pw.Name, Namespace: pw.Namespace}, &dep)
	assert.NoError(t, err)
	assert.Contains(t, dep.Spec.Template.Annotations, "kubectl.kubernetes.io/restartedAt", "expected workload to be restarted (restartedAt annotation)")
}

// assertWorkloadNotRestarted verifies the workload was NOT restarted by checking absence of restartedAt annotation.
func assertWorkloadNotRestarted(t *testing.T, ctx context.Context, c client.Client, pw k8sconsts.PodWorkload) {
	t.Helper()
	var dep appsv1.Deployment
	err := c.Get(ctx, client.ObjectKey{Name: pw.Name, Namespace: pw.Namespace}, &dep)
	assert.NoError(t, err)
	assert.NotContains(t, dep.Spec.Template.Annotations, "kubectl.kubernetes.io/restartedAt", "expected workload NOT to be restarted")
}

func assertTriggeredRolloutWithRequeue(t *testing.T, statusChanged bool, result reconcile.Result, err error) {
	t.Helper()
	assert.NoError(t, err)
	assert.True(t, statusChanged, "expected status change when rollout is triggered")
	assert.NotEqual(t, reconcile.Result{}, result, "expected requeue after rollout")
}

func assertTriggeredRollback(t *testing.T, statusChanged bool, result reconcile.Result, err error, ic *odigosv1alpha1.InstrumentationConfig) {
	t.Helper()
	assert.NoError(t, err)
	assert.True(t, statusChanged, "expected status change after rollback")
	assert.Equal(t, reconcile.Result{RequeueAfter: rollout.RequeueWaitingForWorkloadRollout}, result)
	assert.True(t, ic.Status.RollbackOccurred, "expected RollbackOccurred to be true")
	assert.Equal(t, string(odigosv1alpha1.WorkloadRolloutReasonTriggeredSuccessfully), ic.Status.Conditions[0].Reason)
}

// ****************
// Mock helpers
// ****************

// newMockDeploymentMidRollout creates a deployment in a mid-rollout state (Generation > ObservedGeneration).
func newMockDeploymentMidRollout(ns *corev1.Namespace, name string) *appsv1.Deployment {
	deployment := testutil.NewMockTestDeployment(ns, name)
	deployment.Generation = 2
	deployment.Status.ObservedGeneration = 1
	return deployment
}

// newMockArgoRollout creates an Argo Rollout in a stable state.
func newMockArgoRollout(ns *corev1.Namespace, name string) *argorolloutsv1alpha1.Rollout {
	return &argorolloutsv1alpha1.Rollout{
		ObjectMeta: metav1.ObjectMeta{
			Name:       name,
			Namespace:  ns.Name,
			Generation: 1,
		},
		Spec: argorolloutsv1alpha1.RolloutSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app.kubernetes.io/name": name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app.kubernetes.io/name": name},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "test", Image: "test"},
					},
				},
			},
		},
		Status: argorolloutsv1alpha1.RolloutStatus{
			ObservedGeneration: "1",
		},
	}
}

// newMockArgoRolloutMidRollout creates an Argo Rollout in a mid-rollout state (Generation > ObservedGeneration).
func newMockArgoRolloutMidRollout(ns *corev1.Namespace, name string) *argorolloutsv1alpha1.Rollout {
	return &argorolloutsv1alpha1.Rollout{
		ObjectMeta: metav1.ObjectMeta{
			Name:       name,
			Namespace:  ns.Name,
			Generation: 2,
		},
		Spec: argorolloutsv1alpha1.RolloutSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app.kubernetes.io/name": name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app.kubernetes.io/name": name},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "test", Image: "test"},
					},
				},
			},
		},
		Status: argorolloutsv1alpha1.RolloutStatus{
			ObservedGeneration: "1", // Argo uses string for ObservedGeneration
		},
	}
}

// newMockCrashingPod creates a pod in CrashLoopBackOff state that matches a deployment's selector.
func newMockCrashingPod(ns *corev1.Namespace, deploymentName string, agentsMetaHash string, startTime metav1.Time) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "crashing-pod",
			Namespace: ns.Name,
			Labels: map[string]string{
				"app.kubernetes.io/name":            deploymentName,
				k8sconsts.OdigosAgentsMetaHashLabel: agentsMetaHash,
			},
		},
		Status: corev1.PodStatus{
			StartTime: &startTime,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "test",
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason: "CrashLoopBackOff",
						},
					},
				},
			},
		},
	}
}

// mockICRolloutRequiredDistro creates an InstrumentationConfig with a distribution that requires rollout (java-community).
func mockICRolloutRequiredDistro(base *odigosv1alpha1.InstrumentationConfig) *odigosv1alpha1.InstrumentationConfig {
	base.Spec.Containers = []odigosv1alpha1.ContainerAgentConfig{
		{
			ContainerName:  "test",
			OtelDistroName: "java-community", // This distro requires rollout (noRestartRequired is false)
		},
	}
	base.Spec.AgentsMetaHash = "test-hash-to-trigger-rollout"
	return base
}

// mockICMidRollout creates an InstrumentationConfig in a mid-rollout state
// (rollout-required distro + matching WorkloadRolloutHash to trigger mid-rollout checks).
func mockICMidRollout(base *odigosv1alpha1.InstrumentationConfig) *odigosv1alpha1.InstrumentationConfig {
	ic := mockICRolloutRequiredDistro(base)
	ic.Status.WorkloadRolloutHash = ic.Spec.AgentsMetaHash
	return ic
}

// ****************
// Rate Limiter Fixtures
// ****************

// newRateLimiterNoLimit creates a rate limiter with infinite limit (no rate limiting).
// This is used for tests that don't care about rate limiting behavior.
func newRateLimiterNoLimit() *rollout.RolloutRateLimiter {
	return rollout.NewRolloutRateLimiter(nil) // nil config defaults to rate limiting
}

// newRateLimiterWithLimit creates a rate limiter with a specific concurrent rollout limit.
// Use this for tests that need to verify rate limiting behavior.
func newRateLimiterWithLimit(concurrentRollouts float64) *rollout.RolloutRateLimiter {
	enabled := true
	conf := &common.OdigosConfiguration{
		Rollout: &common.RolloutConfiguration{
			IsConcurrentRolloutsEnabled: &enabled,
			ConcurrentRollouts:          concurrentRollouts,
		},
	}
	return rollout.NewRolloutRateLimiter(conf)
}

// newRateLimiterActive creates a rate limiter with a low limit (1 concurrent rollout).
// First call to Allow() succeeds, subsequent calls fail until token is replenished.
func newRateLimiterActive() *rollout.RolloutRateLimiter {
	return newRateLimiterWithLimit(1.0)
}

// newRateLimiterExhausted creates a rate limiter that has already used its quota.
// Any call to Allow() will return false.
func newRateLimiterExhausted() *rollout.RolloutRateLimiter {
	limiter := newRateLimiterActive()
	limiter.Allow() // Exhaust the single token
	return limiter
}

// newHealthyPod creates a healthy running pod that matches a deployment's selector.
func newHealthyPod(ns *corev1.Namespace, deploymentName, podName string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: ns.Name,
			Labels: map[string]string{
				"app.kubernetes.io/name": deploymentName,
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  "test",
					Ready: true,
					State: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{
							StartedAt: metav1.NewTime(time.Now().Add(-5 * time.Minute)),
						},
					},
				},
			},
		},
	}
}

// newCrashLoopBackOffPod creates a pod in CrashLoopBackOff state WITH odigos label (already instrumented).
func newCrashLoopBackOffPod(ns *corev1.Namespace, deploymentName, podName string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: ns.Name,
			Labels: map[string]string{
				"app.kubernetes.io/name":            deploymentName,
				k8sconsts.OdigosAgentsMetaHashLabel: "some-hash",
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "test",
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason: "CrashLoopBackOff",
						},
					},
				},
			},
		},
	}
}

// newCrashLoopBackOffPodWithoutOdigosLabel creates a pod in CrashLoopBackOff WITHOUT odigos label (not instrumented).
func newCrashLoopBackOffPodWithoutOdigosLabel(ns *corev1.Namespace, deploymentName, podName string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: ns.Name,
			Labels: map[string]string{
				"app.kubernetes.io/name": deploymentName,
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "test",
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason: "CrashLoopBackOff",
						},
					},
				},
			},
		},
	}
}

// newImagePullBackOffPod creates a pod in ImagePullBackOff state.
func newImagePullBackOffPod(ns *corev1.Namespace, deploymentName, podName string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: ns.Name,
			Labels: map[string]string{
				"app.kubernetes.io/name": deploymentName,
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "test",
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason: "ImagePullBackOff",
						},
					},
				},
			},
		},
	}
}

// newInitContainerBackOffPod creates a pod with init container in CrashLoopBackOff state.
func newInitContainerBackOffPod(ns *corev1.Namespace, deploymentName, podName string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: ns.Name,
			Labels: map[string]string{
				"app.kubernetes.io/name": deploymentName,
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
			InitContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "init-container",
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason: "CrashLoopBackOff",
						},
					},
				},
			},
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "test",
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason: "PodInitializing",
						},
					},
				},
			},
		},
	}
}

// newCrashLoopBackOffStaticPod creates a static pod in CrashLoopBackOff state WITHOUT odigos label.
func newCrashLoopBackOffStaticPod(ns *corev1.Namespace, podName string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: ns.Name,
			Annotations: map[string]string{
				"kubernetes.io/config.source": "file",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind: "Node",
					Name: "test-node",
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "test",
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason: "CrashLoopBackOff",
						},
					},
				},
			},
		},
	}
}
