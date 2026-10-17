package config

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/require"
)

func TestAlibabaCloudSignalsAndTransport(t *testing.T) {
	for _, tt := range []struct {
		name      string
		endpoint  string
		tls       string
		signals   []common.ObservabilitySignal
		pipelines []string
		wantError bool
	}{
		{name: "legacy plaintext", endpoint: "tracing-analysis-dc-hz.aliyuncs.com:8090", signals: []common.ObservabilitySignal{common.TracesObservabilitySignal}, pipelines: []string{"traces/alibaba-test"}},
		{name: "documented grpc URL", endpoint: "http://tracing-analysis-dc-hz.aliyuncs.com:8090", signals: []common.ObservabilitySignal{common.MetricsObservabilitySignal}, pipelines: []string{"metrics/alibaba-test"}},
		{name: "TLS traces and metrics", endpoint: "https://example.com:443", tls: "true", pipelines: []string{"traces/alibaba-test", "metrics/alibaba-test"}},
		{name: "no OTLP logs or profiles", endpoint: "example.com:8090", signals: []common.ObservabilitySignal{common.LogsObservabilitySignal, common.ProfilesObservabilitySignal}},
		{name: "TLS with plaintext URL", endpoint: "http://example.com:8090", tls: "true", wantError: true},
		{name: "TLS URL without TLS", endpoint: "https://example.com:443", wantError: true},
		{name: "HTTP ingest path", endpoint: "http://example.com/api/otlp/traces", wantError: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dest := &mockDestination{id: "test", config: map[string]string{ALIBABA_ENDPOINT: tt.endpoint, genericOtlpTlsKey: tt.tls, genericOtlpCaPemKey: "test-ca"}, signals: tt.signals}
			cfg := &Config{Exporters: GenericMap{}, Service: Service{Pipelines: map[string]Pipeline{}}}
			pipelines, err := (&AlibabaCloud{}).ModifyConfig(dest, cfg)
			if tt.wantError {
				require.Error(t, err)
				require.Empty(t, cfg.Exporters)
				require.Empty(t, cfg.Service.Pipelines)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.pipelines, pipelines)
			require.Len(t, cfg.Service.Pipelines, len(tt.pipelines))
			exporter := cfg.Exporters["otlp_grpc/alibaba-test"].(GenericMap)
			require.Equal(t, SecretEnvPlaceholder(ALIBABA_TOKEN, dest), exporter["headers"].(GenericMap)["Authentication"])
			tls := exporter["tls"].(GenericMap)
			require.Equal(t, tt.tls != "true", tls["insecure"])
			if tt.tls == "true" {
				require.Equal(t, "test-ca", tls["ca_pem"])
			} else {
				require.NotContains(t, tls, "ca_pem")
			}
		})
	}
}

func TestAlibabaCloudMissingEndpoint(t *testing.T) {
	cfg := &Config{Exporters: GenericMap{}, Service: Service{Pipelines: map[string]Pipeline{}}}
	_, err := (&AlibabaCloud{}).ModifyConfig(&mockDestination{config: map[string]string{}}, cfg)
	require.Error(t, err)
	require.Empty(t, cfg.Exporters)
}
