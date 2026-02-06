package services

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/kube"

	odigosfake "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/fake"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	versionutil "k8s.io/apimachinery/pkg/version"
	fakediscovery "k8s.io/client-go/discovery/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	crfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const odigosNs = "odigos-system"

func TestMain(m *testing.M) {
	os.Setenv("CURRENT_NS", odigosNs)
	os.Exit(m.Run())
}

// slowReactor adds latency to every API call to simulate a real K8s API server.
func slowReactor(latency time.Duration) k8stesting.ReactionFunc {
	return func(action k8stesting.Action) (bool, runtime.Object, error) {
		time.Sleep(latency)
		return false, nil, nil // pass through to object tracker
	}
}

// --- Data generators ---

func generateNamespaces(count int) []runtime.Object {
	objs := make([]runtime.Object, count)
	for i := 0; i < count; i++ {
		objs[i] = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("ns-%d", i)},
		}
	}
	return objs
}

func generateNamespaceSources(count int) []runtime.Object {
	objs := make([]runtime.Object, count)
	for i := 0; i < count; i++ {
		nsName := fmt.Sprintf("ns-%d", i)
		objs[i] = &odigosv1alpha1.Source{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("source-ns-%d", i),
				Namespace: nsName,
				Labels: map[string]string{
					k8sconsts.WorkloadNamespaceLabel: nsName,
					k8sconsts.WorkloadNameLabel:      nsName,
					k8sconsts.WorkloadKindLabel:      string(k8sconsts.WorkloadKindNamespace),
				},
			},
			Spec: odigosv1alpha1.SourceSpec{
				Workload: k8sconsts.PodWorkload{
					Name:      nsName,
					Namespace: nsName,
					Kind:      k8sconsts.WorkloadKindNamespace,
				},
			},
		}
	}
	return objs
}

func generateDeployments(namespace string, count int) []runtime.Object {
	objs := make([]runtime.Object, count)
	for i := 0; i < count; i++ {
		objs[i] = &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("deploy-%d", i),
				Namespace: namespace,
			},
			Status: appsv1.DeploymentStatus{ReadyReplicas: 2},
		}
	}
	return objs
}

func generateStatefulSets(namespace string, count int) []runtime.Object {
	objs := make([]runtime.Object, count)
	for i := 0; i < count; i++ {
		objs[i] = &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("sts-%d", i),
				Namespace: namespace,
			},
			Status: appsv1.StatefulSetStatus{ReadyReplicas: 1},
		}
	}
	return objs
}

func generateDaemonSets(namespace string, count int) []runtime.Object {
	objs := make([]runtime.Object, count)
	for i := 0; i < count; i++ {
		objs[i] = &appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("ds-%d", i),
				Namespace: namespace,
			},
			Status: appsv1.DaemonSetStatus{NumberReady: 3},
		}
	}
	return objs
}

func generateCronJobs(namespace string, count int) []runtime.Object {
	objs := make([]runtime.Object, count)
	for i := 0; i < count; i++ {
		objs[i] = &batchv1.CronJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("cj-%d", i),
				Namespace: namespace,
			},
		}
	}
	return objs
}

func odigosConfigMap() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.OdigosEffectiveConfigName,
			Namespace: odigosNs,
		},
		Data: map[string]string{
			consts.OdigosConfigurationFileName: "ignoredNamespaces: []",
		},
	}
}

func buildSlowFakeClient(latency time.Duration, k8sObjects []runtime.Object, odigosObjects []runtime.Object) *kube.Client {
	k8sFake := kubefake.NewSimpleClientset(k8sObjects...)
	k8sFake.PrependReactor("*", "*", slowReactor(latency))
	k8sFake.Discovery().(*fakediscovery.FakeDiscovery).FakedServerVersion = &versionutil.Info{
		GitVersion: "v1.28.0",
	}

	odigosFake := odigosfake.NewSimpleClientset(odigosObjects...)
	odigosFake.PrependReactor("*", "*", slowReactor(latency))

	return &kube.Client{
		Interface:     k8sFake,
		OdigosClient:  odigosFake.OdigosV1alpha1(),
		DynamicClient: newFakeDynamicClient(),
	}
}

// newFakeDynamicClient creates a dynamic client that knows about OpenShift
// DeploymentConfigs and Argo Rollouts GVRs so IsDeploymentConfigAvailable()
// doesn't panic. Both will return empty lists (not available).
func newFakeDynamicClient() *fakedynamic.FakeDynamicClient {
	dynScheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(dynScheme)
	return fakedynamic.NewSimpleDynamicClientWithCustomListKinds(dynScheme,
		map[schema.GroupVersionResource]string{
			{Group: "apps.openshift.io", Version: "v1", Resource: "deploymentconfigs"}: "DeploymentConfigList",
			{Group: "argoproj.io", Version: "v1alpha1", Resource: "rollouts"}:           "RolloutList",
		},
	)
}

func buildFakeCacheClient(objects ...ctrlclient.Object) ctrlclient.Client {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = odigosv1alpha1.AddToScheme(scheme)
	return crfake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
}

// ============================================================================
// Benchmarks for Fix 1: GetK8SNamespaces — N+1 Sources elimination
//
// "before": Simulates the OLD code path — per-namespace GetSourceCRD call
//           (1 Sources.List per namespace, each with label selector)
// "after":  Calls the NEW GetK8SNamespaces which does a single batch fetch
// ============================================================================

func BenchmarkGetK8SNamespaces(b *testing.B) {
	for _, nsCount := range []int{10, 100, 500} {
		latency := 5 * time.Millisecond

		// "before": simulate old N+1 pattern explicitly
		b.Run(fmt.Sprintf("before/%dns", nsCount), func(b *testing.B) {
			namespaces := generateNamespaces(nsCount)
			sources := generateNamespaceSources(nsCount)
			k8sObjs := append(namespaces, odigosConfigMap())

			client := buildSlowFakeClient(latency, k8sObjs, sources)
			kube.DefaultClient = client

			ctx := context.Background()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Step 1: List namespaces (same as both old and new)
				nsList, err := kube.DefaultClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
				if err != nil {
					b.Fatal(err)
				}

				// Step 2: OLD pattern — per-namespace Sources.List with label selector
				for _, ns := range nsList.Items {
					_, err := kube.DefaultClient.OdigosClient.Sources(ns.Name).List(ctx, metav1.ListOptions{
						LabelSelector: labels.SelectorFromSet(labels.Set{
							k8sconsts.WorkloadNamespaceLabel: ns.Name,
							k8sconsts.WorkloadNameLabel:      ns.Name,
							k8sconsts.WorkloadKindLabel:      string(k8sconsts.WorkloadKindNamespace),
						}).String(),
					})
					if err != nil {
						b.Fatal(err)
					}
				}
			}
		})

		// "after": new batch code — calls GetK8SNamespaces which does 1 Sources.List
		b.Run(fmt.Sprintf("after/%dns", nsCount), func(b *testing.B) {
			namespaces := generateNamespaces(nsCount)
			sources := generateNamespaceSources(nsCount)
			k8sObjs := append(namespaces, odigosConfigMap())

			client := buildSlowFakeClient(latency, k8sObjs, sources)
			kube.DefaultClient = client

			ctx := context.Background()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				result, err := GetK8SNamespaces(ctx, nil)
				if err != nil {
					b.Fatal(err)
				}
				if len(result) != nsCount {
					b.Fatalf("expected %d namespaces, got %d", nsCount, len(result))
				}
			}
		})
	}
}

// ============================================================================
// Benchmarks for Fix 3: GetWorkloadsInNamespace — Direct API vs Cache
//
// "before": Simulates the OLD code path — 4 direct API list calls with latency
// "after":  Calls the NEW GetWorkloadsInNamespace which uses CacheClient
// ============================================================================

func BenchmarkGetWorkloadsInNamespace(b *testing.B) {
	for _, depCount := range []int{100, 1000} {
		nsName := "test-ns"
		latency := 5 * time.Millisecond

		// "before": simulate old direct API pattern (4 slow list calls)
		b.Run(fmt.Sprintf("before/%ddeps", depCount), func(b *testing.B) {
			var k8sObjs []runtime.Object
			k8sObjs = append(k8sObjs, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName}})
			k8sObjs = append(k8sObjs, generateDeployments(nsName, depCount)...)
			k8sObjs = append(k8sObjs, generateStatefulSets(nsName, depCount/10)...)
			k8sObjs = append(k8sObjs, generateDaemonSets(nsName, depCount/20)...)
			k8sObjs = append(k8sObjs, generateCronJobs(nsName, depCount/20)...)

			client := buildSlowFakeClient(latency, k8sObjs, nil)
			kube.DefaultClient = client

			ctx := context.Background()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Simulate old pattern: Namespace.Get + 4 paginated List calls
				_, err := client.CoreV1().Namespaces().Get(ctx, nsName, metav1.GetOptions{})
				if err != nil {
					b.Fatal(err)
				}
				var total int
				deps, err := client.AppsV1().Deployments(nsName).List(ctx, metav1.ListOptions{})
				if err != nil {
					b.Fatal(err)
				}
				total += len(deps.Items)

				stss, err := client.AppsV1().StatefulSets(nsName).List(ctx, metav1.ListOptions{})
				if err != nil {
					b.Fatal(err)
				}
				total += len(stss.Items)

				dss, err := client.AppsV1().DaemonSets(nsName).List(ctx, metav1.ListOptions{})
				if err != nil {
					b.Fatal(err)
				}
				total += len(dss.Items)

				cjs, err := client.BatchV1().CronJobs(nsName).List(ctx, metav1.ListOptions{})
				if err != nil {
					b.Fatal(err)
				}
				total += len(cjs.Items)

				expected := depCount + depCount/10 + depCount/20 + depCount/20
				if total != expected {
					b.Fatalf("expected %d workloads, got %d", expected, total)
				}
			}
		})

		// "after": new code uses CacheClient (in-memory, no latency)
		b.Run(fmt.Sprintf("after/%ddeps", depCount), func(b *testing.B) {
			var cacheObjs []ctrlclient.Object
			for _, obj := range generateDeployments(nsName, depCount) {
				cacheObjs = append(cacheObjs, obj.(ctrlclient.Object))
			}
			for _, obj := range generateStatefulSets(nsName, depCount/10) {
				cacheObjs = append(cacheObjs, obj.(ctrlclient.Object))
			}
			for _, obj := range generateDaemonSets(nsName, depCount/20) {
				cacheObjs = append(cacheObjs, obj.(ctrlclient.Object))
			}
			for _, obj := range generateCronJobs(nsName, depCount/20) {
				cacheObjs = append(cacheObjs, obj.(ctrlclient.Object))
			}

			kube.CacheClient = buildFakeCacheClient(cacheObjs...)

			// DefaultClient still needed for Discovery (getKubeVersion) and
			// DeploymentConfigs/Rollouts (which stay as direct API calls)
			k8sFake := kubefake.NewSimpleClientset()
			k8sFake.Discovery().(*fakediscovery.FakeDiscovery).FakedServerVersion = &versionutil.Info{
				GitVersion: "v1.28.0",
			}
			kube.DefaultClient = &kube.Client{
				Interface:     k8sFake,
				DynamicClient: newFakeDynamicClient(),
			}

			ctx := context.Background()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				result, err := GetWorkloadsInNamespace(ctx, nsName)
				if err != nil {
					b.Fatal(err)
				}
				expected := depCount + depCount/10 + depCount/20 + depCount/20
				if len(result) != expected {
					b.Fatalf("expected %d workloads, got %d", expected, len(result))
				}
			}
		})
	}
}
