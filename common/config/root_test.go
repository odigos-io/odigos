package config_test

import (
	"os"
	"slices"
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/common/pipelinegen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)


type DummyProcessor struct {
	ID        string
	Type      string
	OrderHint int
}

func (proc DummyProcessor) GetID() string {
	return proc.ID
}

func (proc DummyProcessor) GetConfig() (config.GenericMap, error) {
	return make(config.GenericMap), nil
}

func (proc DummyProcessor) GetSignals() []common.ObservabilitySignal {
	return []common.ObservabilitySignal{common.TracesObservabilitySignal, common.LogsObservabilitySignal}
}

func (proc DummyProcessor) GetType() string {
	if proc.Type != "" {
		return proc.Type
	}
	return "resource"
}

func (proc DummyProcessor) GetOrderHint() int {
	return proc.OrderHint
}

type DummyDestination struct {
	ID string
}

type DummyTraceDestination struct {
	ID string
}

func (dest DummyDestination) GetID() string {
	return dest.ID
}
func (dest DummyDestination) GetType() common.DestinationType {
	return "debug"
}
func (dest DummyDestination) GetConfig() map[string]string {
	return make(map[string]string)
}
func (dest DummyDestination) GetSignals() []common.ObservabilitySignal {
	return []common.ObservabilitySignal{common.LogsObservabilitySignal}
}

func (dest DummyTraceDestination) GetID() string                   { return dest.ID }
func (dest DummyTraceDestination) GetType() common.DestinationType { return "debug" }
func (dest DummyTraceDestination) GetConfig() map[string]string    { return make(map[string]string) }
func (dest DummyTraceDestination) GetSignals() []common.ObservabilitySignal {
	return []common.ObservabilitySignal{common.TracesObservabilitySignal}
}

// a destination whose spec.data is carried verbatim, used to drive a real configer with the
// unvalidated values a Destination CR can hold (there is no validating webhook on Destination).
type rawDataDestination struct {
	ID      string
	Type    common.DestinationType
	Data    map[string]string
	Signals []common.ObservabilitySignal
}

func (dest rawDataDestination) GetID() string                   { return dest.ID }
func (dest rawDataDestination) GetType() common.DestinationType { return dest.Type }
func (dest rawDataDestination) GetConfig() map[string]string    { return dest.Data }
func (dest rawDataDestination) GetSignals() []common.ObservabilitySignal {
	return dest.Signals
}

// A non numeric value in one destination's spec.data must fail only that destination. Previously
// parseInt panicked, which aborted the whole gateway config computation and left every other
// destination unconfigured.
func TestCalculateBadNumericDestinationDataDoesNotAbortTheGatewayConfig(t *testing.T) {
	gatewayOptions := pipelinegen.GatewayConfigOptions{
		OdigosNamespace: "odigos-system",
	}

	badKafka := rawDataDestination{
		ID:   "bad-kafka",
		Type: common.KafkaDestinationType,
		Data: map[string]string{
			"KAFKA_PROTOCOL_VERSION":       "2.0.0",
			"KAFKA_BROKERS":                `["broker-a:9092"]`,
			"KAFKA_PRODUCER_REQUIRED_ACKS": "all",
		},
		Signals: []common.ObservabilitySignal{common.TracesObservabilitySignal},
	}
	healthyJaeger := rawDataDestination{
		ID:   "healthy-jaeger",
		Type: common.JaegerDestinationType,
		Data: map[string]string{
			"JAEGER_URL": "jaeger.tracing:4317",
		},
		Signals: []common.ObservabilitySignal{common.TracesObservabilitySignal},
	}

	cfg, err, statuses, signals := pipelinegen.CalculateGatewayConfig(
		[]config.ExporterConfigurer{badKafka, healthyJaeger},
		[]config.ProcessorConfigurer{},
		nil, nil, &gatewayOptions,
	)

	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, []common.ObservabilitySignal{common.TracesObservabilitySignal}, signals)

	require.Error(t, statuses.Destination["bad-kafka"])
	assert.Contains(t, statuses.Destination["bad-kafka"].Error(), "KAFKA_PRODUCER_REQUIRED_ACKS")
	assert.NoError(t, statuses.Destination["healthy-jaeger"])

	assert.NotContains(t, cfg.Service.Pipelines, "traces/kafka-bad-kafka")
	assert.Contains(t, cfg.Service.Pipelines, "traces/jaeger-healthy-jaeger")
}

func openTestData(t *testing.T, path string) string {
	want, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("Failed to open %s", path)
		t.FailNow()
	}
	return string(want)
}

func marshalConfig(t *testing.T, cfg *config.Config) string {
	t.Helper()
	slices.Sort(cfg.Service.Extensions)
	data, err := yaml.Marshal(cfg)
	require.NoError(t, err)
	return string(data)
}

func TestCalculateMinimal(t *testing.T) {
	want := openTestData(t, "testdata/minimal.yaml")

	gatewayOptions := pipelinegen.GatewayConfigOptions{
		ClusterMetricsEnabled: nil,
		OdigosNamespace:       "odigos-system",
	}
	config, err, statuses, signals := pipelinegen.GetGatewayConfig(
		make([]config.ExporterConfigurer, 0),
		make([]config.ProcessorConfigurer, 0),
		nil,
		nil, &gatewayOptions,
	)
	assert.Nil(t, err)
	assert.Equal(t, config, want)
	assert.Equal(t, len(statuses.Destination), 0)
	assert.Equal(t, len(statuses.Processor), 0)
	assert.Equal(t, len(signals), 0)
}

func TestCalculate(t *testing.T) {
	want := openTestData(t, "testdata/debugexporter.yaml")

	gatewayOptions := pipelinegen.GatewayConfigOptions{
		ClusterMetricsEnabled: nil,
		OdigosNamespace:       "odigos-system",
	}
	config, err, statuses, signals := pipelinegen.GetGatewayConfig(
		[]config.ExporterConfigurer{
			DummyDestination{
				ID: "d1",
			},
		},
		make([]config.ProcessorConfigurer, 0),
		nil,
		nil, &gatewayOptions,
	)
	assert.Nil(t, err)
	assert.Equal(t, want, config)
	assert.Equal(t, len(statuses.Destination), 1)
	assert.Equal(t, len(statuses.Processor), 0)
	assert.Equal(t, len(signals), 1)
	assert.Equal(t, signals[0], common.LogsObservabilitySignal)
}

func TestCalculateDataStreamAndDestinations(t *testing.T) {
	want := openTestData(t, "testdata/withdatastream.yaml")
	gatewayOptions := pipelinegen.GatewayConfigOptions{
		ClusterMetricsEnabled: nil,
		OdigosNamespace:       "odigos-system",
	}
	dummyDest := DummyDestination{
		ID: "dummy",
	}
	dummyProcessors := []config.ProcessorConfigurer{
		DummyProcessor{
			ID: "dummy-processor",
		},
	}

	dataStreamDetails := []pipelinegen.DataStreams{
		{
			Name: "dummy-group",
			Destinations: []pipelinegen.Destination{
				{DestinationName: dummyDest.GetID(), ConfiguredSignals: dummyDest.GetSignals()},
			},
		},
	}

	cfg, err, statuses, signals := pipelinegen.CalculateGatewayConfig(
		[]config.ExporterConfigurer{dummyDest},
		dummyProcessors,
		nil, dataStreamDetails, &gatewayOptions,
	)

	assert.Equal(t, marshalConfig(t, cfg), want)
	assert.Nil(t, err)
	assert.Equal(t, len(statuses.Destination), 1)
	assert.Equal(t, len(statuses.Processor), 0)
	assert.Equal(t, len(signals), 1)
}

// TestCalculateDataStreamMissingSources tests the case where we have a datastream with destination but no sources
// This should test senario where user configures a destination and not yet configured sources
func TestCalculateDataStreamMissingSources(t *testing.T) {
	want := openTestData(t, "testdata/destnosources.yaml")

	gatewayOptions := pipelinegen.GatewayConfigOptions{
		ClusterMetricsEnabled: nil,
		OdigosNamespace:       "odigos-system",
	}
	dummyDest := DummyDestination{
		ID: "dummy",
	}
	dummyProcessors := []config.ProcessorConfigurer{
		DummyProcessor{
			ID: "dummy-processor",
		},
	}

	dataStreamDetails := []pipelinegen.DataStreams{
		{
			Name: "dummy-group",
			Destinations: []pipelinegen.Destination{
				{DestinationName: dummyDest.GetID(), ConfiguredSignals: dummyDest.GetSignals()},
			},
		},
	}

	cfg, err, statuses, signals := pipelinegen.CalculateGatewayConfig(
		[]config.ExporterConfigurer{dummyDest},
		dummyProcessors,
		nil, dataStreamDetails, &gatewayOptions,
	)

	assert.Equal(t, marshalConfig(t, cfg), want)
	assert.Nil(t, err)
	assert.Equal(t, len(statuses.Destination), 1)
	assert.Equal(t, len(statuses.Processor), 0)
	assert.Equal(t, len(signals), 1)
	assert.Equal(t, signals, []common.ObservabilitySignal{common.LogsObservabilitySignal})
}

func strPtr(b bool) *bool { return &b }

func selfMetricsReceiver() func(c *config.Config, _ []string, _ []string) error {
	return func(c *config.Config, _ []string, _ []string) error {
		c.Receivers["prometheus/self-metrics"] = config.GenericMap{
			"config": config.GenericMap{
				"scrape_configs": []config.GenericMap{},
			},
		}
		return nil
	}
}

func TestServiceGraphOptions(t *testing.T) {
	// odigos' widened default naming attributes for uninstrumented virtual peers,
	// applied whenever the operator does not configure their own. Kept in sync
	// with pipelinegen.defaultServiceGraphPeerAttributes (unexported there).
	defaultPeerAttrs := []string{"peer.service", "db.name", "db.system", "server.address", "net.peer.name", "rpc.service"}

	tests := []struct {
		name                     string
		opts                     common.ServiceGraphOptions
		wantConnector            bool
		wantDimensions           []string
		wantVirtualNodePeerAttrs []string
	}{
		{
			// With no operator override, odigos still emits its widened default so
			// plain HTTP/gRPC egress resolves to real hosts instead of "unknown".
			name:                     "enabled by default",
			opts:                     common.ServiceGraphOptions{},
			wantConnector:            true,
			wantDimensions:           []string{"service.name"},
			wantVirtualNodePeerAttrs: defaultPeerAttrs,
		},
		{
			name:          "disabled",
			opts:          common.ServiceGraphOptions{Disabled: strPtr(true)},
			wantConnector: false,
		},
		{
			name: "extra dimensions",
			opts: common.ServiceGraphOptions{
				ExtraDimensions: []string{"k8s.namespace.name", "http.method"},
			},
			wantConnector:            true,
			wantDimensions:           []string{"service.name", "k8s.namespace.name", "http.method"},
			wantVirtualNodePeerAttrs: defaultPeerAttrs,
		},
		{
			name: "custom virtual node peer attributes",
			opts: common.ServiceGraphOptions{
				VirtualNodePeerAttributes: []string{"peer.service", "server.address", "db.system"},
			},
			wantConnector:            true,
			wantDimensions:           []string{"service.name"},
			wantVirtualNodePeerAttrs: []string{"peer.service", "server.address", "db.system"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gatewayOptions := pipelinegen.GatewayConfigOptions{
				OdigosNamespace: "odigos-system",
				ServiceGraph:    tc.opts,
			}
			cfg, err, _, _ := pipelinegen.CalculateGatewayConfig(
				[]config.ExporterConfigurer{DummyTraceDestination{ID: "t1"}},
				[]config.ProcessorConfigurer{},
				selfMetricsReceiver(), nil, &gatewayOptions,
			)
			require.NoError(t, err)
			out := marshalConfig(t, cfg)

			// assert on the YAML string directly — yaml.v2 unmarshals nested maps
			// into map[interface{}]interface{}, making type assertions on GenericMap unreliable.
			if !tc.wantConnector {
				assert.NotContains(t, out, consts.ServiceGraphConnectorName+":\n")
				assert.NotContains(t, out, "metrics/servicegraph:")
				return
			}

			assert.Contains(t, out, consts.ServiceGraphConnectorName+":\n")
			assert.Contains(t, out, "metrics/servicegraph:")

			for _, dim := range tc.wantDimensions {
				assert.Contains(t, out, "- "+dim+"\n")
			}
			if len(tc.wantVirtualNodePeerAttrs) == 0 {
				assert.NotContains(t, out, "virtual_node_peer_attributes:")
			}
			for _, attr := range tc.wantVirtualNodePeerAttrs {
				assert.Contains(t, out, "virtual_node_peer_attributes:")
				assert.Contains(t, out, "- "+attr+"\n")
			}
		})
	}
}

// TestServiceGraphInsights verifies that when insights is active the gateway
// adds the insights OTLP exporter to the existing metrics/servicegraph pipeline
// (alongside prometheus/servicegraph) and forces the k8s.namespace.name
// dimension so edges can be joined to anomalies keyed by (namespace, service).
func TestServiceGraphInsights(t *testing.T) {
	on := true
	gatewayOptions := pipelinegen.GatewayConfigOptions{
		OdigosNamespace:      "odigos-system",
		Insights:             &common.InsightsConfiguration{Enabled: &on},
		InsightsOtlpEndpoint: "dns:///odigos-insights-headless.odigos-system:4317",
		ServiceGraph: common.ServiceGraphOptions{
			ExtraDimensions:           []string{"http.method"},
			VirtualNodePeerAttributes: []string{"peer.service", "server.address"},
		},
	}
	cfg, err, _, _ := pipelinegen.CalculateGatewayConfig(
		[]config.ExporterConfigurer{DummyTraceDestination{ID: "t1"}},
		[]config.ProcessorConfigurer{},
		selfMetricsReceiver(), nil, &gatewayOptions,
	)
	require.NoError(t, err)
	out := marshalConfig(t, cfg)

	// One pipeline, two exporters: prometheus (UI) + OTLP (insights).
	assert.Contains(t, out, "metrics/servicegraph:")
	assert.Contains(t, out, "prometheus/servicegraph")
	assert.Contains(t, out, consts.ServiceGraphInsightsExporterName+":")
	assert.Contains(t, out, "dns:///odigos-insights-headless.odigos-system:4317")
	assert.Contains(t, out, "balancer_name: round_robin")
	assert.NotContains(t, out, "metrics/servicegraph-insights:")
	assert.Contains(t, out, "resource/pod-name:")
	assert.Contains(t, out, "k8s.pod.name")
	assert.Contains(t, out, "${POD_NAME}")

	// Namespace dimension is forced on so edges carry it; user extras and
	// virtual-node peer attributes are preserved.
	assert.Contains(t, out, "- k8s.namespace.name\n")
	assert.Contains(t, out, "- http.method\n")
	assert.Contains(t, out, "virtual_node_peer_attributes:")
	assert.Contains(t, out, "- peer.service\n")
	assert.Contains(t, out, "- server.address\n")

	pipe, ok := cfg.Service.Pipelines["metrics/servicegraph"]
	require.True(t, ok)
	assert.Equal(t, []string{"resource/pod-name"}, pipe.Processors)
	assert.Equal(t, []string{"prometheus/servicegraph", consts.ServiceGraphInsightsExporterName}, pipe.Exporters)
}

// TestInsightsEnablesTracesWithoutDestinations verifies that insights alone
// forces the traces root pipeline and ReceiverSignals, so agents export spans
// even when no destination is configured.
func TestInsightsEnablesTracesWithoutDestinations(t *testing.T) {
	on := true
	gatewayOptions := pipelinegen.GatewayConfigOptions{
		OdigosNamespace:      "odigos-system",
		Insights:             &common.InsightsConfiguration{Enabled: &on},
		InsightsOtlpEndpoint: "dns:///odigos-insights-headless.odigos-system:4317",
	}
	cfg, err, _, signals := pipelinegen.CalculateGatewayConfig(
		nil,
		nil,
		selfMetricsReceiver(), nil, &gatewayOptions,
	)
	require.NoError(t, err)
	require.Contains(t, signals, common.TracesObservabilitySignal)
	require.NotContains(t, signals, common.MetricsObservabilitySignal)
	require.NotContains(t, signals, common.LogsObservabilitySignal)

	_, hasTracesIn := cfg.Service.Pipelines["traces/in"]
	assert.True(t, hasTracesIn, "insights must create the root traces pipeline with no destinations")
	_, hasServiceGraph := cfg.Service.Pipelines["metrics/servicegraph"]
	assert.True(t, hasServiceGraph, "service graph should still be wired when insights enables traces")

	_, hasForward := cfg.Connectors[consts.TracesPostGroupByForwardConnectorName]
	assert.False(t, hasForward, "post-groupby forward must be skipped when there is no traces destination")
	_, hasExporting := cfg.Service.Pipelines[consts.TracesExportingPipelineName]
	assert.False(t, hasExporting, "traces/exporting must be skipped when there is no traces destination")
	_, hasRouter := cfg.Connectors["odigosrouterconnector/traces"]
	assert.False(t, hasRouter, "router must not be wired when there is no traces destination")

	tracesIn := cfg.Service.Pipelines["traces/in"]
	assert.Contains(t, tracesIn.Exporters, consts.ServiceGraphConnectorName)
	assert.Contains(t, tracesIn.Processors, consts.GroupByTraceProcessor)
}

// TestServiceGraphInsightsDisabled verifies the insights exporter is absent
// when insights is not active, so the default service-graph path is unchanged.
func TestServiceGraphInsightsDisabled(t *testing.T) {
	gatewayOptions := pipelinegen.GatewayConfigOptions{OdigosNamespace: "odigos-system"}
	cfg, err, _, _ := pipelinegen.CalculateGatewayConfig(
		[]config.ExporterConfigurer{DummyTraceDestination{ID: "t1"}},
		[]config.ProcessorConfigurer{},
		selfMetricsReceiver(), nil, &gatewayOptions,
	)
	require.NoError(t, err)
	out := marshalConfig(t, cfg)
	assert.NotContains(t, out, consts.ServiceGraphInsightsExporterName+":")
	assert.NotContains(t, out, "metrics/servicegraph-insights:")

	pipe, ok := cfg.Service.Pipelines["metrics/servicegraph"]
	require.True(t, ok)
	assert.Empty(t, pipe.Processors)
	assert.Equal(t, []string{"prometheus/servicegraph"}, pipe.Exporters)
}

func TestServiceGraphInsightsDoesNotDuplicateNamespaceDimension(t *testing.T) {
	on := true
	gatewayOptions := pipelinegen.GatewayConfigOptions{
		OdigosNamespace:      "odigos-system",
		Insights:             &common.InsightsConfiguration{Enabled: &on},
		InsightsOtlpEndpoint: "dns:///odigos-insights-headless.odigos-system:4317",
		ServiceGraph: common.ServiceGraphOptions{
			ExtraDimensions: []string{"k8s.namespace.name"},
		},
	}
	cfg, err, _, _ := pipelinegen.CalculateGatewayConfig(
		[]config.ExporterConfigurer{DummyTraceDestination{ID: "t1"}},
		[]config.ProcessorConfigurer{},
		selfMetricsReceiver(), nil, &gatewayOptions,
	)
	require.NoError(t, err)

	connector, ok := cfg.Connectors[consts.ServiceGraphConnectorName].(config.GenericMap)
	require.True(t, ok)
	dims, ok := connector["dimensions"].([]string)
	require.True(t, ok)
	assert.Equal(t, []string{"service.name", "k8s.namespace.name"}, dims)
}

func TestTracesPipelineSplitAfterGroupByTrace(t *testing.T) {
	ext := "odigosconfigk8s"
	enabled := true
	wait := "45s"
	gatewayOptions := pipelinegen.GatewayConfigOptions{
		OdigosNamespace:              "odigos-system",
		OdigosConfigExtensionName:    &ext,
		TailSamplingEnabled:          &enabled,
		TraceAggregationWaitDuration: &wait,
	}
	cfg, err, _, signals := pipelinegen.CalculateGatewayConfig(
		[]config.ExporterConfigurer{DummyTraceDestination{ID: "t1"}},
		[]config.ProcessorConfigurer{},
		nil, nil, &gatewayOptions,
	)
	require.NoError(t, err)
	require.Contains(t, signals, common.TracesObservabilitySignal)
	out := marshalConfig(t, cfg)

	assert.Contains(t, out, consts.TracesPostGroupByForwardConnectorName+":")
	assert.Contains(t, out, consts.TracesExportingPipelineName+":")
	assert.Contains(t, out, "traces/in:\n")
	assert.Contains(t, out, "- "+consts.GroupByTraceProcessor+"\n")
	assert.Contains(t, out, "- "+consts.TracesPostGroupByForwardConnectorName+"\n")
	assert.Contains(t, out, "- "+consts.OdigosTailSamplingProcessorName+"\n")

	tracesIn, ok := cfg.Service.Pipelines["traces/in"]
	require.True(t, ok)
	assert.Equal(t, []string{"resource/odigos-version", consts.GroupByTraceProcessor}, tracesIn.Processors)
	assert.Contains(t, tracesIn.Exporters, consts.TracesPostGroupByForwardConnectorName)
	assert.NotContains(t, tracesIn.Processors, consts.OdigosTailSamplingProcessorName)
	assert.NotContains(t, tracesIn.Processors, consts.GenericBatchProcessorConfigKey)

	tracesExporting, ok := cfg.Service.Pipelines[consts.TracesExportingPipelineName]
	require.True(t, ok)
	assert.Equal(t, []string{
		consts.OdigosTailSamplingProcessorName,
		consts.OdigosTraceStateProcessorName,
	}, tracesExporting.Processors)
}

func TestTracesPipelineSplitWithAdditionalProcessors(t *testing.T) {
	ext := "odigosconfigk8s"
	enabled := true
	wait := "45s"
	gatewayOptions := pipelinegen.GatewayConfigOptions{
		OdigosNamespace:              "odigos-system",
		OdigosConfigExtensionName:    &ext,
		TailSamplingEnabled:          &enabled,
		TraceAggregationWaitDuration: &wait,
		TraceCorrelationsServiceIO: &common.TraceCorrelationsServiceIOConfiguration{
			Enabled: &enabled,
		},
	}
	processors := []config.ProcessorConfigurer{
		DummyProcessor{ID: "generic-batch-processor", Type: "batch"},
		DummyProcessor{ID: "odigos-url-templatization", Type: "odigosurltemplate", OrderHint: 1},
	}
	cfg, err, _, signals := pipelinegen.CalculateGatewayConfig(
		[]config.ExporterConfigurer{DummyTraceDestination{ID: "t1"}},
		processors,
		nil, nil, &gatewayOptions,
	)
	require.NoError(t, err)
	require.Contains(t, signals, common.TracesObservabilitySignal)

	tracesIn, ok := cfg.Service.Pipelines["traces/in"]
	require.True(t, ok)
	assert.Equal(t, []string{
		"resource/odigos-version",
		consts.GroupByTraceProcessor,
		"odigosurltemplate/odigos-url-templatization",
	}, tracesIn.Processors)
	assert.Contains(t, tracesIn.Exporters, consts.TracesPostGroupByForwardConnectorName)
	assert.Contains(t, tracesIn.Exporters, consts.ServiceIOConnectorName)

	tracesExporting, ok := cfg.Service.Pipelines[consts.TracesExportingPipelineName]
	require.True(t, ok)
	assert.Equal(t, []string{
		consts.OdigosTailSamplingProcessorName,
		consts.GenericBatchProcessorConfigKey,
		consts.OdigosTraceStateProcessorName,
	}, tracesExporting.Processors)
}

func TestTraceCorrelationsServiceIOPipeline(t *testing.T) {
	ext := "odigosconfigk8s"
	enabled := true
	gatewayOptions := pipelinegen.GatewayConfigOptions{
		OdigosNamespace:           "odigos-system",
		OdigosConfigExtensionName: &ext,
		TraceCorrelationsServiceIO: &common.TraceCorrelationsServiceIOConfiguration{
			Enabled:              &enabled,
			InputSpanAttributes:  []string{"http.route"},
			OutputSpanAttributes: []string{"db.system"},
			MetricsFlushInterval: "60s",
		},
	}
	cfg, err, _, signals := pipelinegen.CalculateGatewayConfig(
		[]config.ExporterConfigurer{DummyTraceDestination{ID: "t1"}},
		[]config.ProcessorConfigurer{},
		nil, nil, &gatewayOptions,
	)
	require.NoError(t, err)
	require.Contains(t, signals, common.TracesObservabilitySignal)
	out := marshalConfig(t, cfg)

	assert.Contains(t, out, consts.ServiceIOConnectorName+":")
	assert.Contains(t, out, consts.TraceCorrelationsMetricsPipelineName+":")
	assert.Contains(t, out, consts.TraceCorrelationsVictoriaMetricsExporterName+":")
	assert.Contains(t, out, "http://odigos-correlations-metrics.odigos-system:8428/opentelemetry\n")
	assert.Contains(t, out, "- "+consts.ServiceIOConnectorName+"\n")
	assert.Contains(t, out, "input_span_attributes:\n")
	assert.Contains(t, out, "- http.route\n")
	assert.Contains(t, out, "output_span_attributes:\n")
	assert.Contains(t, out, "- db.system\n")
	assert.Contains(t, out, "metrics_flush_interval: 60s\n")
	assert.Contains(t, out, "odigos_config_extension: odigosconfigk8s\n")

	tracesIn, ok := cfg.Service.Pipelines["traces/in"]
	require.True(t, ok)
	assert.Contains(t, tracesIn.Exporters, consts.TracesPostGroupByForwardConnectorName)
	assert.Contains(t, tracesIn.Exporters, consts.ServiceIOConnectorName)
	assert.Contains(t, tracesIn.Processors, consts.GroupByTraceProcessor)
	assert.NotContains(t, tracesIn.Processors, consts.GenericBatchProcessorConfigKey)

	tracesExporting, ok := cfg.Service.Pipelines[consts.TracesExportingPipelineName]
	require.True(t, ok)
	assert.NotContains(t, tracesExporting.Exporters, consts.ServiceIOConnectorName)
	assert.Equal(t, []string{consts.OdigosTraceStateProcessorName}, tracesExporting.Processors)
}

// TestCalculateDataStreamMissingDestination tests the case where we have a datastream with sources but no destination
func TestCalculateDataStreamMissingDestinatin(t *testing.T) {
	want := openTestData(t, "testdata/sourcesnodest.yaml")

	gatewayOptions := pipelinegen.GatewayConfigOptions{
		ClusterMetricsEnabled: nil,
		OdigosNamespace:       "odigos-system",
	}
	dummyProcessors := []config.ProcessorConfigurer{
		DummyProcessor{
			ID: "dummy-processor",
		},
	}

	dataStreamDetails := []pipelinegen.DataStreams{
		{
			Name: "dummy-group",
		},
	}

	cfg, err, statuses, signals := pipelinegen.CalculateGatewayConfig(
		[]config.ExporterConfigurer{},
		dummyProcessors,
		func(c *config.Config, _ []string, _ []string) error {
			c.Service.Pipelines["metrics/otelcol"] = config.Pipeline{
				Receivers:  []string{"prometheus/self-metrics"},
				Processors: []string{"resource/pod-name"},
				Exporters:  []string{"otlp_grpc/odigos-own-telemetry-ui"},
			}
			return nil
		},
		dataStreamDetails, &gatewayOptions,
	)

	assert.Equal(t, want, marshalConfig(t, cfg))
	assert.Nil(t, err)
	assert.Equal(t, len(statuses.Destination), 0)
	assert.Equal(t, len(statuses.Processor), 0)
	assert.Equal(t, len(signals), 0)
}
