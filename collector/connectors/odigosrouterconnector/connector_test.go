package odigosrouterconnector

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/connector/xconnector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/ptrace"
	collectorpipeline "go.opentelemetry.io/collector/pipeline"
	xpipeline "go.opentelemetry.io/collector/pipeline/xpipeline"
	"go.uber.org/zap"

	commonapi "github.com/odigos-io/odigos/common/api"
	odigoscollector "github.com/odigos-io/odigos/common/collector"
	"github.com/odigos-io/odigos/common/consts"
)

// serviceNameAttr identifies a workload in these tests; the fake extension maps it to data streams.
const serviceNameAttr = "service.name"

// fakeConfigExtension resolves data stream membership from service.name, the way the real
// odigosconfigk8sextension resolves it from the InstrumentationConfig's
// "odigos.io/data-stream-<name>" labels. A workload with no entry is unknown to the extension.
type fakeConfigExtension struct {
	streamsByService map[string][]string
}

func (f *fakeConfigExtension) GetFromResource(pcommon.Resource) (*commonapi.ContainerCollectorConfig, bool) {
	return nil, false
}

func (f *fakeConfigExtension) IsActiveSource(pcommon.Resource) bool { return true }

func (f *fakeConfigExtension) GetWorkloadCacheKey(pcommon.Resource) (string, error) {
	return "", nil
}

func (f *fakeConfigExtension) GetWorkloadIdentityFromResource(pcommon.Resource) (string, pcommon.Map, error) {
	return "", pcommon.NewMap(), nil
}

func (f *fakeConfigExtension) RegisterWorkloadConfigCacheCallback(odigoscollector.WorkloadConfigCacheCallback) {
}

func (f *fakeConfigExtension) UnregisterWorkloadConfigCacheCallback(odigoscollector.WorkloadConfigCacheCallback) {
}

func (f *fakeConfigExtension) WaitForCacheSync(context.Context) bool { return true }

func (f *fakeConfigExtension) GetDataStreamsForWorkload(res pcommon.Resource) ([]string, bool) {
	name, ok := res.Attributes().Get(serviceNameAttr)
	if !ok {
		return nil, false
	}
	streams, found := f.streamsByService[name.Str()]
	return streams, found
}

func (f *fakeConfigExtension) Start(context.Context, component.Host) error { return nil }
func (f *fakeConfigExtension) Shutdown(context.Context) error              { return nil }

func metricsPipelineID(name string) collectorpipeline.ID {
	return collectorpipeline.NewIDWithName(collectorpipeline.SignalMetrics, name)
}

// newMetricsRouterConnector wires a metrics router that only knows the given data stream
// pipelines, mirroring a gateway config where buildDataStreamPipelines only emits
// metrics/<stream> for streams that actually have a metrics destination.
func newMetricsRouterConnector(t *testing.T, ext odigoscollector.OdigosConfigExtension,
	streamSinks map[string]*consumertest.MetricsSink, defaultSink *consumertest.MetricsSink,
) *routerConnector {
	t.Helper()

	consumers := make(map[collectorpipeline.ID]consumer.Metrics, len(streamSinks)+1)
	for name, sink := range streamSinks {
		consumers[metricsPipelineID(name)] = sink
	}
	if defaultSink != nil {
		consumers[metricsPipelineID(consts.DefaultDataStream)] = defaultSink
	}
	router := connector.NewMetricsRouter(consumers)

	var defaultCons consumer.Metrics
	if defaultSink != nil {
		defaultCons = defaultSink
	}

	return &routerConnector{
		metricsConfig: metricsConfig{
			consumers:   router,
			defaultCons: defaultCons,
			logger:      zap.NewNop(),
		},
		odigosConfigExtension: ext,
	}
}

func metricsForService(service string) pmetric.Metrics {
	md := pmetric.NewMetrics()
	rm := md.ResourceMetrics().AppendEmpty()
	rm.Resource().Attributes().PutStr(serviceNameAttr, service)
	m := rm.ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
	m.SetName("some.metric")
	m.SetEmptyGauge().DataPoints().AppendEmpty().SetIntValue(1)
	return md
}

func appendMetricsForService(md pmetric.Metrics, service string) {
	rm := md.ResourceMetrics().AppendEmpty()
	rm.Resource().Attributes().PutStr(serviceNameAttr, service)
	m := rm.ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
	m.SetName("some.metric")
	m.SetEmptyGauge().DataPoints().AppendEmpty().SetIntValue(1)
}

// A data stream whose destinations carry no metrics signal has no metrics/<stream> pipeline, so
// the router holds no consumer for it. The stream must be skipped, not turned into a consume
// error: the error travels back to the gateway's OTLP receiver and fails the whole request.
func TestConsumeMetricsSkipsDataStreamWithNoMetricsPipeline(t *testing.T) {
	ext := &fakeConfigExtension{streamsByService: map[string][]string{
		"traces-only-svc": {"stream-b"},
	}}
	defaultSink := new(consumertest.MetricsSink)
	r := newMetricsRouterConnector(t, ext,
		map[string]*consumertest.MetricsSink{"stream-a": new(consumertest.MetricsSink)},
		defaultSink,
	)

	err := r.ConsumeMetrics(context.Background(), metricsForService("traces-only-svc"))
	require.NoError(t, err, "a data stream with no metrics destination must not fail the batch")

	// The workload is explicitly assigned to stream-b, so its metrics must not leak into the
	// default data stream's destinations either.
	assert.Empty(t, defaultSink.AllMetrics())
}

// The blast radius of the failure: one resource belonging to a metrics-less data stream must not
// stop the rest of the request from being accepted. When the whole request is rejected the sender
// retries it, re-delivering the resources that were already consumed successfully.
func TestConsumeMetricsDeliversOtherStreamsWhenOneStreamHasNoPipeline(t *testing.T) {
	ext := &fakeConfigExtension{streamsByService: map[string][]string{
		"metrics-svc":     {"stream-a"},
		"traces-only-svc": {"stream-b"},
	}}
	streamASink := new(consumertest.MetricsSink)
	r := newMetricsRouterConnector(t, ext,
		map[string]*consumertest.MetricsSink{"stream-a": streamASink},
		new(consumertest.MetricsSink),
	)

	md := metricsForService("metrics-svc")
	appendMetricsForService(md, "traces-only-svc")

	err := r.ConsumeMetrics(context.Background(), md)
	require.NoError(t, err)

	require.Len(t, streamASink.AllMetrics(), 1)
	require.Equal(t, 1, streamASink.AllMetrics()[0].ResourceMetrics().Len())
	svc, ok := streamASink.AllMetrics()[0].ResourceMetrics().At(0).Resource().Attributes().Get(serviceNameAttr)
	require.True(t, ok)
	assert.Equal(t, "metrics-svc", svc.Str())
}

// The routing itself still works: a workload in a stream that does have a metrics pipeline is
// delivered there and nowhere else.
func TestConsumeMetricsRoutesToItsDataStream(t *testing.T) {
	ext := &fakeConfigExtension{streamsByService: map[string][]string{
		"metrics-svc": {"stream-a"},
	}}
	streamASink := new(consumertest.MetricsSink)
	streamBSink := new(consumertest.MetricsSink)
	defaultSink := new(consumertest.MetricsSink)
	r := newMetricsRouterConnector(t, ext, map[string]*consumertest.MetricsSink{
		"stream-a": streamASink,
		"stream-b": streamBSink,
	}, defaultSink)

	require.NoError(t, r.ConsumeMetrics(context.Background(), metricsForService("metrics-svc")))

	assert.Len(t, streamASink.AllMetrics(), 1)
	assert.Empty(t, streamBSink.AllMetrics())
	assert.Empty(t, defaultSink.AllMetrics())
}

// A workload the extension does not know about falls back to the default data stream.
func TestConsumeMetricsUnknownWorkloadGoesToDefault(t *testing.T) {
	ext := &fakeConfigExtension{streamsByService: map[string][]string{}}
	defaultSink := new(consumertest.MetricsSink)
	r := newMetricsRouterConnector(t, ext,
		map[string]*consumertest.MetricsSink{"stream-a": new(consumertest.MetricsSink)},
		defaultSink,
	)

	require.NoError(t, r.ConsumeMetrics(context.Background(), metricsForService("unlabeled-svc")))
	assert.Len(t, defaultSink.AllMetrics(), 1)
}

// The three sibling signals already behave this way. These assertions are what makes the metrics
// behaviour above the correct one rather than a matter of taste, so keep them next to it.
func TestConsumeTracesSkipsDataStreamWithNoTracesPipeline(t *testing.T) {
	ext := &fakeConfigExtension{streamsByService: map[string][]string{
		"svc": {"stream-b"},
	}}
	defaultSink := new(consumertest.TracesSink)
	router := connector.NewTracesRouter(map[collectorpipeline.ID]consumer.Traces{
		collectorpipeline.NewIDWithName(collectorpipeline.SignalTraces, "stream-a"):               new(consumertest.TracesSink),
		collectorpipeline.NewIDWithName(collectorpipeline.SignalTraces, consts.DefaultDataStream): defaultSink,
	})
	r := &routerConnector{
		tracesConfig:          tracesConfig{consumers: router, defaultCons: defaultSink, logger: zap.NewNop()},
		odigosConfigExtension: ext,
	}

	td := ptrace.NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().PutStr(serviceNameAttr, "svc")
	rs.ScopeSpans().AppendEmpty().Spans().AppendEmpty().SetName("span")

	require.NoError(t, r.ConsumeTraces(context.Background(), td))
	assert.Empty(t, defaultSink.AllTraces())
}

func TestConsumeLogsSkipsDataStreamWithNoLogsPipeline(t *testing.T) {
	ext := &fakeConfigExtension{streamsByService: map[string][]string{
		"svc": {"stream-b"},
	}}
	defaultSink := new(consumertest.LogsSink)
	router := connector.NewLogsRouter(map[collectorpipeline.ID]consumer.Logs{
		collectorpipeline.NewIDWithName(collectorpipeline.SignalLogs, "stream-a"):               new(consumertest.LogsSink),
		collectorpipeline.NewIDWithName(collectorpipeline.SignalLogs, consts.DefaultDataStream): defaultSink,
	})
	r := &routerConnector{
		logsConfig:            logsConfig{consumers: router, defaultCons: defaultSink, logger: zap.NewNop()},
		odigosConfigExtension: ext,
	}

	ld := plog.NewLogs()
	rl := ld.ResourceLogs().AppendEmpty()
	rl.Resource().Attributes().PutStr(serviceNameAttr, "svc")
	rl.ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()

	require.NoError(t, r.ConsumeLogs(context.Background(), ld))
	assert.Empty(t, defaultSink.AllLogs())
}

func TestConsumeProfilesSkipsDataStreamWithNoProfilesPipeline(t *testing.T) {
	ext := &fakeConfigExtension{streamsByService: map[string][]string{
		"svc": {"stream-b"},
	}}
	defaultSink := new(consumertest.ProfilesSink)
	router := xconnector.NewProfilesRouter(map[collectorpipeline.ID]xconsumer.Profiles{
		collectorpipeline.NewIDWithName(xpipeline.SignalProfiles, "stream-a"):               new(consumertest.ProfilesSink),
		collectorpipeline.NewIDWithName(xpipeline.SignalProfiles, consts.DefaultDataStream): defaultSink,
	})
	r := &routerConnector{
		profilesConfig:        profilesConfig{consumers: router, defaultCons: defaultSink, logger: zap.NewNop()},
		odigosConfigExtension: ext,
	}

	pd := pprofile.NewProfiles()
	rp := pd.ResourceProfiles().AppendEmpty()
	rp.Resource().Attributes().PutStr(serviceNameAttr, "svc")
	rp.ScopeProfiles().AppendEmpty().Profiles().AppendEmpty()

	require.NoError(t, r.ConsumeProfiles(context.Background(), pd))
	assert.Empty(t, defaultSink.AllProfiles())
}
