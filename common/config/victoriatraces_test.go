package config

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/require"
)

func TestVictoriaTracesConfig(t *testing.T) {
	for _, tt := range []struct {
		name     string
		data     map[string]string
		signals  []common.ObservabilitySignal
		endpoint string
		wantErr  bool
	}{
		{name: "root URL", data: map[string]string{victoriaTracesEndpoint: "http://localhost:10428/"}, endpoint: "http://localhost:10428/insert/opentelemetry/v1/traces"},
		{name: "cloud bearer", data: map[string]string{victoriaTracesEndpoint: "https://example.cloud.victoriametrics.com/insert/opentelemetry/v1/traces", victoriaTracesAuth: "bearer"}, endpoint: "https://example.cloud.victoriametrics.com/insert/opentelemetry/v1/traces"},
		{name: "proxy basic", data: map[string]string{victoriaTracesEndpoint: "https://proxy.example/tenant/insert/opentelemetry/v1/traces", victoriaTracesAuth: "basic", otlpHttpBasicAuthUsernameKey: "user", otlpHttpCaPemKey: "ca", otlpHttpCompression: "none", otlpHttpHeaders: `[{"key":"AccountID","value":"42"}]`}, endpoint: "https://proxy.example/tenant/insert/opentelemetry/v1/traces"},
		{name: "missing endpoint", data: map[string]string{}, wantErr: true},
		{name: "empty endpoint", data: map[string]string{victoriaTracesEndpoint: " "}, wantErr: true},
		{name: "unsupported scheme", data: map[string]string{victoriaTracesEndpoint: "grpc://localhost:4317"}, wantErr: true},
		{name: "missing host", data: map[string]string{victoriaTracesEndpoint: "https:///insert/opentelemetry/v1/traces"}, wantErr: true},
		{name: "URL credentials", data: map[string]string{victoriaTracesEndpoint: "https://user:password@example.com"}, wantErr: true},
		{name: "URL query", data: map[string]string{victoriaTracesEndpoint: "https://example.com?token=secret"}, wantErr: true},
		{name: "URL fragment", data: map[string]string{victoriaTracesEndpoint: "https://example.com/#traces"}, wantErr: true},
		{name: "unknown auth", data: map[string]string{victoriaTracesEndpoint: "http://localhost:10428", victoriaTracesAuth: "bad"}, wantErr: true},
		{name: "basic without username", data: map[string]string{victoriaTracesEndpoint: "http://localhost:10428", victoriaTracesAuth: "basic"}, wantErr: true},
		{name: "unsupported compression", data: map[string]string{victoriaTracesEndpoint: "http://localhost:10428", otlpHttpCompression: "snappy"}, wantErr: true},
		{name: "invalid headers", data: map[string]string{victoriaTracesEndpoint: "http://localhost:10428", otlpHttpHeaders: "invalid"}, wantErr: true},
		{name: "unsupported signal", data: map[string]string{victoriaTracesEndpoint: "http://localhost:10428"}, signals: []common.ObservabilitySignal{common.TracesObservabilitySignal, common.MetricsObservabilitySignal}, wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			signals := tt.signals
			if signals == nil {
				signals = []common.ObservabilitySignal{common.TracesObservabilitySignal}
			}
			dest := &mockDestination{id: "vt-test", config: tt.data, signals: signals}
			cfg := Config{Exporters: GenericMap{}, Extensions: GenericMap{}, Service: Service{Pipelines: map[string]Pipeline{}}}
			pipelines, err := (&VictoriaTraces{}).ModifyConfig(dest, &cfg)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, []string{"traces/victoriatraces-vt-test"}, pipelines)
			require.Len(t, cfg.Service.Pipelines, 1)
			require.Equal(t, []string{"otlp_http/victoriatraces-vt-test"}, cfg.Service.Pipelines[pipelines[0]].Exporters)
			require.Len(t, cfg.Exporters, 1)
			exporter := cfg.Exporters["otlp_http/victoriatraces-vt-test"].(GenericMap)
			require.Equal(t, tt.endpoint, exporter["traces_endpoint"])
			require.NotContains(t, exporter, "endpoint")
			require.Equal(t, len(tt.data[otlpHttpCompression]) == 0 || tt.data[otlpHttpCompression] == "gzip", exporter["compression"] == "gzip")
			require.Equal(t, tt.endpoint[:5] != "https", exporter["tls"].(GenericMap)["insecure"])
			require.NotContains(t, dest.config, otlpHttpTracesEndpointKey)
			if tt.data[victoriaTracesAuth] == "bearer" {
				require.Equal(t, SecretEnvPlaceholder(victoriaTracesToken, dest), cfg.Extensions["bearertokenauth/victoriatraces-vt-test"].(GenericMap)["token"])
				require.Equal(t, GenericMap{"authenticator": "bearertokenauth/victoriatraces-vt-test"}, exporter["auth"])
			}
			if tt.data[victoriaTracesAuth] == "basic" {
				require.Equal(t, "ca", exporter["tls"].(GenericMap)["ca_pem"])
				require.Equal(t, map[string]string{"AccountID": "42"}, exporter["headers"])
				require.Equal(t, SecretEnvPlaceholder(otlpHttpBasicAuthPasswordKey, dest), cfg.Extensions["basicauth/otlphttp-vt-test"].(GenericMap)["client_auth"].(GenericMap)["password"])
			}
		})
	}
	configers, err := LoadConfigers()
	require.NoError(t, err)
	require.IsType(t, &VictoriaTraces{}, configers[common.VictoriaTracesDestinationType])
}
