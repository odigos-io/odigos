package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
)

type adxTestDestination struct{ data map[string]string }

func (d adxTestDestination) GetType() common.DestinationType {
	return common.AzureDataExplorerDestinationType
}
func (d adxTestDestination) GetID() string                { return "adx-test" }
func (d adxTestDestination) GetConfig() map[string]string { return d.data }
func (d adxTestDestination) GetSignals() []common.ObservabilitySignal {
	return []common.ObservabilitySignal{common.TracesObservabilitySignal, common.MetricsObservabilitySignal, common.LogsObservabilitySignal}
}

func TestAzureDataExplorerExporterConfigLoads(t *testing.T) {
	factories, err := components()
	require.NoError(t, err)
	factory := factories.Exporters[component.MustNewType("azuredataexplorer")]
	require.NotNil(t, factory)
	for _, mode := range []string{"service_principal", "managed_identity", "default"} {
		for _, ingestion := range []string{"queued", "managed"} {
			t.Run(mode+"/"+ingestion, func(t *testing.T) {
				dest := adxTestDestination{data: map[string]string{
					config.AzureDataExplorerClusterURI:        "https://test.kusto.windows.net",
					config.AzureDataExplorerAuthMode:          mode,
					config.AzureDataExplorerApplicationID:     "client",
					config.AzureDataExplorerTenantID:          "tenant",
					config.AzureDataExplorerManagedIdentityID: "system",
					config.AzureDataExplorerIngestionType:     ingestion,
				}}
				cfg := config.Config{Exporters: config.GenericMap{}, Service: config.Service{Pipelines: map[string]config.Pipeline{}}}
				_, err := (&config.AzureDataExplorer{}).ModifyConfig(dest, &cfg)
				require.NoError(t, err)
				exp := cfg.Exporters["azuredataexplorer/azuredataexplorer-adx-test"].(config.GenericMap)
				if mode == "service_principal" {
					exp["application_key"] = "test-secret"
				}
				serialized, err := json.Marshal(exp)
				require.NoError(t, err)
				var plain map[string]any
				require.NoError(t, json.Unmarshal(serialized, &plain))
				decoded := factory.CreateDefaultConfig()
				require.NoError(t, confmap.NewFromStringMap(plain).Unmarshal(decoded))
				require.NoError(t, confmap.Validate(decoded))
				if mode == "service_principal" {
					plain["application_key"] = ""
					require.NoError(t, confmap.NewFromStringMap(plain).Unmarshal(decoded))
					require.Error(t, confmap.Validate(decoded), "missing secret must fail collector validation")
				}
				if mode == "managed_identity" {
					plain["managed_identity_id"] = "invalid"
					require.NoError(t, confmap.NewFromStringMap(plain).Unmarshal(decoded))
					require.ErrorContains(t, confmap.Validate(decoded), "UUID")
				}
			})
		}
	}
}
