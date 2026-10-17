package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/confmap"
	envprovider "go.opentelemetry.io/collector/confmap/provider/envprovider"
	fileprovider "go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/otelcol"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"github.com/odigos-io/odigos/common"
	destinationconfig "github.com/odigos-io/odigos/common/config"
	"github.com/odigos-io/odigos/common/config/testconnection"
)

func archiveTestConfig(t *testing.T, data map[string]string) destinationconfig.Config {
	t.Helper()
	dest := &testconnection.TestConnectionConfig{DestinationType: string(common.LocalFileArchiveDestinationType), ID: "archive", Config: data, Signals: []common.ObservabilitySignal{common.TracesObservabilitySignal, common.MetricsObservabilitySignal, common.LogsObservabilitySignal}}
	cfg := destinationconfig.Config{Exporters: destinationconfig.GenericMap{}, Service: destinationconfig.Service{Pipelines: map[string]destinationconfig.Pipeline{}}}
	_, err := (&destinationconfig.LocalFileArchive{}).ModifyConfig(dest, &cfg)
	require.NoError(t, err)
	return cfg
}

func TestLocalFileArchiveCollectorConfigLoads(t *testing.T) {
	cfg := archiveTestConfig(t, map[string]string{destinationconfig.LocalFileArchivePathKey: t.TempDir()})
	cfg.Receivers = destinationconfig.GenericMap{"otlp": destinationconfig.GenericMap{"protocols": destinationconfig.GenericMap{"grpc": destinationconfig.GenericMap{"endpoint": "127.0.0.1:0"}}}}
	for name, pipeline := range cfg.Service.Pipelines {
		pipeline.Receivers = []string{"otlp"}
		cfg.Service.Pipelines[name] = pipeline
	}
	data, err := json.Marshal(cfg)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "archive.json")
	require.NoError(t, os.WriteFile(path, data, 0600))
	col, err := otelcol.NewCollector(otelcol.CollectorSettings{BuildInfo: component.NewDefaultBuildInfo(), Factories: components, ConfigProviderSettings: otelcol.ConfigProviderSettings{ResolverSettings: confmap.ResolverSettings{URIs: []string{path}, ProviderFactories: []confmap.ProviderFactory{fileprovider.NewFactory(), envprovider.NewFactory()}, DefaultScheme: "env"}}})
	require.NoError(t, err)
	require.NoError(t, col.DryRun(context.Background()))
}

func archiveTestPayload(signal, format string) ([]byte, error) {
	switch signal {
	case "traces":
		data := ptrace.NewTraces()
		for _, service := range []string{"frontend", "inventory"} {
			resource := data.ResourceSpans().AppendEmpty()
			resource.Resource().Attributes().PutStr("service.name", service)
			resource.Resource().Attributes().PutStr("deployment.environment.name", "archive-test")
			scope := resource.ScopeSpans().AppendEmpty()
			scope.Scope().SetName("archive-test")
			scope.Scope().SetVersion("1.0")
			scope.Spans().AppendEmpty().SetName("GET /products")
		}
		if format == "proto" {
			return (&ptrace.ProtoMarshaler{}).MarshalTraces(data)
		}
		return (&ptrace.JSONMarshaler{}).MarshalTraces(data)
	case "metrics":
		data := pmetric.NewMetrics()
		for _, service := range []string{"frontend", "inventory"} {
			resource := data.ResourceMetrics().AppendEmpty()
			resource.Resource().Attributes().PutStr("service.name", service)
			scope := resource.ScopeMetrics().AppendEmpty()
			scope.Scope().SetName("archive-test")
			metric := scope.Metrics().AppendEmpty()
			metric.SetName("http.server.request.duration")
			point := metric.SetEmptyHistogram().DataPoints().AppendEmpty()
			point.SetCount(2)
			point.SetSum(0.25)
			point.ExplicitBounds().FromRaw([]float64{0.1, 0.5})
			point.BucketCounts().FromRaw([]uint64{1, 1, 0})
		}
		if format == "proto" {
			return (&pmetric.ProtoMarshaler{}).MarshalMetrics(data)
		}
		return (&pmetric.JSONMarshaler{}).MarshalMetrics(data)
	default:
		data := plog.NewLogs()
		for _, service := range []string{"frontend", "inventory"} {
			resource := data.ResourceLogs().AppendEmpty()
			resource.Resource().Attributes().PutStr("service.name", service)
			scope := resource.ScopeLogs().AppendEmpty()
			scope.Scope().SetName("archive-test")
			log := scope.LogRecords().AppendEmpty()
			log.SetSeverityText("INFO")
			log.Body().SetEmptyMap().PutStr("message", "request complete")
		}
		if format == "proto" {
			return (&plog.ProtoMarshaler{}).MarshalLogs(data)
		}
		return (&plog.JSONMarshaler{}).MarshalLogs(data)
	}
}

func writeArchiveBatch(t *testing.T, cfg destinationconfig.Config, signal string, payload []byte) error {
	t.Helper()
	factories, err := components()
	require.NoError(t, err)
	factory := factories.Exporters[component.MustNewType("file")]
	name := "file/archive-" + signal
	exporterCfg := factory.CreateDefaultConfig()
	require.NoError(t, confmap.NewFromStringMap(map[string]any(cfg.Exporters[name].(destinationconfig.GenericMap))).Unmarshal(exporterCfg))
	require.NoError(t, exporterCfg.(interface{ Validate() error }).Validate())
	settings := exportertest.NewNopSettings(component.MustNewType("file"))
	settings.ID = component.NewIDWithName(component.MustNewType("file"), "archive-"+signal)
	ctx := context.Background()
	var comp component.Component
	var consume func() error
	switch signal {
	case "traces":
		exp, err := factory.CreateTraces(ctx, settings, exporterCfg)
		require.NoError(t, err)
		comp = exp
		data, err := (&ptrace.JSONUnmarshaler{}).UnmarshalTraces(payload)
		require.NoError(t, err)
		consume = func() error { return exp.ConsumeTraces(ctx, data) }
	case "metrics":
		exp, err := factory.CreateMetrics(ctx, settings, exporterCfg)
		require.NoError(t, err)
		comp = exp
		data, err := (&pmetric.JSONUnmarshaler{}).UnmarshalMetrics(payload)
		require.NoError(t, err)
		consume = func() error { return exp.ConsumeMetrics(ctx, data) }
	default:
		exp, err := factory.CreateLogs(ctx, settings, exporterCfg)
		require.NoError(t, err)
		comp = exp
		data, err := (&plog.JSONUnmarshaler{}).UnmarshalLogs(payload)
		require.NoError(t, err)
		consume = func() error { return exp.ConsumeLogs(ctx, data) }
	}
	require.NoError(t, comp.Start(ctx, componenttest.NewNopHost()))
	err = consume()
	require.NoError(t, comp.Shutdown(ctx))
	return err
}

func TestLocalFileArchivePersistedReadback(t *testing.T) {
	for _, format := range []string{"json", "proto"} {
		for _, compression := range []string{"none", "zstd"} {
			for _, rotation := range []string{"0", "1"} {
				t.Run(format+"/"+compression+"/rotation="+rotation, func(t *testing.T) {
					cfg := archiveTestConfig(t, map[string]string{destinationconfig.LocalFileArchivePathKey: t.TempDir(), "FILE_ARCHIVE_FORMAT": format, "FILE_ARCHIVE_COMPRESSION": compression, "FILE_ARCHIVE_MAX_MEGABYTES": rotation})
					for _, signal := range []string{"traces", "metrics", "logs"} {
						payload, err := archiveTestPayload(signal, "json")
						require.NoError(t, err)
						for range 2 {
							require.NoError(t, writeArchiveBatch(t, cfg, signal, payload))
						}
						exporter := cfg.Exporters["file/archive-"+signal].(destinationconfig.GenericMap)
						data, err := os.ReadFile(exporter["path"].(string))
						require.NoError(t, err)
						expected, err := archiveTestPayload(signal, format)
						require.NoError(t, err)
						for range 2 {
							var batch []byte
							if format == "json" && compression == "none" {
								i := bytes.IndexByte(data, '\n')
								require.NotEqual(t, -1, i)
								batch = data[:i]
								data = data[i+1:]
							} else {
								require.GreaterOrEqual(t, len(data), 4)
								size := int(binary.BigEndian.Uint32(data[:4]))
								require.GreaterOrEqual(t, len(data), size+4)
								batch = data[4 : 4+size]
								data = data[4+size:]
								if compression == "zstd" {
									decoder, err := zstd.NewReader(nil)
									require.NoError(t, err)
									batch, err = decoder.DecodeAll(batch, nil)
									decoder.Close()
									require.NoError(t, err)
								}
							}
							if format == "json" {
								assert.JSONEq(t, string(expected), string(batch))
							} else {
								assert.Equal(t, expected, batch)
							}
						}
						assert.Empty(t, data)
					}
				})
			}
		}
	}
}

func TestLocalFileArchiveRotationAndOversizeRejection(t *testing.T) {
	cfg := archiveTestConfig(t, map[string]string{destinationconfig.LocalFileArchivePathKey: t.TempDir(), "FILE_ARCHIVE_MAX_MEGABYTES": "1", "FILE_ARCHIVE_MAX_BACKUPS": "2"})
	logs := plog.NewLogs()
	logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().Body().SetStr(strings.Repeat("x", 600000))
	payload, err := (&plog.JSONMarshaler{}).MarshalLogs(logs)
	require.NoError(t, err)
	for range 4 {
		require.NoError(t, writeArchiveBatch(t, cfg, "logs", payload))
	}
	path := cfg.Exporters["file/archive-logs"].(destinationconfig.GenericMap)["path"].(string)
	files, err := filepath.Glob(filepath.Join(filepath.Dir(path), "logs*.json"))
	require.NoError(t, err)
	require.Greater(t, len(files), 1)
	for _, file := range files {
		stat, err := os.Stat(file)
		require.NoError(t, err)
		assert.LessOrEqual(t, stat.Size(), int64(1024*1024))
	}
	logs.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Body().SetStr(strings.Repeat("x", 2*1024*1024))
	payload, err = (&plog.JSONMarshaler{}).MarshalLogs(logs)
	require.NoError(t, err)
	require.Error(t, writeArchiveBatch(t, cfg, "logs", payload))
}
