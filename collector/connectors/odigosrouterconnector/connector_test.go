package odigosrouterconnector

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	collectorpipeline "go.opentelemetry.io/collector/pipeline"
	"go.uber.org/zap"

	commonapi "github.com/odigos-io/odigos/common/api"
	odigoscollector "github.com/odigos-io/odigos/common/collector"
	"github.com/odigos-io/odigos/common/consts"
)

type mockOdigosConfigExtension struct {
	dataStreams []string
}

func (m *mockOdigosConfigExtension) GetFromResource(pcommon.Resource) (*commonapi.ContainerCollectorConfig, bool) {
	return nil, false
}

func (m *mockOdigosConfigExtension) GetWorkloadCacheKey(pcommon.Resource) (string, error) {
	return "", nil
}

func (m *mockOdigosConfigExtension) RegisterWorkloadConfigCacheCallback(odigoscollector.WorkloadConfigCacheCallback) {
}

func (m *mockOdigosConfigExtension) UnregisterWorkloadConfigCacheCallback(odigoscollector.WorkloadConfigCacheCallback) {
}

func (m *mockOdigosConfigExtension) WaitForCacheSync(context.Context) bool {
	return true
}

func (m *mockOdigosConfigExtension) GetDataStreamsForWorkload(pcommon.Resource) ([]string, bool) {
	return m.dataStreams, true
}

func TestConsumeTracesFallsBackToDefaultWhenNoSignalPipelineMatches(t *testing.T) {
	defaultSink := &consumertest.TracesSink{}
	streamSink := &consumertest.TracesSink{}
	defaultID := collectorpipeline.NewIDWithName(collectorpipeline.SignalTraces, consts.DefaultDataStream)
	unrelatedID := collectorpipeline.NewIDWithName(collectorpipeline.SignalTraces, "unrelated-stream")
	router := connector.NewTracesRouter(map[collectorpipeline.ID]consumer.Traces{
		defaultID:   defaultSink,
		unrelatedID: streamSink,
	})
	rc := &routerConnector{
		tracesConfig:          tracesConfig{consumers: router, defaultCons: defaultSink, logger: zap.NewNop()},
		odigosConfigExtension: &mockOdigosConfigExtension{dataStreams: []string{"logs-only-stream"}},
	}

	err := rc.ConsumeTraces(context.Background(), tracesWithOneSpan())

	require.NoError(t, err)
	require.Equal(t, 1, defaultSink.SpanCount())
	require.Equal(t, 0, streamSink.SpanCount())
}

func TestConsumeMetricsFallsBackToDefaultWhenNoSignalPipelineMatches(t *testing.T) {
	defaultSink := &consumertest.MetricsSink{}
	streamSink := &consumertest.MetricsSink{}
	defaultID := collectorpipeline.NewIDWithName(collectorpipeline.SignalMetrics, consts.DefaultDataStream)
	unrelatedID := collectorpipeline.NewIDWithName(collectorpipeline.SignalMetrics, "unrelated-stream")
	router := connector.NewMetricsRouter(map[collectorpipeline.ID]consumer.Metrics{
		defaultID:   defaultSink,
		unrelatedID: streamSink,
	})
	rc := &routerConnector{
		metricsConfig:         metricsConfig{consumers: router, defaultCons: defaultSink, logger: zap.NewNop()},
		odigosConfigExtension: &mockOdigosConfigExtension{dataStreams: []string{"traces-only-stream"}},
	}

	err := rc.ConsumeMetrics(context.Background(), metricsWithOneDataPoint())

	require.NoError(t, err)
	require.Equal(t, 1, defaultSink.DataPointCount())
	require.Equal(t, 0, streamSink.DataPointCount())
}

func TestConsumeLogsFallsBackToDefaultWhenNoSignalPipelineMatches(t *testing.T) {
	defaultSink := &consumertest.LogsSink{}
	streamSink := &consumertest.LogsSink{}
	defaultID := collectorpipeline.NewIDWithName(collectorpipeline.SignalLogs, consts.DefaultDataStream)
	unrelatedID := collectorpipeline.NewIDWithName(collectorpipeline.SignalLogs, "unrelated-stream")
	router := connector.NewLogsRouter(map[collectorpipeline.ID]consumer.Logs{
		defaultID:   defaultSink,
		unrelatedID: streamSink,
	})
	rc := &routerConnector{
		logsConfig:            logsConfig{consumers: router, defaultCons: defaultSink, logger: zap.NewNop()},
		odigosConfigExtension: &mockOdigosConfigExtension{dataStreams: []string{"traces-only-stream"}},
	}

	err := rc.ConsumeLogs(context.Background(), logsWithOneRecord())

	require.NoError(t, err)
	require.Equal(t, 1, defaultSink.LogRecordCount())
	require.Equal(t, 0, streamSink.LogRecordCount())
}

func tracesWithOneSpan() ptrace.Traces {
	td := ptrace.NewTraces()
	span := td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	span.SetName("test-span")
	return td
}

func metricsWithOneDataPoint() pmetric.Metrics {
	md := pmetric.NewMetrics()
	sum := md.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty().SetEmptySum()
	sum.DataPoints().AppendEmpty().SetIntValue(1)
	return md
}

func logsWithOneRecord() plog.Logs {
	ld := plog.NewLogs()
	record := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
	record.Body().SetStr("test-log")
	return ld
}

var _ odigoscollector.OdigosConfigExtension = (*mockOdigosConfigExtension)(nil)
