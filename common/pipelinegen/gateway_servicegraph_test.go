package pipelinegen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
)

// Spelled out rather than read from consts, for the same reason as the traces
// literals: these are the names the collector config is validated against.
const (
	gpSgConnector        = "servicegraph"
	gpSgPrometheus       = "prometheus/servicegraph"
	gpSgInsightsExporter = "otlp_grpc/servicegraph-insights"
	gpSgPipeline         = "metrics/servicegraph"
	gpSgPodName          = "resource/pod-name"
	gpSelfMetrics        = "prometheus/self-metrics"
)

func gpServiceGraphInput(opts *GatewayConfigOptions) gpInput {
	return gpInput{
		dests:   []config.ExporterConfigurer{gpOtlpDest("d1", common.TracesObservabilitySignal)},
		selfTel: gpSelfTelemetry,
		opts:    opts,
	}
}

// The service graph is the topology map in the UI. It rides on the traces signal, so
// it must be installed whenever traces flow and the operator has not turned it off,
// and must leave nothing behind when they have.
func TestServiceGraphIsInstalledWithTracesUnlessDisabled(t *testing.T) {
	for _, tt := range []struct {
		name  string
		dests []config.ExporterConfigurer
		opts  *GatewayConfigOptions
		want  bool
	}{
		{
			name:  "default - disabled is unset",
			dests: []config.ExporterConfigurer{gpOtlpDest("d1", common.TracesObservabilitySignal)},
			opts:  &GatewayConfigOptions{},
			want:  true,
		},
		{
			name:  "explicitly enabled",
			dests: []config.ExporterConfigurer{gpOtlpDest("d1", common.TracesObservabilitySignal)},
			opts:  &GatewayConfigOptions{ServiceGraph: common.ServiceGraphOptions{Disabled: gpBool(false)}},
			want:  true,
		},
		{
			name:  "explicitly disabled",
			dests: []config.ExporterConfigurer{gpOtlpDest("d1", common.TracesObservabilitySignal)},
			opts:  &GatewayConfigOptions{ServiceGraph: common.ServiceGraphOptions{Disabled: gpBool(true)}},
			want:  false,
		},
		{
			name:  "no traces at all",
			dests: []config.ExporterConfigurer{gpOtlpDest("d1", common.MetricsObservabilitySignal)},
			opts:  &GatewayConfigOptions{},
			want:  false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg, _, _ := gpCalculate(t, gpInput{dests: tt.dests, selfTel: gpSelfTelemetry, opts: tt.opts})

			_, hasConnector := cfg.Connectors[gpSgConnector]
			assert.Equal(t, tt.want, hasConnector, gpSgConnector)

			_, hasPipeline := cfg.Service.Pipelines[gpSgPipeline]
			assert.Equal(t, tt.want, hasPipeline, gpSgPipeline)

			_, hasExporter := cfg.Exporters[gpSgPrometheus]
			assert.Equal(t, tt.want, hasExporter, gpSgPrometheus)
		})
	}
}

// The connector's own built-in virtual-node defaults only cover databases. Odigos
// widens them so plain HTTP/gRPC egress resolves to a real name instead of collapsing
// into a single "unknown" node on the topology map. The expected list is written out
// literally: deriving it from defaultServiceGraphPeerAttributes would not notice an
// attribute being dropped from, or reordered in, that variable.
func TestServiceGraphVirtualNodePeerAttributesDefaultToOdigosWidenedList(t *testing.T) {
	cfg, _, _ := gpCalculate(t, gpServiceGraphInput(&GatewayConfigOptions{}))

	// Asserted as the whole connector config: the store settings bound how long an
	// incomplete edge is retained, and a silent change there drops edges from the
	// topology map without any other symptom.
	assert.Equal(t, config.GenericMap{
		"store":                 config.GenericMap{"ttl": "15s"},
		"store_expiration_loop": "5s",
		"dimensions":            []string{"service.name"},
		"virtual_node_peer_attributes": []string{
			"peer.service",
			"db.name",
			"db.system",
			"server.address",
			"net.peer.name",
			"rpc.service",
		},
	}, cfg.Connectors[gpSgConnector])
}

// First match wins inside the connector, so an operator-supplied list has to be used
// verbatim and in order: appending or prepending the odigos defaults would change
// which attribute names a peer.
func TestServiceGraphUsesTheOperatorPeerAttributeListVerbatim(t *testing.T) {
	cfg, _, _ := gpCalculate(t, gpServiceGraphInput(&GatewayConfigOptions{
		ServiceGraph: common.ServiceGraphOptions{
			VirtualNodePeerAttributes: []string{"rpc.service", "peer.service"},
		},
	}))

	connector, ok := cfg.Connectors[gpSgConnector].(config.GenericMap)
	require.True(t, ok)
	assert.Equal(t, []string{"rpc.service", "peer.service"}, connector["virtual_node_peer_attributes"])
}

// service.name alone is ambiguous across namespaces, so insights needs the namespace
// dimension to key its blast-radius topology. Both halves of the insights gate must
// hold before the dimension is forced in, and it must never be duplicated.
func TestServiceGraphDimensions(t *testing.T) {
	insightsOn := &common.InsightsConfiguration{Enabled: gpBool(true)}
	insightsOff := &common.InsightsConfiguration{Enabled: gpBool(false)}
	const endpoint = "dns:///odigos-insights-headless.odigos-system:4317"

	for _, tt := range []struct {
		name     string
		extra    []string
		insights *common.InsightsConfiguration
		endpoint string
		want     []string
	}{
		{
			name: "no extras, no insights",
			want: []string{"service.name"},
		},
		{
			name:  "operator extras keep their order after service.name",
			extra: []string{"deployment.environment", "k8s.cluster.name"},
			want:  []string{"service.name", "deployment.environment", "k8s.cluster.name"},
		},
		{
			name:     "insights forces the namespace dimension",
			insights: insightsOn,
			endpoint: endpoint,
			want:     []string{"service.name", "k8s.namespace.name"},
		},
		{
			name:     "insights does not duplicate an operator-supplied namespace dimension",
			extra:    []string{"k8s.namespace.name", "deployment.environment"},
			insights: insightsOn,
			endpoint: endpoint,
			want:     []string{"service.name", "k8s.namespace.name", "deployment.environment"},
		},
		{
			name:     "insights enabled but no endpoint resolved",
			insights: insightsOn,
			endpoint: "",
			want:     []string{"service.name"},
		},
		{
			name:     "endpoint resolved but insights disabled",
			insights: insightsOff,
			endpoint: endpoint,
			want:     []string{"service.name"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg, _, _ := gpCalculate(t, gpServiceGraphInput(&GatewayConfigOptions{
				Insights:             tt.insights,
				InsightsOtlpEndpoint: tt.endpoint,
				ServiceGraph:         common.ServiceGraphOptions{ExtraDimensions: tt.extra},
			}))

			connector, ok := cfg.Connectors[gpSgConnector].(config.GenericMap)
			require.True(t, ok)
			assert.Equal(t, tt.want, connector["dimensions"])
		})
	}
}

// The insights fan-out is a second exporter on one pipeline. It needs the feature on
// AND a resolved endpoint; wiring an exporter with an empty endpoint would fail the
// collector at startup and take the whole gateway down.
func TestServiceGraphInsightsFanOutNeedsBothTheFlagAndTheEndpoint(t *testing.T) {
	const endpoint = "dns:///odigos-insights-headless.odigos-system:4317"
	fannedOut := 0

	for _, tt := range []struct {
		name     string
		insights *common.InsightsConfiguration
		endpoint string
		want     bool
	}{
		{name: "enabled with endpoint", insights: &common.InsightsConfiguration{Enabled: gpBool(true)}, endpoint: endpoint, want: true},
		{name: "enabled without endpoint", insights: &common.InsightsConfiguration{Enabled: gpBool(true)}, want: false},
		{name: "disabled with endpoint", insights: &common.InsightsConfiguration{Enabled: gpBool(false)}, endpoint: endpoint, want: false},
		{name: "unset with endpoint", insights: nil, endpoint: endpoint, want: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg, _, _ := gpCalculate(t, gpServiceGraphInput(&GatewayConfigOptions{
				Insights:             tt.insights,
				InsightsOtlpEndpoint: tt.endpoint,
			}))

			pipeline := gpPipeline(t, cfg, gpSgPipeline)
			_, hasExporter := cfg.Exporters[gpSgInsightsExporter]
			assert.Equal(t, tt.want, hasExporter, gpSgInsightsExporter)

			if !tt.want {
				assert.Equal(t, []string{gpSgPrometheus}, pipeline.Exporters)
				assert.Empty(t, pipeline.Processors)
				assert.NotContains(t, cfg.Processors, gpSgPodName)
				return
			}
			fannedOut++

			// Both exporters hang off the single connector, and the gateway pod name
			// is stamped on so insights can tell cumulative counters of different
			// replicas apart.
			assert.Equal(t, []string{gpSgPrometheus, gpSgInsightsExporter}, pipeline.Exporters)
			assert.Equal(t, []string{gpSgPodName}, pipeline.Processors)
			assert.Equal(t, config.GenericMap{
				"endpoint":         endpoint,
				"balancer_name":    "round_robin",
				"tls":              config.GenericMap{"insecure": true},
				"compression":      "none",
				"retry_on_failure": config.GenericMap{"enabled": false},
			}, cfg.Exporters[gpSgInsightsExporter])
		})
	}

	// Guards against an always-true gate: exactly one of the four rows may fan out.
	assert.Equal(t, 1, fannedOut)
}

// resource/pod-name is normally defined by self-telemetry. The service graph only
// defines its own copy as a fallback, and must not clobber the existing one.
func TestServiceGraphReusesAnExistingPodNameProcessor(t *testing.T) {
	existing := config.GenericMap{"attributes": []config.GenericMap{{"key": "already.here"}}}
	cfg, _, _ := gpCalculate(t, gpInput{
		dests: []config.ExporterConfigurer{gpOtlpDest("d1", common.TracesObservabilitySignal)},
		selfTel: func(c *config.Config, a []string, b []string) error {
			c.Processors[gpSgPodName] = existing
			return gpSelfTelemetry(c, a, b)
		},
		opts: &GatewayConfigOptions{
			Insights:             &common.InsightsConfiguration{Enabled: gpBool(true)},
			InsightsOtlpEndpoint: "dns:///insights:4317",
		},
	})

	assert.Equal(t, existing, cfg.Processors[gpSgPodName])
}

// The connector must tap the traces pipeline that actually routes to destinations.
// When the pipeline is forked that is traces/exporting; attaching to traces/in
// instead would feed the service graph pre-sampling spans.
func TestServiceGraphTapsThePipelineThatRoutesToDestinations(t *testing.T) {
	for _, tt := range []struct {
		name       string
		opts       *GatewayConfigOptions
		wantTapped string
		wantClean  string
	}{
		{
			name:       "forked traces pipeline",
			opts:       &GatewayConfigOptions{TailSamplingEnabled: gpBool(true), OdigosConfigExtensionName: gpStr("odigosconfig")},
			wantTapped: gpTracesExporting,
			wantClean:  gpTracesIn,
		},
		{
			name:       "single traces pipeline",
			opts:       &GatewayConfigOptions{},
			wantTapped: gpTracesIn,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg, _, _ := gpCalculate(t, gpServiceGraphInput(tt.opts))

			assert.Contains(t, gpPipeline(t, cfg, tt.wantTapped).Exporters, gpSgConnector)
			if tt.wantClean != "" {
				assert.NotContains(t, gpPipeline(t, cfg, tt.wantClean).Exporters, gpSgConnector)
			}
		})
	}
}

// The UI reads the service graph back off a local prometheus scrape. Without the
// scrape config the metrics are produced and never read, so the topology map is empty
// with no error anywhere.
func TestServiceGraphScrapeConfigIsAppendedToSelfMetrics(t *testing.T) {
	cfg, _, _ := gpCalculate(t, gpServiceGraphInput(&GatewayConfigOptions{}))

	receiver, ok := cfg.Receivers[gpSelfMetrics].(config.GenericMap)
	require.True(t, ok)
	inner, ok := receiver["config"].(config.GenericMap)
	require.True(t, ok)
	scrapes, ok := inner["scrape_configs"].([]config.GenericMap)
	require.True(t, ok)
	require.Len(t, scrapes, 1)

	assert.Equal(t, config.GenericMap{
		"job_name":        "servicegraph",
		"scrape_interval": "10s",
		"static_configs": []config.GenericMap{
			{"targets": []string{"127.0.0.1:9090"}},
		},
		"metric_relabel_configs": []config.GenericMap{
			{
				"source_labels": []string{"__name__"},
				"regex":         "^servicegraph_traces_service_graph_request_total$",
				"action":        "keep",
			},
		},
	}, scrapes[0])

	// The scrape target has to match the endpoint the exporter listens on, or the
	// scrape silently returns nothing.
	assert.Equal(t, config.GenericMap{
		"endpoint":  "localhost:9090",
		"namespace": "servicegraph",
	}, cfg.Exporters[gpSgPrometheus])
}

// AddServiceGraphScrapeConfig is exported and reaches into a receiver built
// elsewhere, so every shape it can be handed has to be rejected rather than panic.
func TestAddServiceGraphScrapeConfigRejectsAMalformedSelfMetricsReceiver(t *testing.T) {
	for _, tt := range []struct {
		name     string
		receiver interface{}
		absent   bool
	}{
		{name: "receiver absent", absent: true},
		{name: "receiver is not a map", receiver: "prometheus"},
		{name: "config key missing", receiver: config.GenericMap{}},
		{name: "config is not a map", receiver: config.GenericMap{"config": "scrape"}},
		{name: "scrape_configs missing", receiver: config.GenericMap{"config": config.GenericMap{}}},
		{
			name:     "scrape_configs is not a list of maps",
			receiver: config.GenericMap{"config": config.GenericMap{"scrape_configs": []string{"a"}}},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c := GetBasicConfig()
			if !tt.absent {
				c.Receivers[gpSelfMetrics] = tt.receiver
			}
			assert.Error(t, AddServiceGraphScrapeConfig(c))
		})
	}
}

// When the scrape config cannot be added the service graph is abandoned half-built:
// the exporter has already been registered. Pinned as the current behaviour so a
// change to it is deliberate rather than accidental.
func TestServiceGraphAbandonsInstallationWhenTheScrapeConfigCannotBeAdded(t *testing.T) {
	cfg, _, _ := gpCalculate(t, gpInput{
		dests: []config.ExporterConfigurer{gpOtlpDest("d1", common.TracesObservabilitySignal)},
		opts:  &GatewayConfigOptions{},
	})

	assert.NotContains(t, cfg.Connectors, gpSgConnector)
	assert.NotContains(t, gpPipelineNames(cfg), gpSgPipeline)
	assert.NotContains(t, gpPipeline(t, cfg, gpTracesIn).Exporters, gpSgConnector)
	assert.Contains(t, cfg.Exporters, gpSgPrometheus)
}
