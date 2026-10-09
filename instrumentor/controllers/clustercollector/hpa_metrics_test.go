package clustercollector

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/instrumentor/controllers/metricshandler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// hcMetric is the part of an HPA metric spec that matters to this contract, flattened so the two
// autoscaling API versions can be compared without hand-written per-version expectations.
type hcMetric struct {
	kind            string
	name            string
	describedObject string
	targetType      string
	target          string
}

func hcFromV2(metrics []autoscalingv2.MetricSpec) []hcMetric {
	out := make([]hcMetric, 0, len(metrics))
	for _, m := range metrics {
		switch {
		case m.Object != nil:
			out = append(out, hcMetric{
				kind: string(m.Type),
				name: m.Object.Metric.Name,
				describedObject: strings.Join([]string{
					m.Object.DescribedObject.APIVersion,
					m.Object.DescribedObject.Kind,
					m.Object.DescribedObject.Name,
				}, "/"),
				targetType: string(m.Object.Target.Type),
				target:     m.Object.Target.Value.String(),
			})
		case m.Resource != nil:
			out = append(out, hcMetric{
				kind:       string(m.Type),
				name:       string(m.Resource.Name),
				targetType: string(m.Resource.Target.Type),
				target:     m.Resource.Target.AverageValue.String(),
			})
		}
	}
	return out
}

func hcFromV2beta2(metrics []autoscalingv2beta2.MetricSpec) []hcMetric {
	out := make([]hcMetric, 0, len(metrics))
	for _, m := range metrics {
		switch {
		case m.Object != nil:
			out = append(out, hcMetric{
				kind: string(m.Type),
				name: m.Object.Metric.Name,
				describedObject: strings.Join([]string{
					m.Object.DescribedObject.APIVersion,
					m.Object.DescribedObject.Kind,
					m.Object.DescribedObject.Name,
				}, "/"),
				targetType: string(m.Object.Target.Type),
				target:     m.Object.Target.Value.String(),
			})
		case m.Resource != nil:
			out = append(out, hcMetric{
				kind:       string(m.Type),
				name:       string(m.Resource.Name),
				targetType: string(m.Resource.Target.Type),
				target:     m.Resource.Target.AverageValue.String(),
			})
		}
	}
	return out
}

func hcQuantities(t *testing.T) (mem, cpu resource.Quantity) {
	t.Helper()
	return resource.MustParse("768Mi"), resource.MustParse("375m")
}

// hcAdvertisedMetricName reads the metric name out of the custom metrics discovery document the
// instrumentor actually serves. The HPA addresses the metric by that name, and nothing but this
// assertion links the two string literals.
func hcAdvertisedMetricName(t *testing.T) string {
	t.Helper()

	rec := httptest.NewRecorder()
	metricshandler.DiscoveryHandler(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	var doc struct {
		Resources []struct {
			Name string `json:"name"`
		} `json:"resources"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &doc))
	require.Len(t, doc.Resources, 1)

	_, metricName, found := strings.Cut(doc.Resources[0].Name, "/")
	require.True(t, found)
	return metricName
}

func TestBuildHPAMetrics_WithoutTheCustomMetric(t *testing.T) {
	mem, cpu := hcQuantities(t)

	want := []hcMetric{
		{kind: "Resource", name: string(corev1.ResourceMemory), targetType: "AverageValue", target: "768Mi"},
		{kind: "Resource", name: string(corev1.ResourceCPU), targetType: "AverageValue", target: "375m"},
	}

	assert.Equal(t, want, hcFromV2(buildv2Metrics(false, mem, cpu)))
	assert.Equal(t, want, hcFromV2beta2(buildv2beta2Metrics(false, mem, cpu)))
}

// The object metric is what lets the gateway scale out while its pods are OOM looping and
// reporting nothing to the metrics server, so every field of the reference is load-bearing.
func TestBuildHPAMetrics_WithTheCustomMetric(t *testing.T) {
	mem, cpu := hcQuantities(t)

	want := []hcMetric{
		{
			kind:            "Object",
			name:            hcAdvertisedMetricName(t),
			describedObject: "apps/v1/Deployment/" + k8sconsts.OdigosClusterCollectorDeploymentName,
			targetType:      "Value",
			target:          "500m",
		},
		{kind: "Resource", name: string(corev1.ResourceMemory), targetType: "AverageValue", target: "768Mi"},
		{kind: "Resource", name: string(corev1.ResourceCPU), targetType: "AverageValue", target: "375m"},
	}

	assert.Equal(t, want, hcFromV2(buildv2Metrics(true, mem, cpu)))
	assert.Equal(t, want, hcFromV2beta2(buildv2beta2Metrics(true, mem, cpu)))
}

// Clusters between 1.23 and 1.25 get the v2beta2 spec and everything newer gets v2. The two are
// built by separate literal-for-literal copies, so they are only kept in step by this check.
func TestBuildHPAMetrics_TheTwoAutoscalingVersionsAgree(t *testing.T) {
	mem, cpu := hcQuantities(t)

	for _, useCustomMetric := range []bool{false, true} {
		v2 := hcFromV2(buildv2Metrics(useCustomMetric, mem, cpu))
		v2beta2 := hcFromV2beta2(buildv2beta2Metrics(useCustomMetric, mem, cpu))

		require.NotEmpty(t, v2)
		assert.Equal(t, v2, v2beta2, "useCustomMetric=%v", useCustomMetric)
	}
}

// metricshandler serves a binary 0-or-1 signal (see
// TestMetricHandler_ServesOnlyTheTwoValuesTheHPATargetSitsBetween). The HPA only scales out when
// the served value exceeds its target, so the target has to fall strictly inside that range.
func TestGatewayHPATargetSitsBetweenTheTwoServedValues(t *testing.T) {
	mem, cpu := hcQuantities(t)

	metrics := buildv2Metrics(true, mem, cpu)
	require.NotEmpty(t, metrics)
	require.NotNil(t, metrics[0].Object)

	target := metrics[0].Object.Target.Value.AsApproximateFloat64()

	assert.Greater(t, target, 0.0, "the idle signal would already trigger scale-out")
	assert.Less(t, target, 1.0, "the rejecting signal would never trigger scale-out")
}
