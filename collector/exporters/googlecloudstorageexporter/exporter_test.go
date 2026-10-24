package googlecloudstorageexporter

import (
	"context"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func TestConfigDecodeAndValidation(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	require.NoError(t, confmap.NewFromStringMap(map[string]any{
		"gcs":            map[string]any{"bucket": "archive", "partition": "hour", "prefix": "data", "file_prefix": "export-"},
		"marshaler_name": "otlp_proto",
	}).Unmarshal(cfg))
	require.NoError(t, cfg.Validate())
	require.Equal(t, "archive", cfg.GCSUploader.GCSBucket)
	cfg.GCSUploader.GCSBucket = ""
	require.ErrorContains(t, cfg.Validate(), "bucket")
	cfg.GCSUploader.GCSBucket = "archive"
	cfg.GCSUploader.GCSPartition = "day"
	require.ErrorContains(t, cfg.Validate(), "partition")
	cfg.GCSUploader.GCSPartition = "minute"
	cfg.MarshalerName = "unknown"
	require.ErrorIs(t, cfg.Validate(), ErrUnknownMarshaler)
}

func TestUploadBatchesAndRetry(t *testing.T) {
	for _, format := range []string{"otlp_json", "otlp_proto", "otlp"} {
		t.Run(format, func(t *testing.T) {
			var mu sync.Mutex
			objects := map[string][]byte{}
			attempts := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				attempts++
				if attempts == 1 {
					w.WriteHeader(http.StatusServiceUnavailable)
					return
				}
				mediaType, params, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
				if err != nil || !strings.HasPrefix(mediaType, "multipart/") {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				reader := multipart.NewReader(req.Body, params["boundary"])
				meta, err := reader.NextPart()
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				var attrs struct {
					Name string `json:"name"`
				}
				if err := json.NewDecoder(meta).Decode(&attrs); err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				part, err := reader.NextPart()
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				body, err := io.ReadAll(part)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				objects[attrs.Name] = body
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"name": attrs.Name, "bucket": "archive"})
			}))
			defer server.Close()
			t.Setenv("STORAGE_EMULATOR_HOST", server.URL)
			cfg := createDefaultConfig().(*Config)
			cfg.GCSUploader.GCSBucket = "archive"
			cfg.GCSUploader.GCSPrefix = "odigos"
			cfg.GCSUploader.FilePrefix = "batch-"
			cfg.MarshalerName = format
			cfg.RetryConfig = configretry.BackOffConfig{Enabled: true, InitialInterval: time.Millisecond, MaxInterval: 5 * time.Millisecond, MaxElapsedTime: time.Second}
			factory := NewFactory()
			settings := exportertest.NewNopSettings(factory.Type())
			tracesExporter, err := factory.CreateTraces(context.Background(), settings, cfg)
			require.NoError(t, err)
			logsExporter, err := factory.CreateLogs(context.Background(), settings, cfg)
			require.NoError(t, err)
			require.NoError(t, tracesExporter.Start(context.Background(), componenttest.NewNopHost()))
			require.NoError(t, logsExporter.Start(context.Background(), componenttest.NewNopHost()))
			defer func() {
				require.NoError(t, tracesExporter.Shutdown(context.Background()))
				require.NoError(t, logsExporter.Shutdown(context.Background()))
			}()
			traces := newTestTraces()
			logs := newTestLogs()
			for range 4 {
				traces.ResourceSpans().At(0).ScopeSpans().At(0).Spans().AppendEmpty().SetName("batch-span")
				logs.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().AppendEmpty().Body().SetStr("batch-log")
			}
			require.NoError(t, tracesExporter.ConsumeTraces(context.Background(), traces))
			require.NoError(t, logsExporter.ConsumeLogs(context.Background(), logs))
			require.Eventually(t, func() bool {
				mu.Lock()
				defer mu.Unlock()
				return len(objects) == 2
			}, 5*time.Second, 10*time.Millisecond)
			mu.Lock()
			defer mu.Unlock()
			require.Greater(t, attempts, 2)
			require.Len(t, objects, 2)
			for name, body := range objects {
				require.True(t, strings.HasPrefix(name, "odigos/year="), name)
				if strings.Contains(name, "batch-traces_") {
					var decoded ptrace.Traces
					if format == "otlp_json" {
						decoded, err = (&ptrace.JSONUnmarshaler{}).UnmarshalTraces(body)
					} else {
						decoded, err = (&ptrace.ProtoUnmarshaler{}).UnmarshalTraces(body)
					}
					require.NoError(t, err)
					require.Equal(t, 5, decoded.SpanCount())
				} else {
					require.Contains(t, name, "batch-logs_")
					var decoded plog.Logs
					if format == "otlp_json" {
						decoded, err = (&plog.JSONUnmarshaler{}).UnmarshalLogs(body)
					} else {
						decoded, err = (&plog.ProtoUnmarshaler{}).UnmarshalLogs(body)
					}
					require.NoError(t, err)
					require.Equal(t, 5, decoded.LogRecordCount())
				}
			}
		})
	}
}

var _ component.Config = (*Config)(nil)
var _ exporter.Factory = NewFactory()

func newTestTraces() ptrace.Traces {
	traces := ptrace.NewTraces()
	traces.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty().SetName("test-span")
	return traces
}

func newTestLogs() plog.Logs {
	logs := plog.NewLogs()
	logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().Body().SetStr("test-log")
	return logs
}

func TestUploadPermissionError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"code":403,"message":"storage.objects.create denied"}}`))
	}))
	defer server.Close()
	t.Setenv("STORAGE_EMULATOR_HOST", server.URL)
	cfg := createDefaultConfig().(*Config)
	cfg.GCSUploader.GCSBucket = "archive"
	cfg.QueueConfig = configoptional.None[exporterhelper.QueueBatchConfig]()
	cfg.RetryConfig.Enabled = false
	factory := NewFactory()
	exp, err := factory.CreateLogs(context.Background(), exportertest.NewNopSettings(factory.Type()), cfg)
	require.NoError(t, err)
	require.NoError(t, exp.Start(context.Background(), componenttest.NewNopHost()))
	defer func() { require.NoError(t, exp.Shutdown(context.Background())) }()
	require.ErrorContains(t, exp.ConsumeLogs(context.Background(), newTestLogs()), "storage.objects.create denied")
}

func TestLegacyObjectKeyLayout(t *testing.T) {
	timestamp := time.Date(2026, 10, 13, 12, 34, 0, 0, time.UTC)
	require.Regexp(t, `^/year=2026/month=10/day=13/hour=12/minute=34/traces_[0-9]+\.json$`, getGCSKey(timestamp, "", "minute", "", "traces", "json"))
	require.Regexp(t, `^archive/year=2026/month=10/day=13/hour=12/export-logs_[0-9]+\.proto$`, getGCSKey(timestamp, "archive", "hour", "export-", "logs", "proto"))
}
