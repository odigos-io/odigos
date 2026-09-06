package clustercollector

import (
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	pipelinegen "github.com/odigos-io/odigos/common/pipelinegen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// brokenPrometheusDest is a metrics destination whose exporter config cannot be rendered (no
// PROMETHEUS_REMOTEWRITE_URL), so CalculateGatewayConfig records an error for it and it
// contributes no metrics pipeline.
type brokenPrometheusDest struct{ id string }

func (d brokenPrometheusDest) GetID() string { return d.id }
func (d brokenPrometheusDest) GetType() common.DestinationType {
	return common.PrometheusDestinationType
}
func (d brokenPrometheusDest) GetConfig() map[string]string { return map[string]string{} }
func (d brokenPrometheusDest) GetSignals() []common.ObservabilitySignal {
	return []common.ObservabilitySignal{common.MetricsObservabilitySignal}
}

// healthyJaegerDest is a traces destination that renders correctly, standing in for the rest of
// the user's pipeline that must keep working when another destination is misconfigured.
type healthyJaegerDest struct{ id string }

func (d healthyJaegerDest) GetID() string                   { return d.id }
func (d healthyJaegerDest) GetType() common.DestinationType { return common.JaegerDestinationType }
func (d healthyJaegerDest) GetConfig() map[string]string {
	return map[string]string{config.JaegerUrlKey: "jaeger.example.com:4317"}
}
func (d healthyJaegerDest) GetSignals() []common.ObservabilitySignal {
	return []common.ObservabilitySignal{common.TracesObservabilitySignal}
}

func newGatewayConfigForOwnMetrics() *config.Config {
	return &config.Config{
		Receivers: config.GenericMap{},
		Exporters: config.GenericMap{},
		Service:   config.Service{Pipelines: map[string]config.Pipeline{}},
	}
}

// A misconfigured metrics destination must not be able to invalidate the whole gateway config.
// With the odigos metrics store disabled, own metrics have only the metrics destinations to go to,
// and a destination that fails to render contributes no metrics pipeline to borrow exporters from.
func TestGatewayConfig_OwnMetricsWithNoRenderedMetricsDestination(t *testing.T) {
	ownMetrics := &odigosv1.OdigosOwnMetricsSettings{
		// ownTelemetry.metricsStore.disabled=true in the helm values
		SendToOdigosMetricsStore: false,
		// a metrics destination sets metricsSettings.odigosOwnMetricsEnabled=true
		SendToMetricsDestinations: true,
		Interval:                  "10s",
	}

	gatewayOptions := pipelinegen.GatewayConfigOptions{OdigosNamespace: "odigos-system"}
	cfg, err, statuses, _ := pipelinegen.CalculateGatewayConfig(
		[]config.ExporterConfigurer{
			brokenPrometheusDest{id: "p1"},
			healthyJaegerDest{id: "j1"},
		},
		nil,
		func(c *config.Config, destinationPipelineNames []string, signalsRootPipelines []string) error {
			if err := addSelfTelemetryPipeline(c, 8888, destinationPipelineNames, signalsRootPipelines); err != nil {
				return err
			}
			return addOwnMetricsPipeline(c, ownMetrics, "odigos-system", 8888, destinationPipelineNames)
		},
		nil, &gatewayOptions)
	require.NoError(t, err)

	// The broken destination is reported and skipped, while the healthy one stays wired up.
	require.Error(t, statuses.Destination["p1"])
	assert.NotContains(t, cfg.Service.Pipelines, "metrics/prometheus-p1")
	require.Contains(t, cfg.Service.Pipelines, "traces/jaeger-j1")

	// The collector requires at least one exporter per pipeline and refuses to start otherwise,
	// which would stop every signal to every destination - not just own metrics.
	assert.NotContains(t, cfg.Service.Pipelines, ownMetricsStorePipelineName)
	for name, pipeline := range cfg.Service.Pipelines {
		assert.NotEmpty(t, pipeline.Exporters, "pipeline %q has no exporters", name)
	}
}

func TestAddOwnMetricsPipeline_NoMetricsDestinationPipeline(t *testing.T) {
	c := newGatewayConfigForOwnMetrics()

	err := addOwnMetricsPipeline(c,
		&odigosv1.OdigosOwnMetricsSettings{SendToMetricsDestinations: true},
		"odigos-system", 8888, []string{"traces/jaeger-j1"})
	require.NoError(t, err)

	assert.NotContains(t, c.Service.Pipelines, ownMetricsStorePipelineName)
	assert.NotContains(t, c.Receivers, odigosOwnTelemetryOtlpReceiverName)
}

func TestAddOwnMetricsPipeline_MetricsDestinationPipeline(t *testing.T) {
	c := newGatewayConfigForOwnMetrics()
	c.Service.Pipelines["metrics/prometheus-p1"] = config.Pipeline{
		Exporters: []string{"prometheusremotewrite/prometheus-p1"},
	}

	err := addOwnMetricsPipeline(c,
		&odigosv1.OdigosOwnMetricsSettings{SendToMetricsDestinations: true},
		"odigos-system", 8888, []string{"metrics/prometheus-p1"})
	require.NoError(t, err)

	require.Contains(t, c.Receivers, odigosOwnTelemetryOtlpReceiverName)
	pipeline := c.Service.Pipelines[ownMetricsStorePipelineName]
	assert.Equal(t, []string{odigosOwnTelemetryOtlpReceiverName}, pipeline.Receivers)
	assert.Equal(t, []string{"prometheusremotewrite/prometheus-p1"}, pipeline.Exporters)
}

func TestAddOwnMetricsPipeline_OdigosMetricsStore(t *testing.T) {
	c := newGatewayConfigForOwnMetrics()

	err := addOwnMetricsPipeline(c,
		&odigosv1.OdigosOwnMetricsSettings{SendToOdigosMetricsStore: true, Interval: "10s"},
		"odigos-system", 8888, nil)
	require.NoError(t, err)

	pipeline := c.Service.Pipelines[ownMetricsStorePipelineName]
	assert.Equal(t,
		[]string{odigosOwnTelemetryOtlpReceiverName, gatewayPrometheusReceiverName},
		pipeline.Receivers)
	assert.Equal(t, []string{odigosVictoriametricsExporterName}, pipeline.Exporters)
	require.Contains(t, c.Exporters, odigosVictoriametricsExporterName)
}
