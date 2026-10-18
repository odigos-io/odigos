package config_test

import (
	"maps"
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	"github.com/stretchr/testify/require"
)

func TestAzureDataExplorerConfig(t *testing.T) {
	base := map[string]string{
		config.AzureDataExplorerClusterURI:    "https://test.kusto.windows.net/",
		config.AzureDataExplorerApplicationID: "client",
		config.AzureDataExplorerTenantID:      "tenant",
	}
	for _, mode := range []string{"service_principal", "managed_identity", "default"} {
		t.Run(mode, func(t *testing.T) {
			data := maps.Clone(base)
			data[config.AzureDataExplorerAuthMode] = mode
			data[config.AzureDataExplorerManagedIdentityID] = "system"
			data[config.AzureDataExplorerTracesTable] = "custom_traces"
			data[config.AzureDataExplorerTracesMapping] = "trace_mapping"
			data[config.AzureDataExplorerIngestionType] = "managed"
			dest := rawDataDestination{ID: "odigos.io.dest.adx-test", Type: common.AzureDataExplorerDestinationType, Data: data, Signals: []common.ObservabilitySignal{common.TracesObservabilitySignal, common.MetricsObservabilitySignal, common.LogsObservabilitySignal}}
			cfg := config.Config{Exporters: config.GenericMap{}, Service: config.Service{Pipelines: map[string]config.Pipeline{}}}
			configers, err := config.LoadConfigers()
			require.NoError(t, err)
			names, err := configers[dest.Type].ModifyConfig(dest, &cfg)
			require.NoError(t, err)
			require.Len(t, names, 3)
			exporterName := "azuredataexplorer/azuredataexplorer-" + dest.ID
			exp := cfg.Exporters[exporterName].(config.GenericMap)
			require.Equal(t, "https://test.kusto.windows.net", exp["cluster_uri"])
			require.Equal(t, "oteldb", exp["db_name"])
			require.Equal(t, "managed", exp["ingestion_type"])
			require.Equal(t, "custom_traces", exp["traces_table_name"])
			require.Equal(t, "trace_mapping", exp["traces_table_json_mapping"])
			require.Equal(t, "OTELMetrics", exp["metrics_table_name"])
			require.Equal(t, "OTELLogs", exp["logs_table_name"])
			for _, name := range names {
				require.Equal(t, []string{exporterName}, cfg.Service.Pipelines[name].Exporters)
			}
			switch mode {
			case "service_principal":
				require.Equal(t, config.SecretEnvPlaceholder(config.AzureDataExplorerApplicationKey, dest), exp["application_key"])
				require.NotContains(t, exp, "managed_identity_id")
				require.NotContains(t, exp, "use_azure_auth")
			case "managed_identity":
				require.Equal(t, "system", exp["managed_identity_id"])
				require.NotContains(t, exp, "application_key")
				require.NotContains(t, exp, "use_azure_auth")
			case "default":
				require.Equal(t, true, exp["use_azure_auth"])
				require.NotContains(t, exp, "application_key")
				require.NotContains(t, exp, "managed_identity_id")
			}
		})
	}
	t.Run("signal selection and defaults", func(t *testing.T) {
		dest := rawDataDestination{ID: "logs", Data: base, Signals: []common.ObservabilitySignal{common.LogsObservabilitySignal}}
		cfg := config.Config{Exporters: config.GenericMap{}, Service: config.Service{Pipelines: map[string]config.Pipeline{}}}
		names, err := (&config.AzureDataExplorer{}).ModifyConfig(dest, &cfg)
		require.NoError(t, err)
		require.Equal(t, []string{"logs/azuredataexplorer-logs"}, names)
		exp := cfg.Exporters["azuredataexplorer/azuredataexplorer-logs"].(config.GenericMap)
		require.Equal(t, "queued", exp["ingestion_type"])
		require.NotContains(t, exp, "traces_table_name")
		require.NotContains(t, exp, "metrics_table_name")
	})
	for _, tc := range []struct{ name, key, value, message string }{
		{"missing URI", config.AzureDataExplorerClusterURI, " ", config.AzureDataExplorerClusterURI},
		{"HTTP URI", config.AzureDataExplorerClusterURI, "http://test.kusto.windows.net", "HTTPS"},
		{"userinfo", config.AzureDataExplorerClusterURI, "https://user:password@test.kusto.windows.net", "HTTPS"},
		{"path", config.AzureDataExplorerClusterURI, "https://test.kusto.windows.net/v1", "HTTPS"},
		{"query", config.AzureDataExplorerClusterURI, "https://test.kusto.windows.net?x=y", "HTTPS"},
		{"unknown auth", config.AzureDataExplorerAuthMode, "password", config.AzureDataExplorerAuthMode},
		{"missing client", config.AzureDataExplorerApplicationID, "", config.AzureDataExplorerApplicationID},
		{"missing tenant", config.AzureDataExplorerTenantID, "", config.AzureDataExplorerTenantID},
		{"missing identity", config.AzureDataExplorerAuthMode, "managed_identity", config.AzureDataExplorerManagedIdentityID},
		{"invalid ingestion", config.AzureDataExplorerIngestionType, "batched", config.AzureDataExplorerIngestionType},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := maps.Clone(base)
			data[tc.key] = tc.value
			dest := rawDataDestination{ID: "bad", Data: data, Signals: []common.ObservabilitySignal{common.LogsObservabilitySignal}}
			cfg := config.Config{Exporters: config.GenericMap{}, Service: config.Service{Pipelines: map[string]config.Pipeline{}}}
			_, err := (&config.AzureDataExplorer{}).ModifyConfig(dest, &cfg)
			require.ErrorContains(t, err, tc.message)
			require.Empty(t, cfg.Exporters)
			require.Empty(t, cfg.Service.Pipelines)
		})
	}
	t.Run("profiles rejected", func(t *testing.T) {
		dest := rawDataDestination{ID: "profiles", Data: base, Signals: []common.ObservabilitySignal{common.ProfilesObservabilitySignal}}
		cfg := config.Config{Exporters: config.GenericMap{}, Service: config.Service{Pipelines: map[string]config.Pipeline{}}}
		_, err := (&config.AzureDataExplorer{}).ModifyConfig(dest, &cfg)
		require.ErrorContains(t, err, "does not support profiles")
		require.Empty(t, cfg.Exporters)
	})
}
