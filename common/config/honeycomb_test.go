package config

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newHoneycombConfig() Config {
	return Config{Exporters: GenericMap{}, Service: Service{Pipelines: map[string]Pipeline{}}}
}

// honeycomb.yaml declares metrics as a supported signal, so a destination created for metrics
// alone must be configurable. It used to be rejected because the configurer required traces.
func TestHoneycombMetricsOnlyEnabled(t *testing.T) {
	dest := &mockDestination{
		id:      "honeycomb.example",
		config:  map[string]string{honeycombEndpoint: "api.eu1.honeycomb.io"},
		signals: []common.ObservabilitySignal{common.MetricsObservabilitySignal},
	}
	cfg := newHoneycombConfig()

	pipelines, err := (&Honeycomb{}).ModifyConfig(dest, &cfg)
	require.NoError(t, err)

	assert.Equal(t, []string{"metrics/honeycomb-honeycomb.example"}, pipelines)
	assert.Equal(t, GenericMap{
		"endpoint": "api.eu1.honeycomb.io:443",
		"headers": GenericMap{
			"x-honeycomb-team": "${ODIGOS_DEST_honeycomb_example_HONEYCOMB_API_KEY}",
		},
	}, cfg.Exporters["otlp_grpc/honeycomb-honeycomb.example"])
	assert.NotContains(t, cfg.Service.Pipelines, "traces/honeycomb-honeycomb.example")
	assert.NotContains(t, cfg.Service.Pipelines, "logs/honeycomb-honeycomb.example")
}

// Same for logs, the other signal honeycomb.yaml declares and the configurer used to reject.
func TestHoneycombLogsOnlyEnabled(t *testing.T) {
	dest := &mockDestination{
		id:      "honeycomb.example",
		config:  map[string]string{honeycombEndpoint: "api.honeycomb.io"},
		signals: []common.ObservabilitySignal{common.LogsObservabilitySignal},
	}
	cfg := newHoneycombConfig()

	pipelines, err := (&Honeycomb{}).ModifyConfig(dest, &cfg)
	require.NoError(t, err)

	assert.Equal(t, []string{"logs/honeycomb-honeycomb.example"}, pipelines)
	assert.Equal(t,
		[]string{"otlp_grpc/honeycomb-honeycomb.example"},
		cfg.Service.Pipelines["logs/honeycomb-honeycomb.example"].Exporters,
	)
	assert.NotContains(t, cfg.Service.Pipelines, "traces/honeycomb-honeycomb.example")
}

func TestHoneycombMetricsAndLogsWithoutTraces(t *testing.T) {
	dest := &mockDestination{
		id:     "honeycomb.example",
		config: map[string]string{honeycombEndpoint: "api.honeycomb.io"},
		signals: []common.ObservabilitySignal{
			common.MetricsObservabilitySignal,
			common.LogsObservabilitySignal,
		},
	}
	cfg := newHoneycombConfig()

	pipelines, err := (&Honeycomb{}).ModifyConfig(dest, &cfg)
	require.NoError(t, err)

	assert.ElementsMatch(t, []string{
		"metrics/honeycomb-honeycomb.example",
		"logs/honeycomb-honeycomb.example",
	}, pipelines)
	assert.NotContains(t, cfg.Service.Pipelines, "traces/honeycomb-honeycomb.example")
}

func TestHoneycombAllSignalsEnabled(t *testing.T) {
	dest := &mockDestination{
		id:     "honeycomb.example",
		config: map[string]string{honeycombEndpoint: "api.honeycomb.io"},
		signals: []common.ObservabilitySignal{
			common.TracesObservabilitySignal,
			common.MetricsObservabilitySignal,
			common.LogsObservabilitySignal,
		},
	}
	cfg := newHoneycombConfig()

	pipelines, err := (&Honeycomb{}).ModifyConfig(dest, &cfg)
	require.NoError(t, err)

	assert.ElementsMatch(t, []string{
		"traces/honeycomb-honeycomb.example",
		"metrics/honeycomb-honeycomb.example",
		"logs/honeycomb-honeycomb.example",
	}, pipelines)
	assert.Len(t, cfg.Exporters, 1)
}

// profiles is declared unsupported in honeycomb.yaml, so a destination with no supported signal
// must still fail instead of leaving an exporter behind with no pipeline to carry it.
func TestHoneycombNoSupportedSignalFails(t *testing.T) {
	dest := &mockDestination{
		id:      "honeycomb.example",
		config:  map[string]string{honeycombEndpoint: "api.honeycomb.io"},
		signals: []common.ObservabilitySignal{common.ProfilesObservabilitySignal},
	}
	cfg := newHoneycombConfig()

	pipelines, err := (&Honeycomb{}).ModifyConfig(dest, &cfg)
	require.ErrorIs(t, err, ErrorHoneycombNoSignals)

	assert.Empty(t, pipelines)
	assert.Empty(t, cfg.Exporters)
	assert.Empty(t, cfg.Service.Pipelines)
}

// The endpoint is a required field in honeycomb.yaml, but a Destination CR can omit it since
// nothing validates it, so the US instance stays the fallback.
func TestHoneycombMissingEndpointFallsBackToUsInstance(t *testing.T) {
	dest := &mockDestination{
		id:      "honeycomb.example",
		config:  map[string]string{},
		signals: []common.ObservabilitySignal{common.MetricsObservabilitySignal},
	}
	cfg := newHoneycombConfig()

	_, err := (&Honeycomb{}).ModifyConfig(dest, &cfg)
	require.NoError(t, err)

	exporter := cfg.Exporters["otlp_grpc/honeycomb-honeycomb.example"].(GenericMap)
	assert.Equal(t, "api.honeycomb.io:443", exporter["endpoint"])
}
