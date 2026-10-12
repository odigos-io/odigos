package testconnection

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	desttest "github.com/odigos-io/odigos/common/config/testconnection"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/exporter/otlphttpexporter"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
)

func newLogScaleTestExporter(t *testing.T, endpoint string, queued bool) exporter.Logs {
	t.Helper()
	dest := &desttest.TestConnectionConfig{DestinationType: "logscale", ID: "test", Config: map[string]string{"LOGSCALE_ENDPOINT": endpoint}, Signals: []common.ObservabilitySignal{common.LogsObservabilitySignal}}
	generated := &config.Config{Exporters: config.GenericMap{}, Service: config.Service{Pipelines: map[string]config.Pipeline{}}}
	_, err := (&config.LogScale{}).ModifyConfig(dest, generated)
	require.NoError(t, err)
	raw := generated.Exporters["otlp_http/logscale-test"].(config.GenericMap)
	raw["headers"] = map[string]any{"Authorization": "Bearer test-ingest-token"}
	factory := otlphttpexporter.NewFactory()
	cfg := factory.CreateDefaultConfig().(*otlphttpexporter.Config)
	encoded, err := json.Marshal(raw)
	require.NoError(t, err)
	var plain map[string]any
	require.NoError(t, json.Unmarshal(encoded, &plain))
	require.NoError(t, confmap.NewFromStringMap(plain).Unmarshal(cfg))
	require.NoError(t, cfg.Validate())
	if !queued {
		cfg.QueueConfig = configoptional.None[exporterhelper.QueueBatchConfig]()
	}
	cfg.RetryConfig.InitialInterval = time.Millisecond
	cfg.RetryConfig.MaxInterval = time.Millisecond
	cfg.RetryConfig.MaxElapsedTime = time.Second
	exp, err := factory.CreateLogs(context.Background(), exportertest.NewNopSettings(factory.Type()), cfg)
	require.NoError(t, err)
	require.NoError(t, exp.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() { require.NoError(t, exp.Shutdown(context.Background())) })
	return exp
}

func logScaleTestLogs(batch string) plog.Logs {
	logs := plog.NewLogs()
	resource := logs.ResourceLogs().AppendEmpty()
	resource.Resource().Attributes().PutStr("service.name", "odigos-logscale-test")
	records := resource.ScopeLogs().AppendEmpty().LogRecords()
	for i := 0; i < 2; i++ {
		record := records.AppendEmpty()
		record.Body().SetStr(batch)
		record.Attributes().PutStr("test.batch", batch)
	}
	return logs
}

func TestLogScaleMultiBatchTransport(t *testing.T) {
	batches := make(chan plog.Logs, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/ingest/otlp/v1/logs" || r.Header.Get("Authorization") != "Bearer test-ingest-token" || r.Header.Get("Content-Type") != "application/x-protobuf" || r.Header.Get("Content-Encoding") != "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		request := plogotlp.NewExportRequest()
		if err := request.UnmarshalProto(body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		batches <- request.Logs()
		w.Header().Set("Content-Type", "application/x-protobuf")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	exp := newLogScaleTestExporter(t, server.URL, false)
	for _, batch := range []string{"batch-1", "batch-2"} {
		require.NoError(t, exp.ConsumeLogs(context.Background(), logScaleTestLogs(batch)))
		received := <-batches
		require.Equal(t, 2, received.LogRecordCount())
		resource := received.ResourceLogs().At(0)
		name, ok := resource.Resource().Attributes().Get("service.name")
		require.True(t, ok)
		require.Equal(t, "odigos-logscale-test", name.Str())
		for i := 0; i < 2; i++ {
			record := resource.ScopeLogs().At(0).LogRecords().At(i)
			require.Equal(t, batch, record.Body().Str())
			attr, ok := record.Attributes().Get("test.batch")
			require.True(t, ok)
			require.Equal(t, batch, attr.Str())
		}
	}
}

func TestLogScaleTransportRejections(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusTooManyRequests, http.StatusServiceUnavailable, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			var attempts atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if attempts.Add(1) == 1 {
					w.WriteHeader(status)
					return
				}
				w.Header().Set("Content-Type", "application/x-protobuf")
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()
			exp := newLogScaleTestExporter(t, server.URL, false)
			err := exp.ConsumeLogs(context.Background(), logScaleTestLogs("rejection-test"))
			if status == http.StatusUnauthorized || status == http.StatusForbidden || status == http.StatusInternalServerError {
				require.Error(t, err)
				require.True(t, consumererror.IsPermanent(err))
				require.Equal(t, int32(1), attempts.Load())
			} else {
				require.NoError(t, err)
				require.Equal(t, int32(2), attempts.Load())
			}
		})
	}
}

func TestLogScaleSplitsLargeBatch(t *testing.T) {
	sizes := make(chan int, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		request := plogotlp.NewExportRequest()
		if err := request.UnmarshalProto(body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		sizes <- request.Logs().LogRecordCount()
		w.Header().Set("Content-Type", "application/x-protobuf")
	}))
	defer server.Close()
	exp := newLogScaleTestExporter(t, server.URL, true)
	logs := logScaleTestLogs("large-batch")
	records := logs.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords()
	for records.Len() < 12001 {
		records.AppendEmpty().Body().SetStr("large-batch")
	}
	require.NoError(t, exp.ConsumeLogs(context.Background(), logs))
	received := 0
	timeout := time.After(10 * time.Second)
	for received < 12001 {
		select {
		case size := <-sizes:
			require.LessOrEqual(t, size, 5000)
			received += size
		case <-timeout:
			t.Fatalf("timed out after receiving %d of 12001 log records", received)
		}
	}
	require.Equal(t, 12001, received)
}
