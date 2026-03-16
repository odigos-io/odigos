package config_test

import (
	"os"
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/common/pipelinegen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var empty = struct{}{}

type DummyProcessor struct {
	ID string
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
	return "resource"
}

func (proc DummyProcessor) GetOrderHint() int {
	return 0
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

func openTestData(t *testing.T, path string) string {
	want, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("Failed to open %s", path)
		t.FailNow()
	}
	return string(want)
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
		nil, gatewayOptions,
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
		nil, gatewayOptions,
	)
	assert.Nil(t, err)
	assert.Equal(t, want, config)
	assert.Equal(t, len(statuses.Destination), 1)
	assert.Equal(t, len(statuses.Processor), 0)
	assert.Equal(t, len(signals), 1)
	assert.Equal(t, signals[0], common.LogsObservabilitySignal)
}

func TestCalculateWithBaseMinimal(t *testing.T) {
	want := openTestData(t, "testdata/withbaseminimal.yaml")

	gatewayOptions := pipelinegen.GatewayConfigOptions{
		ClusterMetricsEnabled: nil,
		OdigosNamespace:       "odigos-system",
	}
	config, err, statuses, signals := pipelinegen.CalculateGatewayConfig(
		&config.Config{
			Receivers: config.GenericMap{
				"otlp": config.GenericMap{
					"protocols": config.GenericMap{
						"grpc": empty,
						"http": empty,
					},
				},
			},
			Processors: config.GenericMap{
				"batch/generic-batch-processor": config.GenericMap{},
			},
			Extensions: config.GenericMap{},
			Exporters:  map[string]interface{}{},
			Service: config.Service{
				Pipelines:  map[string]config.Pipeline{},
				Extensions: []string{},
			},
		},
		[]config.ExporterConfigurer{},
		[]config.ProcessorConfigurer{},
		nil,
		nil, gatewayOptions,
	)
	assert.Nil(t, err)
	assert.Equal(t, config, want)
	assert.Equal(t, len(statuses.Destination), 0)
	assert.Equal(t, len(statuses.Processor), 0)
	assert.Equal(t, len(signals), 0)
}

func TestCalculateWithBaseNoOTLP(t *testing.T) {
	gatewayOptions := pipelinegen.GatewayConfigOptions{
		ClusterMetricsEnabled: nil,
		OdigosNamespace:       "odigos-system",
	}
	_, err, statuses, signals := pipelinegen.CalculateGatewayConfig(
		&config.Config{
			Receivers:  config.GenericMap{},
			Processors: config.GenericMap{},
			Extensions: config.GenericMap{},
			Exporters:  map[string]interface{}{},
			Service: config.Service{
				Pipelines:  map[string]config.Pipeline{},
				Extensions: []string{},
			},
		},
		[]config.ExporterConfigurer{},
		[]config.ProcessorConfigurer{},
		nil,
		nil, gatewayOptions,
	)
	assert.Contains(t, err.Error(), "required receiver")
	assert.Equal(t, len(statuses.Destination), 0)
	assert.Equal(t, len(statuses.Processor), 0)
	assert.Equal(t, len(signals), 0)
}

// TestCalculateDataStreamAndDestinations tests the case where we have a datastream with sources and a destination
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
			Sources: []pipelinegen.SourceFilter{
				{Namespace: "dummy-namespace", Kind: "dummy-kind", Name: "dummy-name"},
			},
			Destinations: []pipelinegen.Destination{
				{DestinationName: dummyDest.GetID(), ConfiguredSignals: dummyDest.GetSignals()},
			},
		},
	}

	config, err, statuses, signals := pipelinegen.CalculateGatewayConfig(
		&config.Config{
			Receivers: config.GenericMap{
				"otlp": config.GenericMap{
					"protocols": config.GenericMap{
						"grpc": empty,
						"http": empty,
					},
				}},
			Processors: config.GenericMap{},
			Extensions: config.GenericMap{},
			Exporters:  map[string]interface{}{},
			Connectors: config.GenericMap{},
			Service: config.Service{
				Pipelines:  map[string]config.Pipeline{},
				Extensions: []string{},
			},
		},
		[]config.ExporterConfigurer{dummyDest},
		dummyProcessors,
		nil, dataStreamDetails, gatewayOptions,
	)

	assert.Equal(t, config, want)
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
			Name:    "dummy-group",
			Sources: []pipelinegen.SourceFilter{},
			Destinations: []pipelinegen.Destination{
				{DestinationName: dummyDest.GetID(), ConfiguredSignals: dummyDest.GetSignals()},
			},
		},
	}

	config, err, statuses, signals := pipelinegen.CalculateGatewayConfig(
		&config.Config{
			Receivers: config.GenericMap{
				"otlp": config.GenericMap{
					"protocols": config.GenericMap{
						"grpc": empty,
						"http": empty,
					},
				}},
			Processors: config.GenericMap{
				"batch/generic-batch-processor": config.GenericMap{},
			},
			Extensions: config.GenericMap{},
			Exporters:  map[string]interface{}{},
			Connectors: config.GenericMap{},
			Service: config.Service{
				Pipelines:  map[string]config.Pipeline{},
				Extensions: []string{},
			},
		},
		[]config.ExporterConfigurer{dummyDest},
		dummyProcessors,
		nil, dataStreamDetails, gatewayOptions,
	)

	assert.Equal(t, config, want)
	assert.Nil(t, err)
	assert.Equal(t, len(statuses.Destination), 1)
	assert.Equal(t, len(statuses.Processor), 0)
	assert.Equal(t, len(signals), 1)
	assert.Equal(t, signals, []common.ObservabilitySignal{common.LogsObservabilitySignal})
}

func strPtr(b bool) *bool { return &b }

// baseConfigWithSelfMetrics returns GetBasicConfig() plus the prometheus/self-metrics
// receiver that AddServiceGraphScrapeConfig requires to attach its scrape job.
func baseConfigWithSelfMetrics() *config.Config {
	base := pipelinegen.GetBasicConfig()
	base.Receivers["prometheus/self-metrics"] = config.GenericMap{
		"config": config.GenericMap{
			"scrape_configs": []config.GenericMap{},
		},
	}
	return base
}

func TestServiceGraphOptions(t *testing.T) {
	tests := []struct {
		name                       string
		opts                       common.ServiceGraphOptions
		wantConnector              bool
		wantDimensions             []string
		wantVirtualNodePeerAttrs   []string
		wantNoVirtualNodePeerAttrs bool
	}{
		{
			name:                       "enabled by default",
			opts:                       common.ServiceGraphOptions{},
			wantConnector:              true,
			wantDimensions:             []string{"service.name"},
			wantNoVirtualNodePeerAttrs: true,
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
			wantConnector:              true,
			wantDimensions:             []string{"service.name", "k8s.namespace.name", "http.method"},
			wantNoVirtualNodePeerAttrs: true,
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
			out, err, _, _ := pipelinegen.CalculateGatewayConfig(
				baseConfigWithSelfMetrics(),
				[]config.ExporterConfigurer{DummyTraceDestination{ID: "t1"}},
				[]config.ProcessorConfigurer{},
				nil, nil, gatewayOptions,
			)
			require.NoError(t, err)

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
			if tc.wantNoVirtualNodePeerAttrs {
				assert.NotContains(t, out, "virtual_node_peer_attributes:")
			}
			for _, attr := range tc.wantVirtualNodePeerAttrs {
				assert.Contains(t, out, "virtual_node_peer_attributes:")
				assert.Contains(t, out, "- "+attr+"\n")
			}
		})
	}
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
			Sources: []pipelinegen.SourceFilter{
				{Namespace: "default", Kind: "dummy-kind", Name: "dummy-name"},
			},
		},
	}

	config, err, statuses, signals := pipelinegen.CalculateGatewayConfig(
		&config.Config{
			Receivers: config.GenericMap{
				"otlp": config.GenericMap{
					"protocols": config.GenericMap{
						"grpc": empty,
						"http": empty,
					},
				}},
			Processors: config.GenericMap{
				"batch/generic-batch-processor": config.GenericMap{},
			},
			Extensions: config.GenericMap{},
			Exporters:  map[string]interface{}{},
			Connectors: config.GenericMap{},
			Service: config.Service{
				Pipelines: map[string]config.Pipeline{
					"metrics/otelcol": {
						Receivers:  []string{"prometheus/self-metrics"},
						Processors: []string{"resource/pod-name"},
						Exporters:  []string{"otlp/odigos-own-telemetry-ui"},
					},
				},
				Extensions: []string{},
			},
		},
		[]config.ExporterConfigurer{},
		dummyProcessors,
		nil, dataStreamDetails, gatewayOptions,
	)

	assert.Equal(t, want, config)
	assert.Nil(t, err)
	assert.Equal(t, len(statuses.Destination), 0)
	assert.Equal(t, len(statuses.Processor), 0)
	assert.Equal(t, len(signals), 0)
}
