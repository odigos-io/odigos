package config

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/require"
)

func TestSkyWalkingConfig(t *testing.T) {
	for _, tc := range []struct {
		name     string
		endpoint string
		tls      bool
		want     string
		fail     bool
	}{
		{name: "default OAP port", endpoint: "oap", want: "oap:11800"},
		{name: "explicit port", endpoint: "oap:11801", want: "oap:11801"},
		{name: "IPv6", endpoint: "[::1]", want: "[::1]:11800"},
		{name: "TLS", endpoint: "https://oap:11800", tls: true, want: "oap:11800"},
		{name: "empty", fail: true},
		{name: "path", endpoint: "oap:11800/v1/traces", fail: true},
		{name: "TLS mismatch", endpoint: "https://oap", fail: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := map[string]string{skyWalkingEndpointKey: tc.endpoint}
			if tc.tls {
				data[skyWalkingTLSKey] = "true"
				data[skyWalkingCAKey] = "test-ca"
			}
			dest := &mockDestination{id: "odigos.io.dest.skywalking", config: data}
			cfg := &Config{Exporters: GenericMap{}, Service: Service{Pipelines: map[string]Pipeline{}}}
			pipelines, err := (&SkyWalking{}).ModifyConfig(dest, cfg)
			if tc.fail {
				require.Error(t, err)
				require.Empty(t, cfg.Exporters)
				require.Empty(t, cfg.Service.Pipelines)
				return
			}
			require.NoError(t, err)
			require.ElementsMatch(t, []string{"traces/skywalking-odigos.io.dest.skywalking", "metrics/skywalking-odigos.io.dest.skywalking", "logs/skywalking-odigos.io.dest.skywalking"}, pipelines)
			exporter := cfg.Exporters["otlp_grpc/skywalking-odigos.io.dest.skywalking"].(GenericMap)
			require.Equal(t, tc.want, exporter["endpoint"])
			require.Equal(t, "none", exporter["compression"])
			require.Equal(t, !tc.tls, exporter["tls"].(GenericMap)["insecure"])
			if tc.tls {
				require.Equal(t, "test-ca", exporter["tls"].(GenericMap)["ca_pem"])
			}
		})
	}
}

func TestSkyWalkingSignalsAndSecrets(t *testing.T) {
	for _, signal := range []common.ObservabilitySignal{common.TracesObservabilitySignal, common.MetricsObservabilitySignal, common.LogsObservabilitySignal, common.ProfilesObservabilitySignal} {
		t.Run(string(signal), func(t *testing.T) {
			dest := &mockDestination{id: "a.b-c", signals: []common.ObservabilitySignal{signal}, config: map[string]string{
				skyWalkingEndpointKey: "oap", skyWalkingAuthKey: "true", skyWalkingCompressionKey: "gzip",
			}}
			cfg := &Config{Exporters: GenericMap{}, Service: Service{Pipelines: map[string]Pipeline{}}}
			pipelines, err := (&SkyWalking{}).ModifyConfig(dest, cfg)
			if signal == common.ProfilesObservabilitySignal {
				require.Error(t, err)
				require.Empty(t, cfg.Exporters)
				require.Empty(t, cfg.Service.Pipelines)
				return
			}
			require.NoError(t, err)
			require.Len(t, pipelines, 1)
			exporter := cfg.Exporters["otlp_grpc/skywalking-a.b-c"].(GenericMap)
			require.Equal(t, "gzip", exporter["compression"])
			require.Equal(t, GenericMap{"authentication": "${ODIGOS_DEST_a_b_c_SKYWALKING_TOKEN}"}, exporter["headers"])
		})
	}
	dest := &mockDestination{config: map[string]string{skyWalkingEndpointKey: "oap", skyWalkingCompressionKey: "snappy"}}
	cfg := &Config{Exporters: GenericMap{}, Service: Service{Pipelines: map[string]Pipeline{}}}
	_, err := (&SkyWalking{}).ModifyConfig(dest, cfg)
	require.Error(t, err)
	require.Empty(t, cfg.Exporters)
	configers, err := LoadConfigers()
	require.NoError(t, err)
	require.IsType(t, &SkyWalking{}, configers[common.SkyWalkingDestinationType])
}
