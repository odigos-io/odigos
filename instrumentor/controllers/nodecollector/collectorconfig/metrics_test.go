package collectorconfig

import (
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
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
		{SpanMetrics: &common.MetricsSourceSpanMetricsConfiguration{}},
	} {
		cfg := MetricsConfig(&odigosv1.CollectorsGroup{}, MetricsConfigOptions{MetricsConfigSettings: settings})
		assert.Nil(t, cfg.Processors)
		assert.NotContains(t, cfg.Service.Pipelines, AgentSpanMetricsDestinationsPipelineName)
		if pl, ok := cfg.Service.Pipelines[odigosMetricsPipelineName]; ok {
			assert.NotContains(t, pl.Processors, dropAgentSpanMetricsName)
		}
	}
}

func TestMetricsConfig_AgentSpanMetricsReachDestinationsOnlyWhenWanted(t *testing.T) {
	spanMetrics := &common.MetricsSourceSpanMetricsConfiguration{Interval: "60s"}
	forSurges := &odigosv1.AgentSpanMetricsSettings{Insights: true}

	cfg := MetricsConfig(&odigosv1.CollectorsGroup{}, MetricsConfigOptions{MetricsConfigSettings: &odigosv1.CollectorsGroupMetricsCollectionSettings{
		AgentsTelemetry:  &odigosv1.AgentsTelemetrySettings{},
		AgentSpanMetrics: forSurges,
	}})
	assert.Contains(t, cfg.Service.Pipelines[odigosMetricsPipelineName].Processors, dropAgentSpanMetricsName,
		"no destination wants span metrics: the ones agents record for trace surges are dropped")

	cfg = MetricsConfig(&odigosv1.CollectorsGroup{}, MetricsConfigOptions{MetricsConfigSettings: &odigosv1.CollectorsGroupMetricsCollectionSettings{
		AgentsTelemetry:  &odigosv1.AgentsTelemetrySettings{},
		AgentSpanMetrics: &odigosv1.AgentSpanMetricsSettings{Destinations: true, Insights: true},
	}})
	assert.NotContains(t, cfg.Service.Pipelines[odigosMetricsPipelineName].Processors, dropAgentSpanMetricsName,
		"span metrics in the agents are enabled: they go to the destinations with the agents' other metrics")

	for _, agentsTelemetry := range []*odigosv1.AgentsTelemetrySettings{{}, nil} {
		cfg = MetricsConfig(&odigosv1.CollectorsGroup{}, MetricsConfigOptions{MetricsConfigSettings: &odigosv1.CollectorsGroupMetricsCollectionSettings{
			AgentsTelemetry:  agentsTelemetry,
			SpanMetrics:      spanMetrics,
			AgentSpanMetrics: forSurges,
		}})
		if agentsTelemetry != nil {
			assert.Contains(t, cfg.Service.Pipelines[odigosMetricsPipelineName].Processors, dropAgentSpanMetricsName,
				"they reach the destinations on their own, not with the agents' other metrics")
		} else {
			assert.NotContains(t, cfg.Service.Pipelines, odigosMetricsPipelineName, "no other metrics source")
		}
		pl, ok := cfg.Service.Pipelines[AgentSpanMetricsDestinationsPipelineName]
		require.True(t, ok, "a destination wants span metrics: the agents' take the place of the connector's")
		assert.Equal(t, []string{OTLPInReceiverName}, pl.Receivers)
		assert.Equal(t, []string{keepAgentSpanMetricsName, agentSpanMetricsRemoveDimensionsName}, pl.Processors[len(pl.Processors)-2:])
	}
}

// The agents' span metrics take the place of the connector's: they lose the resource attributes the
// connector's are aggregated without.
func TestAgentSpanMetricsToDestinationsDropTheConnectorsExcludedResourceAttributes(t *testing.T) {
	included := true
	for _, spanMetrics := range []common.MetricsSourceSpanMetricsConfiguration{
		{Interval: "60s"},
		{Interval: "60s", ExcludedResourceAttributes: []string{"k8s.pod.uid"}, IncludedProcessInDimensions: &included},
	} {
		connector, _, _, _ := GetSpanMetricsConfig(spanMetrics, true)
		cfg := MetricsConfig(&odigosv1.CollectorsGroup{}, MetricsConfigOptions{MetricsConfigSettings: &odigosv1.CollectorsGroupMetricsCollectionSettings{
			SpanMetrics:      &spanMetrics,
			AgentSpanMetrics: &odigosv1.AgentSpanMetricsSettings{Destinations: true},
		}})
		assert.Equal(t, connector.Processors[spanMetricsResourceRemoveDimensionsProcessorName], cfg.Processors[agentSpanMetricsRemoveDimensionsName])
	}
}

func TestSpanMetricsConnectorSkipsAgentRecordedSpans(t *testing.T) {
	cfg, _, _, _ := GetSpanMetricsConfig(common.MetricsSourceSpanMetricsConfiguration{}, true)
	pl := cfg.Service.Pipelines[spanMetricsPipelineName]
	require.NotEmpty(t, pl.Processors)
	assert.Equal(t, spanMetricsSkipAgentRecordedProcessorName, pl.Processors[0], "before anything else in the connector's pipeline")
	assert.Equal(t, []string{`instrumentation_scope.attributes["odigos.span_metrics.recorded"] == true`},
		cfg.Processors[spanMetricsSkipAgentRecordedProcessorName].(config.GenericMap)["traces"].(config.GenericMap)["span"])

	cfg, _, _, _ = GetSpanMetricsConfig(common.MetricsSourceSpanMetricsConfiguration{}, false)
	assert.NotContains(t, cfg.Service.Pipelines[spanMetricsPipelineName].Processors, spanMetricsSkipAgentRecordedProcessorName,
		"no agent records span metrics: the connector's pipeline is unchanged")
	assert.NotContains(t, cfg.Processors, spanMetricsSkipAgentRecordedProcessorName)
}

func TestAgentSpanMetricsToDestinationsGetTheConfiguredMetricsProcessors(t *testing.T) {
	settings := &odigosv1.CollectorsGroupMetricsCollectionSettings{
		AgentsTelemetry:  &odigosv1.AgentsTelemetrySettings{},
		SpanMetrics:      &common.MetricsSourceSpanMetricsConfiguration{Interval: "60s"},
		AgentSpanMetrics: &odigosv1.AgentSpanMetricsSettings{Insights: true},
	}
	opts := MetricsConfigOptions{MetricsConfigSettings: settings}
	opts.ManifestProcessorNames = []string{"attributes/odigos-user-processor"}
	cfg := MetricsConfig(&odigosv1.CollectorsGroup{}, opts)
	assert.Contains(t, cfg.Service.Pipelines[odigosMetricsPipelineName].Processors, "attributes/odigos-user-processor")
	pl := cfg.Service.Pipelines[AgentSpanMetricsDestinationsPipelineName]
	assert.Equal(t, []string{keepAgentSpanMetricsName, "attributes/odigos-user-processor", agentSpanMetricsRemoveDimensionsName}, pl.Processors[len(pl.Processors)-3:],
		"the node's configured metrics processors apply to them too")
}
