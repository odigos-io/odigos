package collectorconfig

import (
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricsConfig_MetricsPipelineProcessorOrder(t *testing.T) {
	cfg := MetricsConfig(&odigosv1.CollectorsGroup{}, MetricsConfigOptions{
		MetricsConfigSettings: &odigosv1.CollectorsGroupMetricsCollectionSettings{
			AgentsTelemetry: &odigosv1.AgentsTelemetrySettings{},
		},
	})

	pl, ok := cfg.Service.Pipelines[odigosMetricsPipelineName]
	require.True(t, ok)
	// traffic accounting must stay last so counts reflect the final metric shape.
	require.NotEmpty(t, pl.Processors)
	assert.Equal(t, odigosTrafficMetricsProcessorName, pl.Processors[len(pl.Processors)-1])
}

func TestMetricsConfig_UnchangedWhenNoAgentRecordsSpanMetrics(t *testing.T) {
	for _, settings := range []*odigosv1.CollectorsGroupMetricsCollectionSettings{
		{AgentsTelemetry: &odigosv1.AgentsTelemetrySettings{}},
		{AgentsTelemetry: &odigosv1.AgentsTelemetrySettings{}, SpanMetrics: &common.MetricsSourceSpanMetricsConfiguration{}},
	} {
		cfg := MetricsConfig(&odigosv1.CollectorsGroup{}, MetricsConfigOptions{MetricsConfigSettings: settings})
		assert.Nil(t, cfg.Processors)
		assert.NotContains(t, cfg.Service.Pipelines[odigosMetricsPipelineName].Processors, dropAgentSpanMetricsName)
	}
}

func TestMetricsConfig_SpanMetricsRecordedForTraceSurgesStayOutOfDestinations(t *testing.T) {
	forSurges := &odigosv1.AgentSpanMetricsSettings{Insights: true}
	for _, spanMetrics := range []*common.MetricsSourceSpanMetricsConfiguration{nil, {Interval: "60s"}} {
		cfg := MetricsConfig(&odigosv1.CollectorsGroup{}, MetricsConfigOptions{MetricsConfigSettings: &odigosv1.CollectorsGroupMetricsCollectionSettings{
			AgentsTelemetry:  &odigosv1.AgentsTelemetrySettings{},
			SpanMetrics:      spanMetrics,
			AgentSpanMetrics: forSurges,
		}})
		pl := cfg.Service.Pipelines[odigosMetricsPipelineName]
		assert.Contains(t, pl.Processors, dropAgentSpanMetricsName, "they go to odigos insights only")
		assert.Equal(t, odigosTrafficMetricsProcessorName, pl.Processors[len(pl.Processors)-1])
		assert.Contains(t, cfg.Processors, dropAgentSpanMetricsName)
	}

	cfg := MetricsConfig(&odigosv1.CollectorsGroup{}, MetricsConfigOptions{MetricsConfigSettings: &odigosv1.CollectorsGroupMetricsCollectionSettings{
		AgentsTelemetry:  &odigosv1.AgentsTelemetrySettings{},
		AgentSpanMetrics: &odigosv1.AgentSpanMetricsSettings{Destinations: true, Insights: true},
	}})
	assert.NotContains(t, cfg.Service.Pipelines[odigosMetricsPipelineName].Processors, dropAgentSpanMetricsName,
		"span metrics in the agents are enabled: they go to the destinations with the agents' other metrics")

	cfg = MetricsConfig(&odigosv1.CollectorsGroup{}, MetricsConfigOptions{MetricsConfigSettings: &odigosv1.CollectorsGroupMetricsCollectionSettings{
		SpanMetrics:      &common.MetricsSourceSpanMetricsConfiguration{},
		AgentSpanMetrics: forSurges,
	}})
	assert.Nil(t, cfg.Processors, "the metrics pipeline does not receive the agents' telemetry")
	assert.NotContains(t, cfg.Service.Pipelines, odigosMetricsPipelineName)
}
