package clustercollector

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// The v2beta1/v2beta2 HPAs are built as unstructured because k8s.io/api v0.36 dropped their Go
// types, which means the compiler no longer checks their field names or value types for us. These
// tests stand in for that: they pin the emitted shape and assert the content is JSON-compatible,
// which is what runtime.DeepCopyJSON (and therefore the apply patch) requires.

func legacyHPATestGateway() *odigosv1.CollectorsGroup {
	return &odigosv1.CollectorsGroup{
		ObjectMeta: buildHPACommonFields(&odigosv1.CollectorsGroup{}),
	}
}

func TestBuildV2Beta1HPA(t *testing.T) {
	gateway := legacyHPATestGateway()
	gateway.Namespace = "odigos-test"

	hpa := buildv2beta1HPA(gateway, "odigos-gateway", intPtr(2), 7,
		resource.MustParse("384Mi"), resource.MustParse("750m"))

	assert.Equal(t, "autoscaling/v2beta1", hpa.GetAPIVersion())
	assert.Equal(t, "HorizontalPodAutoscaler", hpa.GetKind())
	assert.Equal(t, k8sconsts.OdigosClusterCollectorHpaName, hpa.GetName())
	assert.Equal(t, "odigos-test", hpa.GetNamespace())

	assertLegacyScaleTarget(t, hpa, int64(2), int64(7))

	// v2beta1 carries the resource target inline as targetAverageValue, and supports neither
	// behavior nor object metrics.
	_, hasBehavior, _ := unstructured.NestedMap(hpa.Object, "spec", "behavior")
	assert.False(t, hasBehavior, "v2beta1 does not support behavior")

	metrics := nestedSliceOfMaps(t, hpa, "spec", "metrics")
	require.Len(t, metrics, 2)
	assert.Equal(t, "Resource", metrics[0]["type"])
	assert.Equal(t, map[string]interface{}{
		"name":               "memory",
		"targetAverageValue": "384Mi",
	}, metrics[0]["resource"])
	assert.Equal(t, map[string]interface{}{
		"name":               "cpu",
		"targetAverageValue": "750m",
	}, metrics[1]["resource"])

	assertJSONCompatible(t, hpa)
}

func TestBuildV2Beta2HPA(t *testing.T) {
	gateway := legacyHPATestGateway()
	gateway.Namespace = "odigos-test"

	hpa := buildv2beta2HPA(gateway, "odigos-gateway", intPtr(2), 7, false,
		resource.MustParse("384Mi"), resource.MustParse("750m"))

	assert.Equal(t, "autoscaling/v2beta2", hpa.GetAPIVersion())
	assertLegacyScaleTarget(t, hpa, int64(2), int64(7))

	// Behavior must mirror the autoscaling/v2 branch: fast scale-up, slow scale-down.
	scaleUp, found, err := unstructured.NestedMap(hpa.Object, "spec", "behavior", "scaleUp")
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, int64(*ScaleUpStabilizationWindowSeconds), scaleUp["stabilizationWindowSeconds"])
	assert.Equal(t, "Max", scaleUp["selectPolicy"])
	assert.Equal(t, []interface{}{
		map[string]interface{}{"type": "Pods", "value": int64(2), "periodSeconds": int64(15)},
	}, scaleUp["policies"])

	scaleDown, found, err := unstructured.NestedMap(hpa.Object, "spec", "behavior", "scaleDown")
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, int64(*ScaleDownStabilizationWindowSeconds), scaleDown["stabilizationWindowSeconds"])
	assert.Equal(t, "Min", scaleDown["selectPolicy"])
	assert.Equal(t, []interface{}{
		map[string]interface{}{"type": "Pods", "value": int64(1), "periodSeconds": int64(60)},
		map[string]interface{}{"type": "Percent", "value": int64(25), "periodSeconds": int64(60)},
	}, scaleDown["policies"])

	// v2beta2 nests the resource target under target/averageValue rather than inline.
	metrics := nestedSliceOfMaps(t, hpa, "spec", "metrics")
	require.Len(t, metrics, 2)
	assert.Equal(t, map[string]interface{}{
		"name":   "memory",
		"target": map[string]interface{}{"type": "AverageValue", "averageValue": "384Mi"},
	}, metrics[0]["resource"])
	assert.Equal(t, map[string]interface{}{
		"name":   "cpu",
		"target": map[string]interface{}{"type": "AverageValue", "averageValue": "750m"},
	}, metrics[1]["resource"])

	assertJSONCompatible(t, hpa)
}

// The custom metric is only added when the odigos custom metrics API service is available, and it
// must come first so it is evaluated alongside the resource metrics.
func TestBuildV2Beta2HPAWithCustomMetric(t *testing.T) {
	hpa := buildv2beta2HPA(legacyHPATestGateway(), "odigos-gateway", intPtr(1), 10, true,
		resource.MustParse("384Mi"), resource.MustParse("750m"))

	metrics := nestedSliceOfMaps(t, hpa, "spec", "metrics")
	require.Len(t, metrics, 3)
	assert.Equal(t, "Object", metrics[0]["type"])
	assert.Equal(t, map[string]interface{}{
		"describedObject": map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"name":       k8sconsts.OdigosClusterCollectorDeploymentName,
		},
		"metric": map[string]interface{}{"name": "odigos_gateway_rejections"},
		// 500m == the 50%-of-pods-rejecting threshold.
		"target": map[string]interface{}{"type": "Value", "value": "500m"},
	}, metrics[0]["object"])

	assertJSONCompatible(t, hpa)
}

// minReplicas is a pointer on the typed API, and must simply be absent rather than emitted as a
// null the API server would reject.
func TestBuildLegacyHPAOmitsNilMinReplicas(t *testing.T) {
	for name, hpa := range map[string]*unstructured.Unstructured{
		"v2beta1": buildv2beta1HPA(legacyHPATestGateway(), "odigos-gateway", nil, 10,
			resource.MustParse("384Mi"), resource.MustParse("750m")),
		"v2beta2": buildv2beta2HPA(legacyHPATestGateway(), "odigos-gateway", nil, 10, false,
			resource.MustParse("384Mi"), resource.MustParse("750m")),
	} {
		t.Run(name, func(t *testing.T) {
			_, found, err := unstructured.NestedInt64(hpa.Object, "spec", "minReplicas")
			require.NoError(t, err)
			assert.False(t, found, "minReplicas should be omitted, not null")
		})
	}
}

func assertLegacyScaleTarget(t *testing.T, hpa *unstructured.Unstructured, minReplicas, maxReplicas int64) {
	t.Helper()

	scaleTargetRef, found, err := unstructured.NestedMap(hpa.Object, "spec", "scaleTargetRef")
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"name":       "odigos-gateway",
	}, scaleTargetRef)

	// NestedInt64 fails outright if the value was stored as an int32, which is the mistake these
	// hand-built maps are most prone to.
	got, found, err := unstructured.NestedInt64(hpa.Object, "spec", "minReplicas")
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, minReplicas, got)

	got, found, err = unstructured.NestedInt64(hpa.Object, "spec", "maxReplicas")
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, maxReplicas, got)
}

func nestedSliceOfMaps(t *testing.T, hpa *unstructured.Unstructured, fields ...string) []map[string]interface{} {
	t.Helper()

	raw, found, err := unstructured.NestedSlice(hpa.Object, fields...)
	require.NoError(t, err)
	require.True(t, found)

	out := make([]map[string]interface{}, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]interface{})
		require.True(t, ok)
		out = append(out, m)
	}
	return out
}

// runtime.DeepCopyJSON panics on any value that is not JSON-compatible, so it doubles as an
// assertion that every number we stored is an int64 and every quantity is a string.
func assertJSONCompatible(t *testing.T, hpa *unstructured.Unstructured) {
	t.Helper()
	assert.NotPanics(t, func() { runtime.DeepCopyJSON(hpa.Object) })
}
