package clustercollector

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	pipelinegen "github.com/odigos-io/odigos/common/pipelinegen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// configWithTracesIn returns a minimal config with a root traces pipeline
// shaped the way pipelinegen.applyRootPipelineForSignal builds it. We
// exercise addInsightsGatewayExporter against this fixture instead of
// running the full GetGatewayConfig flow because we only care about what
// the tap appends.
func configWithTracesIn() *config.Config {
	rootName := pipelinegen.GetTelemetryRootPipelineName(common.TracesObservabilitySignal)
	return &config.Config{
		Connectors: config.GenericMap{
			"odigosrouterconnector/traces": config.GenericMap{},
		},
		Service: config.Service{
			Pipelines: map[string]config.Pipeline{
				rootName: {
					Receivers:  []string{"otlp"},
					Processors: []string{"resource/odigos-version", "transform/url-template"},
					Exporters:  []string{"odigosrouterconnector/traces"},
				},
			},
		},
	}
}

func TestAddInsightsGatewayExporter_Disabled(t *testing.T) {
	t.Run("nil_config_noop", func(t *testing.T) {
		c := configWithTracesIn()
		require.NoError(t, addInsightsGatewayExporter(c, "odigos-system", nil))

		_, hasExp := c.Exporters[commonconf.InsightsGatewayExporter]
		assert.False(t, hasExp, "exporter must not be registered when feature is off")

		rootPipe := c.Service.Pipelines[pipelinegen.GetTelemetryRootPipelineName(common.TracesObservabilitySignal)]
		assert.Equal(t, []string{"odigosrouterconnector/traces"}, rootPipe.Exporters,
			"root pipeline's exporter list must not be touched when feature is off")
	})

	t.Run("explicit_false_noop", func(t *testing.T) {
		off := false
		c := configWithTracesIn()
		require.NoError(t, addInsightsGatewayExporter(c, "odigos-system", &common.InsightsConfiguration{Enabled: &off}))

		_, hasExp := c.Exporters[commonconf.InsightsGatewayExporter]
		assert.False(t, hasExp)
	})
}

func TestAddInsightsGatewayExporter_NoTracesInPipelineNoop(t *testing.T) {
	on := true
	c := &config.Config{Service: config.Service{Pipelines: map[string]config.Pipeline{}}}
	require.NoError(t, addInsightsGatewayExporter(c, "odigos-system", &common.InsightsConfiguration{Enabled: &on}))

	_, hasExp := c.Exporters[commonconf.InsightsGatewayExporter]
	assert.False(t, hasExp, "exporter must not be registered when there is no root traces pipeline to tap")
}

func TestAddInsightsGatewayExporter_EnabledAppendsExporterToRootPipeline(t *testing.T) {
	on := true
	c := configWithTracesIn()
	require.NoError(t, addInsightsGatewayExporter(c, "odigos-system", &common.InsightsConfiguration{Enabled: &on}))

	exp, ok := c.Exporters[commonconf.InsightsGatewayExporter].(config.GenericMap)
	require.True(t, ok, "exporter must be registered")
	assert.Equal(t, k8sconsts.InsightsOtlpGrpcEndpoint("odigos-system"), exp["endpoint"])
	assert.Equal(t, "none", exp["compression"])
	tls, _ := exp["tls"].(config.GenericMap)
	assert.Equal(t, true, tls["insecure"])

	// The root pipeline must have the new exporter appended after the
	// existing destination router, and its processor chain must be untouched.
	rootPipe := c.Service.Pipelines[pipelinegen.GetTelemetryRootPipelineName(common.TracesObservabilitySignal)]
	assert.Equal(t, []string{"resource/odigos-version", "transform/url-template"}, rootPipe.Processors,
		"root pipeline processors must be preserved verbatim")
	assert.Equal(t, []string{"odigosrouterconnector/traces", commonconf.InsightsGatewayExporter}, rootPipe.Exporters,
		"side-channel exporter must be appended alongside the existing destination router exporter")
}

func TestAddInsightsGatewayExporter_DoesNotCreateExtraPipelineOrConnector(t *testing.T) {
	// Tap is a single exporter, not a pipeline. Lock in that invariant so a
	// future refactor doesn't accidentally reintroduce a per-pipeline batch /
	// groupbytrace / forward connector for the side channel.
	on := true
	c := configWithTracesIn()
	rootName := pipelinegen.GetTelemetryRootPipelineName(common.TracesObservabilitySignal)
	pipelinesBefore := len(c.Service.Pipelines)
	connectorsBefore := len(c.Connectors)

	require.NoError(t, addInsightsGatewayExporter(c, "odigos-system", &common.InsightsConfiguration{Enabled: &on}))

	assert.Equal(t, pipelinesBefore, len(c.Service.Pipelines),
		"no new pipeline must be created; the tap is just an extra exporter on the root pipeline")
	assert.Equal(t, connectorsBefore, len(c.Connectors),
		"no new connector must be created")
	_, hasRoot := c.Service.Pipelines[rootName]
	assert.True(t, hasRoot, "root pipeline must still exist (just with one more exporter)")
}

func TestAddInsightsGatewayExporter_PreservesExistingDestinationConfig(t *testing.T) {
	on := true
	c := configWithTracesIn()
	c.Exporters = config.GenericMap{"otlp/dest1": config.GenericMap{"endpoint": "x:4317"}}
	c.Service.Pipelines["traces/dest1"] = config.Pipeline{
		Receivers: []string{"forward/traces/dest1"}, Processors: []string{"batch"}, Exporters: []string{"otlp/dest1"},
	}

	require.NoError(t, addInsightsGatewayExporter(c, "odigos-system", &common.InsightsConfiguration{Enabled: &on}))

	_, dest := c.Exporters["otlp/dest1"]
	_, ins := c.Exporters[commonconf.InsightsGatewayExporter]
	assert.True(t, dest, "destination exporter must not be removed")
	assert.True(t, ins, "side-channel exporter must be added")

	_, destPipe := c.Service.Pipelines["traces/dest1"]
	assert.True(t, destPipe, "destination pipeline must not be removed")
}
