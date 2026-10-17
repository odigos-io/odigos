package config

import (
	"strings"
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostHogConfig(t *testing.T) {
	for _, region := range []string{"", "us", "eu"} {
		t.Run(region, func(t *testing.T) {
			dest := &mockDestination{id: "posthog.test", config: map[string]string{posthogRegion: region, posthogCompression: "gzip"}, signals: []common.ObservabilitySignal{
				common.TracesObservabilitySignal, common.MetricsObservabilitySignal, common.LogsObservabilitySignal, common.ProfilesObservabilitySignal,
			}}
			cfg := Config{Exporters: GenericMap{}, Service: Service{Pipelines: map[string]Pipeline{}}}
			pipelines, err := (&PostHog{}).ModifyConfig(dest, &cfg)
			require.NoError(t, err)
			assert.Equal(t, []string{"metrics/posthog-posthog.test", "logs/posthog-posthog.test"}, pipelines)
			if region == "" {
				region = "us"
			}
			assert.Equal(t, GenericMap{
				"logs_endpoint":    "https://" + region + ".i.posthog.com/i/v1/logs",
				"metrics_endpoint": "https://" + region + ".i.posthog.com/i/v1/metrics",
				"compression":      "gzip",
				"headers":          GenericMap{"Authorization": "Bearer ${ODIGOS_DEST_posthog_test_POSTHOG_PROJECT_TOKEN}"},
			}, cfg.Exporters["otlp_http/posthog-posthog.test"])
			for _, pipeline := range cfg.Service.Pipelines {
				assert.Equal(t, []string{"otlp_http/posthog-posthog.test"}, pipeline.Exporters)
			}
		})
	}
	configers, err := LoadConfigers()
	require.NoError(t, err)
	assert.IsType(t, &PostHog{}, configers[common.PostHogDestinationType])
}

func TestPostHogSignalSelection(t *testing.T) {
	for _, signal := range []common.ObservabilitySignal{common.LogsObservabilitySignal, common.MetricsObservabilitySignal, common.TracesObservabilitySignal, common.ProfilesObservabilitySignal} {
		t.Run(string(signal), func(t *testing.T) {
			cfg := Config{Exporters: GenericMap{}, Service: Service{Pipelines: map[string]Pipeline{}}}
			pipelines, err := (&PostHog{}).ModifyConfig(&mockDestination{id: "test", config: map[string]string{}, signals: []common.ObservabilitySignal{signal}}, &cfg)
			require.NoError(t, err)
			if signal == common.LogsObservabilitySignal || signal == common.MetricsObservabilitySignal {
				assert.Equal(t, []string{strings.ToLower(string(signal)) + "/posthog-test"}, pipelines)
			} else {
				assert.Empty(t, pipelines)
			}
			assert.Equal(t, "none", cfg.Exporters["otlp_http/posthog-test"].(GenericMap)["compression"])
		})
	}
}

func TestPostHogInvalidConfig(t *testing.T) {
	for _, config := range []map[string]string{{posthogRegion: "https://example.com"}, {posthogCompression: "snappy"}} {
		cfg := Config{Exporters: GenericMap{}, Service: Service{Pipelines: map[string]Pipeline{}}}
		_, err := (&PostHog{}).ModifyConfig(&mockDestination{config: config}, &cfg)
		require.Error(t, err)
		assert.Empty(t, cfg.Exporters)
		assert.Empty(t, cfg.Service.Pipelines)
	}
}
