package tracecorrelations

import (
	"testing"
	"time"

	prommodel "github.com/prometheus/common/model"
	"github.com/stretchr/testify/require"

	"github.com/odigos-io/odigos/frontend/graph/model"
)

func TestWorkloadFromMetric(t *testing.T) {
	labels := prommodel.Metric{
		"__name__":            "traces_service_io_connection_total",
		"k8s_namespace_name":  "default",
		"k8s_container_name":  "app",
		"k8s_deployment_name": "checkout",
	}

	workload, ok := workloadFromMetric(labels)
	require.True(t, ok)
	require.Equal(t, workloadKey{
		namespace: "default",
		kind:      "Deployment",
		name:      "checkout",
		container: "app",
	}, workload)
}

func TestAttributeGroupFromMetric(t *testing.T) {
	labels := prommodel.Metric{
		"input_http_route":        "/auth/login",
		"input_span_kind":         "SPAN_KIND_SERVER",
		"output_db_system":        "postgresql",
		"output_db_statement":     "SELECT 1",
		"odigos_collector_instance_id": "abc",
		"k8s_namespace_name":      "default",
	}

	input := attributeGroupFromMetric(labels, inputAttributePrefix)
	require.Equal(t, map[string]string{
		"http.route": "/auth/login",
		"span.kind":  "SPAN_KIND_SERVER",
	}, input.attrs)

	output := attributeGroupFromMetric(labels, outputAttributePrefix)
	require.Equal(t, map[string]string{
		"db.system":    "postgresql",
		"db.statement": "SELECT 1",
	}, output.attrs)
}

func TestAggregateSeriesAcrossCollectorInstances(t *testing.T) {
	sharedLabels := func(instance string) prommodel.Metric {
		return prommodel.Metric{
			"__name__":                   "traces_service_io_connection_total",
			"k8s_namespace_name":         "default",
			"k8s_container_name":         "app",
			"k8s_deployment_name":        "checkout",
			"input_http_route":           "/login",
			"output_db_system":           "postgresql",
			"odigos_collector_instance_id": prommodel.LabelValue(instance),
		}
	}

	counts := prommodel.Vector{
		&prommodel.Sample{Metric: sharedLabels("a"), Value: 3},
		&prommodel.Sample{Metric: sharedLabels("b"), Value: 5},
	}

	firstSeen := map[string]time.Time{
		mustSeriesIdentityKey(t, sharedLabels("a")): time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC),
		mustSeriesIdentityKey(t, sharedLabels("b")): time.Date(2026, 6, 13, 9, 30, 0, 0, time.UTC),
	}

	result := aggregateSeries(counts, firstSeen, nil)
	require.Len(t, result, 1)

	var series *aggregatedSeries
	for _, byOutput := range result {
		require.Len(t, byOutput, 1)
		for _, value := range byOutput {
			series = value
		}
	}

	require.NotNil(t, series)
	require.Equal(t, int64(8), series.connectionCount)
	require.Equal(t, time.Date(2026, 6, 13, 9, 30, 0, 0, time.UTC), series.firstDetected)
}

func TestBuildResponse(t *testing.T) {
	workload := workloadKey{
		namespace: "default",
		kind:      "Deployment",
		name:      "checkout",
		container: "app",
	}

	aggregated := map[workloadKey]map[string]*aggregatedSeries{
		workload: {
			"input\x00output": {
				workload: workload,
				input: attributeGroup{
					attrs: map[string]string{"http.route": "/login"},
					sig:   "http.route=/login",
				},
				output: attributeGroup{
					attrs: map[string]string{"db.system": "postgresql"},
					sig:   "db.system=postgresql",
				},
				connectionCount: 4,
				firstDetected:   time.Date(2026, 6, 13, 9, 30, 0, 0, time.UTC),
			},
		},
	}

	resp := buildResponse(aggregated)
	require.Len(t, resp.Workloads, 1)
	require.Equal(t, model.K8sResourceKindDeployment, resp.Workloads[0].Kind)
	require.Len(t, resp.Workloads[0].Inputs, 1)
	require.Len(t, resp.Workloads[0].Inputs[0].Outputs, 1)
	require.Equal(t, 4, resp.Workloads[0].Inputs[0].Outputs[0].ConnectionCount)
	require.Equal(t, "2026-06-13T09:30:00Z", resp.Workloads[0].Inputs[0].Outputs[0].FirstDetectedAt)
}

func TestMatchesFilter(t *testing.T) {
	workload := workloadKey{namespace: "prod", kind: "Deployment", name: "api", container: "app"}
	namespace := "prod"
	kind := model.K8sResourceKindDeployment
	name := "api"

	require.True(t, matchesFilter(workload, &model.WorkloadFilter{
		Namespace: &namespace,
		Kind:      &kind,
		Name:      &name,
	}))
	require.False(t, matchesFilter(workload, &model.WorkloadFilter{
		Namespace: &namespace,
		Name:      strPtr("other"),
	}))
}

func strPtr(value string) *string {
	return &value
}

func mustSeriesIdentityKey(t *testing.T, labels prommodel.Metric) string {
	t.Helper()
	key, ok := seriesIdentityKey(labels)
	require.True(t, ok)
	return key
}

func TestFirstSeenMatchesAcrossLabelFormats(t *testing.T) {
	queryLabels := prommodel.Metric{
		"__name__":            "traces_service_io_connection_total",
		"k8s_namespace_name":  "default",
		"k8s_container_name":  "app",
		"k8s_deployment_name": "checkout",
		"input_http_route":    "/login",
		"output_db_system":    "postgresql",
		"odigos_collector_instance_id": "a",
	}

	exportLabels := prommodel.Metric{
		"__name__":                   "traces_service_io_connection_total",
		"k8s.namespace.name":         "default",
		"k8s.container.name":         "app",
		"k8s.deployment.name":        "checkout",
		"input.http.route":           "/login",
		"output.db.system":           "postgresql",
		"odigos.collector.instance.id": "b",
	}

	queryKey, ok := seriesIdentityKey(queryLabels)
	require.True(t, ok)
	exportKey, ok := seriesIdentityKey(exportLabels)
	require.True(t, ok)
	require.Equal(t, queryKey, exportKey)

	firstSeen := map[string]time.Time{
		exportKey: time.Date(2026, 6, 13, 9, 30, 0, 0, time.UTC),
	}

	result := aggregateSeries(prommodel.Vector{
		&prommodel.Sample{Metric: queryLabels, Value: 4},
	}, firstSeen, nil)

	require.Len(t, result, 1)
	for _, byOutput := range result {
		for _, series := range byOutput {
			require.Equal(t, time.Date(2026, 6, 13, 9, 30, 0, 0, time.UTC), series.firstDetected)
		}
	}
}
