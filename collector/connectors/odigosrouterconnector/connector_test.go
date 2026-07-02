package odigosrouterconnector

import (
	"context"
	"testing"

	commonapi "github.com/odigos-io/odigos/common/api"
	odigoscollector "github.com/odigos-io/odigos/common/collector"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pipeline"
	"go.uber.org/zap"
)

type testConfigExtension struct {
	streams []string
}

func (t testConfigExtension) GetFromResource(pcommon.Resource) (*commonapi.ContainerCollectorConfig, bool) {
	return nil, false
}

func (t testConfigExtension) GetWorkloadCacheKey(pcommon.Resource) (string, error) {
	return "", nil
}

func (t testConfigExtension) RegisterWorkloadConfigCacheCallback(odigoscollector.WorkloadConfigCacheCallback) {
}

func (t testConfigExtension) UnregisterWorkloadConfigCacheCallback(odigoscollector.WorkloadConfigCacheCallback) {
}

func (t testConfigExtension) WaitForCacheSync(context.Context) bool {
	return true
}

func (t testConfigExtension) GetDataStreamsForWorkload(pcommon.Resource) ([]string, bool) {
	return t.streams, true
}

func TestConsumeTracesFallsBackWhenStreamHasNoSignalConsumer(t *testing.T) {
	defaultSink := &consumertest.TracesSink{}
	streamSink := &consumertest.TracesSink{}
	router := connector.NewTracesRouter(map[pipeline.ID]consumer.Traces{
		pipeline.NewIDWithName(pipeline.SignalTraces, "default"):     defaultSink,
		pipeline.NewIDWithName(pipeline.SignalTraces, "traces-only"): streamSink,
	})
	conn := &routerConnector{
		odigosConfigExtension: testConfigExtension{streams: []string{"logs-only"}},
		tracesConfig: tracesConfig{
			consumers:   router,
			defaultCons: defaultSink,
			logger:      zap.NewNop(),
		},
	}

	require.NoError(t, conn.ConsumeTraces(context.Background(), oneSpanTraces()))
	require.Equal(t, 1, defaultSink.SpanCount())
	require.Equal(t, 0, streamSink.SpanCount())
}

func TestConsumeMetricsFallsBackWhenStreamHasNoSignalConsumer(t *testing.T) {
	defaultSink := &consumertest.MetricsSink{}
	streamSink := &consumertest.MetricsSink{}
	router := connector.NewMetricsRouter(map[pipeline.ID]consumer.Metrics{
		pipeline.NewIDWithName(pipeline.SignalMetrics, "default"):      defaultSink,
		pipeline.NewIDWithName(pipeline.SignalMetrics, "metrics-only"): streamSink,
	})
	conn := &routerConnector{
		odigosConfigExtension: testConfigExtension{streams: []string{"traces-only"}},
		metricsConfig: metricsConfig{
			consumers:   router,
			defaultCons: defaultSink,
			logger:      zap.NewNop(),
		},
	}

	require.NoError(t, conn.ConsumeMetrics(context.Background(), oneDataPointMetrics()))
	require.Equal(t, 1, defaultSink.DataPointCount())
	require.Equal(t, 0, streamSink.DataPointCount())
}

func TestConsumeLogsFallsBackWhenStreamHasNoSignalConsumer(t *testing.T) {
	defaultSink := &consumertest.LogsSink{}
	streamSink := &consumertest.LogsSink{}
	router := connector.NewLogsRouter(map[pipeline.ID]consumer.Logs{
		pipeline.NewIDWithName(pipeline.SignalLogs, "default"):   defaultSink,
		pipeline.NewIDWithName(pipeline.SignalLogs, "logs-only"): streamSink,
	})
	conn := &routerConnector{
		odigosConfigExtension: testConfigExtension{streams: []string{"traces-only"}},
		logsConfig: logsConfig{
			consumers:   router,
			defaultCons: defaultSink,
			logger:      zap.NewNop(),
		},
	}

	require.NoError(t, conn.ConsumeLogs(context.Background(), oneRecordLogs()))
	require.Equal(t, 1, defaultSink.LogRecordCount())
	require.Equal(t, 0, streamSink.LogRecordCount())
}

func oneSpanTraces() ptrace.Traces {
	td := ptrace.NewTraces()
	td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	return td
}

func oneDataPointMetrics() pmetric.Metrics {
	md := pmetric.NewMetrics()
	metric := md.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
	metric.SetName("test.metric")
	metric.SetEmptyGauge().DataPoints().AppendEmpty().SetIntValue(1)
	return md
}

func oneRecordLogs() plog.Logs {
	ld := plog.NewLogs()
	ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
	return ld
}
