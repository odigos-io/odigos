package dynamicconfig

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	actionsv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
	"github.com/odigos-io/odigos/common"
	commonapiactions "github.com/odigos-io/odigos/common/api/actions"
	"github.com/odigos-io/odigos/common/api/instrumentationrules"
	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/odigos-io/odigos/instrumentor/controllers/agentenabled/signals"
	"github.com/stretchr/testify/require"
)

const dynamicConfigContainerName = "test-container"

var dynamicConfigPodWorkload = k8sconsts.PodWorkload{
	Namespace: "default",
	Name:      "test-workload",
	Kind:      k8sconsts.WorkloadKindDeployment,
}

// the inputs of CalculateDynamicContainerConfig, with defaults that produce an
// instrumentable container with no user configuration at all.
type dynamicConfigInput struct {
	distro                 *distro.OtelDistro
	effectiveConfig        *common.OdigosConfiguration
	irls                   []odigosv1.InstrumentationRule
	agentLevelActions      []odigosv1.Action
	samplingRules          []odigosv1.Sampling
	enabledSignals         signals.EnabledSignals
	nodeCollectorsGroup    *odigosv1.CollectorsGroup
	clusterCollectorsGroup *odigosv1.CollectorsGroup
}

func newDynamicConfigInput() *dynamicConfigInput {
	return &dynamicConfigInput{
		distro:                 &distro.OtelDistro{Name: "test-distro", Language: common.GoProgrammingLanguage},
		effectiveConfig:        &common.OdigosConfiguration{},
		irls:                   []odigosv1.InstrumentationRule{},
		agentLevelActions:      []odigosv1.Action{},
		samplingRules:          []odigosv1.Sampling{},
		enabledSignals:         signals.EnabledSignals{TracesEnabled: true, MetricsEnabled: true},
		nodeCollectorsGroup:    &odigosv1.CollectorsGroup{},
		clusterCollectorsGroup: &odigosv1.CollectorsGroup{},
	}
}

func (in *dynamicConfigInput) calculate() (*DynamicContainerConfigs, *odigosv1.AgentDisabledInfo) {
	runtimeDetails := &odigosv1.RuntimeDetailsByContainer{
		ContainerName: dynamicConfigContainerName,
		Language:      common.GoProgrammingLanguage,
	}

	return CalculateDynamicContainerConfig(
		dynamicConfigContainerName,
		&in.irls,
		in.effectiveConfig,
		runtimeDetails,
		&in.agentLevelActions,
		&in.samplingRules,
		nil, // workload object, only used for the kubelet health probes auto sampling rules
		dynamicConfigPodWorkload,
		in.distro,
		in.enabledSignals,
		in.nodeCollectorsGroup,
		in.clusterCollectorsGroup,
	)
}

func headSamplingDistro() *distro.OtelDistro {
	return &distro.OtelDistro{
		Name:     "head-sampling-distro",
		Language: common.GoProgrammingLanguage,
		Traces:   &distro.Traces{HeadSampling: &distro.HeadSampling{Supported: true}},
	}
}

func agentUrlTemplatizationDistro() *distro.OtelDistro {
	return &distro.OtelDistro{
		Name:     "url-templatization-distro",
		Language: common.GoProgrammingLanguage,
		Traces:   &distro.Traces{UrlTemplatization: &distro.UrlTemplatization{Supported: true}},
	}
}

func agentSpanMetricsEnabledConfig() *common.OdigosConfiguration {
	return &common.OdigosConfiguration{
		MetricsSources: &common.MetricsSourceConfiguration{
			AgentMetrics: &common.MetricsSourceAgentMetricsConfiguration{
				SpanMetrics: &common.MetricsSourceAgentSpanMetricsConfiguration{Enabled: true},
			},
		},
	}
}

func urlTemplatizationDisablingAction() odigosv1.Action {
	return odigosv1.Action{
		Spec: odigosv1.ActionSpec{
			URLTemplatization: &actionsv1.URLTemplatizationConfig{
				Default: []actionsv1.URLTemplatizationDefaultTemplatizationGroup{
					{DefaultTemplatizationConfig: commonapiactions.DefaultTemplatizationConfig{Disabled: true}},
				},
			},
		},
	}
}

func noisyOperationSamplingRule(name string) odigosv1.Sampling {
	percentage := 5.0
	return odigosv1.Sampling{
		Spec: odigosv1.SamplingSpec{
			NoisyOperations: []odigosv1.NoisyOperation{
				{
					Name: name,
					Operation: &commonapisampling.HeadSamplingOperationMatcher{
						HttpServer: &commonapisampling.HeadSamplingHttpServerOperationMatcher{
							Route:  "/healthz",
							Method: "GET",
						},
					},
					PercentageAtMost: &percentage,
				},
			},
		},
	}
}

// each signal is computed independently, and a disabled signal must not leave any
// configuration behind. the collector config is produced only by the traces path,
// so a container with traces disabled must not get one either.
func TestCalculateDynamicContainerConfig_SignalGating(t *testing.T) {
	t.Parallel()

	t.Run("all signals disabled", func(t *testing.T) {
		t.Parallel()

		in := newDynamicConfigInput()
		in.enabledSignals = signals.EnabledSignals{}

		configs, disabledInfo := in.calculate()

		require.Nil(t, disabledInfo)
		require.Nil(t, configs.AgentTracesConfig)
		require.Nil(t, configs.AgentMetricsConfig)
		require.Nil(t, configs.AgentLogsConfig)
		require.Nil(t, configs.CollectorConfig)
	})

	t.Run("traces only", func(t *testing.T) {
		t.Parallel()

		in := newDynamicConfigInput()
		in.enabledSignals = signals.EnabledSignals{TracesEnabled: true}

		configs, disabledInfo := in.calculate()

		require.Nil(t, disabledInfo)
		require.NotNil(t, configs.AgentTracesConfig)
		require.Nil(t, configs.AgentMetricsConfig)
	})

	t.Run("metrics only", func(t *testing.T) {
		t.Parallel()

		in := newDynamicConfigInput()
		in.enabledSignals = signals.EnabledSignals{MetricsEnabled: true}

		configs, disabledInfo := in.calculate()

		require.Nil(t, disabledInfo)
		require.Nil(t, configs.AgentTracesConfig)
		require.NotNil(t, configs.AgentMetricsConfig)
		require.Nil(t, configs.CollectorConfig)
	})
}

// a configuration error must disable the container instead of pushing a broken config
// to the agent, and only for the signal that actually failed to compute.
func TestCalculateDynamicContainerConfig_ErrorsDisableTheContainer(t *testing.T) {
	t.Parallel()

	t.Run("unparsable trace id suffix", func(t *testing.T) {
		t.Parallel()

		in := newDynamicConfigInput()
		in.effectiveConfig = &common.OdigosConfiguration{TraceIdSuffix: "not-hex"}

		configs, disabledInfo := in.calculate()

		require.Nil(t, configs)
		require.NotNil(t, disabledInfo)
		require.Equal(t, "InjectionConflict", string(disabledInfo.AgentEnabledReason))
		require.Contains(t, disabledInfo.AgentEnabledMessage, "failed to parse trace id suffix")
	})

	t.Run("unparsable trace id suffix is not computed when traces are disabled", func(t *testing.T) {
		t.Parallel()

		in := newDynamicConfigInput()
		in.effectiveConfig = &common.OdigosConfiguration{TraceIdSuffix: "not-hex"}
		in.enabledSignals = signals.EnabledSignals{MetricsEnabled: true}

		configs, disabledInfo := in.calculate()

		require.Nil(t, disabledInfo)
		require.NotNil(t, configs)
	})

	t.Run("valid trace id suffix reaches the agent id generator", func(t *testing.T) {
		t.Parallel()

		in := newDynamicConfigInput()
		in.effectiveConfig = &common.OdigosConfiguration{TraceIdSuffix: "A3"}

		configs, disabledInfo := in.calculate()

		require.Nil(t, disabledInfo)
		require.NotNil(t, configs.AgentTracesConfig.IdGenerator)
		require.Equal(t, uint8(0xA3), configs.AgentTracesConfig.IdGenerator.TimedWall.SourceId)
	})

	t.Run("unparsable span metrics interval", func(t *testing.T) {
		t.Parallel()

		in := newDynamicConfigInput()
		in.distro = &distro.OtelDistro{
			Name:         "span-metrics-distro",
			Language:     common.GoProgrammingLanguage,
			AgentMetrics: &distro.AgentMetrics{SpanMetrics: &distro.SpanMetrics{Supported: true}},
		}
		in.effectiveConfig = agentSpanMetricsEnabledConfig()
		in.effectiveConfig.MetricsSources.SpanMetrics = &common.MetricsSourceSpanMetricsConfiguration{Interval: "one minute"}

		configs, disabledInfo := in.calculate()

		require.Nil(t, configs)
		require.NotNil(t, disabledInfo)
		require.Equal(t, "InjectionConflict", string(disabledInfo.AgentEnabledReason))
		require.Contains(t, disabledInfo.AgentEnabledMessage, "failed to parse span metrics interval")
	})

	t.Run("unparsable span metrics interval is not computed when metrics are disabled", func(t *testing.T) {
		t.Parallel()

		in := newDynamicConfigInput()
		in.distro = &distro.OtelDistro{
			Name:         "span-metrics-distro",
			Language:     common.GoProgrammingLanguage,
			AgentMetrics: &distro.AgentMetrics{SpanMetrics: &distro.SpanMetrics{Supported: true}},
		}
		in.effectiveConfig = agentSpanMetricsEnabledConfig()
		in.effectiveConfig.MetricsSources.SpanMetrics = &common.MetricsSourceSpanMetricsConfiguration{Interval: "one minute"}
		in.enabledSignals = signals.EnabledSignals{TracesEnabled: true}

		configs, disabledInfo := in.calculate()

		require.Nil(t, disabledInfo)
		require.NotNil(t, configs)
	})
}

// URL templatization runs either in the agent or in the collector, never in both.
// running it in the collector while the agent computes span metrics would record
// the raw, high cardinality route on the metrics.
func TestCalculateDynamicContainerConfig_UrlTemplatizationPlacement(t *testing.T) {
	t.Parallel()

	t.Run("distro without agent support templatizes in the collector", func(t *testing.T) {
		t.Parallel()

		in := newDynamicConfigInput()
		in.effectiveConfig = agentSpanMetricsEnabledConfig()

		configs, disabledInfo := in.calculate()

		require.Nil(t, disabledInfo)
		require.Nil(t, configs.AgentTracesConfig.UrlTemplatization)
		require.NotNil(t, configs.CollectorConfig)
		require.NotNil(t, configs.CollectorConfig.UrlTemplatization)
	})

	t.Run("agent support without agent span metrics templatizes in the collector", func(t *testing.T) {
		t.Parallel()

		in := newDynamicConfigInput()
		in.distro = agentUrlTemplatizationDistro()

		configs, disabledInfo := in.calculate()

		require.Nil(t, disabledInfo)
		require.Nil(t, configs.AgentTracesConfig.UrlTemplatization)
		require.NotNil(t, configs.CollectorConfig)
		require.NotNil(t, configs.CollectorConfig.UrlTemplatization)
	})

	t.Run("agent support with agent span metrics templatizes in the agent", func(t *testing.T) {
		t.Parallel()

		in := newDynamicConfigInput()
		in.distro = agentUrlTemplatizationDistro()
		in.effectiveConfig = agentSpanMetricsEnabledConfig()

		configs, disabledInfo := in.calculate()

		require.Nil(t, disabledInfo)
		require.NotNil(t, configs.AgentTracesConfig.UrlTemplatization)
		require.Nil(t, configs.CollectorConfig)
	})

	t.Run("templatization disabled by an action produces no config at all", func(t *testing.T) {
		t.Parallel()

		in := newDynamicConfigInput()
		in.agentLevelActions = []odigosv1.Action{urlTemplatizationDisablingAction()}

		configs, disabledInfo := in.calculate()

		require.Nil(t, disabledInfo)
		require.Nil(t, configs.AgentTracesConfig.UrlTemplatization)
		require.Nil(t, configs.CollectorConfig)
	})
}

// db query templatization, inferred db attributes and PII masking have no agent side
// implementation, so they must always end up in the collector config for the container.
func TestCalculateDynamicContainerConfig_CollectorOnlyTraceFeatures(t *testing.T) {
	t.Parallel()

	dbQueryTemplatizationAction := odigosv1.Action{
		Spec: odigosv1.ActionSpec{
			DbQueryTemplatization: &actionsv1.DbQueryTemplatizationConfig{
				DbQueryTemplatizationConfig: commonapiactions.DbQueryTemplatizationConfig{TemplatizeLiterals: true},
			},
		},
	}
	inferDbAttributesAction := odigosv1.Action{
		Spec: odigosv1.ActionSpec{
			InferDbAttributes: &actionsv1.InferDbAttributesConfig{},
		},
	}
	piiMaskingAction := odigosv1.Action{
		Spec: odigosv1.ActionSpec{
			PiiMasking: &actionsv1.PiiMaskingConfig{
				PiiMaskingConfig: commonapiactions.PiiMaskingConfig{
					PiiCategories: []commonapiactions.PiiCategory{commonapiactions.CreditCardMasking},
				},
			},
		},
	}

	t.Run("each feature is placed in the collector config", func(t *testing.T) {
		t.Parallel()

		in := newDynamicConfigInput()
		in.agentLevelActions = []odigosv1.Action{dbQueryTemplatizationAction, inferDbAttributesAction, piiMaskingAction}

		configs, disabledInfo := in.calculate()

		require.Nil(t, disabledInfo)
		require.NotNil(t, configs.CollectorConfig)
		require.Equal(t, &commonapiactions.DbQueryTemplatizationConfig{TemplatizeLiterals: true}, configs.CollectorConfig.DbQueryTemplatization)
		require.Equal(t, &commonapiactions.InferDbAttributesConfig{}, configs.CollectorConfig.InferDbAttributes)
		require.Equal(t, &commonapiactions.PiiMaskingConfig{
			PiiCategories: []commonapiactions.PiiCategory{commonapiactions.CreditCardMasking},
		}, configs.CollectorConfig.PiiMasking)
	})

	// the collector config is created lazily by whichever feature needs it first,
	// so it must still be created when url templatization is handled by the agent.
	t.Run("collector config is created even when templatization runs in the agent", func(t *testing.T) {
		t.Parallel()

		in := newDynamicConfigInput()
		in.distro = agentUrlTemplatizationDistro()
		in.effectiveConfig = agentSpanMetricsEnabledConfig()
		in.agentLevelActions = []odigosv1.Action{piiMaskingAction}

		configs, disabledInfo := in.calculate()

		require.Nil(t, disabledInfo)
		require.NotNil(t, configs.AgentTracesConfig.UrlTemplatization)
		require.NotNil(t, configs.CollectorConfig)
		require.NotNil(t, configs.CollectorConfig.PiiMasking)
		require.Nil(t, configs.CollectorConfig.UrlTemplatization)
	})

	t.Run("no actions produce no collector side features", func(t *testing.T) {
		t.Parallel()

		in := newDynamicConfigInput()
		in.agentLevelActions = []odigosv1.Action{urlTemplatizationDisablingAction()}

		configs, disabledInfo := in.calculate()

		require.Nil(t, disabledInfo)
		require.Nil(t, configs.CollectorConfig)
	})
}

// these features have no collector side implementation, so each one has to land on its
// own field of the agent traces config. mixing two of them up would silently apply the
// wrong feature to every instrumented container.
func TestCalculateDynamicContainerConfig_AgentOnlyTraceFeatures(t *testing.T) {
	t.Parallel()

	collectStacktrace := true

	in := newDynamicConfigInput()
	in.distro = &distro.OtelDistro{
		Name:     "full-traces-distro",
		Language: common.GoProgrammingLanguage,
		Traces: &distro.Traces{
			HeadersCollection:      &distro.HeadersCollection{Supported: true},
			SpanRenamer:            &distro.SpanRenamer{Supported: true},
			PayloadCollection:      &distro.PayloadCollection{Supported: true},
			CodeAttributes:         &distro.CodeAttributes{Supported: true},
			CustomInstrumentations: &distro.CustomInstrumentations{Supported: true},
			TraceVerbosity:         &distro.TraceVerbosity{DisablingAnyScopeSupported: true},
		},
	}
	in.irls = []odigosv1.InstrumentationRule{
		{
			Spec: odigosv1.InstrumentationRuleSpec{
				HeadersCollection: &instrumentationrules.HttpHeadersCollection{HeaderKeys: []string{"x-request-id"}},
				CodeAttributes:    &instrumentationrules.CodeAttributes{Stacktrace: &collectStacktrace},
				PayloadCollection: &instrumentationrules.PayloadCollection{
					HttpRequest: &instrumentationrules.HttpPayloadCollection{MimeTypes: &[]string{"application/json"}},
				},
				CustomInstrumentations: &instrumentationrules.CustomInstrumentations{
					Golang: []instrumentationrules.GolangCustomProbe{{PackageName: "net/http", FunctionName: "ListenAndServe"}},
				},
				TraceVerbosity: &instrumentationrules.TraceVerbosity{
					DisabledLibraries: []instrumentationrules.InstrumentationLibrary{
						{Language: common.GoProgrammingLanguage, LibraryName: "net/http"},
					},
				},
			},
		},
	}
	in.agentLevelActions = []odigosv1.Action{
		{
			Spec: odigosv1.ActionSpec{
				SpanRenamer: &actionsv1.SpanRenamerConfig{
					ProgrammingLanguage: common.GoProgrammingLanguage,
					ScopeName:           "net/http",
					RegexReplacements: []commonapiactions.SpanRenamerRegexReplacement{
						{RegexPattern: "^GET /users/[0-9]+$", TemplateText: "GET /users/{id}"},
					},
				},
			},
		},
	}

	configs, disabledInfo := in.calculate()

	require.Nil(t, disabledInfo)

	tracesConfig := configs.AgentTracesConfig
	require.Equal(t, []string{"x-request-id"}, tracesConfig.HeadersCollection.HeaderKeys)
	require.Equal(t, &collectStacktrace, tracesConfig.CodeAttributes.Stacktrace)
	require.Equal(t, &[]string{"application/json"}, tracesConfig.PayloadCollection.HttpRequest.MimeTypes)
	require.Equal(t, []instrumentationrules.GolangCustomProbe{
		{PackageName: "net/http", FunctionName: "ListenAndServe"},
	}, tracesConfig.CustomInstrumentations.Golang)
	require.Equal(t, []instrumentationrules.InstrumentationLibrary{
		{Language: common.GoProgrammingLanguage, LibraryName: "net/http"},
	}, tracesConfig.TraceVerbosity.DisabledLibraries)
	require.Equal(t, []commonapiactions.SpanRenamerScopeRules{
		{
			ScopeName: "net/http",
			RegexReplacements: []commonapiactions.SpanRenamerRegexReplacement{
				{RegexPattern: "^GET /users/[0-9]+$", TemplateText: "GET /users/{id}"},
			},
		},
	}, tracesConfig.SpanRenamer.ScopeRules)

	// none of these features are handled by the collector.
	require.NotNil(t, configs.CollectorConfig)
	require.Nil(t, configs.CollectorConfig.TailSampling)
	require.Nil(t, configs.CollectorConfig.DbQueryTemplatization)
	require.Nil(t, configs.CollectorConfig.InferDbAttributes)
	require.Nil(t, configs.CollectorConfig.PiiMasking)
}

// noisy operations are dropped either at the agent (head sampling) or at the node
// collector (tail sampling), and configuring both would sample the same trace twice.
func TestCalculateDynamicContainerConfig_SamplingPlacement(t *testing.T) {
	t.Parallel()

	metricsCollectingNodeCollectorsGroup := &odigosv1.CollectorsGroup{
		Spec: odigosv1.CollectorsGroupSpec{
			Metrics: &odigosv1.CollectorsGroupMetricsCollectionSettings{},
		},
		Status: odigosv1.CollectorsGroupStatus{
			ReceiverSignals: []common.ObservabilitySignal{common.MetricsObservabilitySignal},
		},
	}

	t.Run("head sampling distro keeps noisy operations in the agent", func(t *testing.T) {
		t.Parallel()

		in := newDynamicConfigInput()
		in.distro = headSamplingDistro()
		in.samplingRules = []odigosv1.Sampling{noisyOperationSamplingRule("health check")}

		configs, disabledInfo := in.calculate()

		require.Nil(t, disabledInfo)
		require.NotNil(t, configs.AgentTracesConfig.HeadSampling)
		require.Len(t, configs.AgentTracesConfig.HeadSampling.NoisyOperations, 1)
		require.Equal(t, "health check", configs.AgentTracesConfig.HeadSampling.NoisyOperations[0].Name)
		require.Nil(t, configs.CollectorConfig.TailSampling)
	})

	t.Run("distro without head sampling falls back to collector tail sampling", func(t *testing.T) {
		t.Parallel()

		in := newDynamicConfigInput()
		in.samplingRules = []odigosv1.Sampling{noisyOperationSamplingRule("health check")}

		configs, disabledInfo := in.calculate()

		require.Nil(t, disabledInfo)
		require.Nil(t, configs.AgentTracesConfig.HeadSampling)
		require.NotNil(t, configs.CollectorConfig.TailSampling)
		require.Len(t, configs.CollectorConfig.TailSampling.NoisyOperations, 1)
		require.Equal(t, "health check", configs.CollectorConfig.TailSampling.NoisyOperations[0].Name)
	})

	t.Run("no head sampling config when there is nothing to configure", func(t *testing.T) {
		t.Parallel()

		in := newDynamicConfigInput()
		in.distro = headSamplingDistro()

		configs, disabledInfo := in.calculate()

		require.Nil(t, disabledInfo)
		require.Nil(t, configs.AgentTracesConfig.HeadSampling)
	})

	// the span metrics mode has to reach the agent even without noisy operations,
	// because a sampling decision made by an upstream service propagates to this one.
	t.Run("non default span metrics mode is sent without noisy operations", func(t *testing.T) {
		t.Parallel()

		allSpans := commonapisampling.SpanMetricsModeAllSpans
		in := newDynamicConfigInput()
		in.distro = headSamplingDistro()
		in.nodeCollectorsGroup = metricsCollectingNodeCollectorsGroup
		in.effectiveConfig = &common.OdigosConfiguration{
			MetricsSources: &common.MetricsSourceConfiguration{
				SpanMetrics: &common.MetricsSourceSpanMetricsConfiguration{SpanMetricsMode: &allSpans},
			},
		}

		configs, disabledInfo := in.calculate()

		require.Nil(t, disabledInfo)
		require.NotNil(t, configs.AgentTracesConfig.HeadSampling)
		require.Equal(t, commonapisampling.SpanMetricsModeAllSpans, configs.AgentTracesConfig.HeadSampling.SpanMetricsMode)
		require.Empty(t, configs.AgentTracesConfig.HeadSampling.NoisyOperations)
	})

	t.Run("dry run is propagated to the head sampling config", func(t *testing.T) {
		t.Parallel()

		dryRun := true
		in := newDynamicConfigInput()
		in.distro = headSamplingDistro()
		in.samplingRules = []odigosv1.Sampling{noisyOperationSamplingRule("health check")}
		in.effectiveConfig = &common.OdigosConfiguration{
			Sampling: &common.SamplingConfiguration{DryRun: &dryRun},
		}

		configs, disabledInfo := in.calculate()

		require.Nil(t, disabledInfo)
		require.True(t, configs.AgentTracesConfig.HeadSampling.DryRun)
	})

	// highly relevant operations and cost reduction rules are tail sampling only,
	// so they must reach the collector even for a distro that head samples.
	t.Run("relevant operations and cost rules always reach the collector", func(t *testing.T) {
		t.Parallel()

		in := newDynamicConfigInput()
		in.distro = headSamplingDistro()
		in.samplingRules = []odigosv1.Sampling{
			{
				Spec: odigosv1.SamplingSpec{
					HighlyRelevantOperations: []odigosv1.HighlyRelevantOperation{{Name: "checkout"}},
					CostReductionRules:       []odigosv1.CostReductionRule{{Name: "static assets"}},
				},
			},
		}

		configs, disabledInfo := in.calculate()

		require.Nil(t, disabledInfo)
		require.NotNil(t, configs.CollectorConfig)
		require.NotNil(t, configs.CollectorConfig.TailSampling)
		require.Len(t, configs.CollectorConfig.TailSampling.HighlyRelevantOperations, 1)
		require.Len(t, configs.CollectorConfig.TailSampling.CostReductionRules, 1)
		require.Empty(t, configs.CollectorConfig.TailSampling.NoisyOperations)
	})

	t.Run("tail sampling carries noisy, relevant and cost rules together", func(t *testing.T) {
		t.Parallel()

		in := newDynamicConfigInput()
		in.samplingRules = []odigosv1.Sampling{
			noisyOperationSamplingRule("health check"),
			{
				Spec: odigosv1.SamplingSpec{
					HighlyRelevantOperations: []odigosv1.HighlyRelevantOperation{{Name: "checkout"}},
					CostReductionRules:       []odigosv1.CostReductionRule{{Name: "static assets"}},
				},
			},
		}

		configs, disabledInfo := in.calculate()

		require.Nil(t, disabledInfo)
		require.Len(t, configs.CollectorConfig.TailSampling.NoisyOperations, 1)
		require.Len(t, configs.CollectorConfig.TailSampling.HighlyRelevantOperations, 1)
		require.Len(t, configs.CollectorConfig.TailSampling.CostReductionRules, 1)
	})
}

// logs are collected by the cluster gateway, not by the node collector, so the logs
// config is gated on the cluster collectors group rather than on the enabled signals.
func TestCalculateDynamicContainerConfig_Logs(t *testing.T) {
	t.Parallel()

	enabled := true
	ebpfLogCaptureDistro := &distro.OtelDistro{
		Name:     "ebpf-logs-distro",
		Language: common.GoProgrammingLanguage,
		Logs:     &distro.Logs{EbpfLogCapture: &distro.EbpfLogCapture{Supported: true}},
	}
	ebpfLogCaptureRule := odigosv1.InstrumentationRule{
		Spec: odigosv1.InstrumentationRuleSpec{
			EbpfLogCapture: &instrumentationrules.EbpfLogCapture{Enabled: &enabled},
		},
	}
	logsCollectingClusterGroup := &odigosv1.CollectorsGroup{
		Status: odigosv1.CollectorsGroupStatus{
			ReceiverSignals: []common.ObservabilitySignal{common.LogsObservabilitySignal},
		},
	}

	t.Run("cluster gateway collects logs and the distro supports ebpf capture", func(t *testing.T) {
		t.Parallel()

		in := newDynamicConfigInput()
		in.distro = ebpfLogCaptureDistro
		in.irls = []odigosv1.InstrumentationRule{ebpfLogCaptureRule}
		in.clusterCollectorsGroup = logsCollectingClusterGroup

		configs, disabledInfo := in.calculate()

		require.Nil(t, disabledInfo)
		require.NotNil(t, configs.AgentLogsConfig)
		require.Equal(t, &instrumentationrules.EbpfLogCapture{Enabled: &enabled}, configs.AgentLogsConfig.EbpfLogCapture)
	})

	// the node collectors group never reports logs, so EnabledSignals.LogsEnabled
	// must not be what gates the logs config.
	t.Run("logs are configured even when the node collector reports no logs signal", func(t *testing.T) {
		t.Parallel()

		in := newDynamicConfigInput()
		in.distro = ebpfLogCaptureDistro
		in.irls = []odigosv1.InstrumentationRule{ebpfLogCaptureRule}
		in.clusterCollectorsGroup = logsCollectingClusterGroup
		in.enabledSignals = signals.EnabledSignals{LogsEnabled: false}

		configs, disabledInfo := in.calculate()

		require.Nil(t, disabledInfo)
		require.NotNil(t, configs.AgentLogsConfig)
	})

	t.Run("no cluster collectors group", func(t *testing.T) {
		t.Parallel()

		in := newDynamicConfigInput()
		in.distro = ebpfLogCaptureDistro
		in.irls = []odigosv1.InstrumentationRule{ebpfLogCaptureRule}
		in.clusterCollectorsGroup = nil

		configs, disabledInfo := in.calculate()

		require.Nil(t, disabledInfo)
		require.Nil(t, configs.AgentLogsConfig)
	})

	t.Run("cluster gateway does not collect logs", func(t *testing.T) {
		t.Parallel()

		in := newDynamicConfigInput()
		in.distro = ebpfLogCaptureDistro
		in.irls = []odigosv1.InstrumentationRule{ebpfLogCaptureRule}
		in.clusterCollectorsGroup = &odigosv1.CollectorsGroup{
			Status: odigosv1.CollectorsGroupStatus{
				ReceiverSignals: []common.ObservabilitySignal{common.TracesObservabilitySignal},
			},
		}

		configs, disabledInfo := in.calculate()

		require.Nil(t, disabledInfo)
		require.Nil(t, configs.AgentLogsConfig)
	})

	t.Run("distro does not support ebpf log capture", func(t *testing.T) {
		t.Parallel()

		in := newDynamicConfigInput()
		in.irls = []odigosv1.InstrumentationRule{ebpfLogCaptureRule}
		in.clusterCollectorsGroup = logsCollectingClusterGroup

		configs, disabledInfo := in.calculate()

		require.Nil(t, disabledInfo)
		require.Nil(t, configs.AgentLogsConfig)
	})

	t.Run("no rule requests ebpf log capture", func(t *testing.T) {
		t.Parallel()

		in := newDynamicConfigInput()
		in.distro = ebpfLogCaptureDistro
		in.clusterCollectorsGroup = logsCollectingClusterGroup

		configs, disabledInfo := in.calculate()

		require.Nil(t, disabledInfo)
		require.Nil(t, configs.AgentLogsConfig)
	})
}

func TestCalculateDynamicContainerConfig_Metrics(t *testing.T) {
	t.Parallel()

	t.Run("span metrics require both distro support and configuration", func(t *testing.T) {
		t.Parallel()

		spanMetricsDistro := &distro.OtelDistro{
			Name:         "span-metrics-distro",
			Language:     common.GoProgrammingLanguage,
			AgentMetrics: &distro.AgentMetrics{SpanMetrics: &distro.SpanMetrics{Supported: true}},
		}

		tests := []struct {
			name             string
			distro           *distro.OtelDistro
			effectiveConfig  *common.OdigosConfiguration
			expectConfigured bool
		}{
			{
				name:             "distro without support",
				distro:           newDynamicConfigInput().distro,
				effectiveConfig:  agentSpanMetricsEnabledConfig(),
				expectConfigured: false,
			},
			{
				name:             "not enabled in the odigos configuration",
				distro:           spanMetricsDistro,
				effectiveConfig:  &common.OdigosConfiguration{},
				expectConfigured: false,
			},
			{
				name:             "supported and enabled",
				distro:           spanMetricsDistro,
				effectiveConfig:  agentSpanMetricsEnabledConfig(),
				expectConfigured: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				in := newDynamicConfigInput()
				in.distro = tt.distro
				in.effectiveConfig = tt.effectiveConfig

				configs, disabledInfo := in.calculate()

				require.Nil(t, disabledInfo)
				require.NotNil(t, configs.AgentMetricsConfig)
				if tt.expectConfigured {
					require.NotNil(t, configs.AgentMetricsConfig.SpanMetrics)
				} else {
					require.Nil(t, configs.AgentMetricsConfig.SpanMetrics)
				}
			})
		}
	})

	t.Run("runtime metrics are taken from the odigos configuration", func(t *testing.T) {
		t.Parallel()

		runtimeMetrics := &common.MetricsSourceAgentRuntimeMetricsConfiguration{
			Java: &common.MetricsSourceAgentJavaRuntimeMetricsConfiguration{},
		}

		in := newDynamicConfigInput()
		in.distro = &distro.OtelDistro{
			Name:         "runtime-metrics-distro",
			Language:     common.GoProgrammingLanguage,
			AgentMetrics: &distro.AgentMetrics{RuntimeMetrics: &distro.RuntimeMetrics{Supported: true}},
		}
		in.effectiveConfig = &common.OdigosConfiguration{
			MetricsSources: &common.MetricsSourceConfiguration{
				AgentMetrics: &common.MetricsSourceAgentMetricsConfiguration{RuntimeMetrics: runtimeMetrics},
			},
		}

		configs, disabledInfo := in.calculate()

		require.Nil(t, disabledInfo)
		require.Equal(t, runtimeMetrics, configs.AgentMetricsConfig.RuntimeMetrics)
	})

	t.Run("network metrics are taken from the instrumentation rules", func(t *testing.T) {
		t.Parallel()

		networkMetrics := &instrumentationrules.NetworkMetricsConfig{}

		in := newDynamicConfigInput()
		in.irls = []odigosv1.InstrumentationRule{
			{Spec: odigosv1.InstrumentationRuleSpec{NetworkMetrics: networkMetrics}},
		}

		configs, disabledInfo := in.calculate()

		require.Nil(t, disabledInfo)
		require.Equal(t, networkMetrics, configs.AgentMetricsConfig.NetworkMetrics)
	})
}

// the agent diagnostics config controls the agent's own logging, which is how a broken
// agent is debugged. it is not tied to any signal and must survive all of them being off.
func TestCalculateDynamicContainerConfig_AgentDiagnostics(t *testing.T) {
	t.Parallel()

	debugLevel := common.LogLevelDebug

	in := newDynamicConfigInput()
	in.distro = &distro.OtelDistro{
		Name:           "diagnostics-distro",
		Language:       common.GoProgrammingLanguage,
		OwnDiagnostics: &distro.OwnDiagnostics{OdigosAgentOwnLogerSupported: true},
	}
	in.irls = []odigosv1.InstrumentationRule{
		{
			Spec: odigosv1.InstrumentationRuleSpec{
				AgentDiagnostics: &instrumentationrules.AgentDiagnostics{OdigosLogLevel: &debugLevel},
			},
		},
	}
	in.enabledSignals = signals.EnabledSignals{}

	configs, disabledInfo := in.calculate()

	require.Nil(t, disabledInfo)
	require.NotNil(t, configs.AgentDiagnostics)
	require.Equal(t, &debugLevel, configs.AgentDiagnostics.OdigosLogLevel)
}
