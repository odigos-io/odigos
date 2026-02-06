package graph

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/frontend/services"
	"github.com/odigos-io/odigos/frontend/testutil"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const odigosNs = "odigos-system"

func TestMain(m *testing.M) {
	os.Setenv("CURRENT_NS", odigosNs)
	os.Exit(m.Run())
}

// ============================================================================
// Threshold tests — these fail CI if an API path exceeds its time budget.
// ============================================================================

// TestPerfDestinations verifies the batch Destinations+Secrets path completes
// within budget (3 API calls: Destinations.List + ConfigMap.Get + Secrets.List).
func TestPerfDestinations(t *testing.T) {
	destCount := 100
	latency := 5 * time.Millisecond
	budget := 100 * time.Millisecond

	odigosObjs, k8sObjs := testutil.GenerateDestinationsAndSecrets(odigosNs, destCount)
	k8sObjs = append(k8sObjs, testutil.OdigosConfigMap(odigosNs))

	kube.DefaultClient = testutil.SlowFakeClient(latency, k8sObjs, odigosObjs)

	resolver := &computePlatformResolver{}
	ctx := context.Background()

	start := time.Now()
	result, err := resolver.Destinations(ctx, nil)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Destinations failed: %v", err)
	}
	if len(result) != destCount {
		t.Fatalf("expected %d destinations, got %d", destCount, len(result))
	}
	if elapsed > budget {
		t.Fatalf("Destinations took %v, exceeds budget %v (possible N+1)", elapsed, budget)
	}
	t.Logf("Destinations: %v (budget %v)", elapsed, budget)
}

// TestPerfDataStreams verifies the DataStreams resolver completes within budget
// (2 API calls: InstrumentationConfigs.List + Destinations.List).
func TestPerfDataStreams(t *testing.T) {
	icCount := 500
	destCount := 50
	latency := 5 * time.Millisecond
	budget := 100 * time.Millisecond

	var odigosObjs []runtime.Object
	odigosObjs = append(odigosObjs, testutil.GenerateInstrumentationConfigs("test-ns", icCount)...)
	destOdigosObjs, k8sObjs := testutil.GenerateDestinationsAndSecrets(odigosNs, destCount)
	odigosObjs = append(odigosObjs, destOdigosObjs...)
	k8sObjs = append(k8sObjs, testutil.OdigosConfigMap(odigosNs))

	kube.DefaultClient = testutil.SlowFakeClient(latency, k8sObjs, odigosObjs)

	resolver := &computePlatformResolver{}
	ctx := context.Background()

	start := time.Now()
	result, err := resolver.DataStreams(ctx, nil)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("DataStreams failed: %v", err)
	}
	if len(result) == 0 {
		t.Fatal("expected at least 1 data stream")
	}
	if elapsed > budget {
		t.Fatalf("DataStreams took %v, exceeds budget %v", elapsed, budget)
	}
	t.Logf("DataStreams: %v (budget %v), streams: %d", elapsed, budget, len(result))
}

// ============================================================================
// Benchmarks — Destinations: batch Secrets.List vs per-destination N+1
// ============================================================================

func BenchmarkDestinationSecrets_Before(b *testing.B) {
	for _, destCount := range []int{10, 100} {
		b.Run(fmt.Sprintf("%ddests", destCount), func(b *testing.B) {
			latency := 5 * time.Millisecond
			odigosObjs, k8sObjs := testutil.GenerateDestinationsAndSecrets(odigosNs, destCount)
			k8sObjs = append(k8sObjs, testutil.OdigosConfigMap(odigosNs))

			kube.DefaultClient = testutil.SlowFakeClient(latency, k8sObjs, odigosObjs)

			ctx := context.Background()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				dests, err := kube.DefaultClient.OdigosClient.Destinations(odigosNs).List(ctx, metav1.ListOptions{})
				if err != nil {
					b.Fatal(err)
				}
				for _, dest := range dests.Items {
					_, err := services.GetDestinationSecretFields(ctx, odigosNs, &dest)
					if err != nil {
						b.Fatal(err)
					}
				}
				if len(dests.Items) != destCount {
					b.Fatalf("expected %d dests, got %d", destCount, len(dests.Items))
				}
			}
		})
	}
}

func BenchmarkDestinationSecrets_After(b *testing.B) {
	for _, destCount := range []int{10, 100} {
		b.Run(fmt.Sprintf("%ddests", destCount), func(b *testing.B) {
			latency := 5 * time.Millisecond
			odigosObjs, k8sObjs := testutil.GenerateDestinationsAndSecrets(odigosNs, destCount)
			k8sObjs = append(k8sObjs, testutil.OdigosConfigMap(odigosNs))

			kube.DefaultClient = testutil.SlowFakeClient(latency, k8sObjs, odigosObjs)

			ctx := context.Background()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				dests, err := kube.DefaultClient.OdigosClient.Destinations(odigosNs).List(ctx, metav1.ListOptions{})
				if err != nil {
					b.Fatal(err)
				}

				allSecrets, err := kube.DefaultClient.CoreV1().Secrets(odigosNs).List(ctx, metav1.ListOptions{})
				if err != nil {
					b.Fatal(err)
				}

				secretsByName := make(map[string]*corev1.Secret)
				for idx := range allSecrets.Items {
					secretsByName[allSecrets.Items[idx].Name] = &allSecrets.Items[idx]
				}

				for _, dest := range dests.Items {
					if dest.Spec.SecretRef != nil {
						if secret, ok := secretsByName[dest.Spec.SecretRef.Name]; ok {
							fields := services.ExtractSecretFields(secret)
							if len(fields) == 0 {
								b.Fatal("expected secret fields")
							}
						}
					}
				}
				if len(dests.Items) != destCount {
					b.Fatalf("expected %d dests, got %d", destCount, len(dests.Items))
				}
			}
		})
	}
}
