package odigosrouterconnector

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/connector/connectortest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/consumer/xconsumer"

	"github.com/odigos-io/odigos/common/consts"
)

var errDownstream = errors.New("downstream pipeline rejected the batch")

// routingCase drives the same expectations through all four signals, since ConsumeTraces,
// ConsumeMetrics, ConsumeLogs and ConsumeProfiles each reimplement the routing decision.
type routingCase struct {
	name string
	// dataStreams are the pipelines registered on the router, in addition to the default one.
	dataStreams []string
	// streamsByService is the extension's view of the data streams each workload belongs to.
	streamsByService map[string][]string
	// services is the payload, one resource per entry, in order.
	services []string
	// want lists, per pipeline, the batches it must receive, each batch described by the services
	// of the resources it carries. A pipeline missing from the map must receive nothing.
	want map[string][][]string
}

var routingCases = []routingCase{
	{
		name:        "each workload is delivered to the data stream it belongs to",
		dataStreams: []string{routerStreamA, routerStreamB},
		streamsByService: map[string][]string{
			"checkout": {routerStreamA},
			"frontend": {routerStreamB},
		},
		services: []string{"checkout", "frontend"},
		want: map[string][][]string{
			routerStreamA: {{"checkout"}},
			routerStreamB: {{"frontend"}},
		},
	},
	{
		name:             "a workload in two data streams is delivered to both",
		dataStreams:      []string{routerStreamA, routerStreamB},
		streamsByService: map[string][]string{"checkout": {routerStreamA, routerStreamB}},
		services:         []string{"checkout"},
		want: map[string][][]string{
			routerStreamA: {{"checkout"}},
			routerStreamB: {{"checkout"}},
		},
	},
	{
		name:        "workloads sharing a data stream are delivered as a single batch",
		dataStreams: []string{routerStreamA},
		streamsByService: map[string][]string{
			"checkout": {routerStreamA},
			"cart":     {routerStreamA},
		},
		services: []string{"checkout", "cart"},
		want: map[string][][]string{
			routerStreamA: {{"checkout", "cart"}},
		},
	},
	{
		name:             "a workload the extension does not know falls back to the default data stream",
		dataStreams:      []string{routerStreamA},
		streamsByService: nil,
		services:         []string{"checkout"},
		want: map[string][][]string{
			consts.DefaultDataStream: {{"checkout"}},
		},
	},
	{
		name:             "a workload that belongs to no data stream falls back to the default data stream",
		dataStreams:      []string{routerStreamA},
		streamsByService: map[string][]string{"checkout": {}},
		services:         []string{"checkout"},
		want: map[string][][]string{
			consts.DefaultDataStream: {{"checkout"}},
		},
	},
	{
		name:             "unmatched workloads do not follow the matched ones",
		dataStreams:      []string{routerStreamA},
		streamsByService: map[string][]string{"checkout": {routerStreamA}},
		services:         []string{"checkout", "legacy"},
		want: map[string][][]string{
			routerStreamA:            {{"checkout"}},
			consts.DefaultDataStream: {{"legacy"}},
		},
	},
	{
		name:             "a workload explicitly assigned to the default data stream is routed before the fallback",
		dataStreams:      []string{routerStreamA},
		streamsByService: map[string][]string{"checkout": {consts.DefaultDataStream}},
		services:         []string{"checkout", "legacy"},
		want: map[string][][]string{
			consts.DefaultDataStream: {{"checkout"}, {"legacy"}},
		},
	},
	{
		name:             "an empty payload is not delivered anywhere",
		dataStreams:      []string{routerStreamA},
		streamsByService: map[string][]string{"checkout": {routerStreamA}},
		services:         nil,
		want:             nil,
	},
}

// pipelineNames returns the data streams of the case plus the default one.
func (tc routingCase) pipelineNames() []string {
	names := make([]string, 0, len(tc.dataStreams)+1)
	names = append(names, tc.dataStreams...)
	return append(names, consts.DefaultDataStream)
}

func TestConsumeTraces_Routing(t *testing.T) {
	for _, tc := range routingCases {
		t.Run(tc.name, func(t *testing.T) {
			sinks := map[string]*consumertest.TracesSink{}
			pipelines := map[string]consumer.Traces{}
			for _, name := range tc.pipelineNames() {
				sink := new(consumertest.TracesSink)
				sinks[name] = sink
				pipelines[name] = sink
			}

			c := newTracesRouterConnector(t, connectortest.NewNopSettings(typ), pipelines)
			startRouter(t, c, newRouterExtension(tc.streamsByService))

			require.NoError(t, c.ConsumeTraces(t.Context(), newRouterTraces(tc.services...)))

			for name, sink := range sinks {
				assert.Equal(t, tc.want[name], tracesBatches(sink), "pipeline traces/%s", name)
			}
		})
	}
}

func TestConsumeMetrics_Routing(t *testing.T) {
	for _, tc := range routingCases {
		t.Run(tc.name, func(t *testing.T) {
			sinks := map[string]*consumertest.MetricsSink{}
			pipelines := map[string]consumer.Metrics{}
			for _, name := range tc.pipelineNames() {
				sink := new(consumertest.MetricsSink)
				sinks[name] = sink
				pipelines[name] = sink
			}

			c := newMetricsRouterConnector(t, connectortest.NewNopSettings(typ), pipelines)
			startRouter(t, c, newRouterExtension(tc.streamsByService))

			require.NoError(t, c.ConsumeMetrics(t.Context(), newRouterMetrics(tc.services...)))

			for name, sink := range sinks {
				assert.Equal(t, tc.want[name], metricsBatches(sink), "pipeline metrics/%s", name)
			}
		})
	}
}

func TestConsumeLogs_Routing(t *testing.T) {
	for _, tc := range routingCases {
		t.Run(tc.name, func(t *testing.T) {
			sinks := map[string]*consumertest.LogsSink{}
			pipelines := map[string]consumer.Logs{}
			for _, name := range tc.pipelineNames() {
				sink := new(consumertest.LogsSink)
				sinks[name] = sink
				pipelines[name] = sink
			}

			c := newLogsRouterConnector(t, connectortest.NewNopSettings(typ), pipelines)
			startRouter(t, c, newRouterExtension(tc.streamsByService))

			require.NoError(t, c.ConsumeLogs(t.Context(), newRouterLogs(tc.services...)))

			for name, sink := range sinks {
				assert.Equal(t, tc.want[name], logsBatches(sink), "pipeline logs/%s", name)
			}
		})
	}
}

func TestConsumeProfiles_Routing(t *testing.T) {
	for _, tc := range routingCases {
		t.Run(tc.name, func(t *testing.T) {
			sinks := map[string]*consumertest.ProfilesSink{}
			pipelines := map[string]xconsumer.Profiles{}
			for _, name := range tc.pipelineNames() {
				sink := new(consumertest.ProfilesSink)
				sinks[name] = sink
				pipelines[name] = sink
			}

			c := newProfilesRouterConnector(t, connectortest.NewNopSettings(typ), pipelines)
			startRouter(t, c, newRouterExtension(tc.streamsByService))

			require.NoError(t, c.ConsumeProfiles(t.Context(), newRouterProfiles(tc.services...)))

			for name, sink := range sinks {
				assert.Equal(t, tc.want[name], profilesBatches(sink), "pipeline profiles/%s", name)
			}
		})
	}
}

// ***************************** data stream without a pipeline *****************************

// A workload can reference a data stream that has no pipeline in the running collector config, for
// example while the gateway is still rolling out a config that added the data stream. The resource
// is then dropped rather than falling back to the default pipeline, and only ConsumeMetrics reports
// the failure to the receiver.
func TestConsume_DataStreamWithoutAPipelineIsDropped(t *testing.T) {
	const unknownStream = "not-in-this-config"
	streams := map[string][]string{"checkout": {unknownStream}}

	t.Run("traces", func(t *testing.T) {
		routed, fallback := new(consumertest.TracesSink), new(consumertest.TracesSink)
		c := newTracesRouterConnector(t, connectortest.NewNopSettings(typ), map[string]consumer.Traces{
			routerStreamA:            routed,
			consts.DefaultDataStream: fallback,
		})
		startRouter(t, c, newRouterExtension(streams))

		require.NoError(t, c.ConsumeTraces(t.Context(), newRouterTraces("checkout")))
		assert.Empty(t, routed.AllTraces())
		assert.Empty(t, fallback.AllTraces())
	})

	t.Run("metrics", func(t *testing.T) {
		routed, fallback := new(consumertest.MetricsSink), new(consumertest.MetricsSink)
		c := newMetricsRouterConnector(t, connectortest.NewNopSettings(typ), map[string]consumer.Metrics{
			routerStreamA:            routed,
			consts.DefaultDataStream: fallback,
		})
		startRouter(t, c, newRouterExtension(streams))

		err := c.ConsumeMetrics(t.Context(), newRouterMetrics("checkout"))
		require.ErrorContains(t, err, "failed to get metrics consumer for pipeline "+unknownStream)
		require.ErrorContains(t, err, "missing consumer")
		assert.Empty(t, routed.AllMetrics())
		assert.Empty(t, fallback.AllMetrics())
	})

	t.Run("logs", func(t *testing.T) {
		routed, fallback := new(consumertest.LogsSink), new(consumertest.LogsSink)
		c := newLogsRouterConnector(t, connectortest.NewNopSettings(typ), map[string]consumer.Logs{
			routerStreamA:            routed,
			consts.DefaultDataStream: fallback,
		})
		startRouter(t, c, newRouterExtension(streams))

		require.NoError(t, c.ConsumeLogs(t.Context(), newRouterLogs("checkout")))
		assert.Empty(t, routed.AllLogs())
		assert.Empty(t, fallback.AllLogs())
	})

	t.Run("profiles", func(t *testing.T) {
		routed, fallback := new(consumertest.ProfilesSink), new(consumertest.ProfilesSink)
		c := newProfilesRouterConnector(t, connectortest.NewNopSettings(typ), map[string]xconsumer.Profiles{
			routerStreamA:            routed,
			consts.DefaultDataStream: fallback,
		})
		startRouter(t, c, newRouterExtension(streams))

		require.NoError(t, c.ConsumeProfiles(t.Context(), newRouterProfiles("checkout")))
		assert.Empty(t, routed.AllProfiles())
		assert.Empty(t, fallback.AllProfiles())
	})
}

// ***************************** downstream failures *****************************

// A failing data stream must be reported to the receiver so it can retry, and must not stop the
// other data streams of the same batch from being delivered.
func TestConsume_ReportsDownstreamErrorsAndKeepsRoutingOtherDataStreams(t *testing.T) {
	streams := map[string][]string{
		"checkout": {routerStreamA},
		"frontend": {routerStreamB},
	}

	t.Run("traces", func(t *testing.T) {
		healthy := new(consumertest.TracesSink)
		c := newTracesRouterConnector(t, connectortest.NewNopSettings(typ), map[string]consumer.Traces{
			routerStreamA:            consumertest.NewErr(errDownstream),
			routerStreamB:            healthy,
			consts.DefaultDataStream: new(consumertest.TracesSink),
		})
		startRouter(t, c, newRouterExtension(streams))

		err := c.ConsumeTraces(t.Context(), newRouterTraces("checkout", "frontend"))
		require.ErrorIs(t, err, errDownstream)
		assert.Equal(t, [][]string{{"frontend"}}, tracesBatches(healthy))
	})

	t.Run("metrics", func(t *testing.T) {
		healthy := new(consumertest.MetricsSink)
		c := newMetricsRouterConnector(t, connectortest.NewNopSettings(typ), map[string]consumer.Metrics{
			routerStreamA:            consumertest.NewErr(errDownstream),
			routerStreamB:            healthy,
			consts.DefaultDataStream: new(consumertest.MetricsSink),
		})
		startRouter(t, c, newRouterExtension(streams))

		err := c.ConsumeMetrics(t.Context(), newRouterMetrics("checkout", "frontend"))
		require.ErrorIs(t, err, errDownstream)
		assert.Equal(t, [][]string{{"frontend"}}, metricsBatches(healthy))
	})

	t.Run("logs", func(t *testing.T) {
		healthy := new(consumertest.LogsSink)
		c := newLogsRouterConnector(t, connectortest.NewNopSettings(typ), map[string]consumer.Logs{
			routerStreamA:            consumertest.NewErr(errDownstream),
			routerStreamB:            healthy,
			consts.DefaultDataStream: new(consumertest.LogsSink),
		})
		startRouter(t, c, newRouterExtension(streams))

		err := c.ConsumeLogs(t.Context(), newRouterLogs("checkout", "frontend"))
		require.ErrorIs(t, err, errDownstream)
		assert.Equal(t, [][]string{{"frontend"}}, logsBatches(healthy))
	})

	t.Run("profiles", func(t *testing.T) {
		healthy := new(consumertest.ProfilesSink)
		c := newProfilesRouterConnector(t, connectortest.NewNopSettings(typ), map[string]xconsumer.Profiles{
			routerStreamA:            consumertest.NewErr(errDownstream),
			routerStreamB:            healthy,
			consts.DefaultDataStream: new(consumertest.ProfilesSink),
		})
		startRouter(t, c, newRouterExtension(streams))

		err := c.ConsumeProfiles(t.Context(), newRouterProfiles("checkout", "frontend"))
		require.ErrorIs(t, err, errDownstream)
		assert.Equal(t, [][]string{{"frontend"}}, profilesBatches(healthy))
	})
}

// A failure of the default pipeline is only logged: unlike a data stream failure it is not reported
// to the receiver, so the unmatched telemetry is dropped instead of being retried.
func TestConsume_DefaultPipelineFailureIsOnlyLogged(t *testing.T) {
	t.Run("traces", func(t *testing.T) {
		set, logs := newObservedSettings()
		c := newTracesRouterConnector(t, set, map[string]consumer.Traces{
			consts.DefaultDataStream: consumertest.NewErr(errDownstream),
		})
		startRouter(t, c, newRouterExtension(nil))

		require.NoError(t, c.ConsumeTraces(t.Context(), newRouterTraces("checkout")))
		assert.Len(t, logs.FilterMessage("failed to send traces to the default pipeline").All(), 1)
	})

	t.Run("metrics", func(t *testing.T) {
		set, logs := newObservedSettings()
		c := newMetricsRouterConnector(t, set, map[string]consumer.Metrics{
			consts.DefaultDataStream: consumertest.NewErr(errDownstream),
		})
		startRouter(t, c, newRouterExtension(nil))

		require.NoError(t, c.ConsumeMetrics(t.Context(), newRouterMetrics("checkout")))
		assert.Len(t, logs.FilterMessage("failed to send metrics to the default pipeline").All(), 1)
	})

	t.Run("logs", func(t *testing.T) {
		set, logs := newObservedSettings()
		c := newLogsRouterConnector(t, set, map[string]consumer.Logs{
			consts.DefaultDataStream: consumertest.NewErr(errDownstream),
		})
		startRouter(t, c, newRouterExtension(nil))

		require.NoError(t, c.ConsumeLogs(t.Context(), newRouterLogs("checkout")))
		assert.Len(t, logs.FilterMessage("failed to send logs to the default pipeline").All(), 1)
	})

	t.Run("profiles", func(t *testing.T) {
		set, logs := newObservedSettings()
		c := newProfilesRouterConnector(t, set, map[string]xconsumer.Profiles{
			consts.DefaultDataStream: consumertest.NewErr(errDownstream),
		})
		startRouter(t, c, newRouterExtension(nil))

		require.NoError(t, c.ConsumeProfiles(t.Context(), newRouterProfiles("checkout")))
		assert.Len(t, logs.FilterMessage("failed to send profiles to the default pipeline").All(), 1)
	})
}

// A collector config where every data stream is named has no "<signal>/default" pipeline, so the
// default consumer is nil and unmatched telemetry must be dropped without panicking.
func TestConsume_WithoutADefaultPipelineUnmatchedDataIsDropped(t *testing.T) {
	streams := map[string][]string{"checkout": {routerStreamA}}

	t.Run("traces", func(t *testing.T) {
		routed := new(consumertest.TracesSink)
		c := newTracesRouterConnector(t, connectortest.NewNopSettings(typ), map[string]consumer.Traces{
			routerStreamA: routed,
		})
		startRouter(t, c, newRouterExtension(streams))

		require.NoError(t, c.ConsumeTraces(t.Context(), newRouterTraces("checkout", "legacy")))
		assert.Equal(t, [][]string{{"checkout"}}, tracesBatches(routed))
	})

	t.Run("metrics", func(t *testing.T) {
		routed := new(consumertest.MetricsSink)
		c := newMetricsRouterConnector(t, connectortest.NewNopSettings(typ), map[string]consumer.Metrics{
			routerStreamA: routed,
		})
		startRouter(t, c, newRouterExtension(streams))

		require.NoError(t, c.ConsumeMetrics(t.Context(), newRouterMetrics("checkout", "legacy")))
		assert.Equal(t, [][]string{{"checkout"}}, metricsBatches(routed))
	})

	t.Run("logs", func(t *testing.T) {
		routed := new(consumertest.LogsSink)
		c := newLogsRouterConnector(t, connectortest.NewNopSettings(typ), map[string]consumer.Logs{
			routerStreamA: routed,
		})
		startRouter(t, c, newRouterExtension(streams))

		require.NoError(t, c.ConsumeLogs(t.Context(), newRouterLogs("checkout", "legacy")))
		assert.Equal(t, [][]string{{"checkout"}}, logsBatches(routed))
	})

	t.Run("profiles", func(t *testing.T) {
		routed := new(consumertest.ProfilesSink)
		c := newProfilesRouterConnector(t, connectortest.NewNopSettings(typ), map[string]xconsumer.Profiles{
			routerStreamA: routed,
		})
		startRouter(t, c, newRouterExtension(streams))

		require.NoError(t, c.ConsumeProfiles(t.Context(), newRouterProfiles("checkout", "legacy")))
		assert.Equal(t, [][]string{{"checkout"}}, profilesBatches(routed))
	})
}
