package config_test

import (
	"os"
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	"github.com/stretchr/testify/assert"
)

var empty = struct{}{}

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

	config, err, statuses, signals := config.Calculate(
		make([]config.ExporterConfigurer, 0),
		make([]config.ProcessorConfigurer, 0),
		make(config.GenericMap),
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

	config, err, statuses, signals := config.Calculate(
		[]config.ExporterConfigurer{
			DummyDestination{
				ID: "d1",
			},
		},
		make([]config.ProcessorConfigurer, 0),
		make(config.GenericMap),
		nil,
	)
	assert.Nil(t, err)
	assert.Equal(t, config, want)
	assert.Equal(t, len(statuses.Destination), 1)
	assert.Equal(t, len(statuses.Processor), 0)
	assert.Equal(t, len(signals), 1)
	assert.Equal(t, signals[0], common.LogsObservabilitySignal)
}

func TestCalculateWithBaseMinimal(t *testing.T) {
	want := openTestData(t, "testdata/withbaseminimal.yaml")

	config, err, statuses, signals := config.CalculateWithBase(
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
				"batch": empty,
			},
			Extensions: config.GenericMap{},
			Exporters:  map[string]interface{}{},
			Service: config.Service{
				Pipelines:  map[string]config.Pipeline{},
				Extensions: []string{},
			},
		},
		[]string{"batch"},
		[]config.ExporterConfigurer{},
		[]config.ProcessorConfigurer{},
		nil,
	)
	assert.Nil(t, err)
	assert.Equal(t, config, want)
	assert.Equal(t, len(statuses.Destination), 0)
	assert.Equal(t, len(statuses.Processor), 0)
	assert.Equal(t, len(signals), 0)
}

func TestCalculateWithBaseMissingProcessor(t *testing.T) {
	_, err, statuses, signals := config.CalculateWithBase(
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
				"batch": empty,
			},
			Extensions: config.GenericMap{},
			Exporters:  map[string]interface{}{},
			Service: config.Service{
				Pipelines:  map[string]config.Pipeline{},
				Extensions: []string{},
			},
		},
		[]string{"missing"},
		[]config.ExporterConfigurer{},
		[]config.ProcessorConfigurer{},
		nil,
	)
	assert.Contains(t, err.Error(), "'missing'")
	assert.Equal(t, len(statuses.Destination), 0)
	assert.Equal(t, len(statuses.Processor), 0)
	assert.Equal(t, len(signals), 0)
}

func TestCalculateWithBaseNoOTLP(t *testing.T) {
	_, err, statuses, signals := config.CalculateWithBase(
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
		[]string{},
		[]config.ExporterConfigurer{},
		[]config.ProcessorConfigurer{},
		nil,
	)
	assert.Contains(t, err.Error(), "required receiver")
	assert.Equal(t, len(statuses.Destination), 0)
	assert.Equal(t, len(statuses.Processor), 0)
	assert.Equal(t, len(signals), 0)
}
