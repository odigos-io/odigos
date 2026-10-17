package cassandraexporter

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/cassandraexporter/internal/metadata"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/goleak"
	"go.uber.org/zap"
)

// CASSANDRA_TEST_PORT opts into tests against a disposable real Cassandra server.
func TestCassandraIngestion(t *testing.T) {
	portString := os.Getenv("CASSANDRA_TEST_PORT")
	if portString == "" {
		t.Skip("set CASSANDRA_TEST_PORT for real Cassandra ingestion")
	}
	port, err := strconv.Atoi(portString)
	require.NoError(t, err)
	// The driver's asynchronous reconnect attempts outlive Session.Close briefly.
	t.Cleanup(func() {
		deadline := time.Now().Add(20 * time.Second)
		var err error
		for {
			err = goleak.Find()
			if err == nil || time.Now().After(deadline) {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		require.NoError(t, err)
	})
	cfg := createDefaultConfig().(*Config)
	cfg.Port = port
	if compression := os.Getenv("CASSANDRA_TEST_COMPRESSION"); compression != "" {
		cfg.Compression.Algorithm = compression
	}
	cfg.Keyspace = fmt.Sprintf("otel_test_%d", time.Now().UnixNano())
	cfg.Auth.UserName = os.Getenv("CASSANDRA_TEST_USERNAME")
	cfg.Auth.Password = configopaque.String(os.Getenv("CASSANDRA_TEST_PASSWORD"))
	if ca := os.Getenv("CASSANDRA_TEST_CA"); ca != "" {
		cfg.TLS = &configtls.ClientConfig{Config: configtls.Config{CAFile: ca}}
	}
	ctx := context.Background()
	exp := newTracesExporter(zap.NewNop(), cfg)
	require.NoError(t, exp.Start(ctx, componenttest.NewNopHost()))
	defer exp.Shutdown(ctx)
	defer exp.client.Query("DROP KEYSPACE " + cfg.Keyspace).Exec()
	logsExp := newLogsExporter(zap.NewNop(), cfg)
	require.NoError(t, logsExp.Start(ctx, componenttest.NewNopHost()))
	defer logsExp.Shutdown(ctx)
	td := ptrace.NewTraces()
	ld := plog.NewLogs()
	for i := 0; i < 2; i++ {
		rs := td.ResourceSpans().AppendEmpty()
		rs.Resource().Attributes().PutStr("service.name", fmt.Sprintf("odigos-service-%d", i))
		span := rs.ScopeSpans().AppendEmpty().Spans().AppendEmpty()
		span.SetTraceID(pcommon.TraceID{byte(i + 1)})
		span.SetSpanID(pcommon.SpanID{1})
		span.SetName("GET /ordinary-workload")
		span.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now()))
		span.SetEndTimestamp(span.StartTimestamp() + pcommon.Timestamp(5*time.Second))
		span.Attributes().PutInt("http.response.status_code", 200)
		rl := ld.ResourceLogs().AppendEmpty()
		rl.Resource().Attributes().PutStr("service.name", fmt.Sprintf("odigos-service-%d", i))
		for j := 0; j < 3; j++ {
			record := rl.ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
			record.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
			record.SetSeverityNumber(plog.SeverityNumberInfo)
			record.Body().SetStr(fmt.Sprintf("ordinary application log %d", j))
			record.Attributes().PutBool("success", true)
		}
	}
	require.NoError(t, exp.pushTraceData(ctx, td))
	require.NoError(t, logsExp.pushLogsData(ctx, ld))
	// Replaying an entire batch must preserve both resources and uncorrelated logs.
	require.NoError(t, exp.pushTraceData(ctx, td))
	require.NoError(t, logsExp.pushLogsData(ctx, ld))
	var count int
	require.NoError(t, exp.client.Query("SELECT count(*) FROM "+cfg.TraceTable).Scan(&count))
	require.Equal(t, 2, count)
	require.NoError(t, exp.client.Query("SELECT count(*) FROM "+cfg.LogsTable).Scan(&count))
	require.Equal(t, 6, count)
	var duration int64
	require.NoError(t, exp.client.Query("SELECT duration FROM "+cfg.TraceTable+" LIMIT 1").Scan(&duration))
	require.Equal(t, int64(5*time.Second), duration)
	require.NoError(t, exp.client.Query("DROP TABLE "+cfg.TraceTable).Exec())
	require.Error(t, exp.pushTraceData(ctx, td))
	require.NoError(t, exp.client.Query("DROP TABLE "+cfg.LogsTable).Exec())
	require.Error(t, logsExp.pushLogsData(ctx, ld))
	require.NoError(t, initializeTraceKernel(cfg))
	require.NoError(t, initializeLogKernel(cfg))
	require.NoError(t, exp.pushTraceData(ctx, td))
	require.NoError(t, logsExp.pushLogsData(ctx, ld))
	t.Log("verified automatic schema creation, multi-resource batches, 5-second spans, six uncorrelated logs, idempotent replay, write rejection and recovery")
}

func TestTLSAndSchemaValidation(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.TLS = &configtls.ClientConfig{}
	cluster, err := newCluster(cfg)
	require.NoError(t, err)
	require.NotNil(t, cluster.SslOpts)
	require.True(t, cluster.SslOpts.EnableHostVerification)
	cfg.TLS = &configtls.ClientConfig{Config: configtls.Config{CAPem: configopaque.String("invalid")}}
	_, err = newCluster(cfg)
	require.Error(t, err)
	cfg = createDefaultConfig().(*Config)
	cfg.Keyspace = "otel'; DROP KEYSPACE x"
	require.Error(t, cfg.Validate())
}

func TestCassandraSecurityRejections(t *testing.T) {
	if os.Getenv("CASSANDRA_TEST_CA") == "" {
		t.Skip("requires a TLS/auth Cassandra fixture")
	}
	port, err := strconv.Atoi(os.Getenv("CASSANDRA_TEST_PORT"))
	require.NoError(t, err)
	// The driver's asynchronous reconnect attempts outlive Session.Close briefly.
	t.Cleanup(func() {
		deadline := time.Now().Add(20 * time.Second)
		var err error
		for {
			err = goleak.Find()
			if err == nil || time.Now().After(deadline) {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		require.NoError(t, err)
	})
	cfg := createDefaultConfig().(*Config)
	cfg.Port = port
	cfg.Auth = Auth{UserName: os.Getenv("CASSANDRA_TEST_USERNAME"), Password: configopaque.String("incorrect-test-password")}
	cfg.TLS = &configtls.ClientConfig{Config: configtls.Config{CAFile: os.Getenv("CASSANDRA_TEST_CA")}}
	cluster, err := newCluster(cfg)
	require.NoError(t, err)
	session, err := cluster.CreateSession()
	if session != nil {
		session.Close()
	}
	require.Error(t, err, "wrong password must not connect")
	cfg.Auth.Password = configopaque.String(os.Getenv("CASSANDRA_TEST_PASSWORD"))
	cfg.TLS = &configtls.ClientConfig{}
	cluster, err = newCluster(cfg)
	require.NoError(t, err)
	session, err = cluster.CreateSession()
	if session != nil {
		session.Close()
	}
	require.Error(t, err, "untrusted server must not connect")
}

func TestCassandraRetries(t *testing.T) {
	if os.Getenv("CASSANDRA_TEST_PORT") == "" {
		t.Skip("requires a real Cassandra fixture")
	}
	port, err := strconv.Atoi(os.Getenv("CASSANDRA_TEST_PORT"))
	require.NoError(t, err)
	// The driver's asynchronous reconnect attempts outlive Session.Close briefly.
	t.Cleanup(func() {
		deadline := time.Now().Add(20 * time.Second)
		var err error
		for {
			err = goleak.Find()
			if err == nil || time.Now().After(deadline) {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		require.NoError(t, err)
	})
	cfg := createDefaultConfig().(*Config)
	cfg.Port = port
	cfg.Keyspace = fmt.Sprintf("otel_retry_%d", time.Now().UnixNano())
	cfg.Auth = Auth{UserName: os.Getenv("CASSANDRA_TEST_USERNAME"), Password: configopaque.String(os.Getenv("CASSANDRA_TEST_PASSWORD"))}
	if ca := os.Getenv("CASSANDRA_TEST_CA"); ca != "" {
		cfg.TLS = &configtls.ClientConfig{Config: configtls.Config{CAFile: ca}}
	}
	ctx := context.Background()
	host := componenttest.NewNopHost()
	traceExporter, err := createTracesExporter(ctx, exportertest.NewNopSettings(metadata.Type), cfg)
	require.NoError(t, err)
	require.NoError(t, traceExporter.Start(ctx, host))
	defer traceExporter.Shutdown(ctx)
	logExporter, err := createLogsExporter(ctx, exportertest.NewNopSettings(metadata.Type), cfg)
	require.NoError(t, err)
	require.NoError(t, logExporter.Start(ctx, host))
	defer logExporter.Shutdown(ctx)
	cluster, err := newCluster(cfg)
	require.NoError(t, err)
	cluster.Keyspace = cfg.Keyspace
	session, err := cluster.CreateSession()
	require.NoError(t, err)
	defer session.Close()
	defer session.Query("DROP KEYSPACE " + cfg.Keyspace).Exec()
	td := ptrace.NewTraces()
	span := td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	span.SetTraceID(pcommon.TraceID{1})
	span.SetSpanID(pcommon.SpanID{1})
	span.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	span.SetEndTimestamp(span.StartTimestamp() + 100)
	ld := plog.NewLogs()
	record := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
	record.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	record.Body().SetStr("retry proof")
	for _, test := range []struct {
		table      string
		initialize func(*Config) error
		consume    func() error
	}{{cfg.TraceTable, initializeTraceKernel, func() error { return traceExporter.ConsumeTraces(ctx, td) }}, {cfg.LogsTable, initializeLogKernel, func() error { return logExporter.ConsumeLogs(ctx, ld) }}} {
		require.NoError(t, session.Query("DROP TABLE "+test.table).Exec())
		restored := make(chan error, 1)
		go func() { time.Sleep(time.Second); restored <- test.initialize(cfg) }()
		require.NoError(t, test.consume(), "exporterhelper must retry the rejected write after schema recovery")
		require.NoError(t, <-restored)
		var count int
		require.NoError(t, session.Query("SELECT count(*) FROM "+test.table).Scan(&count))
		require.Equal(t, 1, count)
	}
	t.Log("both collector exporter helpers retried real rejected writes and persisted records after recovery")
}
