package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/kafkaexporter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/envprovider"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/otelcol"

	"github.com/odigos-io/odigos/common"
	destinationconfig "github.com/odigos-io/odigos/common/config"
)

type eventHubsTestDestination struct {
	encoding, compression string
}

func (d eventHubsTestDestination) GetID() string { return "azure-test" }
func (d eventHubsTestDestination) GetType() common.DestinationType {
	return common.AzureEventHubsDestinationType
}
func (d eventHubsTestDestination) GetSignals() []common.ObservabilitySignal {
	return []common.ObservabilitySignal{common.TracesObservabilitySignal, common.MetricsObservabilitySignal, common.LogsObservabilitySignal}
}
func (d eventHubsTestDestination) GetConfig() map[string]string {
	return map[string]string{"AZURE_EVENT_HUBS_NAMESPACE": "my-namespace", "AZURE_EVENT_HUBS_ENCODING": d.encoding, "AZURE_EVENT_HUBS_COMPRESSION": d.compression}
}

func TestAzureEventHubsCollectorConfig(t *testing.T) {
	for _, encoding := range []string{"otlp_proto", "otlp_json"} {
		for _, compression := range []string{"none", "gzip"} {
			t.Run(encoding+"/"+compression, func(t *testing.T) {
				t.Setenv("ConnectionString", "must-not-be-used-as-username")
				t.Setenv(destinationconfig.SecretEnvVarName("AZURE_EVENT_HUBS_CONNECTION_STRING", "azure-test"), "Endpoint=sb://my-namespace.servicebus.windows.net/;SharedAccessKeyName=send;SharedAccessKey=test")
				cfg := &destinationconfig.Config{
					Exporters: destinationconfig.GenericMap{},
					Receivers: destinationconfig.GenericMap{"otlp": destinationconfig.GenericMap{"protocols": destinationconfig.GenericMap{"grpc": destinationconfig.GenericMap{"endpoint": "localhost:0"}}}},
					Service:   destinationconfig.Service{Pipelines: map[string]destinationconfig.Pipeline{}},
				}
				pipelines, err := (&destinationconfig.AzureEventHubs{}).ModifyConfig(eventHubsTestDestination{encoding, compression}, cfg)
				require.NoError(t, err)
				for _, name := range pipelines {
					pipe := cfg.Service.Pipelines[name]
					pipe.Receivers = []string{"otlp"}
					cfg.Service.Pipelines[name] = pipe
				}
				data, err := json.Marshal(cfg)
				require.NoError(t, err)
				path := filepath.Join(t.TempDir(), "collector.yaml")
				require.NoError(t, os.WriteFile(path, data, 0600))
				provider, err := otelcol.NewConfigProvider(otelcol.ConfigProviderSettings{ResolverSettings: confmap.ResolverSettings{
					URIs: []string{path}, DefaultScheme: "env", ProviderFactories: []confmap.ProviderFactory{fileprovider.NewFactory(), envprovider.NewFactory()},
				}})
				require.NoError(t, err)
				t.Cleanup(func() { require.NoError(t, provider.Shutdown(context.Background())) })
				factories, err := components()
				require.NoError(t, err)
				loaded, err := provider.Get(context.Background(), factories)
				require.NoError(t, err)
				require.NoError(t, loaded.Validate())
				for _, exporter := range loaded.Exporters {
					kafka := exporter.(*kafkaexporter.Config)
					assert.Equal(t, "$ConnectionString", kafka.ClientConfig.Authentication.SASL.Username)
					assert.Contains(t, string(kafka.ClientConfig.Authentication.SASL.Password), "SharedAccessKeyName=send")
					assert.False(t, kafka.ClientConfig.TLS.Insecure)
					assert.False(t, kafka.ClientConfig.TLS.InsecureSkipVerify)
					assert.Equal(t, 1, kafka.Producer.FlushMaxMessages)
				}
				col, err := otelcol.NewCollector(otelcol.CollectorSettings{BuildInfo: component.NewDefaultBuildInfo(), Factories: components,
					ConfigProviderSettings: otelcol.ConfigProviderSettings{ResolverSettings: confmap.ResolverSettings{URIs: []string{path}, DefaultScheme: "env", ProviderFactories: []confmap.ProviderFactory{fileprovider.NewFactory(), envprovider.NewFactory()}}},
				})
				require.NoError(t, err)
				require.NoError(t, col.DryRun(context.Background()))
			})
		}
	}
}
