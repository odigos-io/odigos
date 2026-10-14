package config

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/require"
)

func TestYandexCloudMonitoringConfig(t *testing.T) {
	dest := &mockDestination{id: "test", config: map[string]string{
		yandexCloudMonitoringWorkspaceIDKey:   "mon-workspace_123",
		prometheusResourceAttributesLabelsKey: `["k8s.namespace.name"]`,
	}}
	cfg := &Config{Exporters: GenericMap{}, Extensions: GenericMap{}, Processors: GenericMap{}, Service: Service{Pipelines: map[string]Pipeline{}}}
	configers, err := LoadConfigers()
	require.NoError(t, err)
	pipelines, err := configers[common.YandexCloudMonitoringDestinationType].ModifyConfig(dest, cfg)
	require.NoError(t, err)
	require.Equal(t, []string{"metrics/yandexcloudmonitoring-test"}, pipelines)
	require.Len(t, cfg.Service.Pipelines, 1)
	exporter := cfg.Exporters["prometheusremotewrite/yandexcloudmonitoring-test"].(GenericMap)
	http := exporter["http"].(GenericMap)
	require.Equal(t, "https://monitoring.api.cloud.yandex.net/prometheus/workspaces/mon-workspace_123/api/v1/write", http["endpoint"])
	require.Equal(t, "snappy", http["compression"])
	require.Equal(t, GenericMap{"token": SecretEnvPlaceholder(yandexCloudMonitoringAPIKey, dest)}, cfg.Extensions["bearertokenauth/yandexcloudmonitoring-test"])
	require.Equal(t, false, exporter["send_metadata"])
	require.Equal(t, GenericMap{"enabled": false}, exporter["target_info"])
	require.Equal(t, []string{"filter/yandexcloudmonitoring-test", "transform/yandexcloudmonitoring-test"}, cfg.Service.Pipelines[pipelines[0]].Processors)
	require.Equal(t, 1, exporter["max_batch_request_parallelism"])
}

func TestYandexCloudMonitoringRejectsInvalidConfig(t *testing.T) {
	for _, workspace := range []string{"", "../other", "https://example.com", "id?x=y", "id\n"} {
		t.Run(workspace, func(t *testing.T) {
			dest := &mockDestination{id: "test", config: map[string]string{yandexCloudMonitoringWorkspaceIDKey: workspace}}
			_, err := (&YandexCloudMonitoring{}).ModifyConfig(dest, &Config{})
			require.Error(t, err)
		})
	}
	dest := &mockDestination{id: "test", signals: []common.ObservabilitySignal{common.TracesObservabilitySignal}, config: map[string]string{yandexCloudMonitoringWorkspaceIDKey: "mon123"}}
	_, err := (&YandexCloudMonitoring{}).ModifyConfig(dest, &Config{})
	require.ErrorContains(t, err, "metrics not enabled")
	dest.signals = []common.ObservabilitySignal{common.MetricsObservabilitySignal}
	dest.config[prometheusResourceAttributesLabelsKey] = "invalid-json"
	_, err = (&YandexCloudMonitoring{}).ModifyConfig(dest, &Config{})
	require.ErrorContains(t, err, "resource attribute labels")
}
