package config

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLegacySplunkUsesOTLP(t *testing.T) {
	for _, realm := range []string{"us0", "eu0"} {
		t.Run(realm, func(t *testing.T) {
			dest := &mockDestination{id: "splunk.example", config: map[string]string{splunkRealm: realm}}
			cfg := Config{Exporters: GenericMap{}, Service: Service{Pipelines: map[string]Pipeline{}}}
			pipelines, err := (&Splunk{}).ModifyConfig(dest, &cfg)
			require.NoError(t, err)
			assert.Equal(t, []string{"traces/splunk-splunk.example"}, pipelines)
			assert.Equal(t, GenericMap{"otlp_http/splunk.example": GenericMap{
				"headers":         GenericMap{"X-SF-Token": "${ODIGOS_DEST_splunk_example_SPLUNK_ACCESS_TOKEN}"},
				"traces_endpoint": "https://ingest." + realm + ".signalfx.com/v2/trace/otlp",
			}}, cfg.Exporters)
			assert.Equal(t, []string{"otlp_http/splunk.example"}, cfg.Service.Pipelines[pipelines[0]].Exporters)
		})
	}
}

func TestLegacySplunkMissingRealm(t *testing.T) {
	cfg := Config{Exporters: GenericMap{}, Service: Service{Pipelines: map[string]Pipeline{}}}
	_, err := (&Splunk{}).ModifyConfig(&mockDestination{id: "splunk"}, &cfg)
	require.Error(t, err)
	assert.Empty(t, cfg.Exporters)
	assert.Empty(t, cfg.Service.Pipelines)
}

func TestLegacySplunkWithoutTraces(t *testing.T) {
	cfg := Config{Exporters: GenericMap{}, Service: Service{Pipelines: map[string]Pipeline{}}}
	pipelines, err := (&Splunk{}).ModifyConfig(&mockDestination{
		id: "splunk", config: map[string]string{splunkRealm: "us0"}, signals: []common.ObservabilitySignal{common.MetricsObservabilitySignal},
	}, &cfg)
	require.NoError(t, err)
	assert.Empty(t, pipelines)
	assert.Empty(t, cfg.Exporters)
	assert.Empty(t, cfg.Service.Pipelines)
}
