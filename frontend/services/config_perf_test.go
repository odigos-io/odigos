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
)

// TestPerfIsSourceCreated verifies the cluster-wide Sources("").List path
// completes within budget (1 API call instead of N+1 per-namespace).
func TestPerfIsSourceCreated(t *testing.T) {
	latency := 5 * time.Millisecond
	budget := 100 * time.Millisecond

	var odigosObjs []runtime.Object
	odigosObjs = append(odigosObjs, testutil.GenerateSources("ns-0", 5)...)
	odigosObjs = append(odigosObjs, testutil.GenerateSources("ns-1", 3)...)

	kube.DefaultClient = testutil.SlowFakeClient(latency, nil, odigosObjs)

	ctx := context.Background()
	start := time.Now()
	result := isSourceCreated(ctx)
	elapsed := time.Since(start)

	if !result {
		t.Fatal("expected isSourceCreated to return true")
	}
	if elapsed > budget {
		t.Fatalf("isSourceCreated took %v, exceeds budget %v (possible N+1)", elapsed, budget)
	}
	t.Logf("isSourceCreated: %v (budget %v)", elapsed, budget)
}

// TestPerfGetConfig verifies the full GetConfig path completes within budget.
func TestPerfGetConfig(t *testing.T) {
	latency := 5 * time.Millisecond
	budget := 200 * time.Millisecond

	var k8sObjs []runtime.Object
	k8sObjs = append(k8sObjs, testutil.OdigosConfigMap(odigosNs))
	k8sObjs = append(k8sObjs, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigosDeploymentConfigMapName,
			Namespace: odigosNs,
		},
		Data: map[string]string{
			k8sconsts.OdigosDeploymentConfigMapTierKey:               "community",
			k8sconsts.OdigosDeploymentConfigMapInstallationMethodKey: "cli",
		},
	})

	var odigosObjs []runtime.Object
	odigosObjs = append(odigosObjs, testutil.GenerateSources("ns-0", 5)...)
	destObjs, destK8sObjs := testutil.GenerateDestinationsAndSecrets(odigosNs, 1)
	odigosObjs = append(odigosObjs, destObjs...)
	k8sObjs = append(k8sObjs, destK8sObjs...)

	kube.DefaultClient = testutil.SlowFakeClient(latency, k8sObjs, odigosObjs)

	ctx := context.Background()
	start := time.Now()
	_ = GetConfig(ctx)
	elapsed := time.Since(start)

	if elapsed > budget {
		t.Fatalf("GetConfig took %v, exceeds budget %v", elapsed, budget)
	}
	t.Logf("GetConfig: %v (budget %v)", elapsed, budget)
}
