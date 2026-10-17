package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	sls "github.com/aliyun/aliyun-log-go-sdk"
	"github.com/gogo/protobuf/proto"
	"github.com/pierrec/lz4/v4"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/envprovider"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/processor/processortest"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/odigos-io/odigos/common"
	odigosconfig "github.com/odigos-io/odigos/common/config"
)

type slsTestDestination struct{ endpoint string }

func (slsTestDestination) GetID() string { return "sls-test" }
func (slsTestDestination) GetType() common.DestinationType {
	return common.AlibabaCloudSLSDestinationType
}
func (slsTestDestination) GetSignals() []common.ObservabilitySignal {
	return []common.ObservabilitySignal{common.LogsObservabilitySignal}
}
func (d slsTestDestination) GetConfig() map[string]string {
	return map[string]string{
		odigosconfig.SLS_ENDPOINT: d.endpoint,
		odigosconfig.SLS_PROJECT:  "project",
		odigosconfig.SLS_LOGSTORE: "logs",
	}
}

func TestAlibabaCloudSLSWire(t *testing.T) {
	var mode atomic.Int32
	var attempts atomic.Int32
	var mu sync.Mutex
	var received []*sls.Log
	var batchCounts []int
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NotNil(t, r.TLS)
		require.Equal(t, "project.example.com", r.Host)
		require.Equal(t, "POST", r.Method)
		require.Equal(t, "/logstores/logs", r.URL.Path)
		require.Equal(t, "lz4", r.Header.Get("x-log-compresstype"))
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		headers := map[string]string{"Date": r.Header.Get("Date"), "Content-Type": r.Header.Get("Content-Type")}
		for key, values := range r.Header {
			if strings.HasPrefix(strings.ToLower(key), "x-log-") || strings.HasPrefix(strings.ToLower(key), "x-acs-") {
				headers[strings.ToLower(key)] = values[0]
			}
		}
		require.NoError(t, sls.NewSignerV1("test-access-key", "test-secret").Sign(r.Method, r.URL.RequestURI(), headers, body))
		require.Equal(t, headers["Authorization"], r.Header.Get("Authorization"))
		size, err := strconv.Atoi(r.Header.Get("x-log-bodyrawsize"))
		require.NoError(t, err)
		raw := make([]byte, size)
		n, err := lz4.UncompressBlock(body, raw)
		require.NoError(t, err)
		require.Equal(t, size, n)
		group := &sls.LogGroup{}
		require.NoError(t, proto.Unmarshal(raw, group))
		attempt := attempts.Add(1)
		w.Header().Set("x-log-requestid", "test-request")
		if mode.Load() == 1 && attempt == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"errorCode":"ServerBusy","errorMessage":"retry"}`))
			return
		}
		if mode.Load() == 2 {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"errorCode":"InvalidParameter","errorMessage":"rejected"}`))
			return
		}
		mu.Lock()
		received = append(received, group.Logs...)
		batchCounts = append(batchCounts, len(group.Logs))
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// The SDK clones DefaultTransport; route its project hostname to our TLS test
	// server while keeping certificate verification enabled with a test root.
	previous := http.DefaultTransport
	transport := previous.(*http.Transport).Clone()
	roots := x509.NewCertPool()
	roots.AddCert(server.Certificate())
	transport.TLSClientConfig = &tls.Config{RootCAs: roots, ServerName: "example.com", MinVersion: tls.VersionTLS12}
	transport.Proxy = nil
	transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, server.Listener.Addr().String())
	}
	http.DefaultTransport = transport
	defer func() { http.DefaultTransport = previous; transport.CloseIdleConnections() }()

	dest := slsTestDestination{endpoint: "https://example.com"}
	t.Setenv(odigosconfig.SecretEnvVarName(odigosconfig.SLS_ACCESS_KEY_ID, dest.GetID()), "test-access-key")
	t.Setenv(odigosconfig.SecretEnvVarName(odigosconfig.SLS_ACCESS_KEY_SECRET, dest.GetID()), "test-secret")
	cfg := &odigosconfig.Config{Exporters: odigosconfig.GenericMap{}, Service: odigosconfig.Service{Pipelines: map[string]odigosconfig.Pipeline{}}}
	pipelines, err := (&odigosconfig.AlibabaCloudSLS{}).ModifyConfig(dest, cfg)
	require.NoError(t, err)
	data, err := json.Marshal(cfg.Exporters["alibabacloud_logservice/alibabacloudsls-sls-test"])
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "exporter.json")
	require.NoError(t, os.WriteFile(path, data, 0600))
	resolver, err := confmap.NewResolver(confmap.ResolverSettings{
		URIs: []string{path}, DefaultScheme: "env",
		ProviderFactories: []confmap.ProviderFactory{fileprovider.NewFactory(), envprovider.NewFactory()},
	})
	require.NoError(t, err)
	defer func() { require.NoError(t, resolver.Shutdown(context.Background())) }()
	resolved, err := resolver.Resolve(context.Background())
	require.NoError(t, err)
	factories, err := components()
	require.NoError(t, err)
	factory := factories.Exporters[component.MustNewType("alibabacloud_logservice")]
	require.NotNil(t, factory)
	exporterConfig := factory.CreateDefaultConfig()
	require.NoError(t, resolved.Unmarshal(exporterConfig))
	core, logs := observer.New(zap.WarnLevel)
	settings := exportertest.NewNopSettings(factory.Type())
	settings.Logger = zap.New(core)
	exp, err := factory.CreateLogs(context.Background(), settings, exporterConfig)
	require.NoError(t, err)
	require.NoError(t, exp.Start(context.Background(), componenttest.NewNopHost()))
	defer func() { require.NoError(t, exp.Shutdown(context.Background())) }()
	batchFactory := factories.Processors[component.MustNewType("batch")]
	batchConfig := batchFactory.CreateDefaultConfig()
	processorName := cfg.Service.Pipelines[pipelines[0]].Processors[0]
	require.NoError(t, confmap.NewFromStringMap(map[string]any(cfg.Processors[processorName].(odigosconfig.GenericMap))).Unmarshal(batchConfig))
	batch, err := batchFactory.CreateLogs(context.Background(), processortest.NewNopSettings(batchFactory.Type()), batchConfig, exp)
	require.NoError(t, err)
	require.NoError(t, batch.Start(context.Background(), componenttest.NewNopHost()))
	defer func() { require.NoError(t, batch.Shutdown(context.Background())) }()

	input := plog.NewLogs()
	for i := 0; i < 2; i++ {
		rl := input.ResourceLogs().AppendEmpty()
		rl.Resource().Attributes().PutStr("service.name", fmt.Sprintf("service-%d", i))
		rl.Resource().Attributes().PutStr("k8s.namespace.name", "demo")
		scope := rl.ScopeLogs().AppendEmpty()
		scope.Scope().SetName("odigos-test")
		for j := 0; j < 3000; j++ {
			log := scope.LogRecords().AppendEmpty()
			log.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
			log.SetSeverityText("INFO")
			log.Attributes().PutInt("sequence", int64(j))
			if j == 0 {
				log.Body().SetEmptyMap().PutStr("message", "structured")
			} else {
				log.Body().SetStr("ordinary odigos log " + strings.Repeat("x", 1024))
			}
		}
		scope.LogRecords().AppendEmpty()
	}
	require.NoError(t, batch.ConsumeLogs(context.Background(), input))
	require.Eventually(t, func() bool { mu.Lock(); defer mu.Unlock(); return len(received) == 6000 }, 15*time.Second, 50*time.Millisecond)
	mu.Lock()
	services := map[string]int{}
	structured := 0
	sequences := map[string]bool{}
	for _, log := range received {
		fields := map[string]string{}
		for _, content := range log.Contents {
			fields[content.GetKey()] = content.GetValue()
		}
		services[fields["service"]]++
		require.Contains(t, fields["resource"], `"k8s.namespace.name":"demo"`)
		require.Equal(t, "odigos-test", fields["otlp.name"])
		require.Equal(t, "INFO", fields["severityText"])
		key := fields["service"] + fields["attribute"]
		require.False(t, sequences[key], "duplicate log record")
		sequences[key] = true
		if strings.Contains(fields["content"], `"message":"structured"`) {
			structured++
		}
	}
	require.Equal(t, map[string]int{"service-0": 3000, "service-1": 3000}, services)
	require.Equal(t, 2, structured)
	require.Greater(t, len(batchCounts), 1)
	for _, count := range batchCounts {
		require.LessOrEqual(t, count, 4096+512)
	}
	mu.Unlock()

	small := func(body string) plog.Logs {
		data := plog.NewLogs()
		data.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().Body().SetStr(body)
		return data
	}
	mode.Store(1)
	attempts.Store(0)
	require.NoError(t, exp.ConsumeLogs(context.Background(), small("retry")))
	require.Eventually(t, func() bool { mu.Lock(); defer mu.Unlock(); return len(received) == 6001 }, 15*time.Second, 50*time.Millisecond)
	require.GreaterOrEqual(t, attempts.Load(), int32(2))
	mode.Store(2)
	attempts.Store(0)
	// Upstream acknowledges enqueueing even when the service subsequently rejects
	// every record; its callback exposes rejection as a warning, not partial success.
	require.NoError(t, exp.ConsumeLogs(context.Background(), small("reject")))
	require.Eventually(t, func() bool { return logs.FilterMessage("Send to LogService failed").Len() == 1 }, 10*time.Second, 50*time.Millisecond)
	require.Equal(t, int32(1), attempts.Load())
	require.Equal(t, "InvalidParameter", logs.FilterMessage("Send to LogService failed").All()[0].ContextMap()["code"])
}
