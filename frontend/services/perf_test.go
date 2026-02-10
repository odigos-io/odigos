package services

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/frontend/testutil"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	versionutil "k8s.io/apimachinery/pkg/version"
	fakediscovery "k8s.io/client-go/discovery/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"

	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const odigosNs = "odigos-system"

func TestMain(m *testing.M) {
	os.Setenv("CURRENT_NS", odigosNs)
	os.Exit(m.Run())
}

func TestPerfGetK8SNamespaces(t *testing.T) {
	nsCount := 100
	latency := 5 * time.Millisecond
	budget := 100 * time.Millisecond

	namespaces := testutil.GenerateNamespaces(nsCount)
	sources := testutil.GenerateNamespaceSources(nsCount)
	k8sObjs := append(namespaces, testutil.OdigosConfigMap(odigosNs))

	kube.DefaultClient = testutil.SlowFakeClient(latency, k8sObjs, sources)

	ctx := context.Background()
	start := time.Now()
	result, err := GetK8SNamespaces(ctx, nil)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("GetK8SNamespaces failed: %v", err)
	}
	if len(result) != nsCount {
		t.Fatalf("expected %d namespaces, got %d", nsCount, len(result))
	}
	if elapsed > budget {
		t.Fatalf("GetK8SNamespaces took %v, exceeds budget %v (possible N+1)", elapsed, budget)
	}
	t.Logf("GetK8SNamespaces: %v (budget %v)", elapsed, budget)
}

func TestPerfGetWorkloadsInNamespace(t *testing.T) {
	depCount := 1000
	nsName := "test-ns"
	budget := 50 * time.Millisecond

	var cacheObjs []ctrlclient.Object
	for _, obj := range testutil.GenerateDeployments(nsName, depCount) {
		cacheObjs = append(cacheObjs, obj.(ctrlclient.Object))
	}
	for _, obj := range testutil.GenerateStatefulSets(nsName, depCount/10) {
		cacheObjs = append(cacheObjs, obj.(ctrlclient.Object))
	}
	for _, obj := range testutil.GenerateDaemonSets(nsName, depCount/20) {
		cacheObjs = append(cacheObjs, obj.(ctrlclient.Object))
	}
	for _, obj := range testutil.GenerateCronJobs(nsName, depCount/20) {
		cacheObjs = append(cacheObjs, obj.(ctrlclient.Object))
	}

	kube.CacheClient = testutil.FakeCacheClient(cacheObjs...)

	k8sFake := kubefake.NewSimpleClientset()
	k8sFake.Discovery().(*fakediscovery.FakeDiscovery).FakedServerVersion = &versionutil.Info{
		GitVersion: "v1.28.0",
	}
	kube.DefaultClient = &kube.Client{
		Interface:     k8sFake,
		DynamicClient: testutil.FakeDynamicClient(),
	}

	ctx := context.Background()
	start := time.Now()
	result, err := GetWorkloadsInNamespace(ctx, nsName)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("GetWorkloadsInNamespace failed: %v", err)
	}
	expected := depCount + depCount/10 + depCount/20 + depCount/20
	if len(result) != expected {
		t.Fatalf("expected %d workloads, got %d", expected, len(result))
	}
	if elapsed > budget {
		t.Fatalf("GetWorkloadsInNamespace took %v, exceeds budget %v", elapsed, budget)
	}
	t.Logf("GetWorkloadsInNamespace: %v (budget %v)", elapsed, budget)
}

func TestPerfCountAppsPerNamespace(t *testing.T) {
	nsCount := 100
	workloadsPerNs := 10
	budget := 100 * time.Millisecond

	var cacheObjs []ctrlclient.Object
	for i := range nsCount {
		ns := fmt.Sprintf("ns-%d", i)
		for _, obj := range testutil.GenerateDeployments(ns, workloadsPerNs) {
			cacheObjs = append(cacheObjs, obj.(ctrlclient.Object))
		}
		for _, obj := range testutil.GenerateStatefulSets(ns, workloadsPerNs/5) {
			cacheObjs = append(cacheObjs, obj.(ctrlclient.Object))
		}
		for _, obj := range testutil.GenerateDaemonSets(ns, workloadsPerNs/5) {
			cacheObjs = append(cacheObjs, obj.(ctrlclient.Object))
		}
	}

	kube.CacheClient = testutil.FakeCacheClient(cacheObjs...)

	ctx := context.Background()
	start := time.Now()
	counts, err := CountAppsPerNamespace(ctx)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("CountAppsPerNamespace failed: %v", err)
	}
	if len(counts) != nsCount {
		t.Fatalf("expected counts for %d namespaces, got %d", nsCount, len(counts))
	}
	if elapsed > budget {
		t.Fatalf("CountAppsPerNamespace took %v, exceeds budget %v", elapsed, budget)
	}
	t.Logf("CountAppsPerNamespace: %v (budget %v)", elapsed, budget)
}

func BenchmarkGetK8SNamespaces(b *testing.B) {
	for _, nsCount := range []int{10, 100, 500} {
		latency := 5 * time.Millisecond

		b.Run(fmt.Sprintf("before/%dns", nsCount), func(b *testing.B) {
			namespaces := testutil.GenerateNamespaces(nsCount)
			sources := testutil.GenerateNamespaceSources(nsCount)
			k8sObjs := append(namespaces, testutil.OdigosConfigMap(odigosNs))

			kube.DefaultClient = testutil.SlowFakeClient(latency, k8sObjs, sources)

			ctx := context.Background()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				nsList, err := kube.DefaultClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
				if err != nil {
					b.Fatal(err)
				}
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

		b.Run(fmt.Sprintf("after/%dns", nsCount), func(b *testing.B) {
			namespaces := testutil.GenerateNamespaces(nsCount)
			sources := testutil.GenerateNamespaceSources(nsCount)
			k8sObjs := append(namespaces, testutil.OdigosConfigMap(odigosNs))

			kube.DefaultClient = testutil.SlowFakeClient(latency, k8sObjs, sources)

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

func BenchmarkGetWorkloadsInNamespace(b *testing.B) {
	for _, depCount := range []int{100, 1000} {
		nsName := "test-ns"
		latency := 5 * time.Millisecond

		b.Run(fmt.Sprintf("before/%ddeps", depCount), func(b *testing.B) {
			var k8sObjs []runtime.Object
			k8sObjs = append(k8sObjs, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName}})
			k8sObjs = append(k8sObjs, testutil.GenerateDeployments(nsName, depCount)...)
			k8sObjs = append(k8sObjs, testutil.GenerateStatefulSets(nsName, depCount/10)...)
			k8sObjs = append(k8sObjs, testutil.GenerateDaemonSets(nsName, depCount/20)...)
			k8sObjs = append(k8sObjs, testutil.GenerateCronJobs(nsName, depCount/20)...)

			kube.DefaultClient = testutil.SlowFakeClient(latency, k8sObjs, nil)

			ctx := context.Background()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := kube.DefaultClient.CoreV1().Namespaces().Get(ctx, nsName, metav1.GetOptions{})
				if err != nil {
					b.Fatal(err)
				}
				var total int
				deps, err := kube.DefaultClient.AppsV1().Deployments(nsName).List(ctx, metav1.ListOptions{})
				if err != nil {
					b.Fatal(err)
				}
				total += len(deps.Items)

				stss, err := kube.DefaultClient.AppsV1().StatefulSets(nsName).List(ctx, metav1.ListOptions{})
				if err != nil {
					b.Fatal(err)
				}
				total += len(stss.Items)

				dss, err := kube.DefaultClient.AppsV1().DaemonSets(nsName).List(ctx, metav1.ListOptions{})
				if err != nil {
					b.Fatal(err)
				}
				total += len(dss.Items)

				cjs, err := kube.DefaultClient.BatchV1().CronJobs(nsName).List(ctx, metav1.ListOptions{})
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

		b.Run(fmt.Sprintf("after/%ddeps", depCount), func(b *testing.B) {
			var cacheObjs []ctrlclient.Object
			for _, obj := range testutil.GenerateDeployments(nsName, depCount) {
				cacheObjs = append(cacheObjs, obj.(ctrlclient.Object))
			}
			for _, obj := range testutil.GenerateStatefulSets(nsName, depCount/10) {
				cacheObjs = append(cacheObjs, obj.(ctrlclient.Object))
			}
			for _, obj := range testutil.GenerateDaemonSets(nsName, depCount/20) {
				cacheObjs = append(cacheObjs, obj.(ctrlclient.Object))
			}
			for _, obj := range testutil.GenerateCronJobs(nsName, depCount/20) {
				cacheObjs = append(cacheObjs, obj.(ctrlclient.Object))
			}

			kube.CacheClient = testutil.FakeCacheClient(cacheObjs...)

			k8sFake := kubefake.NewSimpleClientset()
			k8sFake.Discovery().(*fakediscovery.FakeDiscovery).FakedServerVersion = &versionutil.Info{
				GitVersion: "v1.28.0",
			}
			kube.DefaultClient = &kube.Client{
				Interface:     k8sFake,
				DynamicClient: testutil.FakeDynamicClient(),
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

func BenchmarkCountAppsPerNamespace(b *testing.B) {
	nsCount := 50
	workloadsPerNs := 10

	var cacheObjs []ctrlclient.Object
	for i := range nsCount {
		ns := fmt.Sprintf("ns-%d", i)
		for _, obj := range testutil.GenerateDeployments(ns, workloadsPerNs) {
			cacheObjs = append(cacheObjs, obj.(ctrlclient.Object))
		}
		for _, obj := range testutil.GenerateStatefulSets(ns, workloadsPerNs/5) {
			cacheObjs = append(cacheObjs, obj.(ctrlclient.Object))
		}
		for _, obj := range testutil.GenerateDaemonSets(ns, workloadsPerNs/5) {
			cacheObjs = append(cacheObjs, obj.(ctrlclient.Object))
		}
	}

	kube.CacheClient = testutil.FakeCacheClient(cacheObjs...)

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		counts, err := CountAppsPerNamespace(ctx)
		if err != nil {
			b.Fatal(err)
		}
		if len(counts) != nsCount {
			b.Fatalf("expected %d namespaces, got %d", nsCount, len(counts))
		}
	}
}
