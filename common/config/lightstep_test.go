package config_test

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	"github.com/stretchr/testify/require"
)

func TestLightstepLegacyConfiguration(t *testing.T) {
	dest := rawDataDestination{
		ID:      "legacy",
		Type:    common.LightstepDestinationType,
		Signals: []common.ObservabilitySignal{common.TracesObservabilitySignal},
	}
	cfg := &config.Config{Exporters: config.GenericMap{}, Service: config.Service{Pipelines: map[string]config.Pipeline{}}}
	pipelines, err := (&config.Lightstep{}).ModifyConfig(dest, cfg)
	require.NoError(t, err)
	require.Equal(t, []string{"traces/lightstep-legacy"}, pipelines)
	require.Equal(t, config.GenericMap{
		"endpoint": "ingest.lightstep.com:443",
		"headers":  config.GenericMap{"lightstep-access-token": config.SecretEnvPlaceholder("LIGHTSTEP_ACCESS_TOKEN", dest)},
	}, cfg.Exporters["otlp_grpc/lightstep-legacy"])
	require.Equal(t, []string{"otlp_grpc/lightstep-legacy"}, cfg.Service.Pipelines[pipelines[0]].Exporters)
}
