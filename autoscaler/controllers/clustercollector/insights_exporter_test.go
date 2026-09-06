package clustercollector

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	odigosconsts "github.com/odigos-io/odigos/common/consts"
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
	// Must target the headless Service via dns:/// with round_robin so the
	// gateway load balances across insights replicas instead of pinning one pod.
	assert.Equal(t, k8sconsts.InsightsOtlpGrpcDNSEndpoint("odigos-system"), exp["endpoint"])
	assert.Equal(t, "dns:///odigos-insights-headless.odigos-system:4317", exp["endpoint"])
	assert.Equal(t, "round_robin", exp["balancer_name"])
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

func TestEffectiveInsightsConfig(t *testing.T) {
	on := true
	cfg := &common.InsightsConfiguration{Enabled: &on}

	// odigos-insights ships as the enterprise-insights image and helm renders none of its
	// workloads without an on-prem token, so the flag must be ignored on community tier.
	assert.Nil(t, effectiveInsightsConfig(cfg, common.CommunityOdigosTier),
		"community tier must not wire the gateway for a service that was never deployed")
	assert.Same(t, cfg, effectiveInsightsConfig(cfg, common.OnPremOdigosTier))
	assert.Same(t, cfg, effectiveInsightsConfig(cfg, common.CloudOdigosTier))
	assert.Nil(t, effectiveInsightsConfig(nil, common.OnPremOdigosTier))
}

// insightsGatewayConfig renders a gateway config the way syncConfigMap does, for a cluster
// whose odigos configuration has insights enabled and no destinations at all.
func insightsGatewayConfig(t *testing.T, tier common.OdigosTier) *config.Config {
	t.Helper()

	on := true
	insightsCfg := effectiveInsightsConfig(&common.InsightsConfiguration{Enabled: &on}, tier)

	ext := k8sconsts.OdigosConfigK8sExtensionType
	tailSampling := false
	gatewayOptions := pipelinegen.GatewayConfigOptions{
		OdigosNamespace:           "odigos-system",
		OdigosConfigExtensionName: &ext,
		TailSamplingEnabled:       &tailSampling,
		Insights:                  insightsCfg,
	}
	if common.InsightsPipelineActive(insightsCfg) {
		gatewayOptions.InsightsOtlpEndpoint = k8sconsts.InsightsOtlpGrpcDNSEndpoint("odigos-system")
		waitDuration := k8sconsts.OdigosClusterCollectorTraceAggregationWaitDurationDefault
		gatewayOptions.TraceAggregationWaitDuration = &waitDuration
	}

	cfg, err, _, signals := pipelinegen.CalculateGatewayConfig(nil, nil,
		func(c *config.Config, destinationPipelineNames []string, signalsRootPipelines []string) error {
			if err := addSelfTelemetryPipeline(c, 8888, destinationPipelineNames, signalsRootPipelines); err != nil {
				return err
			}
			return addInsightsGatewayExporter(c, "odigos-system", insightsCfg)
		},
		nil, &gatewayOptions)
	require.NoError(t, err)

	if tier.IsEnterprise() {
		assert.Contains(t, signals, common.TracesObservabilitySignal,
			"insights taps the traces pipeline, so traces must be advertised to the agents")
	} else {
		assert.Empty(t, signals,
			"no destinations and no insights means the gateway should ask the agents for nothing")
	}
	return cfg
}

// Enabling insights without an on-prem token used to leave the odigos configuration saying
// insights is on while helm rendered no odigos-insights workloads, so the gateway turned the
// traces signal on cluster-wide, buffered every span in groupbytrace and exported to a Service
// that does not exist.
func TestGatewayConfig_InsightsIgnoredOnCommunityTier(t *testing.T) {
	cfg := insightsGatewayConfig(t, common.CommunityOdigosTier)

	assert.NotContains(t, cfg.Exporters, commonconf.InsightsGatewayExporter)
	assert.NotContains(t, cfg.Exporters, odigosconsts.ServiceGraphInsightsExporterName)
	assert.NotContains(t, cfg.Processors, odigosconsts.GroupByTraceProcessor)
	assert.NotContains(t, cfg.Service.Pipelines, pipelinegen.GetTelemetryRootPipelineName(common.TracesObservabilitySignal))
}

func TestGatewayConfig_InsightsWiredOnEnterpriseTier(t *testing.T) {
	cfg := insightsGatewayConfig(t, common.OnPremOdigosTier)

	assert.Contains(t, cfg.Exporters, commonconf.InsightsGatewayExporter)
	assert.Contains(t, cfg.Exporters, odigosconsts.ServiceGraphInsightsExporterName)
	assert.Contains(t, cfg.Processors, odigosconsts.GroupByTraceProcessor)

	rootPipe := cfg.Service.Pipelines[pipelinegen.GetTelemetryRootPipelineName(common.TracesObservabilitySignal)]
	assert.Contains(t, rootPipe.Exporters, commonconf.InsightsGatewayExporter)
	// Every pipeline still needs an exporter or the collector refuses to start.
	for name, pipeline := range cfg.Service.Pipelines {
		assert.NotEmpty(t, pipeline.Exporters, "pipeline %q has no exporters", name)
	}
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
