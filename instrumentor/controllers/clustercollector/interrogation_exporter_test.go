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
					Receivers:  []string{"otlp"},
					Exporters:  []string{"odigosrouterconnector/traces"},
				},
			},
		},
	}
}

func TestAddInterrogationExporters_Disabled(t *testing.T) {
	t.Run("nil_config_noop", func(t *testing.T) {
		c := configWithProfilesPipeline()
		require.NoError(t, addInterrogationExporters(c, "odigos-system", nil, nil, "30s"))

		_, hasExp := c.Exporters[pipeline.InterrogationProfilesExporter]
		assert.False(t, hasExp)

		pl := c.Service.Pipelines[gatewayProfilesPipeline]
		assert.Equal(t, []string{pipeline.ProfilingGatewayToUIExporter}, pl.Exporters)
	})

	t.Run("explicit_false_noop", func(t *testing.T) {
		off := false
		c := configWithProfilesPipeline()
		require.NoError(t, addInterrogationExporters(c, "odigos-system", &common.InterrogationConfiguration{Enabled: &off}, nil, "30s"))

		_, hasExp := c.Exporters[pipeline.InterrogationProfilesExporter]
		assert.False(t, hasExp)
	})
}

func TestAddInterrogationExporters_NoPipelinesNoop(t *testing.T) {
	on := true
	c := &config.Config{Service: config.Service{Pipelines: map[string]config.Pipeline{}}}
	require.NoError(t, addInterrogationExporters(c, "odigos-system", &common.InterrogationConfiguration{Enabled: &on}, nil, "30s"))

	_, hasProfiles := c.Exporters[pipeline.InterrogationProfilesExporter]
	_, hasTraces := c.Exporters[pipeline.InterrogationTracesExporter]
	assert.False(t, hasProfiles)
	assert.False(t, hasTraces)
}

func TestAddInterrogationExporters_EnabledAppendsToProfilesPipeline(t *testing.T) {
	on := true
	c := configWithProfilesPipeline()
	require.NoError(t, addInterrogationExporters(c, "odigos-system", &common.InterrogationConfiguration{Enabled: &on}, nil, "30s"))

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

	_, hasTraces := c.Exporters[pipeline.InterrogationTracesExporter]
	assert.False(t, hasTraces, "traces exporter requires a root traces pipeline")
}

func TestAddInterrogationExporters_EnabledAppendsToTracesPipeline(t *testing.T) {
	on := true
	c := configWithTracesRootPipeline()
	require.NoError(t, addInterrogationExporters(c, "odigos-system", &common.InterrogationConfiguration{Enabled: &on}, nil, "30s"))

	exp, ok := c.Exporters[pipeline.InterrogationTracesExporter].(config.GenericMap)
	require.True(t, ok, "traces exporter must be registered")
	_, hasCacheExt := exp["interrogation_cache_extension"]
	assert.False(t, hasCacheExt, "traces exporter does not use in-process cache")
	assert.Equal(t, "tcp://odigos-insights-clickhouse.odigos-system:9000", exp["clickhouse_endpoint"])
	assert.Equal(t, "${ODIGOS_INSIGHTS_CLICKHOUSE_PASSWORD}", exp["clickhouse_password"])
	assert.Equal(t, "odigos-cache.odigos-system:6379", exp["redis_endpoint"])

	rootName := pipelinegen.GetTelemetryRootPipelineName(common.TracesObservabilitySignal)
	pl := c.Service.Pipelines[rootName]
	assert.Equal(t, []string{"odigosrouterconnector/traces", pipeline.InterrogationTracesExporter}, pl.Exporters)

	_, hasProfiles := c.Exporters[pipeline.InterrogationProfilesExporter]
	assert.False(t, hasProfiles, "profiles exporter requires a profiles pipeline")
}

func TestAddInterrogationExporters_BothPipelines(t *testing.T) {
	on := true
	c := configWithProfilesPipeline()
	rootName := pipelinegen.GetTelemetryRootPipelineName(common.TracesObservabilitySignal)
	c.Service.Pipelines[rootName] = config.Pipeline{
		Receivers: []string{"otlp"},
		Exporters: []string{"odigosrouterconnector/traces"},
	}

	require.NoError(t, addInterrogationExporters(c, "odigos-system", &common.InterrogationConfiguration{Enabled: &on}, nil, "45s"))

	_, hasProfiles := c.Exporters[pipeline.InterrogationProfilesExporter]
	_, hasTraces := c.Exporters[pipeline.InterrogationTracesExporter]
	assert.True(t, hasProfiles)
	assert.True(t, hasTraces)
	profilesExp := c.Exporters[pipeline.InterrogationProfilesExporter].(config.GenericMap)
	assert.Equal(t, "45s", profilesExp["group_by_trace_wait_duration"])
}

func TestAddInterrogationExporters_TransactionIdentityDimensions(t *testing.T) {
	on := true
	c := configWithTracesRootPipeline()
	require.NoError(t, addInterrogationExporters(c, "odigos-system", &common.InterrogationConfiguration{Enabled: &on},
		&common.TransactionIdentityConfiguration{
			Dimensions: []string{"http.response.status_code", "rpc.grpc.status_code"},
		}, "30s"))

	exp, ok := c.Exporters[pipeline.InterrogationTracesExporter].(config.GenericMap)
	require.True(t, ok)
	assert.Equal(t, []string{"http.response.status_code", "rpc.grpc.status_code"}, exp["transaction_identity_dimensions"])
}

func TestAddInterrogationExporters_TransactionIdentityDimensionsOmittedWhenEmpty(t *testing.T) {
	on := true
	c := configWithTracesRootPipeline()
	require.NoError(t, addInterrogationExporters(c, "odigos-system", &common.InterrogationConfiguration{Enabled: &on},
		&common.TransactionIdentityConfiguration{Dimensions: nil}, "30s"))

	exp := c.Exporters[pipeline.InterrogationTracesExporter].(config.GenericMap)
	_, hasDims := exp["transaction_identity_dimensions"]
	assert.False(t, hasDims)
}
