package loaders

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/frontend/testutil"

	"k8s.io/apimachinery/pkg/runtime"
)

const odigosNs = "odigos-system"

func TestMain(m *testing.M) {
	os.Setenv("CURRENT_NS", odigosNs)
	os.Exit(m.Run())
}

func TestPerfFetchWorkloadManifests(t *testing.T) {
	nsName := "test-ns"
	depCount := 100
	latency := 5 * time.Millisecond
	budget := 100 * time.Millisecond

	var k8sObjs []runtime.Object
	k8sObjs = append(k8sObjs, testutil.GenerateDeployments(nsName, depCount)...)
	k8sObjs = append(k8sObjs, testutil.GenerateStatefulSets(nsName, depCount/10)...)
	k8sObjs = append(k8sObjs, testutil.GenerateDaemonSets(nsName, depCount/20)...)
	k8sObjs = append(k8sObjs, testutil.GenerateCronJobs(nsName, depCount/20)...)

	kube.DefaultClient = testutil.SlowFakeClient(latency, k8sObjs, nil)

	filters := &WorkloadFilter{
		SingleNamespace: &WorkloadFilterSingleNamespace{Namespace: nsName},
		NamespaceString: nsName,
	}

	ctx := context.Background()
	logger := logr.Discard()

	start := time.Now()
	result, err := fetchWorkloadManifests(ctx, logger, filters)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("fetchWorkloadManifests failed: %v", err)
	}
	expected := depCount + depCount/10 + depCount/20 + depCount/20
	if len(result) != expected {
		t.Fatalf("expected %d workloads, got %d", expected, len(result))
	}
	if elapsed > budget {
		t.Fatalf("fetchWorkloadManifests took %v, exceeds budget %v (not parallel?)", elapsed, budget)
	}
	t.Logf("fetchWorkloadManifests: %v (budget %v)", elapsed, budget)
}
