package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/observiq/bindplane-otel-contrib/exporter/azureloganalyticsexporter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/envprovider"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/otelcol"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"github.com/odigos-io/odigos/common"
	destinationconfig "github.com/odigos-io/odigos/common/config"
)

type sentinelCollectorDestination struct{}

func (sentinelCollectorDestination) GetID() string { return "sentinel-test" }
func (sentinelCollectorDestination) GetType() common.DestinationType {
	return common.MicrosoftSentinelDestinationType
}
func (sentinelCollectorDestination) GetSignals() []common.ObservabilitySignal {
	return []common.ObservabilitySignal{common.LogsObservabilitySignal}
}
func (sentinelCollectorDestination) GetConfig() map[string]string {
	return map[string]string{
		"MICROSOFT_SENTINEL_ENDPOINT":  "https://test.eastus-1.ingest.monitor.azure.com",
		"MICROSOFT_SENTINEL_CLIENT_ID": "client-id", "MICROSOFT_SENTINEL_TENANT_ID": "tenant-id",
		"MICROSOFT_SENTINEL_RULE_ID": "dcr-example", "MICROSOFT_SENTINEL_STREAM_NAME": "Custom-OdigosLogs",
	}
}

func TestMicrosoftSentinelCollectorConfigAndMapping(t *testing.T) {
	ctx := context.Background()
	t.Setenv(destinationconfig.SecretEnvVarName("MICROSOFT_SENTINEL_CLIENT_SECRET", "sentinel-test"), "test-secret")
	cfg := &destinationconfig.Config{
		Receivers: destinationconfig.GenericMap{"otlp": destinationconfig.GenericMap{"protocols": destinationconfig.GenericMap{"grpc": destinationconfig.GenericMap{"endpoint": "localhost:0"}}}},
		Exporters: destinationconfig.GenericMap{}, Processors: destinationconfig.GenericMap{},
		Service: destinationconfig.Service{Pipelines: map[string]destinationconfig.Pipeline{}},
	}
	pipelines, err := (&destinationconfig.MicrosoftSentinel{}).ModifyConfig(sentinelCollectorDestination{}, cfg)
	require.NoError(t, err)
	pipe := cfg.Service.Pipelines[pipelines[0]]
	pipe.Receivers = []string{"otlp"}
	cfg.Service.Pipelines[pipelines[0]] = pipe
	data, err := json.Marshal(cfg)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "collector.yaml")
	require.NoError(t, os.WriteFile(path, data, 0600))
	settings := otelcol.ConfigProviderSettings{ResolverSettings: confmap.ResolverSettings{
		URIs: []string{path}, DefaultScheme: "env", ProviderFactories: []confmap.ProviderFactory{fileprovider.NewFactory(), envprovider.NewFactory()},
	}}
	provider, err := otelcol.NewConfigProvider(settings)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, provider.Shutdown(ctx)) })
	factories, err := components()
	require.NoError(t, err)
	loaded, err := provider.Get(ctx, factories)
	require.NoError(t, err)
	require.NoError(t, loaded.Validate())
	for _, exporter := range loaded.Exporters {
		azure := exporter.(*azureloganalyticsexporter.Config)
		assert.Equal(t, "test-secret", azure.ClientSecret)
		assert.Equal(t, "Custom-OdigosLogs", azure.StreamName)
		assert.Empty(t, azure.RawLogField)
		assert.True(t, azure.QueueConfig.HasValue())
		assert.True(t, azure.BackOffConfig.Enabled)
	}
	col, err := otelcol.NewCollector(otelcol.CollectorSettings{
		BuildInfo: component.NewDefaultBuildInfo(), Factories: components, ConfigProviderSettings: settings,
	})
	require.NoError(t, err)
	require.NoError(t, col.DryRun(ctx))

	var mapped plog.Logs
	sink, err := consumer.NewLogs(func(_ context.Context, logs plog.Logs) error { mapped = logs; return nil })
	require.NoError(t, err)
	id := component.MustNewIDWithName("transform", "microsoftsentinel-sentinel-test")
	transform, err := factories.Processors[id.Type()].CreateLogs(ctx, processor.Settings{
		ID: id, TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop(), MeterProvider: noop.NewMeterProvider(), TracerProvider: trace.NewNoopTracerProvider()},
	}, loaded.Processors[id], sink)
	require.NoError(t, err)
	require.NoError(t, transform.Start(ctx, nil))
	t.Cleanup(func() { require.NoError(t, transform.Shutdown(ctx)) })
	logs := plog.NewLogs()
	observed := pcommon.NewTimestampFromTime(time.Now().Add(-time.Minute))
	for _, service := range []string{"pricing", "currency"} {
		rl := logs.ResourceLogs().AppendEmpty()
		rl.Resource().Attributes().PutStr("service.name", service)
		rl.Resource().Attributes().PutStr("sentinel_rule_id", "dcr-untrusted")
		sl := rl.ScopeLogs().AppendEmpty()
		sl.Scope().SetName("odigos-test")
		for _, kind := range []string{"string", "map", "array", "integer", "missing-timestamp"} {
			lr := sl.LogRecords().AppendEmpty()
			lr.Attributes().PutStr("source", service)
			lr.Attributes().PutStr("sentinel_stream_name", "Custom-Untrusted")
			lr.SetSeverityText("INFO")
			lr.SetSeverityNumber(plog.SeverityNumberInfo)
			if kind != "missing-timestamp" {
				lr.SetObservedTimestamp(observed)
			}
			switch kind {
			case "map":
				lr.Body().SetEmptyMap().PutStr("message", "ordinary log")
			case "array":
				lr.Body().SetEmptySlice().AppendEmpty().SetStr("ordinary log")
			case "integer":
				lr.Body().SetInt(42)
			default:
				lr.Body().SetStr("ordinary log\nsecond line")
			}
		}
	}
	require.NoError(t, transform.ConsumeLogs(ctx, logs))
	require.Equal(t, 10, mapped.LogRecordCount())
	for i, service := range []string{"pricing", "currency"} {
		rl := mapped.ResourceLogs().At(i)
		assert.NotContains(t, rl.Resource().Attributes().AsRaw(), "sentinel_rule_id")
		for j := 0; j < rl.ScopeLogs().At(0).LogRecords().Len(); j++ {
			lr := rl.ScopeLogs().At(0).LogRecords().At(j)
			body := lr.Body().Map().AsRaw()
			assert.Equal(t, service, body["ResourceAttributes"].(map[string]any)["service.name"])
			assert.Equal(t, service, body["LogAttributes"].(map[string]any)["source"])
			assert.Equal(t, "odigos-test", body["ScopeName"])
			assert.IsType(t, "", body["RawData"])
			assert.NotContains(t, lr.Attributes().AsRaw(), "sentinel_stream_name")
			assert.Equal(t, plog.SeverityNumberInfo, lr.SeverityNumber())
			if j < 4 {
				assert.Equal(t, observed, lr.Timestamp())
			} else {
				assert.Greater(t, lr.Timestamp(), observed)
			}
		}
		body := rl.ScopeLogs().At(0).LogRecords().At(1).Body().Map().AsRaw()
		assert.JSONEq(t, `{"message":"ordinary log"}`, body["RawData"].(string))
	}
}
