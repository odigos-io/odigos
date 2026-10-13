package config

import (
	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/require"
	"testing"
)

func victoriaLogsConfig() Config {
	return Config{Exporters: GenericMap{}, Extensions: GenericMap{}, Service: Service{Pipelines: map[string]Pipeline{}}}
}

func TestVictoriaLogs(t *testing.T) {
	configers, err := LoadConfigers()
	require.NoError(t, err)
	require.Contains(t, configers, common.VictoriaLogsDestinationType)
	dest := &mockDestination{id: "logs.one", config: map[string]string{"VICTORIA_LOGS_ENDPOINT": "http://localhost:9428/"}}
	cfg := victoriaLogsConfig()
	pipelines, err := (&VictoriaLogs{}).ModifyConfig(dest, &cfg)
	require.NoError(t, err)
	require.Equal(t, []string{"logs/victorialogs-logs.one"}, pipelines)
	require.Len(t, cfg.Service.Pipelines, 1)
	exporter := cfg.Exporters["otlp_http/victorialogs-logs.one"].(GenericMap)
	require.Equal(t, "http://localhost:9428/insert/opentelemetry/v1/logs", exporter["logs_endpoint"])
	require.Equal(t, "gzip", exporter["compression"])
	require.Equal(t, true, exporter["tls"].(GenericMap)["insecure"])
	require.Empty(t, cfg.Extensions)
}

func TestVictoriaLogsAuthAndHeaders(t *testing.T) {
	for _, auth := range []string{"basic", "bearer"} {
		t.Run(auth, func(t *testing.T) {
			dest := &mockDestination{id: "one", config: map[string]string{
				"VICTORIA_LOGS_ENDPOINT": "https://example.com/proxy/insert/opentelemetry/v1/logs?_stream_fields=service.name",
				"VICTORIA_LOGS_AUTH":     auth, "VICTORIA_LOGS_USERNAME": "test", "VICTORIA_LOGS_CA_PEM": "private-ca",
				"VICTORIA_LOGS_ACCOUNT_ID": "12", "VICTORIA_LOGS_PROJECT_ID": "34", "VICTORIA_LOGS_STREAM_FIELDS": "service.name",
				"VICTORIA_LOGS_IGNORE_FIELDS": "sensitive", "VICTORIA_LOGS_EXTRA_FIELDS": "env=test", "VICTORIA_LOGS_COMPRESSION": "none",
			}}
			cfg := victoriaLogsConfig()
			_, err := (&VictoriaLogs{}).ModifyConfig(dest, &cfg)
			require.NoError(t, err)
			exporter := cfg.Exporters["otlp_http/victorialogs-one"].(GenericMap)
			require.Equal(t, dest.config["VICTORIA_LOGS_ENDPOINT"], exporter["logs_endpoint"])
			require.Equal(t, GenericMap{"insecure": false, "insecure_skip_verify": false, "ca_pem": "private-ca"}, exporter["tls"])
			require.Equal(t, map[string]string{"AccountID": "12", "ProjectID": "34", "VL-Stream-Fields": "service.name", "VL-Ignore-Fields": "sensitive", "VL-Extra-Fields": "env=test"}, exporter["headers"])
			if auth == "basic" {
				require.Equal(t, "${ODIGOS_DEST_one_VICTORIA_LOGS_PASSWORD}", cfg.Extensions["basicauth/victorialogs-one"].(GenericMap)["client_auth"].(GenericMap)["password"])
			} else {
				require.Equal(t, "${ODIGOS_DEST_one_VICTORIA_LOGS_TOKEN}", cfg.Extensions["bearertokenauth/victorialogs-one"].(GenericMap)["token"])
			}
		})
	}
}

func TestVictoriaLogsInvalidConfig(t *testing.T) {
	for _, endpoint := range []string{"", "ftp://localhost", "http:///insert/opentelemetry/v1/logs", "https://user:pass@example.com", "http://localhost:9428/v1/logs", "https://example.com#fragment"} {
		cfg := victoriaLogsConfig()
		_, err := (&VictoriaLogs{}).ModifyConfig(&mockDestination{id: "test", config: map[string]string{"VICTORIA_LOGS_ENDPOINT": endpoint}}, &cfg)
		require.Error(t, err, endpoint)
		require.Empty(t, cfg.Exporters)
	}
	for key, value := range map[string]string{"VICTORIA_LOGS_COMPRESSION": "zstd", "VICTORIA_LOGS_AUTH": "unsupported"} {
		cfg := victoriaLogsConfig()
		_, err := (&VictoriaLogs{}).ModifyConfig(&mockDestination{id: "test", config: map[string]string{"VICTORIA_LOGS_ENDPOINT": "http://localhost:9428", key: value}}, &cfg)
		require.Error(t, err)
	}
	cfg := victoriaLogsConfig()
	_, err := (&VictoriaLogs{}).ModifyConfig(&mockDestination{id: "test", signals: []common.ObservabilitySignal{common.MetricsObservabilitySignal}, config: map[string]string{"VICTORIA_LOGS_ENDPOINT": "http://localhost:9428"}}, &cfg)
	require.NoError(t, err)
	require.Empty(t, cfg.Exporters)
	require.Empty(t, cfg.Service.Pipelines)
}
