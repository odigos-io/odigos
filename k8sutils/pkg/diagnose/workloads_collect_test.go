package diagnose

import (
	"context"
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8stesting "k8s.io/client-go/testing"
	"sigs.k8s.io/yaml"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
)

// dgCollectableKinds is every kind collectWorkload knows how to collect.
var dgCollectableKinds = []k8sconsts.WorkloadKind{
	k8sconsts.WorkloadKindDeployment,
	k8sconsts.WorkloadKindDaemonSet,
	k8sconsts.WorkloadKindStatefulSet,
	k8sconsts.WorkloadKindCronJob,
	k8sconsts.WorkloadKindJob,
	k8sconsts.WorkloadKindDeploymentConfig,
	k8sconsts.WorkloadKindArgoRollout,
}

// Each supported kind is read through its own client call and its own selector shape.
// A kind wired to the wrong getter, or to the wrong selector field, silently produces a
// bundle with the manifest but none of the pods that explain the problem.
func TestCollectWorkloadCollectsTheManifestAndOnlyItsOwnPodsForEveryKind(t *testing.T) {
	for _, tc := range []struct {
		kind         k8sconsts.WorkloadKind
		name         string
		selectorVal  string
		expectedFile string
	}{
		{k8sconsts.WorkloadKindDeployment, "checkout", "checkout-sel", "deployment-checkout.yaml"},
		{k8sconsts.WorkloadKindDaemonSet, "agent", "agent-sel", "daemonset-agent.yaml"},
		{k8sconsts.WorkloadKindStatefulSet, "cart", "cart-sel", "statefulset-cart.yaml"},
		{k8sconsts.WorkloadKindCronJob, "reports", "reports-sel", "cronjob-reports.yaml"},
		{k8sconsts.WorkloadKindJob, "backfill", "backfill-sel", "job-backfill.yaml"},
		{k8sconsts.WorkloadKindDeploymentConfig, "legacy", "legacy-sel", "deploymentconfig-legacy.yaml"},
		{k8sconsts.WorkloadKindArgoRollout, "canary", "canary-sel", "rollout-canary.yaml"},
	} {
		t.Run(string(tc.kind), func(t *testing.T) {
			selector := map[string]string{"app": tc.selectorVal}
			typed := []runtime.Object{
				dgPod(dgAppNs, tc.name+"-pod", selector, "server"),
				dgPod(dgAppNs, "decoy-pod", map[string]string{"app": "someone-else"}, "server"),
			}
			var unstructuredObjs []runtime.Object

			switch tc.kind {
			case k8sconsts.WorkloadKindDeployment:
				typed = append(typed, dgDeployment(dgAppNs, tc.name, selector))
			case k8sconsts.WorkloadKindDaemonSet:
				typed = append(typed, dgDaemonSet(dgAppNs, tc.name, selector))
			case k8sconsts.WorkloadKindStatefulSet:
				typed = append(typed, dgStatefulSet(dgAppNs, tc.name, selector))
			case k8sconsts.WorkloadKindCronJob:
				typed = append(typed, dgCronJob(dgAppNs, tc.name, selector))
			case k8sconsts.WorkloadKindJob:
				typed = append(typed, dgJob(dgAppNs, tc.name, selector))
			case k8sconsts.WorkloadKindDeploymentConfig:
				unstructuredObjs = append(unstructuredObjs, dgDeploymentConfigObject(dgAppNs, tc.name, selector))
			case k8sconsts.WorkloadKindArgoRollout:
				unstructuredObjs = append(unstructuredObjs, dgArgoRolloutObject(dgAppNs, tc.name, selector))
			}

			builder := newDgBuilder()
			dir := GetWorkloadDir(dgRootDir, dgAppNs, "wl")
			err := collectWorkload(context.Background(), dgClientset(typed...), dgDynamicClient(unstructuredObjs...),
				builder, dir, dgAppNs, tc.name, tc.kind, false)

			require.NoError(t, err)
			expected := []string{dir + "/" + tc.expectedFile, dir + "/pod-" + tc.name + "-pod.yaml"}
			sort.Strings(expected)
			assert.Equal(t, expected, builder.paths())
		})
	}
}

// Every kind is read through a different client call, and each has its own error return.
func TestCollectWorkloadReportsAReadFailureForEveryKind(t *testing.T) {
	for _, kind := range dgCollectableKinds {
		t.Run(string(kind), func(t *testing.T) {
			builder := newDgBuilder()

			err := collectWorkload(context.Background(), dgClientset(), dgDynamicClient(), builder,
				"dir", dgAppNs, "vanished", kind, false)

			require.Error(t, err)
			assert.True(t, apierrors.IsNotFound(err), "got %v", err)
			assert.Empty(t, builder.paths())
		})
	}
}

// If the manifest itself cannot be written there is no point listing its pods, and the
// failure must reach the caller rather than leaving a half-collected directory.
func TestCollectWorkloadStopsWhenTheManifestCannotBeWritten(t *testing.T) {
	for _, kind := range dgCollectableKinds {
		t.Run(string(kind), func(t *testing.T) {
			selector := map[string]string{"app": "checkout"}
			client := dgClientset(
				dgDeployment(dgAppNs, "checkout", selector),
				dgDaemonSet(dgAppNs, "checkout", selector),
				dgStatefulSet(dgAppNs, "checkout", selector),
				dgCronJob(dgAppNs, "checkout", selector),
				dgJob(dgAppNs, "checkout", selector),
				dgPod(dgAppNs, "checkout-1", selector, "server"),
			)
			dynamicClient := dgDynamicClient(
				dgDeploymentConfigObject(dgAppNs, "checkout", selector),
				dgArgoRolloutObject(dgAppNs, "checkout", selector),
			)
			builder := newDgBuilder()
			builder.failOn = func(string, string) error { return fmt.Errorf("no space left on device") }

			err := collectWorkload(context.Background(), client, dynamicClient, builder,
				"dir", dgAppNs, "checkout", kind, false)

			assert.ErrorContains(t, err, "no space left on device")
			assert.Empty(t, builder.paths())
		})
	}
}

func TestCollectWorkloadRejectsAKindThatHasNoManifestToCollect(t *testing.T) {
	for _, kind := range []k8sconsts.WorkloadKind{
		k8sconsts.WorkloadKindNamespace,
		k8sconsts.WorkloadKindStaticPod,
		"Pipeline",
	} {
		t.Run(string(kind), func(t *testing.T) {
			builder := newDgBuilder()
			err := collectWorkload(context.Background(), dgClientset(), dgDynamicClient(), builder,
				"dir", dgAppNs, "x", kind, false)

			require.ErrorIs(t, err, workload.ErrKindNotSupported)
			assert.True(t, workload.IsErrorKindNotSupported(err), "callers downgrade this to a debug log")
			assert.Empty(t, builder.paths())
		})
	}
}

// The callers distinguish "the workload is gone" from a real failure, so the NotFound
// must survive as one.
func TestCollectWorkloadReportsAMissingWorkloadAsNotFound(t *testing.T) {
	err := collectWorkload(context.Background(), dgClientset(), dgDynamicClient(), newDgBuilder(),
		"dir", dgAppNs, "vanished", k8sconsts.WorkloadKindDeployment, false)

	require.Error(t, err)
	assert.True(t, apierrors.IsNotFound(err), "got %v", err)
}

// A CronJob or Job without a selector has no way to find its pods. Listing with an
// empty selector would instead pull every pod in the namespace into the bundle.
func TestCollectWorkloadWithoutASelectorCollectsNoPodsAtAll(t *testing.T) {
	for _, tc := range []struct {
		kind         k8sconsts.WorkloadKind
		name         string
		object       runtime.Object
		expectedFile string
	}{
		{k8sconsts.WorkloadKindCronJob, "reports", dgCronJob(dgAppNs, "reports", nil), "cronjob-reports.yaml"},
		{k8sconsts.WorkloadKindJob, "backfill", dgJob(dgAppNs, "backfill", nil), "job-backfill.yaml"},
	} {
		t.Run(string(tc.kind), func(t *testing.T) {
			client := dgClientset(tc.object, dgPod(dgAppNs, "unrelated-pod", map[string]string{"app": "other"}, "c"))
			builder := newDgBuilder()

			require.NoError(t, collectWorkload(context.Background(), client, dgDynamicClient(), builder,
				"dir", dgAppNs, tc.name, tc.kind, false))

			assert.Equal(t, []string{"dir/" + tc.expectedFile}, builder.paths())
		})
	}
}

func TestExtractSelectorFromUnstructured(t *testing.T) {
	// A DeploymentConfig's selector is a flat map; an Argo Rollout nests it under
	// matchLabels. Reading the wrong shape yields no selector and no pods.
	deploymentConfigSpec := map[string]interface{}{
		"spec": map[string]interface{}{"selector": map[string]interface{}{"app": "legacy"}},
	}
	argoRolloutSpec := map[string]interface{}{
		"spec": map[string]interface{}{"selector": map[string]interface{}{
			"matchLabels": map[string]interface{}{"app": "canary"},
		}},
	}

	for _, tc := range []struct {
		name           string
		obj            map[string]interface{}
		useMatchLabels bool
		expected       string
	}{
		{"deploymentconfig flat selector", deploymentConfigSpec, false, "app=legacy"},
		{"deploymentconfig read as matchLabels", deploymentConfigSpec, true, ""},
		{"argo rollout matchLabels", argoRolloutSpec, true, "app=canary"},
		{"argo rollout read as a flat selector", argoRolloutSpec, false, ""},
		{"no spec", map[string]interface{}{"kind": "Rollout"}, true, ""},
		{"spec is not an object", map[string]interface{}{"spec": "oops"}, true, ""},
		{"no selector", map[string]interface{}{"spec": map[string]interface{}{"replicas": int64(2)}}, true, ""},
		{"empty selector", map[string]interface{}{"spec": map[string]interface{}{"selector": map[string]interface{}{}}}, false, ""},
		{
			"non-string label values are dropped",
			map[string]interface{}{"spec": map[string]interface{}{"selector": map[string]interface{}{
				"app":      "legacy",
				"replicas": int64(3),
			}}},
			false,
			"app=legacy",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			selector := extractSelectorFromUnstructured(tc.obj, tc.useMatchLabels)
			if tc.expected == "" {
				assert.Nil(t, selector)
				return
			}
			require.NotNil(t, selector)
			assert.Equal(t, tc.expected, selector.String())
		})
	}
}

// The "argo rollout read as a flat selector" case above returns nil only because the
// nested map holds no string values; this proves the two shapes cannot be confused even
// when both are present on one object.
func TestExtractSelectorFromUnstructuredPrefersTheRequestedShape(t *testing.T) {
	obj := map[string]interface{}{"spec": map[string]interface{}{"selector": map[string]interface{}{
		"app":         "flat",
		"matchLabels": map[string]interface{}{"app": "nested"},
	}}}

	assert.Equal(t, "app=flat", extractSelectorFromUnstructured(obj, false).String())
	assert.Equal(t, "app=nested", extractSelectorFromUnstructured(obj, true).String())
}

func TestCollectedObjectsLoseTheirManagedFieldsAndTheStoredObjectIsNotMutated(t *testing.T) {
	for name, obj := range map[string]interface{}{
		"deployment":  dgDeployment(dgAppNs, "checkout", map[string]string{"app": "a"}),
		"daemonset":   dgDaemonSet(dgAppNs, "agent", map[string]string{"app": "a"}),
		"statefulset": dgStatefulSet(dgAppNs, "cart", map[string]string{"app": "a"}),
		"cronjob":     dgCronJob(dgAppNs, "reports", map[string]string{"app": "a"}),
		"job":         dgJob(dgAppNs, "backfill", map[string]string{"app": "a"}),
		"pod":         dgPod(dgAppNs, "checkout-1", map[string]string{"app": "a"}, "server"),
		"configmap": &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Namespace: dgAppNs, Name: "cm", ManagedFields: dgManagedFields()},
		},
	} {
		t.Run(name, func(t *testing.T) {
			cleaned := cleanObjectForExport(obj)

			accessor, err := meta.Accessor(cleaned)
			require.NoError(t, err)
			assert.Empty(t, accessor.GetManagedFields(), "managedFields is apiserver noise that bloats every bundle")

			original, err := meta.Accessor(obj)
			require.NoError(t, err)
			assert.NotEmpty(t, original.GetManagedFields(), "the cached object handed in must not be modified")
			assert.NotSame(t, obj, cleaned)
		})
	}
}

// An object of a type cleanObjectForExport does not know is passed through untouched;
// it is only reached for the unstructured kinds, which strip managedFields themselves.
func TestCleanObjectForExportPassesThroughAnUnknownType(t *testing.T) {
	obj := map[string]interface{}{"kind": "Rollout", "metadata": map[string]interface{}{"managedFields": []interface{}{}}}
	assert.Equal(t, obj, cleanObjectForExport(obj))
}

func TestCollectedDeploymentConfigAndRolloutYamlHaveNoManagedFields(t *testing.T) {
	dynamicClient := dgDynamicClient(
		dgDeploymentConfigObject(dgAppNs, "legacy", map[string]string{"app": "legacy"}),
		dgArgoRolloutObject(dgAppNs, "canary", map[string]string{"app": "canary"}),
	)

	for _, tc := range []struct {
		kind k8sconsts.WorkloadKind
		name string
		file string
	}{
		{k8sconsts.WorkloadKindDeploymentConfig, "legacy", "deploymentconfig-legacy.yaml"},
		{k8sconsts.WorkloadKindArgoRollout, "canary", "rollout-canary.yaml"},
	} {
		t.Run(string(tc.kind), func(t *testing.T) {
			builder := newDgBuilder()
			require.NoError(t, collectWorkload(context.Background(), dgClientset(), dynamicClient, builder,
				"dir", dgAppNs, tc.name, tc.kind, false))

			var collected map[string]interface{}
			require.NoError(t, yaml.Unmarshal(builder.body(t, "dir/"+tc.file), &collected))
			metadata := collected["metadata"].(map[string]interface{})
			assert.NotContains(t, metadata, "managedFields")
			assert.Equal(t, tc.name, metadata["name"])
		})
	}
}

func TestAddWorkloadYAMLNamesTheFileAfterItsKindAndWorkload(t *testing.T) {
	builder := newDgBuilder()
	deployment := dgDeployment(dgAppNs, "checkout", map[string]string{"app": "checkout"})

	require.NoError(t, addWorkloadYAML(builder, "bundle/shop/deployment-checkout", "deployment", "checkout", deployment))

	assert.Equal(t, []string{"bundle/shop/deployment-checkout/deployment-checkout.yaml"}, builder.paths())
	var round appsv1.Deployment
	require.NoError(t, yaml.Unmarshal(builder.body(t, "bundle/shop/deployment-checkout/deployment-checkout.yaml"), &round))
	assert.Equal(t, "checkout", round.Name)
	assert.Equal(t, map[string]string{"app": "checkout"}, round.Spec.Selector.MatchLabels)
}

func TestAddWorkloadYAMLReportsWhatItCouldNotMarshal(t *testing.T) {
	err := addWorkloadYAML(newDgBuilder(), "dir", "cronjob", "reports", map[string]interface{}{
		"schedule": func() {},
	})
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to marshal cronjob reports to YAML")
}

func TestAddPodYAMLNamesTheFileAfterThePod(t *testing.T) {
	builder := newDgBuilder()
	pod := dgPod(dgAppNs, "checkout-abc123", map[string]string{"app": "checkout"}, "server")

	require.NoError(t, addPodYAML(builder, "bundle/shop/deployment-checkout", pod))

	assert.Equal(t, []string{"bundle/shop/deployment-checkout/pod-checkout-abc123.yaml"}, builder.paths())
	var round corev1.Pod
	require.NoError(t, yaml.Unmarshal(builder.body(t, "bundle/shop/deployment-checkout/pod-checkout-abc123.yaml"), &round))
	assert.Empty(t, round.ManagedFields)
	assert.Equal(t, pod.ManagedFields, dgManagedFields(), "the pod from the lister must not be mutated")
}

func TestCollectPodsReportsWhyItCouldNotListPods(t *testing.T) {
	client := dgClientset()
	client.PrependReactor("list", "pods", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: "pods"}, "", fmt.Errorf("no access"))
	})

	err := collectPods(context.Background(), client, newDgBuilder(), dgAppNs, "dir",
		labels.SelectorFromSet(labels.Set{"app": "checkout"}), false)

	assert.ErrorContains(t, err, "failed to list pods")
}

func TestCollectPodsKeepsGoingWhenOnePodCannotBeWritten(t *testing.T) {
	client := dgClientset(
		dgPod(dgAppNs, "checkout-1", map[string]string{"app": "checkout"}, "server"),
		dgPod(dgAppNs, "checkout-2", map[string]string{"app": "checkout"}, "server"),
	)
	builder := newDgBuilder()
	builder.failOn = func(_, filename string) error {
		if filename == "pod-checkout-1.yaml" {
			return fmt.Errorf("disk full")
		}
		return nil
	}

	require.NoError(t, collectPods(context.Background(), client, builder, dgAppNs, "dir",
		labels.SelectorFromSet(labels.Set{"app": "checkout"}), false))

	assert.Equal(t, []string{"dir/pod-checkout-2.yaml"}, builder.paths())
}

func TestFetchOdigosWorkloadsCollectsTheComponentDeploymentsAndDaemonSets(t *testing.T) {
	client := dgClientset(
		dgDeployment(dgNamespace, "odigos-ui", map[string]string{"app": "odigos-ui"}),
		dgDaemonSet(dgNamespace, "odiglet", map[string]string{"app.kubernetes.io/name": "odiglet"}),
		dgPod(dgNamespace, "odigos-ui-1", map[string]string{"app": "odigos-ui"}, "ui"),
		dgPod(dgNamespace, "odiglet-1", map[string]string{"app.kubernetes.io/name": "odiglet"}, "odiglet"),
		dgDeployment(dgAppNs, "checkout", map[string]string{"app": "checkout"}),
	)
	builder := newDgBuilder()

	require.NoError(t, FetchOdigosWorkloads(context.Background(), client, dgDynamicClient(), builder, dgRootDir, dgNamespace, false))

	assert.Equal(t, []string{
		dgRootDir + "/odigos-system/daemonset-odiglet/daemonset-odiglet.yaml",
		dgRootDir + "/odigos-system/daemonset-odiglet/pod-odiglet-1.yaml",
		dgRootDir + "/odigos-system/deployment-odigos-ui/deployment-odigos-ui.yaml",
		dgRootDir + "/odigos-system/deployment-odigos-ui/pod-odigos-ui-1.yaml",
	}, builder.paths(), "workloads outside the odigos namespace are collected by the source workloads stage instead")
}

// Known gap, pinned as-is: the odigos namespace sweep lists only Deployments and
// DaemonSets, so a StatefulSet running there (the docstring says it is collected) never
// reaches the bundle.
func TestFetchOdigosWorkloadsSkipsAStatefulSetInTheOdigosNamespace(t *testing.T) {
	client := dgClientset(
		dgStatefulSet(dgNamespace, "odigos-clickhouse", map[string]string{"app": "clickhouse"}),
		dgPod(dgNamespace, "odigos-clickhouse-0", map[string]string{"app": "clickhouse"}, "clickhouse"),
	)
	builder := newDgBuilder()

	require.NoError(t, FetchOdigosWorkloads(context.Background(), client, dgDynamicClient(), builder, dgRootDir, dgNamespace, false))

	assert.Empty(t, builder.paths())
}

func TestFetchOdigosWorkloadsStillCollectsDaemonSetsWhenDeploymentsCannotBeListed(t *testing.T) {
	client := dgClientset(
		dgDeployment(dgNamespace, "odigos-ui", map[string]string{"app": "odigos-ui"}),
		dgDaemonSet(dgNamespace, "odiglet", map[string]string{"app.kubernetes.io/name": "odiglet"}),
	)
	client.PrependReactor("list", "deployments", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: "deployments"}, "", fmt.Errorf("no access"))
	})
	builder := newDgBuilder()

	require.NoError(t, FetchOdigosWorkloads(context.Background(), client, dgDynamicClient(), builder, dgRootDir, dgNamespace, false))

	assert.Equal(t, []string{dgRootDir + "/odigos-system/daemonset-odiglet/daemonset-odiglet.yaml"}, builder.paths())
}

func TestFetchOdigosWorkloadsStillCollectsDeploymentsWhenDaemonSetsCannotBeListed(t *testing.T) {
	client := dgClientset(
		dgDeployment(dgNamespace, "odigos-ui", map[string]string{"app": "odigos-ui"}),
		dgDaemonSet(dgNamespace, "odiglet", map[string]string{"app.kubernetes.io/name": "odiglet"}),
	)
	client.PrependReactor("list", "daemonsets", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: "daemonsets"}, "", fmt.Errorf("no access"))
	})
	builder := newDgBuilder()

	require.NoError(t, FetchOdigosWorkloads(context.Background(), client, dgDynamicClient(), builder, dgRootDir, dgNamespace, false))

	assert.Equal(t, []string{dgRootDir + "/odigos-system/deployment-odigos-ui/deployment-odigos-ui.yaml"}, builder.paths())
}

func TestFetchOdigosWorkloadsKeepsGoingWhenOneComponentCannotBeRead(t *testing.T) {
	client := dgClientset(
		dgDeployment(dgNamespace, "odigos-ui", map[string]string{"app": "odigos-ui"}),
		dgDaemonSet(dgNamespace, "odiglet", map[string]string{"app.kubernetes.io/name": "odiglet"}),
	)
	client.PrependReactor("get", "deployments", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: "deployments"}, "odigos-ui", fmt.Errorf("no access"))
	})
	builder := newDgBuilder()

	require.NoError(t, FetchOdigosWorkloads(context.Background(), client, dgDynamicClient(), builder, dgRootDir, dgNamespace, false))

	assert.Equal(t, []string{dgRootDir + "/odigos-system/daemonset-odiglet/daemonset-odiglet.yaml"}, builder.paths())
}

func TestFetchOdigosWorkloadsCollectsComponentLogsOnlyWhenAsked(t *testing.T) {
	for _, includeLogs := range []bool{false, true} {
		t.Run(fmt.Sprintf("includeLogs=%v", includeLogs), func(t *testing.T) {
			client := dgClientset(
				dgDeployment(dgNamespace, "odigos-ui", map[string]string{"app": "odigos-ui"}),
				dgPod(dgNamespace, "odigos-ui-1", map[string]string{"app": "odigos-ui"}, "ui"),
			)
			builder := newDgBuilder()

			require.NoError(t, FetchOdigosWorkloads(context.Background(), client, dgDynamicClient(), builder, dgRootDir, dgNamespace, includeLogs))

			if includeLogs {
				assert.Equal(t, []string{dgRootDir + "/odigos-system/deployment-odigos-ui/pod-odigos-ui-1.ui.log.gz"}, builder.gzippedPaths())
			} else {
				assert.Empty(t, builder.gzippedPaths())
			}
		})
	}
}

func TestFetchSourceWorkloadsCollectsAWorkloadThatHasItsOwnSource(t *testing.T) {
	client := dgClientset(
		dgDeployment(dgAppNs, "checkout", map[string]string{"app": "checkout"}),
		dgPod(dgAppNs, "checkout-1", map[string]string{"app": "checkout"}, "server"),
		dgDeployment(dgAppNs, "uninstrumented", map[string]string{"app": "uninstrumented"}),
	)
	odigosClient := dgOdigosClient(dgSource(dgAppNs, k8sconsts.WorkloadKindDeployment, dgAppNs, "checkout"))
	builder := newDgBuilder()

	require.NoError(t, FetchSourceWorkloads(context.Background(), client, dgDynamicClient(), odigosClient,
		builder, dgRootDir, nil, false))

	assert.Equal(t, []string{
		dgRootDir + "/shop/deployment-checkout/deployment-checkout.yaml",
		dgRootDir + "/shop/deployment-checkout/pod-checkout-1.yaml",
	}, builder.paths())
}

func TestFetchSourceWorkloadsExpandsANamespaceSourceAndHonoursDisabledWorkloads(t *testing.T) {
	client := dgClientset(
		dgDeployment(dgAppNs, "checkout", map[string]string{"app": "checkout"}),
		dgDeployment(dgAppNs, "opted-out", map[string]string{"app": "opted-out"}),
		dgDaemonSet(dgAppNs, "logger", map[string]string{"app": "logger"}),
		dgStatefulSet(dgAppNs, "cart", map[string]string{"app": "cart"}),
		dgCronJob(dgAppNs, "reports", map[string]string{"app": "reports"}),
	)
	namespaceSource := dgSource(dgAppNs, k8sconsts.WorkloadKindNamespace, dgAppNs, dgAppNs)
	optedOut := dgSource(dgAppNs, k8sconsts.WorkloadKindDeployment, dgAppNs, "opted-out")
	optedOut.Spec.DisableInstrumentation = true
	builder := newDgBuilder()

	require.NoError(t, FetchSourceWorkloads(context.Background(), client, dgDynamicClient(),
		dgOdigosClient(namespaceSource, optedOut), builder, dgRootDir, nil, false))

	assert.Equal(t, []string{
		dgRootDir + "/shop/cronjob-reports/cronjob-reports.yaml",
		dgRootDir + "/shop/daemonset-logger/daemonset-logger.yaml",
		dgRootDir + "/shop/deployment-checkout/deployment-checkout.yaml",
		dgRootDir + "/shop/statefulset-cart/statefulset-cart.yaml",
	}, builder.paths())
}

func TestFetchSourceWorkloadsCollectsAWorkloadCoveredTwiceOnlyOnce(t *testing.T) {
	client := dgClientset(dgDeployment(dgAppNs, "checkout", map[string]string{"app": "checkout"}))
	builder := newDgBuilder()

	require.NoError(t, FetchSourceWorkloads(context.Background(), client, dgDynamicClient(),
		dgOdigosClient(
			dgSource(dgAppNs, k8sconsts.WorkloadKindNamespace, dgAppNs, dgAppNs),
			dgSource(dgAppNs, k8sconsts.WorkloadKindDeployment, dgAppNs, "checkout"),
		), builder, dgRootDir, nil, false))

	assert.Equal(t, 1, builder.count(dgRootDir+"/shop/deployment-checkout/deployment-checkout.yaml"))
}

// Two workloads of different kinds may share a name in one namespace, so the
// already-collected key has to include the kind or the second one is silently dropped.
func TestFetchSourceWorkloadsCollectsTwoKindsThatShareAName(t *testing.T) {
	client := dgClientset(
		dgDeployment(dgAppNs, "checkout", map[string]string{"app": "checkout-deploy"}),
		dgStatefulSet(dgAppNs, "checkout", map[string]string{"app": "checkout-sts"}),
	)
	builder := newDgBuilder()

	require.NoError(t, FetchSourceWorkloads(context.Background(), client, dgDynamicClient(),
		dgOdigosClient(
			dgSource(dgAppNs, k8sconsts.WorkloadKindDeployment, dgAppNs, "checkout"),
			dgSource(dgAppNs, k8sconsts.WorkloadKindStatefulSet, dgAppNs, "checkout"),
		), builder, dgRootDir, nil, false))

	assert.Equal(t, []string{
		dgRootDir + "/shop/deployment-checkout/deployment-checkout.yaml",
		dgRootDir + "/shop/statefulset-checkout/statefulset-checkout.yaml",
	}, builder.paths())
}

func TestFetchSourceWorkloadsOnlyExpandsTheRequestedNamespaces(t *testing.T) {
	client := dgClientset(
		dgDeployment(dgAppNs, "checkout", map[string]string{"app": "checkout"}),
		dgDeployment("payments", "ledger", map[string]string{"app": "ledger"}),
	)
	builder := newDgBuilder()

	require.NoError(t, FetchSourceWorkloads(context.Background(), client, dgDynamicClient(),
		dgOdigosClient(
			dgSource(dgAppNs, k8sconsts.WorkloadKindNamespace, dgAppNs, dgAppNs),
			dgSource("payments", k8sconsts.WorkloadKindNamespace, "payments", "payments"),
		), builder, dgRootDir, []string{dgAppNs}, false))

	assert.Equal(t, []string{dgRootDir + "/shop/deployment-checkout/deployment-checkout.yaml"}, builder.paths())
}

func TestFetchSourceWorkloadsReportsWhyItCouldNotReadTheSources(t *testing.T) {
	err := FetchSourceWorkloads(context.Background(), dgClientset(), dgDynamicClient(),
		dgFailingOdigosClient(), newDgBuilder(), dgRootDir, nil, false)

	assert.ErrorContains(t, err, "failed to list Source CRDs")
}

func TestFetchSourceWorkloadsCollectsLogsOnlyWhenTheCallerAsksForThem(t *testing.T) {
	client := dgClientset(
		dgDeployment(dgAppNs, "checkout", map[string]string{"app": "checkout"}),
		dgPod(dgAppNs, "checkout-1", map[string]string{"app": "checkout"}, "server"),
	)
	builder := newDgBuilder()

	require.NoError(t, FetchSourceWorkloads(context.Background(), client, dgDynamicClient(),
		dgOdigosClient(dgSource(dgAppNs, k8sconsts.WorkloadKindDeployment, dgAppNs, "checkout")),
		builder, dgRootDir, nil, true))

	assert.Equal(t, []string{dgRootDir + "/shop/deployment-checkout/pod-checkout-1.server.log.gz"}, builder.gzippedPaths())
}

// Known gap, pinned as-is: workload.IsValidWorkloadKind has no Job case, so a Source for
// a Job is dropped here even though collectWorkload knows how to collect one.
func TestFetchSourceWorkloadsDropsAJobSource(t *testing.T) {
	client := dgClientset(dgJob(dgAppNs, "backfill", map[string]string{"app": "backfill"}))
	builder := newDgBuilder()

	require.NoError(t, FetchSourceWorkloads(context.Background(), client, dgDynamicClient(),
		dgOdigosClient(dgSource(dgAppNs, k8sconsts.WorkloadKindJob, dgAppNs, "backfill")),
		builder, dgRootDir, nil, false))

	assert.Empty(t, builder.paths())
	assert.False(t, workload.IsValidWorkloadKind(k8sconsts.WorkloadKindJob))
}

func TestFetchSourceWorkloadsKeepsGoingWhenOneWorkloadCannotBeRead(t *testing.T) {
	client := dgClientset(
		dgDeployment(dgAppNs, "checkout", map[string]string{"app": "checkout"}),
		dgStatefulSet(dgAppNs, "cart", map[string]string{"app": "cart"}),
	)
	client.PrependReactor("get", "deployments", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: "deployments"}, "checkout", fmt.Errorf("no access"))
	})
	builder := newDgBuilder()

	require.NoError(t, FetchSourceWorkloads(context.Background(), client, dgDynamicClient(),
		dgOdigosClient(
			dgSource(dgAppNs, k8sconsts.WorkloadKindDeployment, dgAppNs, "checkout"),
			dgSource(dgAppNs, k8sconsts.WorkloadKindStatefulSet, dgAppNs, "cart"),
		), builder, dgRootDir, nil, false))

	assert.Equal(t, []string{dgRootDir + "/shop/statefulset-cart/statefulset-cart.yaml"}, builder.paths())
}

func TestCategorizeSourcesForDiagnoseIgnoresIncompleteAndUnknownWorkloads(t *testing.T) {
	sources := []odigosv1.Source{
		{Spec: odigosv1.SourceSpec{Workload: k8sconsts.PodWorkload{Kind: "", Name: "n", Namespace: "ns"}}},
		{Spec: odigosv1.SourceSpec{Workload: k8sconsts.PodWorkload{Kind: k8sconsts.WorkloadKindDeployment, Name: "", Namespace: "ns"}}},
		{Spec: odigosv1.SourceSpec{Workload: k8sconsts.PodWorkload{Kind: k8sconsts.WorkloadKindDeployment, Name: "n", Namespace: ""}}},
		{Spec: odigosv1.SourceSpec{Workload: k8sconsts.PodWorkload{Kind: "Pipeline", Name: "n", Namespace: "ns"}}},
	}

	plan := categorizeSourcesForDiagnose(sources, nil)

	assert.Empty(t, plan.explicitWorkloads)
	assert.Empty(t, plan.namespaceSources)
	assert.Empty(t, plan.disabledExclusions)
}

func TestCategorizeSourcesForDiagnoseSkipsADisabledNamespaceSource(t *testing.T) {
	disabled := dgSource(dgAppNs, k8sconsts.WorkloadKindNamespace, dgAppNs, dgAppNs)
	disabled.Spec.DisableInstrumentation = true

	plan := categorizeSourcesForDiagnose([]odigosv1.Source{disabled}, nil)

	assert.Empty(t, plan.namespaceSources, "an uninstrumented namespace is not expanded")
	assert.Empty(t, plan.disabledExclusions, "and it is not an exclusion either")
}

func TestCategorizeSourcesForDiagnoseTreatsARegexSourceAsAPatternNotAWorkload(t *testing.T) {
	regexSource := dgSource(dgAppNs, k8sconsts.WorkloadKindDeployment, dgAppNs, "checkout-.*")
	regexSource.Spec.MatchWorkloadNameAsRegex = true
	disabledRegex := dgSource(dgAppNs, k8sconsts.WorkloadKindDeployment, dgAppNs, "legacy-.*")
	disabledRegex.Spec.MatchWorkloadNameAsRegex = true
	disabledRegex.Spec.DisableInstrumentation = true

	plan := categorizeSourcesForDiagnose([]odigosv1.Source{regexSource, disabledRegex}, nil)

	assert.Empty(t, plan.explicitWorkloads, "there is no single object named checkout-.* to GET")
	assert.Equal(t, []disabledWorkloadExclusion{{
		Namespace: dgAppNs, Kind: k8sconsts.WorkloadKindDeployment, Name: "legacy-.*", Regex: true,
	}}, plan.disabledExclusions)
}

func TestIsWorkloadExcludedRequiresTheSameNamespaceAndKind(t *testing.T) {
	exclusions := []disabledWorkloadExclusion{
		{Namespace: dgAppNs, Kind: k8sconsts.WorkloadKindDeployment, Name: "checkout"},
	}

	assert.False(t, isWorkloadExcluded(k8sconsts.PodWorkload{
		Namespace: "payments", Name: "checkout", Kind: k8sconsts.WorkloadKindDeployment,
	}, exclusions), "a same-named workload in another namespace is a different workload")
	assert.False(t, isWorkloadExcluded(k8sconsts.PodWorkload{
		Namespace: dgAppNs, Name: "checkout", Kind: k8sconsts.WorkloadKindStatefulSet,
	}, exclusions), "a same-named workload of another kind is a different workload")
	assert.True(t, isWorkloadExcluded(k8sconsts.PodWorkload{
		Namespace: dgAppNs, Name: "checkout", Kind: k8sconsts.WorkloadKindDeployment,
	}, exclusions))
}

func TestIsWorkloadExcludedIgnoresAnUnparsableRegexAndKeepsChecking(t *testing.T) {
	exclusions := []disabledWorkloadExclusion{
		{Namespace: dgAppNs, Kind: k8sconsts.WorkloadKindDeployment, Name: "checkout-[", Regex: true},
		{Namespace: dgAppNs, Kind: k8sconsts.WorkloadKindDeployment, Name: "checkout", Regex: false},
	}

	assert.True(t, isWorkloadExcluded(k8sconsts.PodWorkload{
		Namespace: dgAppNs, Name: "checkout", Kind: k8sconsts.WorkloadKindDeployment,
	}, exclusions), "one broken pattern must not hide the exclusions after it")
	assert.False(t, isWorkloadExcluded(k8sconsts.PodWorkload{
		Namespace: dgAppNs, Name: "checkout-1", Kind: k8sconsts.WorkloadKindDeployment,
	}, exclusions))
}

// A regex exclusion is not anchored, so it must not be mistaken for an exact match.
func TestIsWorkloadExcludedMatchesARegexAnywhereInTheName(t *testing.T) {
	exclusions := []disabledWorkloadExclusion{
		{Namespace: dgAppNs, Kind: k8sconsts.WorkloadKindDeployment, Name: "^legacy-", Regex: true},
	}

	assert.True(t, isWorkloadExcluded(k8sconsts.PodWorkload{
		Namespace: dgAppNs, Name: "legacy-checkout", Kind: k8sconsts.WorkloadKindDeployment,
	}, exclusions))
	assert.False(t, isWorkloadExcluded(k8sconsts.PodWorkload{
		Namespace: dgAppNs, Name: "new-legacy-checkout", Kind: k8sconsts.WorkloadKindDeployment,
	}, exclusions))
}

func TestListCollectableWorkloadsInNamespaceCoversEveryExpandableKind(t *testing.T) {
	client := dgClientset(
		dgDeployment(dgAppNs, "checkout", nil),
		dgDaemonSet(dgAppNs, "logger", nil),
		dgStatefulSet(dgAppNs, "cart", nil),
		dgCronJob(dgAppNs, "reports", nil),
		dgJob(dgAppNs, "backfill", nil),
		dgDeployment("payments", "ledger", nil),
	)
	dynamicClient := dgDynamicClient(
		dgDeploymentConfigObject(dgAppNs, "legacy", nil),
		dgArgoRolloutObject(dgAppNs, "canary", nil),
	)

	found := listCollectableWorkloadsInNamespace(context.Background(), client, dynamicClient, dgAppNs)

	assert.Equal(t, []k8sconsts.PodWorkload{
		{Namespace: dgAppNs, Name: "checkout", Kind: k8sconsts.WorkloadKindDeployment},
		{Namespace: dgAppNs, Name: "logger", Kind: k8sconsts.WorkloadKindDaemonSet},
		{Namespace: dgAppNs, Name: "cart", Kind: k8sconsts.WorkloadKindStatefulSet},
		{Namespace: dgAppNs, Name: "reports", Kind: k8sconsts.WorkloadKindCronJob},
		{Namespace: dgAppNs, Name: "legacy", Kind: k8sconsts.WorkloadKindDeploymentConfig},
		{Namespace: dgAppNs, Name: "canary", Kind: k8sconsts.WorkloadKindArgoRollout},
	}, found, "Jobs are deliberately absent: a namespace Source does not instrument bare Jobs")
}

// Four kinds are listed one after another, each guarded separately; one denied kind must
// not cost the namespace expansion the other three.
func TestListCollectableWorkloadsInNamespaceKeepsGoingWhenAnyOneKindCannotBeListed(t *testing.T) {
	all := []k8sconsts.PodWorkload{
		{Namespace: dgAppNs, Name: "checkout", Kind: k8sconsts.WorkloadKindDeployment},
		{Namespace: dgAppNs, Name: "logger", Kind: k8sconsts.WorkloadKindDaemonSet},
		{Namespace: dgAppNs, Name: "cart", Kind: k8sconsts.WorkloadKindStatefulSet},
		{Namespace: dgAppNs, Name: "reports", Kind: k8sconsts.WorkloadKindCronJob},
	}

	for i, denied := range []string{"deployments", "daemonsets", "statefulsets", "cronjobs"} {
		t.Run(denied, func(t *testing.T) {
			client := dgClientset(
				dgDeployment(dgAppNs, "checkout", nil),
				dgDaemonSet(dgAppNs, "logger", nil),
				dgStatefulSet(dgAppNs, "cart", nil),
				dgCronJob(dgAppNs, "reports", nil),
			)
			client.PrependReactor("list", denied, func(k8stesting.Action) (bool, runtime.Object, error) {
				return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: denied}, "", fmt.Errorf("no access"))
			})

			var expected []k8sconsts.PodWorkload
			for j, workload := range all {
				if j != i {
					expected = append(expected, workload)
				}
			}
			assert.Equal(t, expected, listCollectableWorkloadsInNamespace(context.Background(), client, dgDynamicClient(), dgAppNs))
		})
	}
}

// Neither OpenShift nor Argo Rollouts is installed on a typical cluster, and the CLI may
// not be allowed to list them; none of that may abort the namespace expansion.
func TestListDynamicWorkloadsInNamespaceToleratesAMissingOrForbiddenCRD(t *testing.T) {
	for name, injected := range map[string]error{
		"crd not installed": apierrors.NewNotFound(schema.GroupResource{Resource: "rollouts"}, ""),
		"no permission":     apierrors.NewForbidden(schema.GroupResource{Resource: "rollouts"}, "", fmt.Errorf("no access")),
		"no rest mapping":   &meta.NoKindMatchError{GroupKind: schema.GroupKind{Group: "argoproj.io", Kind: "Rollout"}},
		"unexpected":        apierrors.NewInternalError(fmt.Errorf("etcd is down")),
	} {
		t.Run(name, func(t *testing.T) {
			dynamicClient := dgDynamicClient(dgArgoRolloutObject(dgAppNs, "canary", nil))
			dynamicClient.PrependReactor("list", "rollouts", func(k8stesting.Action) (bool, runtime.Object, error) {
				return true, nil, injected
			})

			assert.Nil(t, listDynamicWorkloadsInNamespace(context.Background(), dynamicClient, dgAppNs, argoRolloutGVR, k8sconsts.WorkloadKindArgoRollout))
		})
	}
}

func TestListDynamicWorkloadsInNamespaceWithoutADynamicClient(t *testing.T) {
	assert.Nil(t, listDynamicWorkloadsInNamespace(context.Background(), nil, dgAppNs, argoRolloutGVR, k8sconsts.WorkloadKindArgoRollout))
}

func TestWorkloadTargetKindsMapToTheDirectoryNamesSupportEngineersLookFor(t *testing.T) {
	// The folder name is <kind lowercase>-<name>; it is how a support engineer finds a
	// workload in an extracted bundle.
	for kind, expected := range map[k8sconsts.WorkloadKind]string{
		k8sconsts.WorkloadKindDeployment:       "deployment",
		k8sconsts.WorkloadKindDaemonSet:        "daemonset",
		k8sconsts.WorkloadKindStatefulSet:      "statefulset",
		k8sconsts.WorkloadKindCronJob:          "cronjob",
		k8sconsts.WorkloadKindJob:              "job",
		k8sconsts.WorkloadKindDeploymentConfig: "deploymentconfig",
		k8sconsts.WorkloadKindArgoRollout:      "rollout",
	} {
		assert.Equal(t, expected, string(workload.WorkloadKindLowerCaseFromKind(kind)))
	}
}
