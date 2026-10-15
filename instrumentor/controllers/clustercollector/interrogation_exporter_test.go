package clustercollector

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	pipelinegen "github.com/odigos-io/odigos/common/pipelinegen"
	"github.com/odigos-io/odigos/instrumentor/controllers/pipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func configWithProfilesPipeline() *config.Config {
	return &config.Config{
		Exporters: config.GenericMap{
			pipeline.ProfilingGatewayToUIExporter: config.GenericMap{},
		},
		Service: config.Service{
			Pipelines: map[string]config.Pipeline{
				gatewayProfilesPipeline: {
					Receivers: []string{"otlp"},
					Exporters: []string{pipeline.ProfilingGatewayToUIExporter},
				},
			},
		},
	}
}

func configWithTracesRootPipeline() *config.Config {
	rootName := pipelinegen.GetTelemetryRootPipelineName(common.TracesObservabilitySignal)
	return &config.Config{
		Exporters: config.GenericMap{
			"odigosrouterconnector/traces": config.GenericMap{},
		},
		Service: config.Service{
			Pipelines: map[string]config.Pipeline{
				rootName: {
					Receivers: []string{"otlp"},
					Exporters: []string{"odigosrouterconnector/traces"},
				},
			},
		},
	}
}

func TestAddInterrogationExporters_Disabled(t *testing.T) {
	t.Run("nil_config_noop", func(t *testing.T) {
		c := configWithProfilesPipeline()
		require.NoError(t, addInterrogationExporters(c, "odigos-system", nil, "30s"))

		_, hasExp := c.Exporters[pipeline.InterrogationProfilesExporter]
		assert.False(t, hasExp)

		pl := c.Service.Pipelines[gatewayProfilesPipeline]
		assert.Equal(t, []string{pipeline.ProfilingGatewayToUIExporter}, pl.Exporters)
	})

	t.Run("explicit_false_noop", func(t *testing.T) {
		off := false
		c := configWithProfilesPipeline()
		require.NoError(t, addInterrogationExporters(c, "odigos-system", &common.InterrogationConfiguration{Enabled: &off}, "30s"))

		_, hasExp := c.Exporters[pipeline.InterrogationProfilesExporter]
		assert.False(t, hasExp)
	})
}

func TestAddInterrogationExporters_NoPipelinesNoop(t *testing.T) {
	on := true
	c := &config.Config{Service: config.Service{Pipelines: map[string]config.Pipeline{}}}
	require.NoError(t, addInterrogationExporters(c, "odigos-system", &common.InterrogationConfiguration{Enabled: &on}, "30s"))

	_, hasProfiles := c.Exporters[pipeline.InterrogationProfilesExporter]
	assert.False(t, hasProfiles)
}

func TestAddInterrogationExporters_EnabledAppendsToProfilesPipeline(t *testing.T) {
	on := true
	c := configWithProfilesPipeline()
	require.NoError(t, addInterrogationExporters(c, "odigos-system", &common.InterrogationConfiguration{Enabled: &on}, "30s"))

	exp, ok := c.Exporters[pipeline.InterrogationProfilesExporter].(config.GenericMap)
	require.True(t, ok, "profiles exporter must be registered")
	assert.Equal(t, "odigos-cache.odigos-system:6379", exp["redis_endpoint"])
	assert.Equal(t, "30s", exp["group_by_trace_wait_duration"])
	_, hasCH := exp["clickhouse_endpoint"]
	assert.False(t, hasCH, "profiles exporter writes stacks to Redis only")
	_, hasCacheExt := exp["interrogation_cache_extension"]
	assert.False(t, hasCacheExt, "profiles exporter does not use in-process cache")

	pl := c.Service.Pipelines[gatewayProfilesPipeline]
	assert.Equal(t, []string{"otlp"}, pl.Receivers)
	assert.Equal(t, []string{pipeline.ProfilingGatewayToUIExporter, pipeline.InterrogationProfilesExporter}, pl.Exporters)
}

func TestAddInterrogationExporters_TracesPipelineDoesNotAddTracesExporter(t *testing.T) {
	on := true
	c := configWithTracesRootPipeline()
	require.NoError(t, addInterrogationExporters(c, "odigos-system", &common.InterrogationConfiguration{Enabled: &on}, "30s"))

	_, hasTraces := c.Exporters[pipeline.InterrogationTracesExporter]
	assert.False(t, hasTraces, "call-trie writes moved to insights; traces exporter not wired")

	rootName := pipelinegen.GetTelemetryRootPipelineName(common.TracesObservabilitySignal)
	pl := c.Service.Pipelines[rootName]
	assert.Equal(t, []string{"odigosrouterconnector/traces"}, pl.Exporters)

	_, hasProfiles := c.Exporters[pipeline.InterrogationProfilesExporter]
	assert.False(t, hasProfiles, "profiles exporter requires a profiles pipeline")
}

func TestAddInterrogationExporters_BothPipelinesProfilesOnly(t *testing.T) {
	on := true
	c := configWithProfilesPipeline()
	rootName := pipelinegen.GetTelemetryRootPipelineName(common.TracesObservabilitySignal)
	c.Service.Pipelines[rootName] = config.Pipeline{
		Receivers: []string{"otlp"},
		Exporters: []string{"odigosrouterconnector/traces"},
	}

	require.NoError(t, addInterrogationExporters(c, "odigos-system", &common.InterrogationConfiguration{Enabled: &on}, "45s"))

	_, hasProfiles := c.Exporters[pipeline.InterrogationProfilesExporter]
	_, hasTraces := c.Exporters[pipeline.InterrogationTracesExporter]
	assert.True(t, hasProfiles)
	assert.False(t, hasTraces)
	profilesExp := c.Exporters[pipeline.InterrogationProfilesExporter].(config.GenericMap)
	assert.Equal(t, "45s", profilesExp["group_by_trace_wait_duration"])

	pl := c.Service.Pipelines[rootName]
	assert.Equal(t, []string{"odigosrouterconnector/traces"}, pl.Exporters)
}
