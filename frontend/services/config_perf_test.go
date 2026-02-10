package services

import (
	"context"
	"testing"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/frontend/testutil"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// TestPerfIsSourceCreated verifies that isSourceCreated reads from cache
// (instant in-memory read, no API calls).
func TestPerfIsSourceCreated(t *testing.T) {
	budget := 10 * time.Millisecond

	var cacheObjs []ctrlclient.Object
	for _, obj := range testutil.GenerateSources("ns-0", 5) {
		cacheObjs = append(cacheObjs, obj.(ctrlclient.Object))
	}
	for _, obj := range testutil.GenerateSources("ns-1", 3) {
		cacheObjs = append(cacheObjs, obj.(ctrlclient.Object))
	}
	kube.CacheClient = testutil.FakeCacheClient(cacheObjs...)

	ctx := context.Background()
	start := time.Now()
	result := isSourceCreated(ctx)
	elapsed := time.Since(start)

	if !result {
		t.Fatal("expected isSourceCreated to return true")
	}
	if elapsed > budget {
		t.Fatalf("isSourceCreated took %v, exceeds budget %v (should be instant cache read)", elapsed, budget)
	}
	t.Logf("isSourceCreated: %v (budget %v)", elapsed, budget)
}

// TestPerfGetConfig verifies GetConfig completes within budget.
// Most reads are from cache (ConfigMaps, Deployment, Sources).
// Only isDestinationConnected makes an API call (with Limit:1).
func TestPerfGetConfig(t *testing.T) {
	latency := 5 * time.Millisecond
	budget := 50 * time.Millisecond

	// Cache objects: ConfigMaps + Sources
	var cacheObjs []ctrlclient.Object
	cacheObjs = append(cacheObjs, testutil.OdigosConfigMap(odigosNs))
	cacheObjs = append(cacheObjs, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigosDeploymentConfigMapName,
			Namespace: odigosNs,
		},
		Data: map[string]string{
			k8sconsts.OdigosDeploymentConfigMapTierKey:               "community",
			k8sconsts.OdigosDeploymentConfigMapInstallationMethodKey: "cli",
		},
	})
	for _, obj := range testutil.GenerateSources("ns-0", 5) {
		cacheObjs = append(cacheObjs, obj.(ctrlclient.Object))
	}
	kube.CacheClient = testutil.FakeCacheClient(cacheObjs...)

	// DefaultClient: only used for isDestinationConnected
	var odigosObjs []runtime.Object
	destObjs, destK8sObjs := testutil.GenerateDestinationsAndSecrets(odigosNs, 1)
	odigosObjs = append(odigosObjs, destObjs...)
	kube.DefaultClient = testutil.SlowFakeClient(latency, destK8sObjs, odigosObjs)

	ctx := context.Background()
	start := time.Now()
	_ = GetConfig(ctx)
	elapsed := time.Since(start)

	if elapsed > budget {
		t.Fatalf("GetConfig took %v, exceeds budget %v", elapsed, budget)
	}
	t.Logf("GetConfig: %v (budget %v)", elapsed, budget)
}
