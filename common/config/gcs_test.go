package config

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/odigos-io/odigos/common"
)

func TestGoogleCloudStorageConfig(t *testing.T) {
	for _, signals := range [][]common.ObservabilitySignal{
		{common.TracesObservabilitySignal}, {common.LogsObservabilitySignal},
		{common.TracesObservabilitySignal, common.LogsObservabilitySignal},
	} {
		for _, fields := range []map[string]string{
			{},
			{"GCS_BUCKET": ""},
			{"GCS_BUCKET": "archive", "GCS_PREFIX": "odigos", "GCS_PARTITION": "hour", "GCS_FILE_PREFIX": "export-", "GCS_MARSHALER": "otlp_proto"},
		} {
			cfg := &Config{Exporters: GenericMap{}, Service: Service{Pipelines: map[string]Pipeline{}}}
			dest := &mockDestination{id: "test-id", signals: signals, config: fields}
			names, err := (&GoogleCloudStorage{}).ModifyConfig(dest, cfg)
			require.NoError(t, err)
			require.Len(t, names, len(signals))
			exp := cfg.Exporters["googlecloudstorage/test-id"].(GenericMap)
			require.NotContains(t, exp, "pcs")
			upload := exp["gcs"].(GenericMap)
			if fields["GCS_BUCKET"] == "" {
				require.Equal(t, GenericMap{"bucket": "odigos-otlp"}, upload)
			} else {
				require.Equal(t, GenericMap{"bucket": "archive", "prefix": "odigos", "partition": "hour", "file_prefix": "export-"}, upload)
				require.Equal(t, "otlp_proto", exp["marshaler_name"])
			}
			for _, signal := range signals {
				prefix := "traces"
				if signal == common.LogsObservabilitySignal {
					prefix = "logs"
				}
				require.Equal(t, []string{"googlecloudstorage/test-id"}, cfg.Service.Pipelines[prefix+"/gcs-test-id"].Exporters)
			}
			require.NotContains(t, cfg.Service.Pipelines, "metrics/gcs-test-id")
		}
	}
}

func TestGoogleCloudStorageUnsupportedSignals(t *testing.T) {
	for _, signal := range []common.ObservabilitySignal{common.MetricsObservabilitySignal, common.ProfilesObservabilitySignal} {
		cfg := &Config{Exporters: GenericMap{}, Service: Service{Pipelines: map[string]Pipeline{}}}
		_, err := (&GoogleCloudStorage{}).ModifyConfig(&mockDestination{signals: []common.ObservabilitySignal{signal}}, cfg)
		require.Error(t, err)
		require.Empty(t, cfg.Exporters)
	}
}
