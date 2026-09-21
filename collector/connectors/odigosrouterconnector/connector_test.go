package odigosrouterconnector

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/connector/connectortest"
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
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	commonapi "github.com/odigos-io/odigos/common/api"
	odigoscollector "github.com/odigos-io/odigos/common/collector"
	"github.com/odigos-io/odigos/common/consts"
)

const (
	// the mock extension resolves data streams by this resource attribute, standing in for the
	// namespace/kind/name workload identity the real extension derives from the resource.
	routerServiceAttr = "service.name"

	routerStreamA = "stream-a"
	routerStreamB = "stream-b"
)

var routerExtensionID = component.MustNewID("odigos_config_k8s")

func routerConfig() *Config {
	extID := routerExtensionID
	return &Config{OdigosConfigExtension: &extID}
}

type routerTestHost struct {
	extensions map[component.ID]component.Component
}

func (h *routerTestHost) GetExtensions() map[component.ID]component.Component {
	return h.extensions
}

func (h *routerTestHost) GetFactory(component.Kind, component.Type) component.Factory {
	return nil
}

// mockRouterConfigExtension resolves data streams from the routerServiceAttr resource attribute.
// A service present in streamsByService is reported as found (even with an empty stream list),
// an absent one is reported as not found.
type mockRouterConfigExtension struct {
	streamsByService map[string][]string
	cacheSyncCalls   int
}

func (m *mockRouterConfigExtension) GetFromResource(pcommon.Resource) (*commonapi.ContainerCollectorConfig, bool) {
	return nil, false
}

func (m *mockRouterConfigExtension) IsActiveSource(pcommon.Resource) bool { return true }

func (m *mockRouterConfigExtension) GetWorkloadCacheKey(pcommon.Resource) (string, error) {
	return "", nil
}

func (m *mockRouterConfigExtension) GetWorkloadIdentityFromResource(pcommon.Resource) (string, pcommon.Map, error) {
	return "", pcommon.NewMap(), nil
}

func (m *mockRouterConfigExtension) RegisterWorkloadConfigCacheCallback(odigoscollector.WorkloadConfigCacheCallback) {
}

func (m *mockRouterConfigExtension) UnregisterWorkloadConfigCacheCallback(odigoscollector.WorkloadConfigCacheCallback) {
}

func (m *mockRouterConfigExtension) WaitForCacheSync(context.Context) bool {
	m.cacheSyncCalls++
	return true
}

func (m *mockRouterConfigExtension) GetDataStreamsForWorkload(res pcommon.Resource) ([]string, bool) {
	service, ok := res.Attributes().Get(routerServiceAttr)
	if !ok {
		return nil, false
	}
	streams, found := m.streamsByService[service.Str()]
	if !found {
		return nil, false
	}
	return streams, true
}

func (m *mockRouterConfigExtension) Start(context.Context, component.Host) error { return nil }
func (m *mockRouterConfigExtension) Shutdown(context.Context) error              { return nil }

// notOdigosExtension is a component registered under the configured extension ID that does not
// implement odigoscollector.OdigosConfigExtension.
type notOdigosExtension struct{}

func (notOdigosExtension) Start(context.Context, component.Host) error { return nil }
func (notOdigosExtension) Shutdown(context.Context) error              { return nil }

func newRouterExtension(streamsByService map[string][]string) *mockRouterConfigExtension {
	return &mockRouterConfigExtension{streamsByService: streamsByService}
}

func startRouter(t *testing.T, c component.Component, ext component.Component) {
	t.Helper()
	host := &routerTestHost{extensions: map[component.ID]component.Component{routerExtensionID: ext}}
	require.NoError(t, c.Start(t.Context(), host))
	t.Cleanup(func() {
		require.NoError(t, c.Shutdown(t.Context()))
	})
}

func newObservedSettings() (connector.Settings, *observer.ObservedLogs) {
	core, logs := observer.New(zapcore.DebugLevel)
	set := connectortest.NewNopSettings(typ)
	set.Logger = zap.New(core)
	return set, logs
}

// ***************************** data builders *****************************

func newRouterTraces(services ...string) ptrace.Traces {
	td := ptrace.NewTraces()
	for _, service := range services {
		rs := td.ResourceSpans().AppendEmpty()
		rs.Resource().Attributes().PutStr(routerServiceAttr, service)
		rs.ScopeSpans().AppendEmpty().Spans().AppendEmpty().SetName(service)
	}
	return td
}

func newRouterMetrics(services ...string) pmetric.Metrics {
	md := pmetric.NewMetrics()
	for _, service := range services {
		rm := md.ResourceMetrics().AppendEmpty()
		rm.Resource().Attributes().PutStr(routerServiceAttr, service)
		metric := rm.ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
		metric.SetName(service)
		metric.SetEmptyGauge().DataPoints().AppendEmpty().SetIntValue(1)
	}
	return md
}

func newRouterLogs(services ...string) plog.Logs {
	ld := plog.NewLogs()
	for _, service := range services {
		rl := ld.ResourceLogs().AppendEmpty()
		rl.Resource().Attributes().PutStr(routerServiceAttr, service)
		rl.ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().Body().SetStr(service)
	}
	return ld
}

func newRouterProfiles(services ...string) pprofile.Profiles {
	pd := pprofile.NewProfiles()
	pd.Dictionary().StringTable().Append("")
	for _, service := range services {
		rp := pd.ResourceProfiles().AppendEmpty()
		rp.Resource().Attributes().PutStr(routerServiceAttr, service)
		rp.ScopeProfiles().AppendEmpty().Profiles().AppendEmpty().Samples().AppendEmpty()
	}
	return pd
}

// ***************************** batch readers *****************************

// resourceServices reads the routerServiceAttr of every resource in one delivered batch, so that a
// test can assert both which resources were delivered and how they were grouped into batches.
func resourceServices(count int, resourceAt func(int) pcommon.Resource) []string {
	services := make([]string, 0, count)
	for i := 0; i < count; i++ {
		service, ok := resourceAt(i).Attributes().Get(routerServiceAttr)
		if !ok {
			services = append(services, "")
			continue
		}
		services = append(services, service.Str())
	}
	return services
}

func tracesBatches(sink *consumertest.TracesSink) [][]string {
	var batches [][]string
	for _, td := range sink.AllTraces() {
		batches = append(batches, resourceServices(td.ResourceSpans().Len(), func(i int) pcommon.Resource {
			return td.ResourceSpans().At(i).Resource()
		}))
	}
	return batches
}

func metricsBatches(sink *consumertest.MetricsSink) [][]string {
	var batches [][]string
	for _, md := range sink.AllMetrics() {
		batches = append(batches, resourceServices(md.ResourceMetrics().Len(), func(i int) pcommon.Resource {
			return md.ResourceMetrics().At(i).Resource()
		}))
	}
	return batches
}

func logsBatches(sink *consumertest.LogsSink) [][]string {
	var batches [][]string
	for _, ld := range sink.AllLogs() {
		batches = append(batches, resourceServices(ld.ResourceLogs().Len(), func(i int) pcommon.Resource {
			return ld.ResourceLogs().At(i).Resource()
		}))
	}
	return batches
}

func profilesBatches(sink *consumertest.ProfilesSink) [][]string {
	var batches [][]string
	for _, pd := range sink.AllProfiles() {
		batches = append(batches, resourceServices(pd.ResourceProfiles().Len(), func(i int) pcommon.Resource {
			return pd.ResourceProfiles().At(i).Resource()
		}))
	}
	return batches
}

// ***************************** connector builders *****************************

func tracesPipelineID(dataStream string) collectorpipeline.ID {
	return collectorpipeline.NewIDWithName(collectorpipeline.SignalTraces, dataStream)
}

func newTracesRouterConnector(t *testing.T, set connector.Settings, pipelines map[string]consumer.Traces) connector.Traces {
	t.Helper()
	byID := make(map[collectorpipeline.ID]consumer.Traces, len(pipelines))
	for dataStream, cons := range pipelines {
		byID[tracesPipelineID(dataStream)] = cons
	}
	c, err := createTracesConnector(t.Context(), set, routerConfig(), connector.NewTracesRouter(byID))
	require.NoError(t, err)
	return c
}

func newMetricsRouterConnector(t *testing.T, set connector.Settings, pipelines map[string]consumer.Metrics) connector.Metrics {
	t.Helper()
	byID := make(map[collectorpipeline.ID]consumer.Metrics, len(pipelines))
	for dataStream, cons := range pipelines {
		byID[collectorpipeline.NewIDWithName(collectorpipeline.SignalMetrics, dataStream)] = cons
	}
	c, err := createMetricsConnector(t.Context(), set, routerConfig(), connector.NewMetricsRouter(byID))
	require.NoError(t, err)
	return c
}

func newLogsRouterConnector(t *testing.T, set connector.Settings, pipelines map[string]consumer.Logs) connector.Logs {
	t.Helper()
	byID := make(map[collectorpipeline.ID]consumer.Logs, len(pipelines))
	for dataStream, cons := range pipelines {
		byID[collectorpipeline.NewIDWithName(collectorpipeline.SignalLogs, dataStream)] = cons
	}
	c, err := createLogsConnector(t.Context(), set, routerConfig(), connector.NewLogsRouter(byID))
	require.NoError(t, err)
	return c
}

func newProfilesRouterConnector(t *testing.T, set connector.Settings, pipelines map[string]xconsumer.Profiles) xconnector.Profiles {
	t.Helper()
	byID := make(map[collectorpipeline.ID]xconsumer.Profiles, len(pipelines))
	for dataStream, cons := range pipelines {
		byID[collectorpipeline.NewIDWithName(xpipeline.SignalProfiles, dataStream)] = cons
	}
	c, err := createProfilesConnector(t.Context(), set, routerConfig(), xconnector.NewProfilesRouter(byID))
	require.NoError(t, err)
	return c
}

// ***************************** Start *****************************

func TestStart_ResolvesOdigosConfigExtension(t *testing.T) {
	ext := newRouterExtension(nil)
	c := newTracesRouterConnector(t, connectortest.NewNopSettings(typ), map[string]consumer.Traces{
		consts.DefaultDataStream: consumertest.NewNop(),
	})

	startRouter(t, c, ext)

	router, ok := c.(*routerConnector)
	require.True(t, ok)
	assert.Same(t, ext, router.odigosConfigExtension)
	// routing reads a cache the extension fills asynchronously, so Start must block on the sync.
	assert.Equal(t, 1, ext.cacheSyncCalls)
}

func TestStart_ExtensionMissingFromHost(t *testing.T) {
	c := newTracesRouterConnector(t, connectortest.NewNopSettings(typ), map[string]consumer.Traces{
		consts.DefaultDataStream: consumertest.NewNop(),
	})

	host := &routerTestHost{extensions: map[component.ID]component.Component{}}
	err := c.Start(t.Context(), host)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "odigos config extension")
	assert.Contains(t, err.Error(), routerExtensionID.String())
}

func TestStart_ExtensionRegisteredAsNil(t *testing.T) {
	c := newTracesRouterConnector(t, connectortest.NewNopSettings(typ), map[string]consumer.Traces{
		consts.DefaultDataStream: consumertest.NewNop(),
	})

	host := &routerTestHost{extensions: map[component.ID]component.Component{routerExtensionID: nil}}
	err := c.Start(t.Context(), host)

	require.ErrorContains(t, err, "not found")
}

func TestStart_ExtensionIsNotAnOdigosConfigExtension(t *testing.T) {
	c := newTracesRouterConnector(t, connectortest.NewNopSettings(typ), map[string]consumer.Traces{
		consts.DefaultDataStream: consumertest.NewNop(),
	})

	host := &routerTestHost{extensions: map[component.ID]component.Component{
		routerExtensionID: notOdigosExtension{},
	}}
	err := c.Start(t.Context(), host)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not a valid odigos config extension")
}

func TestStart_WithoutConfiguredExtensionIsANoop(t *testing.T) {
	byID := map[collectorpipeline.ID]consumer.Traces{tracesPipelineID(consts.DefaultDataStream): consumertest.NewNop()}
	c, err := createTracesConnector(t.Context(), connectortest.NewNopSettings(typ), &Config{}, connector.NewTracesRouter(byID))
	require.NoError(t, err)

	// Config.Validate rejects a missing extension, so this only happens in the generated tests.
	require.NoError(t, c.Start(t.Context(), &routerTestHost{}))
	assert.Nil(t, c.(*routerConnector).odigosConfigExtension)
}

// ***************************** default consumer resolution *****************************

func TestCreateConnector_WithoutDefaultPipelineWarnsAndHasNoDefaultConsumer(t *testing.T) {
	// a collector config whose data streams are all named by the user has no "<signal>/default"
	// pipeline, which is a supported setup, not an error.
	set, logs := newObservedSettings()
	c := newTracesRouterConnector(t, set, map[string]consumer.Traces{
		routerStreamA: consumertest.NewNop(),
	})

	assert.Nil(t, c.(*routerConnector).tracesConfig.defaultCons)
	require.Len(t, logs.FilterMessage("failed to get default traces consumer").All(), 1)
}

func TestCreateConnector_RejectsAConsumerThatIsNotARouter(t *testing.T) {
	set := connectortest.NewNopSettings(typ)
	cfg := routerConfig()

	_, err := createTracesConnector(t.Context(), set, cfg, consumertest.NewNop())
	require.ErrorContains(t, err, "expected consumer to be a connector router")

	_, err = createMetricsConnector(t.Context(), set, cfg, consumertest.NewNop())
	require.ErrorContains(t, err, "expected consumer to be a connector router")

	_, err = createLogsConnector(t.Context(), set, cfg, consumertest.NewNop())
	require.ErrorContains(t, err, "expected consumer to be a connector router")

	_, err = createProfilesConnector(t.Context(), set, cfg, consumertest.NewNop())
	require.ErrorContains(t, err, "expected consumer to be a connector router")
}

// ***************************** capabilities *****************************

func TestCapabilities_DoesNotMutateTheIncomingData(t *testing.T) {
	sink := new(consumertest.TracesSink)
	c := newTracesRouterConnector(t, connectortest.NewNopSettings(typ), map[string]consumer.Traces{
		routerStreamA: sink,
	})
	startRouter(t, c, newRouterExtension(map[string][]string{"checkout": {routerStreamA}}))

	assert.False(t, c.Capabilities().MutatesData)

	td := newRouterTraces("checkout")
	require.NoError(t, c.ConsumeTraces(t.Context(), td))

	// the connector declares it does not mutate data, so the routed batch must be a copy: a
	// downstream pipeline mutating it must not be visible to the other pipelines it was routed to.
	require.Len(t, sink.AllTraces(), 1)
	sink.AllTraces()[0].ResourceSpans().At(0).Resource().Attributes().PutStr(routerServiceAttr, "mutated")
	service, ok := td.ResourceSpans().At(0).Resource().Attributes().Get(routerServiceAttr)
	require.True(t, ok)
	assert.Equal(t, "checkout", service.Str())
}
