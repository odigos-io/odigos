package config

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newKafkaConfig() *Config {
	return &Config{
		Receivers:  make(GenericMap),
		Exporters:  make(GenericMap),
		Processors: make(GenericMap),
		Extensions: make(GenericMap),
		Connectors: make(GenericMap),
		Service: Service{
			Pipelines: make(map[string]Pipeline),
		},
	}
}

func TestKafkaBrokersAreRenderedAsAList(t *testing.T) {
	tests := []struct {
		name        string
		rawBrokers  string
		setBrokers  bool
		wantBrokers []string
		wantErr     bool
	}{
		{
			name:        "omitted brokers use the documented default",
			setBrokers:  false,
			wantBrokers: []string{"localhost:9092"},
		},
		{
			name:        "json array from the UI multi input",
			setBrokers:  true,
			rawBrokers:  `["broker-a:9092","broker-b:9092"]`,
			wantBrokers: []string{"broker-a:9092", "broker-b:9092"},
		},
		{
			name:        "single plain broker address",
			setBrokers:  true,
			rawBrokers:  "broker-a:9092",
			wantBrokers: []string{"broker-a:9092"},
		},
		{
			name:        "comma separated plain broker addresses",
			setBrokers:  true,
			rawBrokers:  "broker-a:9092, broker-b:9092",
			wantBrokers: []string{"broker-a:9092", "broker-b:9092"},
		},
		{
			name:       "empty broker list is rejected instead of crash looping the gateway",
			setBrokers: true,
			rawBrokers: "[]",
			wantErr:    true,
		},
		{
			name:       "malformed json is rejected",
			setBrokers: true,
			rawBrokers: `["broker-a:9092"`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			destConfig := map[string]string{KAFKA_PROTOCOL_VERSION: "2.0.0"}
			if tt.setBrokers {
				destConfig[KAFKA_BROKERS] = tt.rawBrokers
			}
			dest := &mockDestination{
				id:      "test-id",
				config:  destConfig,
				signals: []common.ObservabilitySignal{common.TracesObservabilitySignal},
			}

			currentConfig := newKafkaConfig()
			_, err := (&Kafka{}).ModifyConfig(dest, currentConfig)
			if tt.wantErr {
				require.Error(t, err)
				assert.Empty(t, currentConfig.Exporters)
				return
			}
			require.NoError(t, err)

			exporter, ok := currentConfig.Exporters["kafka/kafka-test-id-traces"].(GenericMap)
			require.True(t, ok, "expected a kafka exporter, got %v", currentConfig.Exporters)
			assert.Equal(t, tt.wantBrokers, exporter["brokers"])
		})
	}
}

func TestKafkaNonNumericValuesAreRejected(t *testing.T) {
	numericKeys := []string{
		KAFKA_METADATA_MAX_RETRY,
		KAFKA_PRODUCER_MAX_MESSAGE_BYTES,
		KAFKA_PRODUCER_REQUIRED_ACKS,
		KAFKA_PRODUCER_FLUSH_MAX_MESSAGES,
	}
	badValues := []string{"all", "", "1_000_000", "1000000 "}

	for _, key := range numericKeys {
		for _, badValue := range badValues {
			t.Run(key+"="+badValue, func(t *testing.T) {
				dest := &mockDestination{
					id: "test-id",
					config: map[string]string{
						KAFKA_PROTOCOL_VERSION: "2.0.0",
						KAFKA_BROKERS:          `["broker-a:9092"]`,
						key:                    badValue,
					},
					signals: []common.ObservabilitySignal{common.TracesObservabilitySignal},
				}

				currentConfig := newKafkaConfig()
				_, err := (&Kafka{}).ModifyConfig(dest, currentConfig)
				require.Error(t, err, "a non numeric %s must fail the destination instead of panicking", key)
				assert.Contains(t, err.Error(), key)
				assert.Empty(t, currentConfig.Exporters)
			})
		}
	}
}

func TestKafkaNumericValuesAreRenderedAsNumbers(t *testing.T) {
	dest := &mockDestination{
		id: "test-id",
		config: map[string]string{
			KAFKA_PROTOCOL_VERSION:            "2.0.0",
			KAFKA_BROKERS:                     `["broker-a:9092"]`,
			KAFKA_METADATA_MAX_RETRY:          "7",
			KAFKA_PRODUCER_MAX_MESSAGE_BYTES:  "2000000",
			KAFKA_PRODUCER_REQUIRED_ACKS:      "-1",
			KAFKA_PRODUCER_FLUSH_MAX_MESSAGES: "100",
		},
		signals: []common.ObservabilitySignal{common.TracesObservabilitySignal},
	}

	currentConfig := newKafkaConfig()
	_, err := (&Kafka{}).ModifyConfig(dest, currentConfig)
	require.NoError(t, err)

	exporter, ok := currentConfig.Exporters["kafka/kafka-test-id-traces"].(GenericMap)
	require.True(t, ok, "expected a kafka exporter, got %v", currentConfig.Exporters)

	metadata, ok := exporter["metadata"].(GenericMap)
	require.True(t, ok)
	retry, ok := metadata["retry"].(GenericMap)
	require.True(t, ok)
	assert.Equal(t, 7, retry["max"])

	producer, ok := exporter["producer"].(GenericMap)
	require.True(t, ok)
	assert.Equal(t, 2000000, producer["max_message_bytes"])
	assert.Equal(t, -1, producer["required_acks"])
	assert.Equal(t, 100, producer["flush_max_messages"])
}

func TestKafkaDefaultTopicIsPerSignal(t *testing.T) {
	dest := &mockDestination{
		id: "test-id",
		config: map[string]string{
			KAFKA_PROTOCOL_VERSION: "2.0.0",
			KAFKA_BROKERS:          `["broker-a:9092"]`,
		},
		signals: []common.ObservabilitySignal{
			common.TracesObservabilitySignal,
			common.MetricsObservabilitySignal,
			common.LogsObservabilitySignal,
		},
	}

	currentConfig := newKafkaConfig()
	pipelineNames, err := (&Kafka{}).ModifyConfig(dest, currentConfig)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"traces/kafka-test-id", "metrics/kafka-test-id", "logs/kafka-test-id"}, pipelineNames)

	wantTopics := map[string]string{
		"traces/kafka-test-id":  "otlp_spans",
		"metrics/kafka-test-id": "otlp_metrics",
		"logs/kafka-test-id":    "otlp_logs",
	}
	for pipelineName, wantTopic := range wantTopics {
		pipeline, ok := currentConfig.Service.Pipelines[pipelineName]
		require.True(t, ok, "missing pipeline %s", pipelineName)
		require.Len(t, pipeline.Exporters, 1)

		exporter, ok := currentConfig.Exporters[pipeline.Exporters[0]].(GenericMap)
		require.True(t, ok, "missing exporter %s", pipeline.Exporters[0])
		assert.Equal(t, wantTopic, exporter["topic"], "pipeline %s exports to the wrong topic", pipelineName)
	}
}

func TestKafkaExplicitTopicSharesASingleExporter(t *testing.T) {
	dest := &mockDestination{
		id: "test-id",
		config: map[string]string{
			KAFKA_PROTOCOL_VERSION: "2.0.0",
			KAFKA_BROKERS:          `["broker-a:9092"]`,
			KAFKA_TOPIC:            "my-topic",
		},
		signals: []common.ObservabilitySignal{
			common.TracesObservabilitySignal,
			common.LogsObservabilitySignal,
		},
	}

	currentConfig := newKafkaConfig()
	_, err := (&Kafka{}).ModifyConfig(dest, currentConfig)
	require.NoError(t, err)

	require.Len(t, currentConfig.Exporters, 1)
	exporter, ok := currentConfig.Exporters["kafka/kafka-test-id"].(GenericMap)
	require.True(t, ok, "expected the shared kafka exporter, got %v", currentConfig.Exporters)
	assert.Equal(t, "my-topic", exporter["topic"])

	for _, pipelineName := range []string{"traces/kafka-test-id", "logs/kafka-test-id"} {
		pipeline, ok := currentConfig.Service.Pipelines[pipelineName]
		require.True(t, ok, "missing pipeline %s", pipelineName)
		assert.Equal(t, []string{"kafka/kafka-test-id"}, pipeline.Exporters)
	}
}
