package config

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newBrontoConfig() Config {
	return Config{Exporters: GenericMap{}, Service: Service{Pipelines: map[string]Pipeline{}}}
}

func TestBrontoAllSignalsEnabled(t *testing.T) {
	dest := &mockDestination{
		id:     "bronto.example",
		config: map[string]string{brontoRegion: "us"},
		signals: []common.ObservabilitySignal{
			common.TracesObservabilitySignal,
			common.MetricsObservabilitySignal,
			common.LogsObservabilitySignal,
		},
	}
	cfg := newBrontoConfig()

	pipelines, err := (&Bronto{}).ModifyConfig(dest, &cfg)
	require.NoError(t, err)

	assert.ElementsMatch(t, []string{
		"traces/bronto-bronto.example",
		"metrics/bronto-bronto.example",
		"logs/bronto-bronto.example",
	}, pipelines)
	assert.Equal(t, GenericMap{
		"endpoint": "https://ingestion.us.bronto.io",
		"headers": GenericMap{
			"x-bronto-api-key": "${ODIGOS_DEST_bronto_example_BRONTO_API_KEY}",
		},
	}, cfg.Exporters["otlp_http/bronto-bronto.example"])
}

func TestBrontoMissingRegionFallsBackToEu(t *testing.T) {
	dest := &mockDestination{
		id:      "bronto.example",
		config:  map[string]string{},
		signals: []common.ObservabilitySignal{common.LogsObservabilitySignal},
	}
	cfg := newBrontoConfig()

	_, err := (&Bronto{}).ModifyConfig(dest, &cfg)
	require.NoError(t, err)

	exporter := cfg.Exporters["otlp_http/bronto-bronto.example"].(GenericMap)
	assert.Equal(t, "https://ingestion.eu.bronto.io", exporter["endpoint"])
}

func TestBrontoInvalidRegionFails(t *testing.T) {
	dest := &mockDestination{
		id:      "bronto.example",
		config:  map[string]string{brontoRegion: "ingestion.eu.bronto.io"},
		signals: []common.ObservabilitySignal{common.LogsObservabilitySignal},
	}
	cfg := newBrontoConfig()

	_, err := (&Bronto{}).ModifyConfig(dest, &cfg)
	require.ErrorIs(t, err, ErrorBrontoRegionInvalid)
	assert.Empty(t, cfg.Exporters)
}

func TestBrontoCollectionHeader(t *testing.T) {
	dest := &mockDestination{
		id:      "bronto.example",
		config:  map[string]string{brontoRegion: "eu", brontoCollection: "prod-eu-west-1"},
		signals: []common.ObservabilitySignal{common.LogsObservabilitySignal},
	}
	cfg := newBrontoConfig()

	_, err := (&Bronto{}).ModifyConfig(dest, &cfg)
	require.NoError(t, err)

	exporter := cfg.Exporters["otlp_http/bronto-bronto.example"].(GenericMap)
	assert.Equal(t, GenericMap{
		"x-bronto-api-key":    "${ODIGOS_DEST_bronto_example_BRONTO_API_KEY}",
		"x-bronto-collection": "prod-eu-west-1",
	}, exporter["headers"])
}

func TestBrontoNoSupportedSignalFails(t *testing.T) {
	dest := &mockDestination{
		id:      "bronto.example",
		config:  map[string]string{brontoRegion: "eu"},
		signals: []common.ObservabilitySignal{common.ProfilesObservabilitySignal},
	}
	cfg := newBrontoConfig()

	_, err := (&Bronto{}).ModifyConfig(dest, &cfg)
	require.ErrorIs(t, err, ErrorBrontoNoSignals)
	assert.Empty(t, cfg.Exporters)
}
