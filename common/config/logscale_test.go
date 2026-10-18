package config

import (
	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestLogScaleConfig(t *testing.T) {
	for _, endpoint := range []string{"https://cloud.humio.com", "https://cloud.humio.com/", "https://cloud.humio.com/api/v1/ingest/otlp/", "http://logscale:8080"} {
		t.Run(endpoint, func(t *testing.T) {
			dest := &mockDestination{id: "odigos.io.dest.logscale-test", config: map[string]string{logScaleEndpointKey: endpoint}, signals: []common.ObservabilitySignal{common.LogsObservabilitySignal}}
			cfg := &Config{Exporters: GenericMap{}, Service: Service{Pipelines: map[string]Pipeline{}}}
			pipelines, err := (&LogScale{}).ModifyConfig(dest, cfg)
			require.NoError(t, err)
			require.Equal(t, []string{"logs/logscale-odigos.io.dest.logscale-test"}, pipelines)
			exporter := cfg.Exporters["otlp_http/logscale-odigos.io.dest.logscale-test"].(GenericMap)
			want := "https://cloud.humio.com/api/v1/ingest/otlp/v1/logs"
			if endpoint == "http://logscale:8080" {
				want = "http://logscale:8080/api/v1/ingest/otlp/v1/logs"
			}
			require.Equal(t, want, exporter["endpoint"].(string)+"/v1/logs")
			require.Equal(t, "proto", exporter["encoding"])
			require.Equal(t, "none", exporter["compression"])
			require.Equal(t, "Bearer ${ODIGOS_DEST_odigos_io_dest_logscale_test_LOGSCALE_INGEST_TOKEN}", exporter["headers"].(GenericMap)["Authorization"])
			require.Len(t, cfg.Service.Pipelines, 1)
		})
	}
}

func TestLogScaleInvalidEndpoint(t *testing.T) {
	for _, endpoint := range []string{"", "cloud.humio.com", "https://", "https:///api/v1/ingest/otlp", "ftp://cloud.humio.com", "https://cloud.humio.com/api/v1/ingest/otlp/v1/logs", "https://cloud.humio.com/wrong", "https://user:password@cloud.humio.com", "https://cloud.humio.com?token=secret", "https://cloud.humio.com#fragment"} {
		t.Run(endpoint, func(t *testing.T) {
			dest := &mockDestination{id: "test", config: map[string]string{logScaleEndpointKey: endpoint}, signals: []common.ObservabilitySignal{common.LogsObservabilitySignal}}
			cfg := &Config{Exporters: GenericMap{}, Service: Service{Pipelines: map[string]Pipeline{}}}
			_, err := (&LogScale{}).ModifyConfig(dest, cfg)
			require.Error(t, err)
			require.Contains(t, err.Error(), logScaleEndpointKey)
			require.Empty(t, cfg.Exporters)
			require.Empty(t, cfg.Service.Pipelines)
		})
	}
}

func TestLogScaleRejectsUnsupportedSignals(t *testing.T) {
	for _, signal := range []common.ObservabilitySignal{common.TracesObservabilitySignal, common.MetricsObservabilitySignal, common.ProfilesObservabilitySignal} {
		dest := &mockDestination{id: "test", config: map[string]string{logScaleEndpointKey: "https://cloud.humio.com"}, signals: []common.ObservabilitySignal{common.LogsObservabilitySignal, signal}}
		cfg := &Config{Exporters: GenericMap{}, Service: Service{Pipelines: map[string]Pipeline{}}}
		_, err := (&LogScale{}).ModifyConfig(dest, cfg)
		require.ErrorContains(t, err, "supports only logs")
		require.Empty(t, cfg.Exporters)
		require.Empty(t, cfg.Service.Pipelines)
	}
}
