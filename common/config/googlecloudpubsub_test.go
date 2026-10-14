package config

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/require"
)

func TestGoogleCloudPubSubConfig(t *testing.T) {
	configers, err := LoadConfigers()
	require.NoError(t, err)
	configer := configers[common.GoogleCloudPubSubDestinationType]
	require.NotNil(t, configer)
	dest := &mockDestination{id: "pubsub", config: map[string]string{
		gcpPubSubTopicKey:       "projects/test-project/topics/otlp",
		gcpPubSubCompressionKey: "gzip",
		gcpPubSubOrderingKey:    "service.name",
		gcpPubSubEndpointKey:    "us-east1-pubsub.googleapis.com:443",
		gcpTimeoutKey:           "20s",
	}}
	cfg := &Config{Exporters: GenericMap{}, Service: Service{Pipelines: map[string]Pipeline{}}}
	names, err := configer.ModifyConfig(dest, cfg)
	require.NoError(t, err)
	require.Equal(t, []string{"traces/googlecloudpubsub-pubsub", "metrics/googlecloudpubsub-pubsub", "logs/googlecloudpubsub-pubsub"}, names)
	require.Len(t, cfg.Exporters, 1)
	exporter := cfg.Exporters["googlecloudpubsub/pubsub"].(GenericMap)
	require.Equal(t, "test-project", exporter["project"])
	require.Equal(t, "gzip", exporter["compression"])
	require.Equal(t, "20s", exporter["timeout"])
	require.Equal(t, "us-east1-pubsub.googleapis.com:443", exporter["endpoint"])
	require.Equal(t, GenericMap{"enabled": true, "from_resource_attribute": "service.name"}, exporter["ordering"])
	require.NotContains(t, exporter, "insecure")
	require.NotContains(t, exporter, "GCP_APPLICATION_CREDENTIALS")
	for _, name := range names {
		require.Equal(t, []string{"googlecloudpubsub/pubsub"}, cfg.Service.Pipelines[name].Exporters)
	}
}

func TestGoogleCloudPubSubValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  map[string]string
		signals []common.ObservabilitySignal
		invalid bool
	}{
		{name: "missing topic", config: map[string]string{}, invalid: true},
		{name: "empty topic", config: map[string]string{gcpPubSubTopicKey: "  "}, invalid: true},
		{name: "invalid project", config: map[string]string{gcpPubSubTopicKey: "projects/INVALID/topics/otlp"}, invalid: true},
		{name: "short topic", config: map[string]string{gcpPubSubTopicKey: "otlp"}, invalid: true},
		{name: "empty topic name", config: map[string]string{gcpPubSubTopicKey: "projects/test/topics/"}, invalid: true},
		{name: "invalid compression", config: map[string]string{gcpPubSubTopicKey: "projects/test/topics/otlp", gcpPubSubCompressionKey: "snappy"}, invalid: true},
		{name: "invalid timeout", config: map[string]string{gcpPubSubTopicKey: "projects/test/topics/otlp", gcpTimeoutKey: "-1s"}, invalid: true},
		{name: "profiles only", config: map[string]string{gcpPubSubTopicKey: "projects/test/topics/otlp"}, signals: []common.ObservabilitySignal{common.ProfilesObservabilitySignal}, invalid: true},
		{name: "metrics only with project override", config: map[string]string{gcpPubSubTopicKey: "projects/test/topics/otlp", gcpProjectIdKey: "client-project", gcpPubSubCompressionKey: "none"}, signals: []common.ObservabilitySignal{common.MetricsObservabilitySignal}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Exporters: GenericMap{}, Service: Service{Pipelines: map[string]Pipeline{}}}
			names, err := (&GoogleCloudPubSub{}).ModifyConfig(&mockDestination{id: "pubsub", config: tt.config, signals: tt.signals}, cfg)
			if tt.invalid {
				require.Error(t, err)
				require.Empty(t, cfg.Exporters)
				require.Empty(t, cfg.Service.Pipelines)
				return
			}
			require.NoError(t, err)
			require.Equal(t, []string{"metrics/googlecloudpubsub-pubsub"}, names)
			exporter := cfg.Exporters["googlecloudpubsub/pubsub"].(GenericMap)
			require.Equal(t, "client-project", exporter["project"])
			require.NotContains(t, exporter, "compression")
			require.NotContains(t, exporter, "ordering")
		})
	}
}
