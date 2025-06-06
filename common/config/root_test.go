package config_test

import (
	"os"
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	"github.com/odigos-io/odigos/common/pipelinegen"
	"github.com/stretchr/testify/assert"
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

type DummyDestination struct {
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

	config, err, statuses, signals := pipelinegen.GetGatewayConfig(
		make([]config.ExporterConfigurer, 0),
		make([]config.ProcessorConfigurer, 0),
		make(config.GenericMap),
		nil,
		nil,
	)
	assert.Nil(t, err)
	assert.Equal(t, config, want)
	assert.Equal(t, len(statuses.Destination), 0)
	assert.Equal(t, len(statuses.Processor), 0)
	assert.Equal(t, len(signals), 0)
}

func TestCalculate(t *testing.T) {
	want := openTestData(t, "testdata/debugexporter.yaml")

	config, err, statuses, signals := pipelinegen.GetGatewayConfig(
		[]config.ExporterConfigurer{
			DummyDestination{
				ID: "d1",
			},
		},
		make([]config.ProcessorConfigurer, 0),
		make(config.GenericMap),
		nil,
		nil,
	)
	assert.Nil(t, err)
	assert.Equal(t, config, want)
	assert.Equal(t, len(statuses.Destination), 0)
	assert.Equal(t, len(statuses.Processor), 0)
	assert.Equal(t, len(signals), 1)
	assert.Equal(t, signals[0], common.LogsObservabilitySignal)
}

func TestCalculateWithBaseMinimal(t *testing.T) {
	want := openTestData(t, "testdata/withbaseminimal.yaml")

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
		nil,
	)
	assert.Nil(t, err)
	assert.Equal(t, config, want)
	assert.Equal(t, len(statuses.Destination), 0)
	assert.Equal(t, len(statuses.Processor), 0)
	assert.Equal(t, len(signals), 0)
}

func TestCalculateWithBaseNoOTLP(t *testing.T) {
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
		nil, nil,
	)
	assert.Contains(t, err.Error(), "required receiver")
	assert.Equal(t, len(statuses.Destination), 0)
	assert.Equal(t, len(statuses.Processor), 0)
	assert.Equal(t, len(signals), 0)
}

// TestCalculateDataStreamAndDestinations tests the case where we have a datastream with sources and a destination
func TestCalculateDataStreamAndDestinations(t *testing.T) {
	want := openTestData(t, "testdata/withdatastream.yaml")
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
		nil, dataStreamDetails,
	)

	assert.Equal(t, config, want)
	assert.Nil(t, err)
	assert.Equal(t, len(statuses.Destination), 0)
	assert.Equal(t, len(statuses.Processor), 0)
	assert.Equal(t, len(signals), 1)
}

// TestCalculateDataStreamUsingNamespaceSources tests the case where we have a datastream with sources and a destination
// The sources are configured using namespace source object
func TestCalculateDataStreamUsingNamespaceSources(t *testing.T) {

	want := openTestData(t, "testdata/namespacesource.yaml")
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
			Name:    "groupA",
			Sources: []pipelinegen.SourceFilter{},
			Namespaces: []pipelinegen.NamespaceFilter{
				{Namespace: "default"},
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
		nil, dataStreamDetails,
	)

	assert.Equal(t, config, want)
	assert.Nil(t, err)
	assert.Equal(t, len(statuses.Destination), 0)
	assert.Equal(t, len(statuses.Processor), 0)
	assert.Equal(t, len(signals), 1)
	assert.Equal(t, signals, []common.ObservabilitySignal{common.LogsObservabilitySignal})
}

// TestCalculateDataStreamMissingSources tests the case where we have a datastream with destination but no sources
// This should test senario where user configures a destination and not yet configured sources
func TestCalculateDataStreamMissingSources(t *testing.T) {
	want := openTestData(t, "testdata/destnosources.yaml")

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
		nil, dataStreamDetails,
	)

	assert.Equal(t, config, want)
	assert.Nil(t, err)
	assert.Equal(t, len(statuses.Destination), 0)
	assert.Equal(t, len(statuses.Processor), 0)
	assert.Equal(t, len(signals), 1)
	assert.Equal(t, signals, []common.ObservabilitySignal{common.LogsObservabilitySignal})
}

// TestCalculateDataStreamMissingDestination tests the case where we have a datastream with sources but no destination
func TestCalculateDataStreamMissingDestinatin(t *testing.T) {
	want := openTestData(t, "testdata/sourcesnodest.yaml")

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
		nil, dataStreamDetails,
	)

	assert.Equal(t, config, want)
	assert.Nil(t, err)
	assert.Equal(t, len(statuses.Destination), 0)
	assert.Equal(t, len(statuses.Processor), 0)
	assert.Equal(t, len(signals), 0)
}
