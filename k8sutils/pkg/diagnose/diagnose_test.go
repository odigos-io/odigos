package diagnose

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8stesting "k8s.io/client-go/testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
)

// collectStages drains onStageComplete until RunDiagnose returns and every result has
// been read, the way both the CLI and the UI consume it.
func dgRunDiagnose(t *testing.T, client *dgClients, opts Options) ([]Stage, *dgBuilder, error) {
	t.Helper()

	builder := newDgBuilder()
	results := make(chan StageResult)
	var stages []Stage
	var drained sync.WaitGroup
	drained.Add(1)
	go func() {
		defer drained.Done()
		for r := range results {
			stages = append(stages, r.Stage)
		}
	}()

	err := RunDiagnose(context.Background(), client.kube, client.dynamic, client.kube.Discovery(), client.odigos, builder, dgRootDir, opts, results)
	close(results)
	drained.Wait()

	sort.Slice(stages, func(i, j int) bool { return stages[i] < stages[j] })
	return stages, builder, err
}

// The UI sizes its progress bar from RequestedStages and ticks it once per StageResult,
// and the CLI prints one line per requested stage and updates it in place by index.
// A stage that runs without being announced overshoots the bar and is dropped by the
// CLI; an announced stage that never runs leaves both stuck below 100% forever.
func TestEveryOptionCombinationRunsExactlyTheStagesItAnnounced(t *testing.T) {
	toggles := []struct {
		name string
		set  func(*Options)
	}{
		{"CRDs", func(o *Options) { o.IncludeCRDs = true }},
		{"Profiles", func(o *Options) { o.IncludeProfiles = true }},
		{"Metrics", func(o *Options) { o.IncludeMetrics = true }},
		{"ConfigMaps", func(o *Options) { o.IncludeConfigMaps = true }},
		{"SourceWorkloads", func(o *Options) { o.IncludeSourceWorkloads = true }},
	}

	for mask := 0; mask < 1<<len(toggles); mask++ {
		opts := Options{OdigosNamespace: dgNamespace}
		var enabled []string
		for i, toggle := range toggles {
			if mask&(1<<i) != 0 {
				toggle.set(&opts)
				enabled = append(enabled, toggle.name)
			}
		}
		name := "none"
		if len(enabled) > 0 {
			name = strings.Join(enabled, "+")
		}

		t.Run(name, func(t *testing.T) {
			announced := RequestedStages(opts)
			sorted := append([]Stage(nil), announced...)
			sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

			reported, _, err := dgRunDiagnose(t, dgEmptyCluster(), opts)
			require.NoError(t, err)
			assert.Equal(t, sorted, reported)
		})
	}
}

func TestRequestedStagesAreUniqueAndAlwaysLedByWorkloads(t *testing.T) {
	for _, opts := range []Options{
		{OdigosNamespace: dgNamespace},
		DefaultOptions(),
		{
			OdigosNamespace: dgNamespace, IncludeCRDs: true, IncludeProfiles: true,
			IncludeMetrics: true, IncludeConfigMaps: true, IncludeSourceWorkloads: true,
		},
	} {
		stages := RequestedStages(opts)
		require.NotEmpty(t, stages)
		// The CLI maps stage -> line index, so a repeated stage would overwrite a line.
		assert.Len(t, dgUniqueStages(stages), len(stages))
		assert.Equal(t, StageWorkloads, stages[0], "odigos component workloads are always collected")
	}
}

func TestRunDiagnoseCollectsComponentWorkloadsEvenWithEveryToggleOff(t *testing.T) {
	cluster := dgEmptyCluster()
	cluster.kube = dgClientset(
		dgDeployment(dgNamespace, "odigos-ui", map[string]string{"app": "odigos-ui"}),
		dgPod(dgNamespace, "odigos-ui-1", map[string]string{"app": "odigos-ui"}, "ui"),
	)

	stages, builder, err := dgRunDiagnose(t, cluster, Options{OdigosNamespace: dgNamespace})

	require.NoError(t, err)
	assert.Equal(t, []Stage{StageWorkloads}, stages)
	assert.Equal(t, []string{
		dgRootDir + "/odigos-system/deployment-odigos-ui/deployment-odigos-ui.yaml",
		dgRootDir + "/odigos-system/deployment-odigos-ui/pod-odigos-ui-1.yaml",
	}, builder.paths())
}

func TestRunDiagnoseRejectsAnEmptyOdigosNamespace(t *testing.T) {
	cluster := dgEmptyCluster()
	builder := newDgBuilder()

	err := RunDiagnose(context.Background(), cluster.kube, cluster.dynamic, cluster.kube.Discovery(),
		cluster.odigos, builder, dgRootDir, DefaultOptions(), nil)

	require.ErrorContains(t, err, "odigos namespace is required")
	assert.Empty(t, builder.paths(), "nothing may be collected without knowing which namespace odigos runs in")
}

// The UI's dry run passes a nil channel.
func TestRunDiagnoseWithoutAStageChannelStillCollects(t *testing.T) {
	cluster := dgEmptyCluster()
	cluster.kube = dgClientset(
		dgDeployment(dgNamespace, "odigos-ui", map[string]string{"app": "odigos-ui"}),
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Namespace: dgNamespace, Name: "odigos-config"}},
	)
	builder := newDgBuilder()

	opts := Options{OdigosNamespace: dgNamespace, IncludeConfigMaps: true}
	require.NoError(t, RunDiagnose(context.Background(), cluster.kube, cluster.dynamic,
		cluster.kube.Discovery(), cluster.odigos, builder, dgRootDir, opts, nil))

	assert.Contains(t, builder.paths(), dgRootDir+"/odigos-system/ConfigMaps/configmap-odigos-config.yaml")
}

func TestRunStageReportsTheStageErrorVerbatim(t *testing.T) {
	stageErr := fmt.Errorf("failed to list configmaps: forbidden")
	results := make(chan StageResult, 1)
	var wg sync.WaitGroup

	runStage(&wg, StageConfigMaps, results, func() error { return stageErr })
	wg.Wait()

	result := <-results
	assert.Equal(t, StageConfigMaps, result.Stage)
	// The UI renders result.Status.Error() in the failure notification.
	require.ErrorIs(t, result.Status, stageErr)
}

func TestRunStageReportsSuccessAsANilStatus(t *testing.T) {
	results := make(chan StageResult, 1)
	var wg sync.WaitGroup

	runStage(&wg, StageMetrics, results, func() error { return nil })
	wg.Wait()

	assert.Equal(t, StageResult{Stage: StageMetrics}, <-results)
}

func TestRunStageWithoutAChannelWaitsForTheFetchAnyway(t *testing.T) {
	var wg sync.WaitGroup
	finished := false

	runStage(&wg, StageCRDs, nil, func() error {
		time.Sleep(10 * time.Millisecond)
		finished = true
		return nil
	})
	wg.Wait()

	assert.True(t, finished, "RunDiagnose must not return before its stages complete")
}

// A failing stage must not stop the other stages: a support bundle collected while one
// CRD group is forbidden is still worth having.
func TestRunDiagnoseKeepsCollectingWhenAStageFails(t *testing.T) {
	cluster := dgEmptyCluster()
	cluster.kube = dgClientset(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Namespace: dgNamespace, Name: "odigos-config"},
	})
	cluster.kube.PrependReactor("list", "configmaps", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: "configmaps"}, "", fmt.Errorf("nope"))
	})

	builder := newDgBuilder()
	results := make(chan StageResult, 8)
	opts := Options{OdigosNamespace: dgNamespace, IncludeConfigMaps: true}

	require.NoError(t, RunDiagnose(context.Background(), cluster.kube, cluster.dynamic,
		cluster.kube.Discovery(), cluster.odigos, builder, dgRootDir, opts, results))
	close(results)

	byStage := map[Stage]error{}
	for r := range results {
		byStage[r.Stage] = r.Status
	}
	require.Len(t, byStage, 2)
	assert.NoError(t, byStage[StageWorkloads])
	require.Error(t, byStage[StageConfigMaps])
	assert.True(t, apierrors.IsForbidden(byStage[StageConfigMaps]), "the reported status must stay recognisable as a permission problem")
}

// Source workloads are deliberately collected without logs even when IncludeLogs is on:
// streaming logs from every instrumented application saturates the shared client-go
// rate limiter and cancels the in-flight profile and metrics lists.
func TestRunDiagnoseNeverCollectsLogsFromInstrumentedApplications(t *testing.T) {
	cluster := dgEmptyCluster()
	cluster.kube = dgClientset(
		dgDeployment(dgNamespace, "odigos-ui", map[string]string{"app": "odigos-ui"}),
		dgPod(dgNamespace, "odigos-ui-1", map[string]string{"app": "odigos-ui"}, "ui"),
		dgDeployment(dgAppNs, "checkout", map[string]string{"app": "checkout"}),
		dgPod(dgAppNs, "checkout-1", map[string]string{"app": "checkout"}, "server"),
	)
	cluster.odigos = dgOdigosClient(dgSource(dgAppNs, k8sconsts.WorkloadKindDeployment, dgAppNs, "checkout"))

	opts := Options{OdigosNamespace: dgNamespace, IncludeLogs: true, IncludeSourceWorkloads: true}
	_, builder, err := dgRunDiagnose(t, cluster, opts)
	require.NoError(t, err)

	assert.Contains(t, builder.paths(), dgRootDir+"/shop/deployment-checkout/pod-checkout-1.yaml",
		"the instrumented workload's manifests are still collected")
	assert.Equal(t, []string{dgRootDir + "/odigos-system/deployment-odigos-ui/pod-odigos-ui-1.ui.log.gz"},
		builder.gzippedPaths(), "only odigos component logs belong in the bundle")
}

// One full collection against a cluster that looks like a real odigos install. This is
// the layout a support engineer navigates in the extracted tar.gz, and it is the only
// place where every stage's target directory is checked against the stage that fills it.
func TestRunDiagnoseProducesTheWholeBundleLayout(t *testing.T) {
	odigletPod := dgPod(dgNamespace, "odiglet-1", map[string]string{
		"app.kubernetes.io/name":           k8sconsts.OdigletAppLabelValue,
		k8sconsts.OdigosCollectorRoleLabel: string(k8sconsts.CollectorsRoleNodeCollector),
	}, k8sconsts.OdigletContainerName, k8sconsts.OdigosNodeCollectorContainerName)
	gatewayPod := dgPod(dgNamespace, "odigos-gateway-1", map[string]string{
		"app":                              "odigos-gateway",
		k8sconsts.OdigosCollectorRoleLabel: string(k8sconsts.CollectorsRoleClusterGateway),
	}, "gateway")
	uiPod := dgPod(dgNamespace, "odigos-ui-1", map[string]string{"app": k8sconsts.UIAppLabelValue}, "ui")

	cluster := &dgCluster{
		pods: []corev1.Pod{
			*odigletPod, *gatewayPod, *uiPod,
			*dgPod(dgAppNs, "checkout-1", map[string]string{"app": "checkout"}, "server"),
		},
		deployments: []appsv1.Deployment{
			*dgDeployment(dgNamespace, "odigos-ui", map[string]string{"app": k8sconsts.UIAppLabelValue}),
			*dgDeployment(dgNamespace, "odigos-gateway", map[string]string{"app": "odigos-gateway"}),
			*dgDeployment(dgAppNs, "checkout", map[string]string{"app": "checkout"}),
		},
		daemonsets: []appsv1.DaemonSet{
			*dgDaemonSet(dgNamespace, "odiglet", map[string]string{"app.kubernetes.io/name": k8sconsts.OdigletAppLabelValue}),
		},
		configMaps: []corev1.ConfigMap{
			{ObjectMeta: metav1.ObjectMeta{Namespace: dgNamespace, Name: "odigos-config"}},
		},
	}

	client := dgRESTClientset(t, cluster.handler(t))
	dynamicClient := dgDynamicClient(dgCRDObject("odigos.io/v1alpha1", "Destination", "", "jaeger"))
	discoveryClient := &dgDiscovery{lists: []*metav1.APIResourceList{
		dgResourceList("odigos.io/v1alpha1", "destinations"),
	}}
	odigosClient := dgOdigosClient(dgSource(dgAppNs, k8sconsts.WorkloadKindDeployment, dgAppNs, "checkout"))

	opts := DefaultOptions()
	opts.OdigosNamespace = dgNamespace
	opts.IncludeSourceWorkloads = true
	builder := newDgBuilder()
	// Unbuffered, drained concurrently, the way both callers consume the channel.
	results := make(chan StageResult)
	var failures []string
	var drained sync.WaitGroup
	drained.Add(1)
	go func() {
		defer drained.Done()
		for result := range results {
			if result.Status != nil {
				failures = append(failures, string(result.Stage)+": "+result.Status.Error())
			}
		}
	}()

	require.NoError(t, RunDiagnose(context.Background(), client, dynamicClient, discoveryClient,
		odigosClient, builder, dgRootDir, opts, results))
	close(results)
	drained.Wait()
	require.Empty(t, failures)

	assert.Equal(t, []string{
		// The odigos components, each with its pods and their logs.
		dgRootDir + "/odigos-system/ConfigMaps/configmap-odigos-config.yaml",
		dgRootDir + "/odigos-system/Destinations/jaeger.yaml",
		dgRootDir + "/odigos-system/Metrics/odiglet-1-data-collection",
		dgRootDir + "/odigos-system/Metrics/odiglet-1-odiglet",
		dgRootDir + "/odigos-system/Metrics/odigos-gateway-1",
		dgRootDir + "/odigos-system/Profile/odiglet-1-node-a-data-collection/allocs_profile.prof",
		dgRootDir + "/odigos-system/Profile/odiglet-1-node-a-data-collection/cpu_profile.prof",
		dgRootDir + "/odigos-system/Profile/odiglet-1-node-a-data-collection/goroutine_profile.prof",
		dgRootDir + "/odigos-system/Profile/odiglet-1-node-a-data-collection/heap_profile.prof",
		dgRootDir + "/odigos-system/Profile/odiglet-1-node-a-odiglet/allocs_profile.prof",
		dgRootDir + "/odigos-system/Profile/odiglet-1-node-a-odiglet/cpu_profile.prof",
		dgRootDir + "/odigos-system/Profile/odiglet-1-node-a-odiglet/goroutine_profile.prof",
		dgRootDir + "/odigos-system/Profile/odiglet-1-node-a-odiglet/heap_profile.prof",
		dgRootDir + "/odigos-system/Profile/odigos-gateway-1-node-a-gateway/allocs_profile.prof",
		dgRootDir + "/odigos-system/Profile/odigos-gateway-1-node-a-gateway/cpu_profile.prof",
		dgRootDir + "/odigos-system/Profile/odigos-gateway-1-node-a-gateway/goroutine_profile.prof",
		dgRootDir + "/odigos-system/Profile/odigos-gateway-1-node-a-gateway/heap_profile.prof",
		dgRootDir + "/odigos-system/Profile/odigos-ui-1-node-a-ui/allocs_profile.prof",
		dgRootDir + "/odigos-system/Profile/odigos-ui-1-node-a-ui/cpu_profile.prof",
		dgRootDir + "/odigos-system/Profile/odigos-ui-1-node-a-ui/goroutine_profile.prof",
		dgRootDir + "/odigos-system/Profile/odigos-ui-1-node-a-ui/heap_profile.prof",
		dgRootDir + "/odigos-system/daemonset-odiglet/daemonset-odiglet.yaml",
		dgRootDir + "/odigos-system/daemonset-odiglet/pod-odiglet-1.data-collection.log.gz",
		dgRootDir + "/odigos-system/daemonset-odiglet/pod-odiglet-1.odiglet.log.gz",
		dgRootDir + "/odigos-system/daemonset-odiglet/pod-odiglet-1.yaml",
		dgRootDir + "/odigos-system/deployment-odigos-gateway/deployment-odigos-gateway.yaml",
		dgRootDir + "/odigos-system/deployment-odigos-gateway/pod-odigos-gateway-1.gateway.log.gz",
		dgRootDir + "/odigos-system/deployment-odigos-gateway/pod-odigos-gateway-1.yaml",
		dgRootDir + "/odigos-system/deployment-odigos-ui/deployment-odigos-ui.yaml",
		dgRootDir + "/odigos-system/deployment-odigos-ui/pod-odigos-ui-1.ui.log.gz",
		dgRootDir + "/odigos-system/deployment-odigos-ui/pod-odigos-ui-1.yaml",
		// The instrumented application, under its own namespace and without its logs.
		dgRootDir + "/shop/deployment-checkout/deployment-checkout.yaml",
		dgRootDir + "/shop/deployment-checkout/pod-checkout-1.yaml",
	}, builder.paths())
}

// Each profiled service is its own stage so the progress UI moves while pprof is being
// collected. A service that fails must be reported as that service's stage, not as a
// single opaque "profiles" failure.
func TestRunDiagnoseReportsAFailedProfileServiceAgainstThatServiceAlone(t *testing.T) {
	cluster := dgEmptyCluster()
	cluster.kube = dgClientset(
		dgPod(dgNamespace, "odigos-ui-1", map[string]string{"app": k8sconsts.UIAppLabelValue}, "ui"),
	)
	cluster.kube.PrependReactor("list", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		listAction, ok := action.(k8stesting.ListActionImpl)
		if ok && listAction.GetListRestrictions().Labels.String() == "app=odigos-ui" {
			return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: "pods"}, "", fmt.Errorf("no access"))
		}
		return false, nil, nil
	})

	builder := newDgBuilder()
	results := make(chan StageResult, 8)
	opts := Options{OdigosNamespace: dgNamespace, IncludeProfiles: true}

	require.NoError(t, RunDiagnose(context.Background(), cluster.kube, cluster.dynamic,
		cluster.kube.Discovery(), cluster.odigos, builder, dgRootDir, opts, results))
	close(results)

	failed := map[Stage]string{}
	succeeded := 0
	for result := range results {
		if result.Status != nil {
			failed[result.Stage] = result.Status.Error()
			continue
		}
		succeeded++
	}

	require.Len(t, failed, 1)
	assert.Contains(t, failed[StageProfileService("ui")], "failed to list pods for service ui")
	assert.Equal(t, len(RequestedStages(opts))-1, succeeded, "every other stage still reports success")
}

func dgUniqueStages(stages []Stage) []Stage {
	seen := map[Stage]bool{}
	var out []Stage
	for _, s := range stages {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
