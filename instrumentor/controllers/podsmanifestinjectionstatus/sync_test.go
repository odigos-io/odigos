package podsmanifestinjectionstatus

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	podsManifestInjection "github.com/odigos-io/odigos/status/instrumentationconfig/generated"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

const (
	injectionNamespace   = "app-ns"
	injectionOtherNS     = "other-ns"
	injectionCurrentHash = "agents-hash-current"
	injectionStaleHash   = "agents-hash-stale"
)

// ****************
// Setup helpers
// ****************

func newInjectionTestContext() context.Context {
	return logr.NewContext(context.Background(), logr.Discard())
}

// newEffectiveConfigMap builds the odigos effective config ConfigMap that syncWorkload reads on
// every reconcile. An empty configYAML yields the zero-value configuration.
func newEffectiveConfigMap(configYAML string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.OdigosEffectiveConfigName,
			Namespace: consts.DefaultOdigosNamespace,
		},
		Data: map[string]string{consts.OdigosConfigurationFileName: configYAML},
	}
}

func newInjectionTestClient(t *testing.T, objects ...client.Object) client.WithWatch {
	t.Helper()
	t.Setenv(consts.CurrentNamespaceEnvVar, consts.DefaultOdigosNamespace)

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, odigosv1.AddToScheme(scheme))

	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objects...).
		WithStatusSubresource(&odigosv1.InstrumentationConfig{}).
		Build()
}

// ****************
// Mock helpers
// ****************

// newInjectionInstrumentationConfig builds the InstrumentationConfig that syncWorkload writes the
// observed pod injection status onto. Its name encodes the workload it belongs to.
func newInjectionInstrumentationConfig(name string, kind k8sconsts.WorkloadKind) *odigosv1.InstrumentationConfig {
	return &odigosv1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workload.CalculateWorkloadRuntimeObjectName(name, kind),
			Namespace: injectionNamespace,
		},
		Spec: odigosv1.InstrumentationConfigSpec{
			AgentInjectionEnabled: true,
			AgentsMetaHash:        injectionCurrentHash,
		},
	}
}

// newInjectionPod builds a running pod. An empty agentsHash means the pod carries no odigos agents
// hash label at all, which is how an uninjected pod is recognized.
func newInjectionPod(namespace, name, agentsHash string) *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: map[string]string{}},
		Status:     corev1.PodStatus{Phase: corev1.PodRunning},
	}
	if agentsHash != "" {
		pod.Labels[k8sconsts.OdigosAgentsMetaHashLabel] = agentsHash
	}
	return pod
}

func withOwner(pod *corev1.Pod, apiVersion, kind, name string) *corev1.Pod {
	pod.OwnerReferences = append(pod.OwnerReferences, metav1.OwnerReference{
		APIVersion: apiVersion,
		Kind:       kind,
		Name:       name,
		UID:        types.UID(name),
	})
	return pod
}

func withPodLabel(pod *corev1.Pod, key, value string) *corev1.Pod {
	pod.Labels[key] = value
	return pod
}

func withPhase(pod *corev1.Pod, phase corev1.PodPhase) *corev1.Pod {
	pod.Status.Phase = phase
	return pod
}

func newInjectionDeployment(name string, selector map[string]string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: injectionNamespace},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: selector},
			Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: selector}},
		},
	}
}

func newInjectionCronJob(name string) *batchv1.CronJob {
	return &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: injectionNamespace},
		Spec:       batchv1.CronJobSpec{Schedule: "*/5 * * * *"},
	}
}

// syncedInstrumentationConfig re-reads the InstrumentationConfig that syncWorkload updated.
func syncedInstrumentationConfig(t *testing.T, ctx context.Context, c client.Client, name string,
	kind k8sconsts.WorkloadKind) *odigosv1.InstrumentationConfig {
	t.Helper()
	ic := &odigosv1.InstrumentationConfig{}
	require.NoError(t, c.Get(ctx, types.NamespacedName{
		Namespace: injectionNamespace,
		Name:      workload.CalculateWorkloadRuntimeObjectName(name, kind),
	}, ic))
	return ic
}

// requireInjectionStatus asserts the full observed injection status. All three flags are always
// asserted together: a pod that leaks into the result set flips a flag that should stay false.
func requireInjectionStatus(t *testing.T, ic *odigosv1.InstrumentationConfig,
	upToDate, outOfDate, uninjected bool) {
	t.Helper()
	require.NotNil(t, ic.Status.PodsManifestInjectionStatus)
	assert.Equal(t, upToDate, ic.Status.PodsManifestInjectionStatus.HasInjectedUpToDatePods,
		"HasInjectedUpToDatePods")
	assert.Equal(t, outOfDate, ic.Status.PodsManifestInjectionStatus.HasInjectedOutOfDatePods,
		"HasInjectedOutOfDatePods")
	assert.Equal(t, uninjected, ic.Status.PodsManifestInjectionStatus.HasUninjectedPods,
		"HasUninjectedPods")
}

func injectionConditionReason(ic *odigosv1.InstrumentationConfig) string {
	cond := meta.FindStatusCondition(ic.Status.Conditions, podsManifestInjection.PodsManifestInjectionType)
	if cond == nil {
		return ""
	}
	return cond.Reason
}

// ****************
// CronJob pod resolution
// ****************

// CronJobs have no label selector, so their pods can only be found through the Job that owns them.
func TestSyncWorkloadResolvesCronJobPodsThroughJobOwnership(t *testing.T) {
	ctx := newInjectionTestContext()
	c := newInjectionTestClient(t,
		newEffectiveConfigMap(""),
		newInjectionCronJob("backup"),
		newInjectionInstrumentationConfig("backup", k8sconsts.WorkloadKindCronJob),
		// pods of the CronJob under test, each owned by one of its Job runs
		withOwner(newInjectionPod(injectionNamespace, "backup-28001-abc", injectionCurrentHash),
			"batch/v1", "Job", "backup-28001"),
		withOwner(newInjectionPod(injectionNamespace, "backup-28002-def", ""),
			"batch/v1", "Job", "backup-28002"),
		// every pod below belongs to something else and carries the stale hash, so counting any of
		// them would flip HasInjectedOutOfDatePods
		withOwner(newInjectionPod(injectionNamespace, "cleanup-28001-ghi", injectionStaleHash),
			"batch/v1", "Job", "cleanup-28001"),
		withOwner(newInjectionPod(injectionNamespace, "web-7d4c8-jkl", injectionStaleHash),
			"apps/v1", "ReplicaSet", "web-7d4c8"),
		newInjectionPod(injectionNamespace, "standalone", injectionStaleHash),
	)

	require.NoError(t, syncWorkload(ctx, c, k8sconsts.PodWorkload{
		Namespace: injectionNamespace, Name: "backup", Kind: k8sconsts.WorkloadKindCronJob,
	}))

	ic := syncedInstrumentationConfig(t, ctx, c, "backup", k8sconsts.WorkloadKindCronJob)
	requireInjectionStatus(t, ic, true, false, true)
}

func TestSyncWorkloadCronJobIgnoresPodsInOtherNamespaces(t *testing.T) {
	ctx := newInjectionTestContext()
	c := newInjectionTestClient(t,
		newEffectiveConfigMap(""),
		newInjectionCronJob("backup"),
		newInjectionInstrumentationConfig("backup", k8sconsts.WorkloadKindCronJob),
		withOwner(newInjectionPod(injectionNamespace, "backup-28001-abc", injectionCurrentHash),
			"batch/v1", "Job", "backup-28001"),
		// a same-named CronJob in another namespace is a different workload
		withOwner(newInjectionPod(injectionOtherNS, "backup-28001-zzz", injectionStaleHash),
			"batch/v1", "Job", "backup-28001"),
	)

	require.NoError(t, syncWorkload(ctx, c, k8sconsts.PodWorkload{
		Namespace: injectionNamespace, Name: "backup", Kind: k8sconsts.WorkloadKindCronJob,
	}))

	ic := syncedInstrumentationConfig(t, ctx, c, "backup", k8sconsts.WorkloadKindCronJob)
	requireInjectionStatus(t, ic, true, false, false)
}

// A Job whose name carries no generated suffix cannot be attributed to a CronJob. It must be
// skipped without aborting the sync, or one manually created Job breaks the whole CronJob status.
func TestSyncWorkloadCronJobSkipsJobOwnerWithoutGeneratedSuffix(t *testing.T) {
	ctx := newInjectionTestContext()
	c := newInjectionTestClient(t,
		newEffectiveConfigMap(""),
		newInjectionCronJob("backup"),
		newInjectionInstrumentationConfig("backup", k8sconsts.WorkloadKindCronJob),
		withOwner(newInjectionPod(injectionNamespace, "backup-manual", injectionStaleHash),
			"batch/v1", "Job", "backup"),
		withOwner(newInjectionPod(injectionNamespace, "backup-28001-abc", injectionCurrentHash),
			"batch/v1", "Job", "backup-28001"),
	)

	require.NoError(t, syncWorkload(ctx, c, k8sconsts.PodWorkload{
		Namespace: injectionNamespace, Name: "backup", Kind: k8sconsts.WorkloadKindCronJob,
	}))

	ic := syncedInstrumentationConfig(t, ctx, c, "backup", k8sconsts.WorkloadKindCronJob)
	requireInjectionStatus(t, ic, true, false, false)
}

// ****************
// Terminated pods
// ****************

func TestSyncWorkloadSkipsSucceededAndFailedPods(t *testing.T) {
	t.Run("terminated pods do not contribute to the status of a running workload", func(t *testing.T) {
		ctx := newInjectionTestContext()
		selector := map[string]string{"app": "web"}
		c := newInjectionTestClient(t,
			newEffectiveConfigMap(""),
			newInjectionDeployment("web", selector),
			newInjectionInstrumentationConfig("web", k8sconsts.WorkloadKindDeployment),
			withPodLabel(newInjectionPod(injectionNamespace, "web-running", injectionCurrentHash),
				"app", "web"),
			withPhase(withPodLabel(newInjectionPod(injectionNamespace, "web-succeeded",
				injectionStaleHash), "app", "web"), corev1.PodSucceeded),
			withPhase(withPodLabel(newInjectionPod(injectionNamespace, "web-failed", ""),
				"app", "web"), corev1.PodFailed),
		)

		require.NoError(t, syncWorkload(ctx, c, k8sconsts.PodWorkload{
			Namespace: injectionNamespace, Name: "web", Kind: k8sconsts.WorkloadKindDeployment,
		}))

		ic := syncedInstrumentationConfig(t, ctx, c, "web", k8sconsts.WorkloadKindDeployment)
		requireInjectionStatus(t, ic, true, false, false)
	})

	// A CronJob between runs has only completed pods. Counting them would leave the workload
	// permanently reporting the injection state of a Job run that already finished.
	t.Run("a cronjob whose runs all finished reports no pods", func(t *testing.T) {
		ctx := newInjectionTestContext()
		c := newInjectionTestClient(t,
			newEffectiveConfigMap(""),
			newInjectionCronJob("backup"),
			newInjectionInstrumentationConfig("backup", k8sconsts.WorkloadKindCronJob),
			withPhase(withOwner(newInjectionPod(injectionNamespace, "backup-28001-abc",
				injectionStaleHash), "batch/v1", "Job", "backup-28001"), corev1.PodSucceeded),
			withPhase(withOwner(newInjectionPod(injectionNamespace, "backup-28002-def", ""),
				"batch/v1", "Job", "backup-28002"), corev1.PodFailed),
		)

		require.NoError(t, syncWorkload(ctx, c, k8sconsts.PodWorkload{
			Namespace: injectionNamespace, Name: "backup", Kind: k8sconsts.WorkloadKindCronJob,
		}))

		ic := syncedInstrumentationConfig(t, ctx, c, "backup", k8sconsts.WorkloadKindCronJob)
		requireInjectionStatus(t, ic, false, false, false)
		assert.Equal(t, string(podsManifestInjection.PodsManifestInjectionReasonNoPods),
			injectionConditionReason(ic))
	})
}

// ****************
// Label selector pod resolution
// ****************

func TestSyncWorkloadLabelSelectorListIsScopedToTheWorkloadNamespace(t *testing.T) {
	ctx := newInjectionTestContext()
	selector := map[string]string{"app": "web"}
	c := newInjectionTestClient(t,
		newEffectiveConfigMap(""),
		newInjectionDeployment("web", selector),
		newInjectionInstrumentationConfig("web", k8sconsts.WorkloadKindDeployment),
		withPodLabel(newInjectionPod(injectionNamespace, "web-1", injectionCurrentHash), "app", "web"),
		// an unrelated workload in another namespace that happens to use the same labels
		withPodLabel(newInjectionPod(injectionOtherNS, "web-2", ""), "app", "web"),
	)

	require.NoError(t, syncWorkload(ctx, c, k8sconsts.PodWorkload{
		Namespace: injectionNamespace, Name: "web", Kind: k8sconsts.WorkloadKindDeployment,
	}))

	ic := syncedInstrumentationConfig(t, ctx, c, "web", k8sconsts.WorkloadKindDeployment)
	requireInjectionStatus(t, ic, true, false, false)
}

func TestSyncWorkloadLabelSelectorMatchesOnlySelectedPods(t *testing.T) {
	ctx := newInjectionTestContext()
	c := newInjectionTestClient(t,
		newEffectiveConfigMap(""),
		newInjectionDeployment("web", map[string]string{"app": "web"}),
		newInjectionInstrumentationConfig("web", k8sconsts.WorkloadKindDeployment),
		withPodLabel(newInjectionPod(injectionNamespace, "web-1", injectionStaleHash), "app", "web"),
		withPodLabel(newInjectionPod(injectionNamespace, "api-1", injectionCurrentHash), "app", "api"),
	)

	require.NoError(t, syncWorkload(ctx, c, k8sconsts.PodWorkload{
		Namespace: injectionNamespace, Name: "web", Kind: k8sconsts.WorkloadKindDeployment,
	}))

	ic := syncedInstrumentationConfig(t, ctx, c, "web", k8sconsts.WorkloadKindDeployment)
	requireInjectionStatus(t, ic, false, true, false)
}

// A workload kind with no label selector cannot have its pods resolved. Listing with an empty
// selector would match every pod in the namespace, so the sync has to bail out instead.
func TestSyncWorkloadSkipsWorkloadWithoutLabelSelector(t *testing.T) {
	ctx := newInjectionTestContext()
	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "db", Namespace: injectionNamespace},
	}
	c := newInjectionTestClient(t,
		newEffectiveConfigMap(""),
		statefulSet,
		newInjectionInstrumentationConfig("db", k8sconsts.WorkloadKindStatefulSet),
		newInjectionPod(injectionNamespace, "unrelated", injectionStaleHash),
	)

	require.NoError(t, syncWorkload(ctx, c, k8sconsts.PodWorkload{
		Namespace: injectionNamespace, Name: "db", Kind: k8sconsts.WorkloadKindStatefulSet,
	}))

	ic := syncedInstrumentationConfig(t, ctx, c, "db", k8sconsts.WorkloadKindStatefulSet)
	assert.Nil(t, ic.Status.PodsManifestInjectionStatus)
	assert.Empty(t, injectionConditionReason(ic))
}

// ****************
// Static pods
// ****************

func TestSyncWorkloadStaticPodUsesTheWorkloadObjectItself(t *testing.T) {
	ctx := newInjectionTestContext()
	staticPod := newInjectionPod(injectionNamespace, "static-app", injectionCurrentHash)
	c := newInjectionTestClient(t,
		newEffectiveConfigMap(""),
		staticPod,
		newInjectionInstrumentationConfig("static-app", k8sconsts.WorkloadKindStaticPod),
		// a static pod is not selected by labels, so this pod must never be considered
		newInjectionPod(injectionNamespace, "other-static-app", injectionStaleHash),
	)

	require.NoError(t, syncWorkload(ctx, c, k8sconsts.PodWorkload{
		Namespace: injectionNamespace, Name: "static-app", Kind: k8sconsts.WorkloadKindStaticPod,
	}))

	ic := syncedInstrumentationConfig(t, ctx, c, "static-app", k8sconsts.WorkloadKindStaticPod)
	requireInjectionStatus(t, ic, true, false, false)
}

// ****************
// Missing objects
// ****************

func TestSyncWorkloadIgnoresMissingObjects(t *testing.T) {
	t.Run("no instrumentation config", func(t *testing.T) {
		ctx := newInjectionTestContext()
		c := newInjectionTestClient(t,
			newEffectiveConfigMap(""),
			newInjectionDeployment("web", map[string]string{"app": "web"}),
		)

		require.NoError(t, syncWorkload(ctx, c, k8sconsts.PodWorkload{
			Namespace: injectionNamespace, Name: "web", Kind: k8sconsts.WorkloadKindDeployment,
		}))
	})

	t.Run("no workload object", func(t *testing.T) {
		ctx := newInjectionTestContext()
		c := newInjectionTestClient(t,
			newEffectiveConfigMap(""),
			newInjectionInstrumentationConfig("web", k8sconsts.WorkloadKindDeployment),
			withPodLabel(newInjectionPod(injectionNamespace, "web-1", injectionStaleHash), "app", "web"),
		)

		require.NoError(t, syncWorkload(ctx, c, k8sconsts.PodWorkload{
			Namespace: injectionNamespace, Name: "web", Kind: k8sconsts.WorkloadKindDeployment,
		}))

		ic := syncedInstrumentationConfig(t, ctx, c, "web", k8sconsts.WorkloadKindDeployment)
		assert.Nil(t, ic.Status.PodsManifestInjectionStatus)
	})
}

// ****************
// Effective configuration plumbing
// ****************

// The reason depends on the cluster-wide rollout configuration, which is read from the odigos
// effective config ConfigMap on every sync.
func TestSyncWorkloadReadsRolloutConfigurationFromEffectiveConfig(t *testing.T) {
	ctx := newInjectionTestContext()
	selector := map[string]string{"app": "web"}
	c := newInjectionTestClient(t,
		newEffectiveConfigMap("rollout:\n  automaticRolloutDisabled: true\n"),
		newInjectionDeployment("web", selector),
		newInjectionInstrumentationConfig("web", k8sconsts.WorkloadKindDeployment),
		withPodLabel(newInjectionPod(injectionNamespace, "web-1", ""), "app", "web"),
	)

	require.NoError(t, syncWorkload(ctx, c, k8sconsts.PodWorkload{
		Namespace: injectionNamespace, Name: "web", Kind: k8sconsts.WorkloadKindDeployment,
	}))

	ic := syncedInstrumentationConfig(t, ctx, c, "web", k8sconsts.WorkloadKindDeployment)
	assert.Equal(t,
		string(podsManifestInjection.PodsManifestInjectionReasonRestartRequiredAutoRolloutDisabled_Enabled),
		injectionConditionReason(ic))
}

// ****************
// Status update gating
// ****************

// The controller reconciles on every pod event, so a sync that changes nothing must not issue a
// write. Otherwise each write re-triggers the watch and the controller spins against the API server.
func TestSyncWorkloadOnlyWritesStatusWhenItChanged(t *testing.T) {
	ctx := newInjectionTestContext()
	selector := map[string]string{"app": "web"}
	statusUpdates := 0
	c := interceptor.NewClient(newInjectionTestClient(t,
		newEffectiveConfigMap(""),
		newInjectionDeployment("web", selector),
		newInjectionInstrumentationConfig("web", k8sconsts.WorkloadKindDeployment),
		withPodLabel(newInjectionPod(injectionNamespace, "web-1", injectionCurrentHash), "app", "web"),
	), interceptor.Funcs{
		SubResourceUpdate: func(ctx context.Context, cl client.Client, subResourceName string,
			obj client.Object, opts ...client.SubResourceUpdateOption) error {
			statusUpdates++
			return cl.SubResource(subResourceName).Update(ctx, obj, opts...)
		},
	})

	pw := k8sconsts.PodWorkload{
		Namespace: injectionNamespace, Name: "web", Kind: k8sconsts.WorkloadKindDeployment,
	}

	require.NoError(t, syncWorkload(ctx, c, pw))
	require.Equal(t, 1, statusUpdates, "the first sync must persist the observed status")

	require.NoError(t, syncWorkload(ctx, c, pw))
	assert.Equal(t, 1, statusUpdates, "a sync that observes no change must not write")

	ic := syncedInstrumentationConfig(t, ctx, c, "web", k8sconsts.WorkloadKindDeployment)
	requireInjectionStatus(t, ic, true, false, false)
	assert.Equal(t,
		string(podsManifestInjection.PodsManifestInjectionReasonPodsAppliedSuccessfully_Enabled),
		injectionConditionReason(ic))
}

func TestSyncWorkloadWritesStatusWhenPodsChange(t *testing.T) {
	ctx := newInjectionTestContext()
	selector := map[string]string{"app": "web"}
	inner := newInjectionTestClient(t,
		newEffectiveConfigMap(""),
		newInjectionDeployment("web", selector),
		newInjectionInstrumentationConfig("web", k8sconsts.WorkloadKindDeployment),
		withPodLabel(newInjectionPod(injectionNamespace, "web-1", injectionCurrentHash), "app", "web"),
	)
	statusUpdates := 0
	c := interceptor.NewClient(inner, interceptor.Funcs{
		SubResourceUpdate: func(ctx context.Context, cl client.Client, subResourceName string,
			obj client.Object, opts ...client.SubResourceUpdateOption) error {
			statusUpdates++
			return cl.SubResource(subResourceName).Update(ctx, obj, opts...)
		},
	})

	pw := k8sconsts.PodWorkload{
		Namespace: injectionNamespace, Name: "web", Kind: k8sconsts.WorkloadKindDeployment,
	}
	require.NoError(t, syncWorkload(ctx, c, pw))
	require.Equal(t, 1, statusUpdates)

	require.NoError(t, inner.Create(ctx,
		withPodLabel(newInjectionPod(injectionNamespace, "web-2", ""), "app", "web")))

	require.NoError(t, syncWorkload(ctx, c, pw))
	assert.Equal(t, 2, statusUpdates, "a newly uninjected pod must be persisted")

	ic := syncedInstrumentationConfig(t, ctx, c, "web", k8sconsts.WorkloadKindDeployment)
	requireInjectionStatus(t, ic, true, false, true)
}

// ****************
// podsManifestInjectionStatusNeedsUpdate
// ****************

func TestPodsManifestInjectionStatusNeedsUpdate(t *testing.T) {
	desired := odigosv1.PodsManifestInjectionStatus{
		HasInjectedUpToDatePods:  true,
		HasInjectedOutOfDatePods: false,
		HasUninjectedPods:        true,
	}

	tests := []struct {
		name     string
		current  *odigosv1.PodsManifestInjectionStatus
		expected bool
	}{
		{
			name:     "never persisted",
			current:  nil,
			expected: true,
		},
		{
			name:     "identical",
			current:  desired.DeepCopy(),
			expected: false,
		},
		{
			name: "up to date flag differs",
			current: &odigosv1.PodsManifestInjectionStatus{
				HasInjectedUpToDatePods: false, HasUninjectedPods: true,
			},
			expected: true,
		},
		{
			name: "out of date flag differs",
			current: &odigosv1.PodsManifestInjectionStatus{
				HasInjectedUpToDatePods: true, HasInjectedOutOfDatePods: true, HasUninjectedPods: true,
			},
			expected: true,
		},
		{
			name: "uninjected flag differs",
			current: &odigosv1.PodsManifestInjectionStatus{
				HasInjectedUpToDatePods: true, HasUninjectedPods: false,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, podsManifestInjectionStatusNeedsUpdate(tt.current, desired))
		})
	}
}
