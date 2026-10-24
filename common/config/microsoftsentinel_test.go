package config

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sentinelTestDestination() *mockDestination {
	return &mockDestination{id: "sentinel-test", signals: []common.ObservabilitySignal{common.LogsObservabilitySignal}, config: map[string]string{
		sentinelEndpoint: "https://odigos.eastus-1.ingest.monitor.azure.com/",
		sentinelTenantID: "tenant-id", sentinelClientID: "client-id",
		sentinelRuleID: "dcr-example", sentinelStreamName: "Custom-OdigosLogs",
	}}
}

func TestMicrosoftSentinelConfig(t *testing.T) {
	configers, err := LoadConfigers()
	require.NoError(t, err)
	require.Contains(t, configers, common.MicrosoftSentinelDestinationType)
	dest := sentinelTestDestination()
	cfg := newKafkaConfig()
	pipelines, err := configers[common.MicrosoftSentinelDestinationType].ModifyConfig(dest, cfg)
	require.NoError(t, err)
	assert.Equal(t, []string{"logs/microsoftsentinel-sentinel-test"}, pipelines)
	require.Len(t, cfg.Exporters, 1)
	exporter := cfg.Exporters["azureloganalytics/microsoftsentinel-sentinel-test"].(GenericMap)
	assert.Equal(t, "https://odigos.eastus-1.ingest.monitor.azure.com", exporter["endpoint"])
	assert.Equal(t, "Custom-OdigosLogs", exporter["stream_name"])
	assert.Equal(t, "${ODIGOS_DEST_sentinel_test_MICROSOFT_SENTINEL_CLIENT_SECRET}", exporter["client_secret"])
	assert.Equal(t, Pipeline{Processors: []string{"transform/microsoftsentinel-sentinel-test"}, Exporters: []string{"azureloganalytics/microsoftsentinel-sentinel-test"}}, cfg.Service.Pipelines[pipelines[0]])
	assert.Len(t, dest.config, 5)

	dest.id = "other"
	_, err = (&MicrosoftSentinel{}).ModifyConfig(dest, cfg)
	require.NoError(t, err)
	assert.Len(t, cfg.Exporters, 2)
	assert.Len(t, cfg.Processors, 2)
	assert.Equal(t, "${ODIGOS_DEST_other_MICROSOFT_SENTINEL_CLIENT_SECRET}", cfg.Exporters["azureloganalytics/microsoftsentinel-other"].(GenericMap)["client_secret"])
}

func TestMicrosoftSentinelRejectsInvalidConfig(t *testing.T) {
	for _, tc := range []struct{ name, key, value string }{
		{"missing endpoint", sentinelEndpoint, ""},
		{"missing tenant", sentinelTenantID, " "},
		{"missing client", sentinelClientID, ""},
		{"missing rule", sentinelRuleID, ""},
		{"missing stream", sentinelStreamName, ""},
		{"http endpoint", sentinelEndpoint, "http://ingest.example.com"},
		{"missing host", sentinelEndpoint, "https:///"},
		{"full API path", sentinelEndpoint, "https://ingest.example.com/datacollectionRules/dcr-example/streams/Custom-OdigosLogs"},
		{"URL credentials", sentinelEndpoint, "https://user:pass@ingest.example.com"},
		{"query", sentinelEndpoint, "https://ingest.example.com?key=value"},
		{"fragment", sentinelEndpoint, "https://ingest.example.com#fragment"},
		{"resource ID instead of immutable ID", sentinelRuleID, "/subscriptions/example/providers/Microsoft.Insights/dataCollectionRules/rule"},
		{"empty immutable ID", sentinelRuleID, "dcr-"},
		{"table instead of input stream", sentinelStreamName, "OdigosLogs_CL"},
		{"empty stream", sentinelStreamName, "Custom-"},
		{"invalid stream", sentinelStreamName, "Custom-invalid/path"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dest := sentinelTestDestination()
			dest.config[tc.key] = tc.value
			cfg := newKafkaConfig()
			_, err := (&MicrosoftSentinel{}).ModifyConfig(dest, cfg)
			require.Error(t, err)
			assert.Empty(t, cfg.Exporters)
			assert.Empty(t, cfg.Processors)
			assert.Empty(t, cfg.Service.Pipelines)
		})
	}
	for _, signal := range []common.ObservabilitySignal{common.TracesObservabilitySignal, common.MetricsObservabilitySignal, common.ProfilesObservabilitySignal} {
		t.Run(string(signal), func(t *testing.T) {
			dest := sentinelTestDestination()
			dest.signals = append(dest.signals, signal)
			cfg := newKafkaConfig()
			_, err := (&MicrosoftSentinel{}).ModifyConfig(dest, cfg)
			require.Error(t, err)
			assert.Empty(t, cfg.Exporters)
		})
	}
}
