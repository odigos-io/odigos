package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/exporter/otlphttpexporter"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/odigos-io/odigos/common"
	cfg "github.com/odigos-io/odigos/common/config"
	"github.com/odigos-io/odigos/common/config/testconnection"
)

func victoriaLogsExporter(t *testing.T, endpoint string, logger ...*zap.Logger) exporter.Logs {
	t.Helper()
	dest := &testconnection.TestConnectionConfig{DestinationType: "victorialogs", ID: "e2e", Signals: []common.ObservabilitySignal{common.LogsObservabilitySignal}, Config: map[string]string{"VICTORIA_LOGS_ENDPOINT": endpoint, "VICTORIA_LOGS_STREAM_FIELDS": "service.name", "VICTORIA_LOGS_IGNORE_FIELDS": "sensitive", "VICTORIA_LOGS_EXTRA_FIELDS": "integration=odigos"}}
	current := cfg.Config{Exporters: cfg.GenericMap{}, Extensions: cfg.GenericMap{}, Service: cfg.Service{Pipelines: map[string]cfg.Pipeline{}}}
	_, err := (&cfg.VictoriaLogs{}).ModifyConfig(dest, &current)
	require.NoError(t, err)
	factory := otlphttpexporter.NewFactory()
	raw := current.Exporters["otlp_http/victorialogs-e2e"].(cfg.GenericMap)
	raw["sending_queue"] = map[string]any{"enabled": false}
	settings := factory.CreateDefaultConfig()
	encoded, err := json.Marshal(raw)
	require.NoError(t, err)
	var normalized map[string]any
	require.NoError(t, json.Unmarshal(encoded, &normalized))
	require.NoError(t, confmap.NewFromStringMap(normalized).Unmarshal(settings))
	require.NoError(t, settings.(*otlphttpexporter.Config).Validate())
	exporterSettings := exportertest.NewNopSettings(factory.Type())
	if len(logger) > 0 {
		exporterSettings.Logger = logger[0]
	}
	exp, err := factory.CreateLogs(context.Background(), exporterSettings, settings)
	require.NoError(t, err)
	require.NoError(t, exp.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() { require.NoError(t, exp.Shutdown(context.Background())) })
	return exp
}

func victoriaLogsBatch(run string, batch, count int, timestamp time.Time) plog.Logs {
	logs := plog.NewLogs()
	resource := logs.ResourceLogs().AppendEmpty()
	resource.Resource().Attributes().PutStr("service.name", "odigos-victorialogs-e2e")
	records := resource.ScopeLogs().AppendEmpty().LogRecords()
	for i := 0; i < count; i++ {
		record := records.AppendEmpty()
		record.SetTimestamp(pcommon.NewTimestampFromTime(timestamp))
		record.Body().SetStr(fmt.Sprintf("CORE-1731 batch=%d record=%d", batch, i))
		record.Attributes().PutStr("test.run", run)
		record.Attributes().PutStr("sensitive", "must-be-ignored")
		record.SetSeverityText("INFO")
	}
	return logs
}

func TestVictoriaLogsPartialSuccess(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		require.Equal(t, "/insert/opentelemetry/v1/logs", r.URL.Path)
		require.Equal(t, "gzip", r.Header.Get("Content-Encoding"))
		response := plogotlp.NewExportResponse()
		response.PartialSuccess().SetRejectedLogRecords(1)
		response.PartialSuccess().SetErrorMessage("one record rejected")
		body, err := response.MarshalProto()
		require.NoError(t, err)
		w.Header().Set("Content-Type", "application/x-protobuf")
		_, err = w.Write(body)
		require.NoError(t, err)
	}))
	defer server.Close()
	core, logs := observer.New(zap.WarnLevel)
	exp := victoriaLogsExporter(t, server.URL, zap.New(core))
	require.NoError(t, exp.ConsumeLogs(context.Background(), victoriaLogsBatch("partial", 0, 2, time.Now())))
	require.EqualValues(t, 1, attempts.Load(), "partial success must not retry accepted records")
	require.Equal(t, 1, logs.FilterMessage("Partial success response").Len())
	require.EqualValues(t, 1, logs.All()[0].ContextMap()["dropped_log_records"])
}

func TestVictoriaLogsLiveIngestion(t *testing.T) {
	endpoint := os.Getenv("VICTORIA_LOGS_E2E_ENDPOINT")
	if endpoint == "" {
		t.Skip("set VICTORIA_LOGS_E2E_ENDPOINT to a disposable VictoriaLogs instance with 1d retention")
	}
	run := fmt.Sprintf("core1731-%d", time.Now().UnixNano())
	exp := victoriaLogsExporter(t, endpoint)
	for batch := 0; batch < 3; batch++ {
		require.NoError(t, exp.ConsumeLogs(context.Background(), victoriaLogsBatch(run, batch, 10, time.Now())))
	}
	query := func(id string) []map[string]any {
		response, err := http.PostForm(endpoint+"/select/logsql/query", url.Values{"query": {`test.run:="` + id + `"`}, "limit": {"100"}})
		require.NoError(t, err)
		defer response.Body.Close()
		require.Equal(t, http.StatusOK, response.StatusCode)
		body, err := io.ReadAll(response.Body)
		require.NoError(t, err)
		var rows []map[string]any
		for _, line := range strings.Split(strings.TrimSpace(string(body)), "\n") {
			if line != "" {
				var row map[string]any
				require.NoError(t, json.Unmarshal([]byte(line), &row))
				rows = append(rows, row)
			}
		}
		return rows
	}
	require.Eventually(t, func() bool { return len(query(run)) == 30 }, 15*time.Second, 250*time.Millisecond)
	for _, row := range query(run) {
		require.Equal(t, "odigos-victorialogs-e2e", row["service.name"])
		require.Equal(t, "odigos", row["integration"])
		require.NotContains(t, row, "sensitive")
	}
	mixed := victoriaLogsBatch(run+"-mixed", 3, 1, time.Now())
	old := mixed.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().AppendEmpty()
	old.SetTimestamp(pcommon.NewTimestampFromTime(time.Now().Add(-72 * time.Hour)))
	old.Body().SetStr("CORE-1731 expired record")
	old.Attributes().PutStr("test.run", run+"-mixed")
	require.NoError(t, exp.ConsumeLogs(context.Background(), mixed))
	require.Eventually(t, func() bool { return len(query(run+"-mixed")) == 1 }, 15*time.Second, 250*time.Millisecond)
	t.Logf("Persisted and queried 30 records in 3 batches; mixed valid/expired batch stored 1 of 2 records. LogsQL: test.run:%q", run)
}
