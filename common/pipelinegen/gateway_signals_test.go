package pipelinegen

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
)

// The signals the gateway reports back are what the collectors group status
// advertises to the agents. Getting them wrong either starves a destination of data
// or has agents emit a signal nothing consumes. They are derived purely from the
// pipeline-name prefixes the destination configurers hand back.
func TestEnabledSignalsAreDerivedFromTheDestinationPipelinePrefixes(t *testing.T) {
	for _, tt := range []struct {
		name  string
		dests []config.ExporterConfigurer
		opts  *GatewayConfigOptions
		want  []common.ObservabilitySignal
	}{
		{
			name:  "no destinations",
			dests: nil,
			want:  []common.ObservabilitySignal{},
		},
		{
			name:  "traces only",
			dests: []config.ExporterConfigurer{gpOtlpDest("d1", common.TracesObservabilitySignal)},
			want:  []common.ObservabilitySignal{common.TracesObservabilitySignal},
		},
		{
			name:  "metrics only",
			dests: []config.ExporterConfigurer{gpOtlpDest("d1", common.MetricsObservabilitySignal)},
			want:  []common.ObservabilitySignal{common.MetricsObservabilitySignal},
		},
		{
			name:  "logs only",
			dests: []config.ExporterConfigurer{gpOtlpDest("d1", common.LogsObservabilitySignal)},
			want:  []common.ObservabilitySignal{common.LogsObservabilitySignal},
		},
		{
			name:  "profiles only",
			dests: []config.ExporterConfigurer{gpPyroscopeDest("p1")},
			want:  []common.ObservabilitySignal{common.ProfilesObservabilitySignal},
		},
		{
			name: "every signal, across two destinations",
			dests: []config.ExporterConfigurer{
				gpOtlpDest("d1", common.TracesObservabilitySignal, common.MetricsObservabilitySignal, common.LogsObservabilitySignal),
				gpPyroscopeDest("p1"),
			},
			want: []common.ObservabilitySignal{
				common.TracesObservabilitySignal,
				common.MetricsObservabilitySignal,
				common.LogsObservabilitySignal,
				common.ProfilesObservabilitySignal,
			},
		},
		{
			name:  "insights turns traces on with no traces destination",
			dests: []config.ExporterConfigurer{gpOtlpDest("d1", common.LogsObservabilitySignal)},
			opts:  &GatewayConfigOptions{Insights: &common.InsightsConfiguration{Enabled: gpBool(true)}},
			want: []common.ObservabilitySignal{
				common.TracesObservabilitySignal,
				common.LogsObservabilitySignal,
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, _, signals := gpCalculate(t, gpInput{dests: tt.dests, selfTel: gpSelfTelemetry, opts: tt.opts})
			assert.Equal(t, tt.want, signals)
		})
	}
}

// The batch processor does not support profiles in the pinned collector build, so
// putting it on a profiles pipeline fails the collector at startup. There are two
// places that decide this - the per-destination pipeline and the data stream
// pipeline - and both are driven here.
func TestProfilesPipelinesCarryNoBatchProcessorWhileOtherSignalsDo(t *testing.T) {
	cfg, _, _ := gpCalculate(t, gpInput{
		dests: []config.ExporterConfigurer{
			gpOtlpDest("d1", common.TracesObservabilitySignal, common.MetricsObservabilitySignal, common.LogsObservabilitySignal),
			gpPyroscopeDest("p1"),
		},
		selfTel: gpSelfTelemetry,
		streams: []DataStreams{{Name: "ds", Destinations: []Destination{
			{DestinationName: "d1"}, {DestinationName: "p1"},
		}}},
	})

	for _, name := range []string{
		"traces/generic-d1", "metrics/generic-d1", "logs/generic-d1",
		"traces/ds", "metrics/ds", "logs/ds",
	} {
		assert.Equalf(t, []string{gpBatch}, gpPipeline(t, cfg, name).Processors, "pipeline %q", name)
	}

	for _, name := range []string{"profiles/pyroscope-p1", "profiles/ds"} {
		assert.Emptyf(t, gpPipeline(t, cfg, name).Processors, "pipeline %q", name)
	}
}

// A data stream fans one routing connector out to the destinations that belong to it.
// A stream is only materialised per signal that actually has a destination, so a
// stream naming an unknown destination, or a signal no destination serves, must not
// leave an exporter-less pipeline behind for the collector to reject.
func TestDataStreamPipelinesOnlyExistForSignalsTheyCanActuallyReach(t *testing.T) {
	cfg, _, _ := gpCalculate(t, gpInput{
		dests: []config.ExporterConfigurer{
			gpOtlpDest("traces-dest", common.TracesObservabilitySignal),
			gpOtlpDest("logs-dest", common.LogsObservabilitySignal),
		},
		selfTel: gpSelfTelemetry,
		streams: []DataStreams{
			{Name: "mixed", Destinations: []Destination{
				{DestinationName: "traces-dest"},
				{DestinationName: "logs-dest"},
			}},
			{Name: "traces-only", Destinations: []Destination{{DestinationName: "traces-dest"}}},
			{Name: "ghost", Destinations: []Destination{{DestinationName: "no-such-destination"}}},
		},
	})

	assert.Equal(t, config.Pipeline{
		Receivers:  []string{"odigosrouterconnector/traces"},
		Processors: []string{gpBatch},
		Exporters:  []string{"forward/traces/generic-traces-dest"},
	}, gpPipeline(t, cfg, "traces/mixed"))

	assert.Equal(t, config.Pipeline{
		Receivers:  []string{"odigosrouterconnector/logs"},
		Processors: []string{gpBatch},
		Exporters:  []string{"forward/logs/generic-logs-dest"},
	}, gpPipeline(t, cfg, "logs/mixed"))

	assert.Equal(t,
		[]string{"forward/traces/generic-traces-dest"},
		gpPipeline(t, cfg, "traces/traces-only").Exporters)

	names := gpPipelineNames(cfg)
	for _, absent := range []string{
		"metrics/mixed", "profiles/mixed",
		"logs/traces-only", "metrics/traces-only", "profiles/traces-only",
		"traces/ghost", "metrics/ghost", "logs/ghost", "profiles/ghost",
	} {
		assert.NotContainsf(t, names, absent, "pipeline %q should not exist", absent)
	}
}

// Each destination gets its own forward connector per signal, wired as a receiver on
// the destination's own pipeline. Two destinations serving the same signal must not
// collapse onto one connector.
func TestEachDestinationGetsItsOwnForwardConnectorPerSignal(t *testing.T) {
	cfg, _, _ := gpCalculate(t, gpInput{
		dests: []config.ExporterConfigurer{
			gpOtlpDest("a", common.TracesObservabilitySignal, common.MetricsObservabilitySignal),
			gpOtlpDest("b", common.TracesObservabilitySignal),
		},
		selfTel: gpSelfTelemetry,
	})

	for connector, pipeline := range map[string]string{
		"forward/traces/generic-a":  "traces/generic-a",
		"forward/metrics/generic-a": "metrics/generic-a",
		"forward/traces/generic-b":  "traces/generic-b",
	} {
		assert.Containsf(t, cfg.Connectors, connector, "connector %q", connector)
		assert.Equalf(t, []string{connector}, gpPipeline(t, cfg, pipeline).Receivers, "pipeline %q", pipeline)
	}

	assert.NotContains(t, cfg.Connectors, "forward/metrics/generic-b")
}

// A destination that cannot be configured must be reported and skipped, never allowed
// to abort the whole config: one broken destination would otherwise stop telemetry
// reaching every healthy one.
func TestABrokenDestinationIsReportedAndSkippedWithoutAffectingTheHealthyOnes(t *testing.T) {
	cfg, status, signals := gpCalculate(t, gpInput{
		dests: []config.ExporterConfigurer{
			gpExporter{
				id:      "unknown-type",
				typ:     common.DestinationType("not-a-real-destination"),
				signals: []common.ObservabilitySignal{common.TracesObservabilitySignal},
			},
			gpExporter{
				id:      "misconfigured",
				typ:     common.GenericOTLPDestinationType,
				signals: []common.ObservabilitySignal{common.MetricsObservabilitySignal},
				cfg:     map[string]string{},
			},
			gpOtlpDest("healthy", common.TracesObservabilitySignal),
		},
		selfTel: gpSelfTelemetry,
	})

	require.Len(t, status.Destination, 3)
	assert.ErrorContains(t, status.Destination["unknown-type"], "no configer for not-a-real-destination")
	assert.Error(t, status.Destination["misconfigured"])
	assert.NoError(t, status.Destination["healthy"])

	// The two failures contributed nothing: no pipelines, and metrics never became an
	// advertised signal even though the misconfigured destination asked for it.
	assert.Equal(t, []common.ObservabilitySignal{common.TracesObservabilitySignal}, signals)
	assert.Contains(t, gpPipelineNames(cfg), "traces/generic-healthy")
	assert.NotContains(t, gpPipelineNames(cfg), "metrics/in")
}

// Same contract on the processor side: a processor whose config cannot be rendered is
// recorded against its own id and dropped from every pipeline.
func TestABrokenProcessorIsReportedAndLeftOutOfThePipelines(t *testing.T) {
	traces := []common.ObservabilitySignal{common.TracesObservabilitySignal}
	cfg, status, _ := gpCalculate(t, gpInput{
		dests: []config.ExporterConfigurer{gpOtlpDest("d1", common.TracesObservabilitySignal)},
		processors: []config.ProcessorConfigurer{
			gpProcessor{id: "bad", typ: "transform", signals: traces, err: errors.New("bad rule")},
			gpTracesProcessor("good", "filter", 1),
		},
		opts: &GatewayConfigOptions{ServiceGraph: common.ServiceGraphOptions{Disabled: gpBool(true)}},
	})

	require.Len(t, status.Processor, 1)
	assert.ErrorContains(t, status.Processor["bad"], `failed to convert processor "bad" to collector config`)

	assert.NotContains(t, cfg.Processors, "transform/bad")
	assert.Contains(t, cfg.Processors, "filter/good")
	assert.Equal(t,
		[]string{gpOdigosVersion, "filter/good", gpTraceState},
		gpPipeline(t, cfg, gpTracesIn).Processors)
}

// A processor only joins the pipelines of the signals it declares.
func TestProcessorsOnlyJoinThePipelinesOfTheirOwnSignals(t *testing.T) {
	cfg, _, _ := gpCalculate(t, gpInput{
		dests: []config.ExporterConfigurer{
			gpOtlpDest("d1", common.TracesObservabilitySignal, common.MetricsObservabilitySignal, common.LogsObservabilitySignal),
			gpPyroscopeDest("p1"),
		},
		processors: []config.ProcessorConfigurer{
			gpProcessor{id: "t", typ: "transform", signals: []common.ObservabilitySignal{common.TracesObservabilitySignal}, cfg: config.GenericMap{}},
			gpProcessor{id: "m", typ: "transform", signals: []common.ObservabilitySignal{common.MetricsObservabilitySignal}, cfg: config.GenericMap{}},
			gpProcessor{id: "l", typ: "transform", signals: []common.ObservabilitySignal{common.LogsObservabilitySignal}, cfg: config.GenericMap{}},
			gpProcessor{id: "p", typ: "transform", signals: []common.ObservabilitySignal{common.ProfilesObservabilitySignal}, cfg: config.GenericMap{}},
		},
		opts: &GatewayConfigOptions{ServiceGraph: common.ServiceGraphOptions{Disabled: gpBool(true)}},
	})

	assert.Equal(t, []string{gpOdigosVersion, "transform/t", gpTraceState}, gpPipeline(t, cfg, gpTracesIn).Processors)
	assert.Equal(t, []string{gpOdigosVersion, "transform/m"}, gpPipeline(t, cfg, "metrics/in").Processors)
	assert.Equal(t, []string{gpOdigosVersion, "transform/l"}, gpPipeline(t, cfg, "logs/in").Processors)
	assert.Equal(t, []string{gpOdigosVersion, "transform/p"}, gpPipeline(t, cfg, "profiles/in").Processors)
}

// Cluster metrics are scraped by one elected gateway replica. Both halves have to
// hold: the receiver is only useful on a metrics pipeline, and it is opt-in.
func TestClusterMetricsNeedBothTheMetricsSignalAndTheFlag(t *testing.T) {
	for _, tt := range []struct {
		name    string
		signal  common.ObservabilitySignal
		enabled *bool
		want    bool
	}{
		{name: "metrics and enabled", signal: common.MetricsObservabilitySignal, enabled: gpBool(true), want: true},
		{name: "metrics but not enabled", signal: common.MetricsObservabilitySignal, enabled: gpBool(false), want: false},
		{name: "metrics with the flag unset", signal: common.MetricsObservabilitySignal, enabled: nil, want: false},
		{name: "enabled but no metrics destination", signal: common.TracesObservabilitySignal, enabled: gpBool(true), want: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg, _, _ := gpCalculate(t, gpInput{
				dests:   []config.ExporterConfigurer{gpOtlpDest("d1", tt.signal)},
				selfTel: gpSelfTelemetry,
				opts: &GatewayConfigOptions{
					ClusterMetricsEnabled: tt.enabled,
					OdigosNamespace:       "odigos-ns",
				},
			})

			_, hasReceiver := cfg.Receivers["k8s_cluster"]
			assert.Equal(t, tt.want, hasReceiver, "k8s_cluster receiver")
			_, hasExtension := cfg.Extensions["k8s_leader_elector"]
			assert.Equal(t, tt.want, hasExtension, "k8s_leader_elector extension")

			if !tt.want {
				assert.NotContains(t, cfg.Service.Extensions, "k8s_leader_elector")
				return
			}
			assert.Contains(t, cfg.Service.Extensions, "k8s_leader_elector")

			// Only one replica may scrape, and the lease has to live in the odigos
			// namespace the caller passed rather than a hardcoded one.
			assert.Equal(t, config.GenericMap{
				"lease_name":      "odigos-gateway-leader",
				"lease_namespace": "odigos-ns",
			}, cfg.Extensions["k8s_leader_elector"])
			assert.Equal(t, config.GenericMap{
				"k8s_leader_elector": "k8s_leader_elector",
			}, cfg.Receivers["k8s_cluster"])
			assert.Equal(t,
				[]string{"otlp", "k8s_cluster"},
				gpPipeline(t, cfg, "metrics/in").Receivers)
		})
	}
}

// The odigos config extension is what the router connectors and the tail sampling
// processor read their runtime rules from. Every router connector has to name it, or
// that connector silently routes nothing.
func TestOdigosConfigExtensionIsRegisteredAndNamedByEveryRouterConnector(t *testing.T) {
	cfg, _, _ := gpCalculate(t, gpInput{
		dests: []config.ExporterConfigurer{
			gpOtlpDest("d1", common.TracesObservabilitySignal, common.MetricsObservabilitySignal, common.LogsObservabilitySignal),
			gpPyroscopeDest("p1"),
		},
		selfTel: gpSelfTelemetry,
		opts:    &GatewayConfigOptions{OdigosConfigExtensionName: gpStr("odigosconfigk8sextension")},
	})

	assert.Contains(t, cfg.Extensions, "odigosconfigk8sextension")
	assert.Contains(t, cfg.Service.Extensions, "odigosconfigk8sextension")

	for _, signal := range []string{"traces", "metrics", "logs", "profiles"} {
		assert.Equalf(t,
			config.GenericMap{"odigos_config_extension": "odigosconfigk8sextension"},
			cfg.Connectors["odigosrouterconnector/"+signal],
			"router connector for %s", signal)
	}
}

// Without an extension name the connectors are still created, just unconfigured.
func TestRouterConnectorsAreCreatedWithoutAnOdigosConfigExtension(t *testing.T) {
	cfg, _, _ := gpCalculate(t, gpInput{
		dests:   []config.ExporterConfigurer{gpOtlpDest("d1", common.TracesObservabilitySignal)},
		selfTel: gpSelfTelemetry,
	})

	assert.Equal(t, config.GenericMap{}, cfg.Connectors[gpTracesRouter])
	assert.Equal(t, []string{"health_check", "pprof"}, cfg.Service.Extensions)
}

// A failing self-telemetry hook aborts the whole render rather than shipping a
// gateway config missing its own observability.
func TestSelfTelemetryFailureAbortsTheRender(t *testing.T) {
	cfg, err, status, signals := CalculateGatewayConfig(
		[]config.ExporterConfigurer{gpOtlpDest("d1", common.TracesObservabilitySignal)},
		nil,
		func(*config.Config, []string, []string) error { return errors.New("self telemetry exploded") },
		nil,
		&GatewayConfigOptions{},
	)

	require.ErrorContains(t, err, "self telemetry exploded")
	assert.Nil(t, cfg)
	assert.Nil(t, signals)
	// The per-destination statuses are still reported so the caller can surface them.
	assert.NoError(t, status.Destination["d1"])

	// The YAML wrapper must propagate the failure rather than write an empty
	// ConfigMap body over a working one.
	data, err, status, signals := GetGatewayConfig(
		[]config.ExporterConfigurer{gpOtlpDest("d1", common.TracesObservabilitySignal)},
		nil,
		func(*config.Config, []string, []string) error { return errors.New("self telemetry exploded") },
		nil,
		&GatewayConfigOptions{},
	)
	require.ErrorContains(t, err, "self telemetry exploded")
	assert.Empty(t, data)
	assert.Nil(t, signals)
	assert.NoError(t, status.Destination["d1"])
}

// GetGatewayConfig renders the ConfigMap body the cluster collector controller
// server-side-applies on every reconcile, and it owns the Deployment it patches. A
// render that is not byte-stable re-triggers its own reconcile forever, which is what
// CORE-1716 was. service.extensions in particular is accumulated by append and only
// made deterministic by the sort in GetGatewayConfig.
func TestGatewayConfigRendersIdenticallyEveryTime(t *testing.T) {
	render := func() string {
		data, err, _, _ := GetGatewayConfig(
			[]config.ExporterConfigurer{
				gpOtlpDest("a", common.TracesObservabilitySignal, common.MetricsObservabilitySignal),
				gpOtlpDest("b", common.LogsObservabilitySignal),
				gpPyroscopeDest("p"),
			},
			[]config.ProcessorConfigurer{
				gpTracesProcessor("early", "transform", 1),
				gpTracesProcessor("late", "filter", 20),
			},
			gpSelfTelemetry,
			[]DataStreams{
				{Name: "one", Destinations: []Destination{{DestinationName: "a"}, {DestinationName: "p"}}},
				{Name: "two", Destinations: []Destination{{DestinationName: "b"}}},
			},
			&GatewayConfigOptions{
				TailSamplingEnabled:       gpBool(true),
				OdigosConfigExtensionName: gpStr("odigosconfigk8sextension"),
				ClusterMetricsEnabled:     gpBool(true),
				OdigosNamespace:           "odigos-system",
				Insights:                  &common.InsightsConfiguration{Enabled: gpBool(true)},
				InsightsOtlpEndpoint:      "dns:///odigos-insights-headless.odigos-system:4317",
				ServiceGraph:              common.ServiceGraphOptions{ExtraDimensions: []string{"deployment.environment"}},
			},
		)
		require.NoError(t, err)
		return data
	}

	first := render()

	// Anti-vacuity: comparing an empty render against itself would pass forever.
	require.Contains(t, first, "traces/exporting")
	require.Contains(t, first, "metrics/servicegraph")
	require.Contains(t, first, "k8s_leader_elector")
	require.Contains(t, first, "profiles/one")

	for i := 0; i < 50; i++ {
		assert.Equal(t, first, render(), "render %d differed", i)
	}
}

// The extensions list is built by append from several independent places, so the sort
// is the only thing keeping it stable. Asserted on the rendered YAML, because
// CalculateGatewayConfig deliberately leaves it in insertion order.
func TestGatewayConfigSortsServiceExtensions(t *testing.T) {
	cfg, _, _ := gpCalculate(t, gpInput{
		dests:   []config.ExporterConfigurer{gpOtlpDest("d1", common.MetricsObservabilitySignal)},
		selfTel: gpSelfTelemetry,
		opts: &GatewayConfigOptions{
			OdigosConfigExtensionName: gpStr("odigosconfigk8sextension"),
			ClusterMetricsEnabled:     gpBool(true),
			OdigosNamespace:           "odigos-system",
		},
	})
	require.Equal(t,
		[]string{"health_check", "pprof", "k8s_leader_elector", "odigosconfigk8sextension"},
		cfg.Service.Extensions)

	data, err, _, _ := GetGatewayConfig(
		[]config.ExporterConfigurer{gpOtlpDest("d1", common.MetricsObservabilitySignal)},
		nil, gpSelfTelemetry, nil,
		&GatewayConfigOptions{
			OdigosConfigExtensionName: gpStr("odigosconfigk8sextension"),
			ClusterMetricsEnabled:     gpBool(true),
			OdigosNamespace:           "odigos-system",
		},
	)
	require.NoError(t, err)
	assert.Contains(t, data, "extensions:\n  - health_check\n  - k8s_leader_elector\n  - odigosconfigk8sextension\n  - pprof\n")
}

// A pipeline naming a component that was never defined fails the collector at
// startup, which takes the whole gateway down rather than degrading one signal. The
// production code adds names to pipelines and definitions to the component maps in
// separate statements, so nothing but this invariant ties the two together.
func TestEveryComponentAPipelineNamesIsAlsoDefined(t *testing.T) {
	cfg, _, _ := gpCalculate(t, gpInput{
		dests: []config.ExporterConfigurer{
			gpOtlpDest("a", common.TracesObservabilitySignal, common.MetricsObservabilitySignal),
			gpOtlpDest("b", common.LogsObservabilitySignal),
			gpPyroscopeDest("p"),
		},
		processors: []config.ProcessorConfigurer{
			gpTracesProcessor("early", "transform", 1),
			gpTracesProcessor("late", "filter", 20),
		},
		selfTel: gpSelfTelemetry,
		streams: []DataStreams{
			{Name: "one", Destinations: []Destination{{DestinationName: "a"}, {DestinationName: "p"}}},
			{Name: "two", Destinations: []Destination{{DestinationName: "b"}}},
		},
		opts: &GatewayConfigOptions{
			TailSamplingEnabled:       gpBool(true),
			OdigosConfigExtensionName: gpStr("odigosconfigk8sextension"),
			ClusterMetricsEnabled:     gpBool(true),
			OdigosNamespace:           "odigos-system",
			Insights:                  &common.InsightsConfiguration{Enabled: gpBool(true)},
			InsightsOtlpEndpoint:      "dns:///odigos-insights-headless.odigos-system:4317",
		},
	})

	defined := func(maps ...map[string]interface{}) func(string) bool {
		return func(name string) bool {
			for _, m := range maps {
				if _, ok := m[name]; ok {
					return true
				}
			}
			return false
		}
	}
	receivers := defined(cfg.Receivers, cfg.Connectors)
	processors := defined(cfg.Processors)
	exporters := defined(cfg.Exporters, cfg.Connectors)

	checked := 0
	for name, pipeline := range cfg.Service.Pipelines {
		for _, r := range pipeline.Receivers {
			assert.Truef(t, receivers(r), "pipeline %q receives from undefined %q", name, r)
			checked++
		}
		for _, p := range pipeline.Processors {
			assert.Truef(t, processors(p), "pipeline %q uses undefined processor %q", name, p)
			checked++
		}
		for _, e := range pipeline.Exporters {
			assert.Truef(t, exporters(e), "pipeline %q exports to undefined %q", name, e)
			checked++
		}
	}

	// Anti-vacuity: this fixture wires every signal, the traces fork, data streams,
	// cluster metrics and the insights fan-out, so it must have plenty to check.
	assert.Greater(t, checked, 40)

	// Every extension the service enables must be defined too.
	for _, e := range cfg.Service.Extensions {
		assert.Containsf(t, cfg.Extensions, e, "service enables undefined extension %q", e)
	}
	assert.NotEmpty(t, cfg.Service.Extensions)
}

// The root pipeline names are the single source of truth shared with the callers'
// self-telemetry hooks; a mismatch there attaches odigos' own metrics to a pipeline
// that does not exist.
func TestRootPipelineNames(t *testing.T) {
	assert.Equal(t, "traces/in", GetTelemetryRootPipelineName(common.TracesObservabilitySignal))
	assert.Equal(t, "metrics/in", GetTelemetryRootPipelineName(common.MetricsObservabilitySignal))
	assert.Equal(t, "logs/in", GetTelemetryRootPipelineName(common.LogsObservabilitySignal))
	assert.Equal(t, "profiles/in", GetTelemetryRootPipelineName(common.ProfilesObservabilitySignal))
	assert.Empty(t, GetTelemetryRootPipelineName(common.ObservabilitySignal("UNKNOWN")))

	assert.Equal(t,
		[]string{"traces/in", "metrics/in", "logs/in", "profiles/in"},
		GetSignalsRootPipelineNames())
}
