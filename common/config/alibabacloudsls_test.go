package config

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/require"
)

func TestAlibabaCloudSLS(t *testing.T) {
	for _, tt := range []struct {
		name      string
		endpoint  string
		wantError bool
	}{
		{name: "regional host", endpoint: "eu-central-1.log.aliyuncs.com"},
		{name: "HTTPS", endpoint: "https://eu-central-1.log.aliyuncs.com/"},
		{name: "project endpoint", endpoint: "https://project.eu-central-1.log.aliyuncs.com"},
		{name: "plaintext", endpoint: "http://eu-central-1.log.aliyuncs.com", wantError: true},
		{name: "path", endpoint: "https://example.com/logstores/logs", wantError: true},
		{name: "credentials", endpoint: "https://user:pass@example.com", wantError: true},
		{name: "query", endpoint: "https://example.com?secret=value", wantError: true},
		{name: "fragment", endpoint: "https://example.com#logs", wantError: true},
		{name: "missing host", endpoint: "https:///", wantError: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dest := &mockDestination{id: "test", config: map[string]string{SLS_ENDPOINT: tt.endpoint, SLS_PROJECT: "project", SLS_LOGSTORE: "logs"}, signals: []common.ObservabilitySignal{common.LogsObservabilitySignal, common.TracesObservabilitySignal, common.MetricsObservabilitySignal, common.ProfilesObservabilitySignal}}
			cfg := &Config{Exporters: GenericMap{}, Service: Service{Pipelines: map[string]Pipeline{}}}
			pipelines, err := (&AlibabaCloudSLS{}).ModifyConfig(dest, cfg)
			if tt.wantError {
				require.Error(t, err)
				require.Empty(t, cfg.Exporters)
				require.Empty(t, cfg.Service.Pipelines)
				return
			}
			require.NoError(t, err)
			require.Equal(t, []string{"logs/alibabacloudsls-test"}, pipelines)
			require.Len(t, cfg.Service.Pipelines, 1)
			exporter := cfg.Exporters["alibabacloud_logservice/alibabacloudsls-test"].(GenericMap)
			require.Equal(t, "https://eu-central-1.log.aliyuncs.com", exporter["endpoint"])
			require.Equal(t, SecretEnvPlaceholder(SLS_ACCESS_KEY_ID, dest), exporter["access_key_id"])
			require.Equal(t, SecretEnvPlaceholder(SLS_ACCESS_KEY_SECRET, dest), exporter["access_key_secret"])
		})
	}
}

func TestAlibabaCloudSLSRequiredFields(t *testing.T) {
	for _, key := range []string{SLS_ENDPOINT, SLS_PROJECT, SLS_LOGSTORE} {
		t.Run(key, func(t *testing.T) {
			data := map[string]string{SLS_ENDPOINT: "example.com", SLS_PROJECT: "project", SLS_LOGSTORE: "logs"}
			data[key] = " "
			cfg := &Config{Exporters: GenericMap{}, Service: Service{Pipelines: map[string]Pipeline{}}}
			_, err := (&AlibabaCloudSLS{}).ModifyConfig(&mockDestination{config: data}, cfg)
			require.ErrorContains(t, err, key)
			require.Empty(t, cfg.Exporters)
		})
	}
}

func TestAlibabaCloudSLSRegistrationAndSignals(t *testing.T) {
	configers, err := LoadConfigers()
	require.NoError(t, err)
	require.IsType(t, &AlibabaCloudSLS{}, configers[common.AlibabaCloudSLSDestinationType])
	cfg := &Config{Exporters: GenericMap{}, Service: Service{Pipelines: map[string]Pipeline{}}}
	pipelines, err := (&AlibabaCloudSLS{}).ModifyConfig(&mockDestination{config: map[string]string{SLS_ENDPOINT: "example.com", SLS_PROJECT: "project", SLS_LOGSTORE: "logs"}, signals: []common.ObservabilitySignal{common.TracesObservabilitySignal, common.MetricsObservabilitySignal, common.ProfilesObservabilitySignal}}, cfg)
	require.NoError(t, err)
	require.Empty(t, pipelines)
	require.Empty(t, cfg.Exporters)
}
