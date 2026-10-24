package config

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAzureEventHubsConfig(t *testing.T) {
	for _, encoding := range []string{"otlp_proto", "otlp_json"} {
		for _, compression := range []string{"none", "gzip"} {
			t.Run(encoding+"/"+compression, func(t *testing.T) {
				dest := &mockDestination{id: "azure-test", config: map[string]string{
					azureEventHubsNamespace: "my-namespace", azureEventHubsEncoding: encoding,
					azureEventHubsCompression: compression, azureEventHubsTracesTopic: "custom-spans",
				}, signals: []common.ObservabilitySignal{common.TracesObservabilitySignal, common.MetricsObservabilitySignal, common.LogsObservabilitySignal}}
				cfg := newKafkaConfig()
				pipelines, err := (&AzureEventHubs{}).ModifyConfig(dest, cfg)
				require.NoError(t, err)
				require.Len(t, pipelines, 3)
				require.Len(t, cfg.Exporters, 3)
				for signal, topic := range map[string]string{"traces": "custom-spans", "metrics": "otlp_metrics", "logs": "otlp_logs"} {
					pipe := cfg.Service.Pipelines[signal+"/azureeventhubs-azure-test"]
					require.Len(t, pipe.Exporters, 1)
					exporter := cfg.Exporters[pipe.Exporters[0]].(GenericMap)
					assert.Equal(t, []string{"my-namespace.servicebus.windows.net:9093"}, exporter["brokers"])
					assert.Equal(t, GenericMap{"topic": topic, "encoding": encoding}, exporter[signal])
					assert.NotContains(t, exporter, "topic")
					assert.NotContains(t, exporter, "encoding")
					assert.Equal(t, GenericMap{"insecure": false, "insecure_skip_verify": false, "min_version": "1.2"}, exporter["tls"])
					sasl := exporter["auth"].(GenericMap)["sasl"].(GenericMap)
					assert.Equal(t, "PLAIN", sasl["mechanism"])
					assert.Equal(t, "$$ConnectionString", sasl["username"])
					assert.Equal(t, "${ODIGOS_DEST_azure_test_AZURE_EVENT_HUBS_CONNECTION_STRING}", sasl["password"])
					producer := exporter["producer"].(GenericMap)
					assert.Equal(t, compression, producer["compression"])
					assert.Equal(t, 1, producer["flush_max_messages"])
					assert.Equal(t, 1000000, producer["max_message_bytes"])
					assert.Equal(t, false, producer["allow_auto_topic_creation"])
				}
				assert.Len(t, dest.config, 4, "delegating to Kafka must not mutate destination data")
			})
		}
	}
}

func TestAzureEventHubsInvalidConfig(t *testing.T) {
	for _, tc := range []struct {
		name, key, value string
	}{
		{"missing namespace", azureEventHubsNamespace, ""},
		{"endpoint instead of namespace", azureEventHubsNamespace, "my-namespace.servicebus.windows.net:9093"},
		{"invalid encoding", azureEventHubsEncoding, "raw"},
		{"unsupported compression", azureEventHubsCompression, "snappy"},
		{"invalid topic", azureEventHubsTracesTopic, "bad/topic"},
		{"duplicate topics", azureEventHubsTracesTopic, "otlp_metrics"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dest := &mockDestination{id: "test", config: map[string]string{azureEventHubsNamespace: "my-namespace"},
				signals: []common.ObservabilitySignal{common.TracesObservabilitySignal, common.MetricsObservabilitySignal}}
			dest.config[tc.key] = tc.value
			cfg := newKafkaConfig()
			_, err := (&AzureEventHubs{}).ModifyConfig(dest, cfg)
			require.Error(t, err)
			assert.Empty(t, cfg.Exporters)
			assert.Empty(t, cfg.Service.Pipelines)
		})
	}
}

func TestAzureEventHubsRegisteredAndProfilesRejected(t *testing.T) {
	configers, err := LoadConfigers()
	require.NoError(t, err)
	require.Contains(t, configers, common.AzureEventHubsDestinationType)
	dest := &mockDestination{id: "test", config: map[string]string{azureEventHubsNamespace: "my-namespace"}, signals: []common.ObservabilitySignal{common.ProfilesObservabilitySignal}}
	_, err = configers[common.AzureEventHubsDestinationType].ModifyConfig(dest, newKafkaConfig())
	require.Error(t, err)
}
