package serviceioconnector

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/connector/connectortest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"github.com/odigos-io/odigos/collector/connectors/serviceioconnector/internal/metadata"
	"github.com/odigos-io/odigos/collector/pkg/completetrace"
)

func TestAggregateConnectionsFromTree(t *testing.T) {
	td := buildServiceIOTestTrace(t)

	tree, err := completetrace.BuildTraceTree(td, nil)
	require.NoError(t, err)

	connector := &serviceioConnector{
		keyToMetric:          make(map[uint64]metricSeries),
		inputSpanAttributes:  []string{"http.route"},
		outputSpanAttributes: []string{"rpc.service"},
		odigosConfig: &mockOdigosConfigExtension{
			activeSources: map[string]struct{}{"svc-1": {}},
		},
	}

	require.True(t, connector.aggregateConnectionsFromTree(tree))
	require.Len(t, connector.keyToMetric, 2)

	outputServices := make([]string, 0, 2)
	for _, series := range connector.keyToMetric {
		require.EqualValues(t, 1, series.count)
		serviceName, ok := series.dimensions.Get(serviceNameAttribute)
		require.True(t, ok)
		require.Equal(t, "svc-1", serviceName.Str())
		route, ok := series.dimensions.Get(inputAttributePrefix + "http.route")
		require.True(t, ok)
		require.Equal(t, "/root", route.Str())
		outputService, ok := series.dimensions.Get(outputAttributePrefix + "rpc.service")
		require.True(t, ok)
		outputServices = append(outputServices, outputService.Str())
	}
	require.ElementsMatch(t, []string{"Users", "Orders"}, outputServices)
}

func TestConnectorConsumeTraces_EmitsConnectionMetrics(t *testing.T) {
	sink := &consumertest.MetricsSink{}
	flushImmediately := time.Duration(0)
	cfg := &Config{
		InputSpanAttributes:   []string{"http.route"},
		OutputSpanAttributes:  []string{"rpc.service"},
		MetricsFlushInterval:  &flushImmediately,
		OdigosConfigExtension: &odigosConfigExtensionID,
	}
	require.NoError(t, cfg.Validate())

	connector, err := NewFactory().CreateTracesToMetrics(
		t.Context(),
		connectortest.NewNopSettings(metadata.Type),
		cfg,
		sink,
	)
	require.NoError(t, err)
	startConnectorWithMockExtension(t, connector, &mockOdigosConfigExtension{
		activeSources: map[string]struct{}{"svc-1": {}},
	})

	require.NoError(t, connector.ConsumeTraces(t.Context(), buildServiceIOTestTrace(t)))
	require.Len(t, sink.AllMetrics(), 1)

	metric := sink.AllMetrics()[0].ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0)
	require.Equal(t, metricNameConnectionTotal, metric.Name())
	require.Equal(t, pmetric.AggregationTemporalityCumulative, metric.Sum().AggregationTemporality())
	require.Equal(t, 2, metric.Sum().DataPoints().Len())
}

func TestConnectorConsumeTraces_AggregatesBeforeFlush(t *testing.T) {
	sink := &consumertest.MetricsSink{}
	flushImmediately := time.Duration(0)
	cfg := &Config{
		InputSpanAttributes:   []string{"http.route"},
		OutputSpanAttributes:  []string{"rpc.service"},
		MetricsFlushInterval:  &flushImmediately,
		OdigosConfigExtension: &odigosConfigExtensionID,
	}
	require.NoError(t, cfg.Validate())

	connector, err := NewFactory().CreateTracesToMetrics(
		t.Context(),
		connectortest.NewNopSettings(metadata.Type),
		cfg,
		sink,
	)
	require.NoError(t, err)
	startConnectorWithMockExtension(t, connector, &mockOdigosConfigExtension{
		activeSources: map[string]struct{}{"svc-1": {}},
	})

	trace := buildServiceIOTestTrace(t)
	require.NoError(t, connector.ConsumeTraces(t.Context(), trace))
	require.NoError(t, connector.ConsumeTraces(t.Context(), trace))
	require.Len(t, sink.AllMetrics(), 2)

	lastMetric := sink.AllMetrics()[1].ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0)
	require.Equal(t, 2, lastMetric.Sum().DataPoints().Len())

	counts := make(map[int64]struct{})
	for i := 0; i < lastMetric.Sum().DataPoints().Len(); i++ {
		counts[lastMetric.Sum().DataPoints().At(i).IntValue()] = struct{}{}
	}
	require.Contains(t, counts, int64(2))
}

func buildServiceIOTestTrace(t *testing.T) ptrace.Traces {
	t.Helper()

	td := ptrace.NewTraces()
	rootID := pcommon.SpanID([8]byte{1})
	client1ID := pcommon.SpanID([8]byte{2})
	server1ID := pcommon.SpanID([8]byte{3})
	client2ID := pcommon.SpanID([8]byte{4})
	server2ID := pcommon.SpanID([8]byte{5})

	appendSpan := func(serviceName, name string, spanID, parentID pcommon.SpanID, kind ptrace.SpanKind, attrs map[string]string) {
		rs := td.ResourceSpans().AppendEmpty()
		rs.Resource().Attributes().PutStr("service.name", serviceName)
		ss := rs.ScopeSpans().AppendEmpty()
		span := ss.Spans().AppendEmpty()
		span.SetSpanID(spanID)
		if !parentID.IsEmpty() {
			span.SetParentSpanID(parentID)
		}
		span.SetName(name)
		span.SetKind(kind)
		for key, value := range attrs {
			span.Attributes().PutStr(key, value)
		}
	}

	appendSpan("svc-1", "root", rootID, pcommon.SpanID{}, ptrace.SpanKindServer, map[string]string{"http.route": "/root"})
	appendSpan("svc-1", "client-1", client1ID, rootID, ptrace.SpanKindClient, map[string]string{"rpc.service": "Users"})
	appendSpan("svc-2", "server-1", server1ID, client1ID, ptrace.SpanKindServer, nil)
	appendSpan("svc-1", "client-2", client2ID, rootID, ptrace.SpanKindClient, map[string]string{"rpc.service": "Orders"})
	appendSpan("svc-2", "server-2", server2ID, client2ID, ptrace.SpanKindServer, nil)

	return td
}

func TestConnectorConsumeTraces_InvalidTraceDoesNotEmitMetrics(t *testing.T) {
	sink := &consumertest.MetricsSink{}
	connector, err := NewFactory().CreateTracesToMetrics(
		t.Context(),
		connectortest.NewNopSettings(metadata.Type),
		&Config{},
		sink,
	)
	require.NoError(t, err)
	require.NoError(t, connector.Start(t.Context(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		require.NoError(t, connector.Shutdown(t.Context()))
	})

	td := ptrace.NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	ss := rs.ScopeSpans().AppendEmpty()
	span1 := ss.Spans().AppendEmpty()
	span1.SetSpanID(pcommon.SpanID([8]byte{1}))
	span2 := ss.Spans().AppendEmpty()
	span2.SetSpanID(pcommon.SpanID([8]byte{2}))
	span2.SetTraceID(pcommon.TraceID([16]byte{2}))

	require.NoError(t, connector.ConsumeTraces(t.Context(), td))
	require.Empty(t, sink.AllMetrics())
}
