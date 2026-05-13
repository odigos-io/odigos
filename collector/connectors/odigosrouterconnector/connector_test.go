package odigosrouterconnector

import (
	"context"
	"errors"
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
	"go.uber.org/zap/zaptest"

	commonapi "github.com/odigos-io/odigos/common/api"
	odigoscollector "github.com/odigos-io/odigos/common/collector"
	"github.com/odigos-io/odigos/common/consts"
)

type fakeOdigosConfigExtension struct {
	streams []string
}

func (f fakeOdigosConfigExtension) GetFromResource(_ pcommon.Resource) (*commonapi.ContainerCollectorConfig, bool) {
	return nil, false
}

func (f fakeOdigosConfigExtension) GetWorkloadCacheKey(_ pcommon.Resource) (string, error) {
	return "", errors.New("not implemented")
}

func (f fakeOdigosConfigExtension) RegisterWorkloadConfigCacheCallback(odigoscollector.WorkloadConfigCacheCallback) {
}

func (f fakeOdigosConfigExtension) UnregisterWorkloadConfigCacheCallback(odigoscollector.WorkloadConfigCacheCallback) {
}

func (f fakeOdigosConfigExtension) WaitForCacheSync(context.Context) bool {
	return true
}

func (f fakeOdigosConfigExtension) GetDataStreamsForWorkload(_ pcommon.Resource) ([]string, bool) {
	return f.streams, true
}

func TestConsumeLogsFallsBackWhenLabeledStreamsHaveNoLogsPipeline(t *testing.T) {
	defaultSink := &consumertest.LogsSink{}
	tracesOnlySink := &consumertest.LogsSink{}
	logRouter := connector.NewLogsRouter(map[collectorpipeline.ID]consumer.Logs{
		collectorpipeline.NewIDWithName(collectorpipeline.SignalLogs, consts.DefaultDataStream): defaultSink,
		collectorpipeline.NewIDWithName(collectorpipeline.SignalLogs, "logs-enabled"):           tracesOnlySink,
	})
	rc := &routerConnector{
		odigosConfigExtension: fakeOdigosConfigExtension{streams: []string{"traces-only"}},
		logsConfig: logsConfig{
			consumers:   logRouter,
			defaultCons: defaultSink,
			logger:      zaptest.NewLogger(t),
		},
	}

	err := rc.ConsumeLogs(context.Background(), logsWithWorkloadResource())

	require.NoError(t, err)
	require.Equal(t, 1, defaultSink.LogRecordCount())
	require.Equal(t, 0, tracesOnlySink.LogRecordCount())
}

func TestConsumeMetricsFallsBackWhenLabeledStreamsHaveNoMetricsPipeline(t *testing.T) {
	defaultSink := &consumertest.MetricsSink{}
	logsOnlySink := &consumertest.MetricsSink{}
	metricsRouter := connector.NewMetricsRouter(map[collectorpipeline.ID]consumer.Metrics{
		collectorpipeline.NewIDWithName(collectorpipeline.SignalMetrics, consts.DefaultDataStream): defaultSink,
		collectorpipeline.NewIDWithName(collectorpipeline.SignalMetrics, "metrics-enabled"):        logsOnlySink,
	})
	rc := &routerConnector{
		odigosConfigExtension: fakeOdigosConfigExtension{streams: []string{"logs-only"}},
		metricsConfig: metricsConfig{
			consumers:   metricsRouter,
			defaultCons: defaultSink,
			logger:      zaptest.NewLogger(t),
		},
	}

	err := rc.ConsumeMetrics(context.Background(), metricsWithWorkloadResource())

	require.NoError(t, err)
	require.Equal(t, 1, defaultSink.DataPointCount())
	require.Equal(t, 0, logsOnlySink.DataPointCount())
}

func TestConsumeTracesFallsBackWhenLabeledStreamsHaveNoTracesPipeline(t *testing.T) {
	defaultSink := &consumertest.TracesSink{}
	metricsOnlySink := &consumertest.TracesSink{}
	tracesRouter := connector.NewTracesRouter(map[collectorpipeline.ID]consumer.Traces{
		collectorpipeline.NewIDWithName(collectorpipeline.SignalTraces, consts.DefaultDataStream): defaultSink,
		collectorpipeline.NewIDWithName(collectorpipeline.SignalTraces, "traces-enabled"):         metricsOnlySink,
	})
	rc := &routerConnector{
		odigosConfigExtension: fakeOdigosConfigExtension{streams: []string{"metrics-only"}},
		tracesConfig: tracesConfig{
			consumers:   tracesRouter,
			defaultCons: defaultSink,
			logger:      zaptest.NewLogger(t),
		},
	}

	err := rc.ConsumeTraces(context.Background(), tracesWithWorkloadResource())

	require.NoError(t, err)
	require.Equal(t, 1, defaultSink.SpanCount())
	require.Equal(t, 0, metricsOnlySink.SpanCount())
}

func logsWithWorkloadResource() plog.Logs {
	ld := plog.NewLogs()
	rl := ld.ResourceLogs().AppendEmpty()
	putWorkloadAttrs(rl.Resource().Attributes())
	lr := rl.ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
	lr.Body().SetStr("log")
	return ld
}

func metricsWithWorkloadResource() pmetric.Metrics {
	md := pmetric.NewMetrics()
	rm := md.ResourceMetrics().AppendEmpty()
	putWorkloadAttrs(rm.Resource().Attributes())
	metric := rm.ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
	metric.SetName("requests")
	metric.SetEmptyGauge().DataPoints().AppendEmpty().SetIntValue(1)
	return md
}

func tracesWithWorkloadResource() ptrace.Traces {
	td := ptrace.NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	putWorkloadAttrs(rs.Resource().Attributes())
	rs.ScopeSpans().AppendEmpty().Spans().AppendEmpty().SetName("span")
	return td
}

func putWorkloadAttrs(attrs pcommon.Map) {
	attrs.PutStr("k8s.namespace.name", "default")
	attrs.PutStr("k8s.deployment.name", "frontend")
}
