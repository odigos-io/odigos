package pipelinegen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/common/config"
	"github.com/odigos-io/odigos/common/consts"
)

// gpExporter is a minimal ExporterConfigurer. It is deliberately driven through a
// real entry in the destination catalog rather than a stub configurer, so the
// pipeline names it produces are the ones the gateway really gets.
type gpExporter struct {
	id      string
	typ     common.DestinationType
	signals []common.ObservabilitySignal
	cfg     map[string]string
}

func (e gpExporter) GetID() string                            { return e.id }
func (e gpExporter) GetType() common.DestinationType          { return e.typ }
func (e gpExporter) GetSignals() []common.ObservabilitySignal { return e.signals }
func (e gpExporter) GetConfig() map[string]string             { return e.cfg }

// gpOtlpDest is the cheapest real destination in the catalog: one config key, and
// one pipeline per requested signal (traces, metrics, logs).
func gpOtlpDest(id string, signals ...common.ObservabilitySignal) gpExporter {
	return gpExporter{
		id:      id,
		typ:     common.GenericOTLPDestinationType,
		signals: signals,
		cfg:     map[string]string{"OTLP_GRPC_ENDPOINT": "otel.example.com:4317"},
	}
}

// gpPyroscopeDest is the only catalog entry that produces a profiles/ pipeline.
func gpPyroscopeDest(id string) gpExporter {
	return gpExporter{
		id:      id,
		typ:     common.PyroscopeDestinationType,
		signals: []common.ObservabilitySignal{common.ProfilesObservabilitySignal},
		cfg:     map[string]string{"PYROSCOPE_URL": "pyroscope.example.com"},
	}
}

type gpProcessor struct {
	id        string
	typ       string
	signals   []common.ObservabilitySignal
	cfg       config.GenericMap
	orderHint int
	err       error
}

func (p gpProcessor) GetID() string                            { return p.id }
func (p gpProcessor) GetType() string                          { return p.typ }
func (p gpProcessor) GetSignals() []common.ObservabilitySignal { return p.signals }
func (p gpProcessor) GetOrderHint() int                        { return p.orderHint }
func (p gpProcessor) GetConfig() (config.GenericMap, error) {
	if p.err != nil {
		return nil, p.err
	}
	return p.cfg, nil
}

func gpTracesProcessor(id, typ string, orderHint int) gpProcessor {
	return gpProcessor{
		id:        id,
		typ:       typ,
		signals:   []common.ObservabilitySignal{common.TracesObservabilitySignal},
		cfg:       config.GenericMap{"from": id},
		orderHint: orderHint,
	}
}

// gpSelfTelemetry stands in for the callers' applySelfTelemetry hook. It seeds the
// prometheus/self-metrics receiver in the shape AddServiceGraphScrapeConfig expects,
// which is what lets the service graph pipeline be installed at all.
func gpSelfTelemetry(c *config.Config, _ []string, _ []string) error {
	c.Receivers["prometheus/self-metrics"] = config.GenericMap{
		"config": config.GenericMap{"scrape_configs": []config.GenericMap{}},
	}
	return nil
}

func gpBool(b bool) *bool    { return &b }
func gpStr(s string) *string { return &s }

type gpInput struct {
	dests      []config.ExporterConfigurer
	processors []config.ProcessorConfigurer
	streams    []DataStreams
	selfTel    func(*config.Config, []string, []string) error
	opts       *GatewayConfigOptions
}

func gpCalculate(t *testing.T, in gpInput) (*config.Config, *config.ResourceStatuses, []common.ObservabilitySignal) {
	t.Helper()
	opts := in.opts
	if opts == nil {
		opts = &GatewayConfigOptions{}
	}
	cfg, err, status, signals := CalculateGatewayConfig(in.dests, in.processors, in.selfTel, in.streams, opts)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	return cfg, status, signals
}

func gpPipeline(t *testing.T, cfg *config.Config, name string) config.Pipeline {
	t.Helper()
	p, ok := cfg.Service.Pipelines[name]
	require.Truef(t, ok, "pipeline %q is missing; have %v", name, gpPipelineNames(cfg))
	return p
}

func gpPipelineNames(cfg *config.Config) []string {
	names := make([]string, 0, len(cfg.Service.Pipelines))
	for name := range cfg.Service.Pipelines {
		names = append(names, name)
	}
	return names
}

// The literals below are spelled out on purpose instead of being built from the
// consts the production code uses: these names are the collector's wire contract,
// and a test that reads the same constant cannot see that constant being renamed.
const (
	gpTracesIn        = "traces/in"
	gpTracesExporting = "traces/exporting"
	gpPostGroupBy     = "forward/traces-post-groupby"
	gpGroupByTrace    = "groupbytrace"
	gpTailSampling    = "odigostailsampling"
	gpTraceState      = "odigostracestate"
	gpBatch           = "batch/generic-batch-processor"
	gpOdigosVersion   = "resource/odigos-version"
	gpTracesRouter    = "odigosrouterconnector/traces"
)

// Tail sampling and the insights side channel both decide on a whole trace, so the
// gateway has to buffer spans in groupbytrace before anything downstream looks at
// them. Forking traces/in -> traces/exporting is how that buffering is kept ahead of
// the sampling decision; when nothing aggregates, the fork must not appear at all,
// or every span pays the groupbytrace wait for nothing.
func TestTracesRootPipelineForksOnlyWhenSomethingAggregatesWholeTraces(t *testing.T) {
	for _, tt := range []struct {
		name       string
		opts       *GatewayConfigOptions
		wantForked bool
	}{
		{
			name:       "nothing aggregates",
			opts:       &GatewayConfigOptions{ServiceGraph: common.ServiceGraphOptions{Disabled: gpBool(true)}},
			wantForked: false,
		},
		{
			name: "tail sampling alone",
			opts: &GatewayConfigOptions{
				ServiceGraph:              common.ServiceGraphOptions{Disabled: gpBool(true)},
				TailSamplingEnabled:       gpBool(true),
				OdigosConfigExtensionName: gpStr("odigosconfig"),
			},
			wantForked: true,
		},
		{
			name: "insights alone",
			opts: &GatewayConfigOptions{
				ServiceGraph: common.ServiceGraphOptions{Disabled: gpBool(true)},
				Insights:     &common.InsightsConfiguration{Enabled: gpBool(true)},
			},
			wantForked: true,
		},
		{
			name: "both explicitly off",
			opts: &GatewayConfigOptions{
				ServiceGraph:        common.ServiceGraphOptions{Disabled: gpBool(true)},
				TailSamplingEnabled: gpBool(false),
				Insights:            &common.InsightsConfiguration{Enabled: gpBool(false)},
			},
			wantForked: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg, _, _ := gpCalculate(t, gpInput{
				dests: []config.ExporterConfigurer{gpOtlpDest("d1", common.TracesObservabilitySignal)},
				opts:  tt.opts,
			})

			// Anti-vacuity: the root traces pipeline exists in every row, so an
			// assertion about the fork is never passing because nothing was built.
			require.Contains(t, gpPipelineNames(cfg), gpTracesIn)

			_, forked := cfg.Service.Pipelines[gpTracesExporting]
			assert.Equal(t, tt.wantForked, forked, "traces/exporting")

			_, hasForwardConnector := cfg.Connectors[gpPostGroupBy]
			assert.Equal(t, tt.wantForked, hasForwardConnector, gpPostGroupBy)

			_, hasGroupByTrace := cfg.Processors[gpGroupByTrace]
			assert.Equal(t, tt.wantForked, hasGroupByTrace, gpGroupByTrace)
		})
	}
}

// The whole point of the fork: groupbytrace has to run before it and tail sampling
// after it. If tail sampling ever ends up on the aggregating side, or groupbytrace on
// the exporting side, the sampler decides on trace fragments and silently drops spans
// of traces it would have kept.
func TestGroupByTraceRunsBeforeTheForkAndTailSamplingAfterIt(t *testing.T) {
	cfg, _, _ := gpCalculate(t, gpInput{
		dests: []config.ExporterConfigurer{gpOtlpDest("d1", common.TracesObservabilitySignal)},
		opts: &GatewayConfigOptions{
			ServiceGraph:              common.ServiceGraphOptions{Disabled: gpBool(true)},
			TailSamplingEnabled:       gpBool(true),
			OdigosConfigExtensionName: gpStr("odigosconfig"),
		},
	})

	in := gpPipeline(t, cfg, gpTracesIn)
	exporting := gpPipeline(t, cfg, gpTracesExporting)

	assert.Equal(t, []string{gpOdigosVersion, gpGroupByTrace}, in.Processors)
	assert.Equal(t, []string{gpTailSampling, gpTraceState}, exporting.Processors)

	// The two halves are joined by exactly one connector, in one direction.
	assert.Equal(t, []string{"otlp"}, in.Receivers)
	assert.Equal(t, []string{gpPostGroupBy}, in.Exporters)
	assert.Equal(t, []string{gpPostGroupBy}, exporting.Receivers)
	assert.Equal(t, []string{gpTracesRouter}, exporting.Exporters)

	assert.NotContains(t, exporting.Processors, gpGroupByTrace)
	assert.NotContains(t, in.Processors, gpTailSampling)
}

// orderExportingPipelineProcessors is the only thing keeping tail sampling ahead of
// the batch processor. Batching first would hand the sampler pre-assembled batches it
// can no longer split per trace decision.
func TestExportingPipelineAlwaysOrdersTailSamplingBeforeBatch(t *testing.T) {
	for _, tt := range []struct {
		name string
		in   []string
		want []string
	}{
		{name: "already ordered", in: []string{gpTailSampling, gpBatch}, want: []string{gpTailSampling, gpBatch}},
		{name: "reversed input is re-ordered", in: []string{gpBatch, gpTailSampling}, want: []string{gpTailSampling, gpBatch}},
		{
			name: "unknown processors keep their relative order, after the two known ones",
			in:   []string{"transform/a", gpBatch, "filter/b", gpTailSampling},
			want: []string{gpTailSampling, gpBatch, "transform/a", "filter/b"},
		},
		{name: "only tail sampling", in: []string{gpTailSampling}, want: []string{gpTailSampling}},
		{name: "only batch", in: []string{gpBatch}, want: []string{gpBatch}},
		{name: "neither", in: []string{"transform/a"}, want: []string{"transform/a"}},
		{name: "empty", in: []string{}, want: []string{}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, orderExportingPipelineProcessors(tt.in))
		})
	}
}

// splitTracesProcessorsForPipelines decides, per processor, which side of the fork it
// lands on. Everything that is not tail sampling or batching has to stay on the
// aggregating side so it sees the complete trace.
func TestSplitTracesProcessorsKeepsEnrichmentOnTheAggregatingSide(t *testing.T) {
	in, exporting := splitTracesProcessorsForPipelines(
		[]string{gpGroupByTrace, "transform/a", gpTailSampling, gpBatch, "filter/b"},
		[]string{gpTraceState},
	)

	assert.Equal(t, []string{gpGroupByTrace, "transform/a", "filter/b"}, in)
	assert.Equal(t, []string{gpTailSampling, gpBatch, gpTraceState}, exporting)
}

// A traces processor's order hint is what puts it before or after span metrics are
// calculated, which after the fork means before or after the forward connector. 10 is
// the boundary, so both sides of it need a fixture.
func TestTracesProcessorOrderHintDecidesWhichSideOfTheForkItRunsOn(t *testing.T) {
	cfg, _, _ := gpCalculate(t, gpInput{
		dests: []config.ExporterConfigurer{gpOtlpDest("d1", common.TracesObservabilitySignal)},
		processors: []config.ProcessorConfigurer{
			gpTracesProcessor("early", "transform", 9),
			gpTracesProcessor("late", "filter", 10),
		},
		opts: &GatewayConfigOptions{
			ServiceGraph:              common.ServiceGraphOptions{Disabled: gpBool(true)},
			TailSamplingEnabled:       gpBool(true),
			OdigosConfigExtensionName: gpStr("odigosconfig"),
		},
	})

	assert.Equal(t,
		[]string{gpOdigosVersion, gpGroupByTrace, "transform/early"},
		gpPipeline(t, cfg, gpTracesIn).Processors)
	assert.Equal(t,
		[]string{gpTailSampling, "filter/late", gpTraceState},
		gpPipeline(t, cfg, gpTracesExporting).Processors)
}

// Without the fork there is a single traces/in carrying every processor, and none of
// the fork's machinery may be left behind.
func TestTracesPipelineWithoutAggregationIsSingleRooted(t *testing.T) {
	cfg, _, _ := gpCalculate(t, gpInput{
		dests: []config.ExporterConfigurer{gpOtlpDest("d1", common.TracesObservabilitySignal)},
		processors: []config.ProcessorConfigurer{
			gpTracesProcessor("early", "transform", 9),
			gpTracesProcessor("late", "filter", 10),
		},
		opts: &GatewayConfigOptions{ServiceGraph: common.ServiceGraphOptions{Disabled: gpBool(true)}},
	})

	in := gpPipeline(t, cfg, gpTracesIn)
	assert.Equal(t, []string{gpOdigosVersion, "transform/early", "filter/late", gpTraceState}, in.Processors)
	assert.Equal(t, []string{gpTracesRouter}, in.Exporters)

	assert.NotContains(t, gpPipelineNames(cfg), gpTracesExporting)
	assert.NotContains(t, cfg.Connectors, gpPostGroupBy)
	assert.NotContains(t, cfg.Processors, gpGroupByTrace)
}

// Insights taps traces without any traces destination. The gateway still has to
// receive and aggregate spans, but there is nothing to export to, so the fork and the
// router connector must be skipped rather than built with a dangling exporter.
func TestInsightsOnlyGatewayAggregatesTracesWithNoExportPath(t *testing.T) {
	cfg, _, signals := gpCalculate(t, gpInput{
		opts: &GatewayConfigOptions{
			ServiceGraph: common.ServiceGraphOptions{Disabled: gpBool(true)},
			Insights:     &common.InsightsConfiguration{Enabled: gpBool(true)},
		},
	})

	in := gpPipeline(t, cfg, gpTracesIn)
	assert.Equal(t, []string{gpOdigosVersion, gpGroupByTrace}, in.Processors)
	assert.Equal(t, []string{"otlp"}, in.Receivers)
	assert.Empty(t, in.Exporters)

	assert.NotContains(t, gpPipelineNames(cfg), gpTracesExporting)
	assert.NotContains(t, cfg.Connectors, gpPostGroupBy)
	assert.NotContains(t, cfg.Connectors, gpTracesRouter)

	// Traces must still be advertised, otherwise the agents are never told to send
	// spans and the insights side channel stays empty.
	assert.Equal(t, []common.ObservabilitySignal{common.TracesObservabilitySignal}, signals)
}

// The wait duration is how long the gateway holds a trace before releasing it. An
// empty string reaching the collector config would be rejected at startup, so the
// canonical default has to cover both "unset" and "set to empty".
func TestGroupByTraceWaitDurationFallsBackToTheCanonicalDefault(t *testing.T) {
	for _, tt := range []struct {
		name string
		give *string
		want string
	}{
		{name: "unset", give: nil, want: "30s"},
		{name: "empty string", give: gpStr(""), want: "30s"},
		{name: "resolved by the caller", give: gpStr("7s"), want: "7s"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg, _, _ := gpCalculate(t, gpInput{
				dests: []config.ExporterConfigurer{gpOtlpDest("d1", common.TracesObservabilitySignal)},
				opts: &GatewayConfigOptions{
					ServiceGraph:                 common.ServiceGraphOptions{Disabled: gpBool(true)},
					TailSamplingEnabled:          gpBool(true),
					OdigosConfigExtensionName:    gpStr("odigosconfig"),
					TraceAggregationWaitDuration: tt.give,
				},
			})
			assert.Equal(t,
				config.GenericMap{"wait_duration": tt.want},
				cfg.Processors[consts.GroupByTraceProcessor])
		})
	}
}

// The tail sampling processor is only installed when the caller both turns it on and
// names the odigos config extension it reads its rules from. Installing it without the
// extension name would panic in getTailSamplingProcessors.
func TestTailSamplingProcessorNeedsBothTheFlagAndTheConfigExtension(t *testing.T) {
	for _, tt := range []struct {
		name          string
		enabled       *bool
		extensionName *string
		want          bool
	}{
		{name: "enabled with extension", enabled: gpBool(true), extensionName: gpStr("odigosconfig"), want: true},
		{name: "enabled without extension", enabled: gpBool(true), extensionName: nil, want: false},
		{name: "disabled with extension", enabled: gpBool(false), extensionName: gpStr("odigosconfig"), want: false},
		{name: "unset with extension", enabled: nil, extensionName: gpStr("odigosconfig"), want: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg, _, _ := gpCalculate(t, gpInput{
				dests: []config.ExporterConfigurer{gpOtlpDest("d1", common.TracesObservabilitySignal)},
				opts: &GatewayConfigOptions{
					ServiceGraph:              common.ServiceGraphOptions{Disabled: gpBool(true)},
					TailSamplingEnabled:       tt.enabled,
					OdigosConfigExtensionName: tt.extensionName,
				},
			})
			_, installed := cfg.Processors[consts.OdigosTailSamplingProcessorName]
			assert.Equal(t, tt.want, installed)
		})
	}
}

// Each span sampling attribute toggle lands under its own key. They are four
// consecutive same-typed pointer fields, and a bool only has two values, so a fixture
// that sets several of them at once cannot tell a cross-wired assignment from a
// correct one. Each row therefore sets exactly one field and asserts the whole map.
func TestTailSamplingSpanAttributesAreEmittedPerField(t *testing.T) {
	for _, tt := range []struct {
		name string
		give sampling.SpanSamplingAttributesConfiguration
		want config.GenericMap
	}{
		{
			name: "nothing set",
			want: config.GenericMap{},
		},
		{
			name: "disabled only",
			give: sampling.SpanSamplingAttributesConfiguration{Disabled: gpBool(true)},
			want: config.GenericMap{"disabled": true},
		},
		{
			name: "sampling category only",
			give: sampling.SpanSamplingAttributesConfiguration{SamplingCategoryDisabled: gpBool(true)},
			want: config.GenericMap{"sampling_category_disabled": true},
		},
		{
			name: "trace deciding rule only",
			give: sampling.SpanSamplingAttributesConfiguration{TraceDecidingRuleDisabled: gpBool(true)},
			want: config.GenericMap{"trace_deciding_rule_disabled": true},
		},
		{
			name: "span decision attributes only",
			give: sampling.SpanSamplingAttributesConfiguration{SpanDecisionAttributesDisabled: gpBool(true)},
			want: config.GenericMap{"span_decision_attributes_disabled": true},
		},
		{
			name: "all four, mixed values",
			give: sampling.SpanSamplingAttributesConfiguration{
				Disabled:                       gpBool(true),
				SamplingCategoryDisabled:       gpBool(false),
				TraceDecidingRuleDisabled:      gpBool(true),
				SpanDecisionAttributesDisabled: gpBool(false),
			},
			want: config.GenericMap{
				"disabled":                          true,
				"sampling_category_disabled":        false,
				"trace_deciding_rule_disabled":      true,
				"span_decision_attributes_disabled": false,
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			give := tt.give
			cfg, _, _ := gpCalculate(t, gpInput{
				dests: []config.ExporterConfigurer{gpOtlpDest("d1", common.TracesObservabilitySignal)},
				opts: &GatewayConfigOptions{
					ServiceGraph:              common.ServiceGraphOptions{Disabled: gpBool(true)},
					TailSamplingEnabled:       gpBool(true),
					OdigosConfigExtensionName: gpStr("odigosconfig"),
					SamplingDryRun:            true,
					SamplingSpanAttributes:    &give,
				},
			})

			assert.Equal(t, config.GenericMap{
				"odigos_config_extension":  "odigosconfig",
				"dry_run":                  true,
				"span_sampling_attributes": tt.want,
			}, cfg.Processors[consts.OdigosTailSamplingProcessorName])
		})
	}
}

// Both halves of the forked traces pipeline are built by their own code path, so the
// router connector that routes to destinations exists in two literal copies. They have
// to agree, or turning tail sampling on silently unconfigures the router.
func TestTheForkedTracesPipelineBuildsTheSameRouterConnectorAsTheUnforkedOne(t *testing.T) {
	render := func(tailSampling bool) config.GenericMap {
		cfg, _, _ := gpCalculate(t, gpInput{
			dests: []config.ExporterConfigurer{gpOtlpDest("d1", common.TracesObservabilitySignal)},
			opts: &GatewayConfigOptions{
				ServiceGraph:              common.ServiceGraphOptions{Disabled: gpBool(true)},
				TailSamplingEnabled:       gpBool(tailSampling),
				OdigosConfigExtensionName: gpStr("odigosconfigk8sextension"),
			},
		})
		connector, ok := cfg.Connectors[gpTracesRouter].(config.GenericMap)
		require.Truef(t, ok, "traces router connector missing (tailSampling=%v)", tailSampling)
		return connector
	}

	want := config.GenericMap{"odigos_config_extension": "odigosconfigk8sextension"}
	assert.Equal(t, want, render(false))
	assert.Equal(t, want, render(true))
}

// dry_run and span_sampling_attributes are both omitted when the caller says nothing,
// so the collector falls back to its own defaults instead of being handed zero values.
func TestTailSamplingProcessorOmitsUnsetOptionalConfig(t *testing.T) {
	cfg, _, _ := gpCalculate(t, gpInput{
		dests: []config.ExporterConfigurer{gpOtlpDest("d1", common.TracesObservabilitySignal)},
		opts: &GatewayConfigOptions{
			ServiceGraph:              common.ServiceGraphOptions{Disabled: gpBool(true)},
			TailSamplingEnabled:       gpBool(true),
			OdigosConfigExtensionName: gpStr("odigosconfig"),
		},
	})

	assert.Equal(t,
		config.GenericMap{"odigos_config_extension": "odigosconfig"},
		cfg.Processors[consts.OdigosTailSamplingProcessorName])
}
