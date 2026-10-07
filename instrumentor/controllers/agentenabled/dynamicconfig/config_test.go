package dynamicconfig

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/odigos-io/odigos/instrumentor/controllers/agentenabled/signals"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTraceSurgeSpanMetricsInterval(t *testing.T) {
	enabled := true
	pw := k8sconsts.PodWorkload{Namespace: "shop", Kind: k8sconsts.WorkloadKindDeployment, Name: "payments"}
	runtimeDetails := &odigosv1.RuntimeDetailsByContainer{ContainerName: "app", Language: common.JavaProgrammingLanguage}
	d := &distro.OtelDistro{AgentMetrics: &distro.AgentMetrics{SpanMetrics: &distro.SpanMetrics{Supported: true}}}
	surgeRules := &[]odigosv1.Sampling{{Spec: odigosv1.SamplingSpec{NoisyOperations: []odigosv1.NoisyOperation{{
		Name: "shop",
		Surge: &odigosv1.TraceSurgeSettings{Metric: odigosv1.TraceSurgeMetricErrorRate, Threshold: 5, SustainedSeconds: 60,
			MinimumRequests: 100, BoostPercent: 50, RecoveryThreshold: 2, RecoverySeconds: 60, MinimumBoostSeconds: 120},
	}}}}}

	intervalMs := func(conf *common.OdigosConfiguration, rules *[]odigosv1.Sampling) int {
		t.Helper()
		configs, disabled := CalculateDynamicContainerConfig("app", &[]odigosv1.InstrumentationRule{}, conf, runtimeDetails,
			&[]odigosv1.Action{}, rules, nil, pw, d, signals.EnabledSignals{}, nil, nil)
		require.Nil(t, disabled)
		if configs.AgentMetricsConfig == nil || configs.AgentMetricsConfig.SpanMetrics == nil {
			return 0
		}
		return configs.AgentMetricsConfig.SpanMetrics.IntervalMs
	}

	insights := &common.InsightsConfiguration{Enabled: &enabled}
	assert.Equal(t, 10_000, intervalMs(&common.OdigosConfiguration{Insights: insights}, surgeRules),
		"a covered agent reports at the default evaluation interval, not the 60s span metrics default")
	assert.Equal(t, 5_000, intervalMs(&common.OdigosConfiguration{Insights: insights,
		Sampling: &common.SamplingConfiguration{TraceSurge: &common.TraceSurgeConfiguration{EvaluationInterval: "5s"}}}, surgeRules))
	assert.Equal(t, 2_000, intervalMs(&common.OdigosConfiguration{Insights: insights,
		MetricsSources: &common.MetricsSourceConfiguration{SpanMetrics: &common.MetricsSourceSpanMetricsConfiguration{Interval: "2s"}}}, surgeRules),
		"a shorter span metrics interval is kept")
	assert.Equal(t, 0, intervalMs(&common.OdigosConfiguration{Insights: insights}, &[]odigosv1.Sampling{}),
		"without a surge rule nothing is recorded")
	assert.Equal(t, 0, intervalMs(&common.OdigosConfiguration{}, surgeRules), "without insights nothing is recorded")
}

func TestAgentRecordsSpanMetricsForDestinations(t *testing.T) {
	pw := k8sconsts.PodWorkload{Namespace: "shop", Kind: k8sconsts.WorkloadKindDeployment, Name: "frontend"}
	runtimeDetails := &odigosv1.RuntimeDetailsByContainer{ContainerName: "app", Language: common.JavaProgrammingLanguage}
	recording := &distro.OtelDistro{
		AgentMetrics: &distro.AgentMetrics{SpanMetrics: &distro.SpanMetrics{Supported: true}},
		Traces:       &distro.Traces{HeadSampling: &distro.HeadSampling{Supported: true}},
	}
	wantsSpanMetrics := &odigosv1.CollectorsGroup{
		Spec: odigosv1.CollectorsGroupSpec{Metrics: &odigosv1.CollectorsGroupMetricsCollectionSettings{
			SpanMetrics: &common.MetricsSourceSpanMetricsConfiguration{Interval: "60s"},
		}},
		Status: odigosv1.CollectorsGroupStatus{ReceiverSignals: []common.ObservabilitySignal{common.MetricsObservabilitySignal}},
	}
	allSpans := commonapisampling.SpanMetricsModeAllSpans
	conf := func(enabled bool) *common.OdigosConfiguration {
		return &common.OdigosConfiguration{MetricsSources: &common.MetricsSourceConfiguration{
			SpanMetrics:  &common.MetricsSourceSpanMetricsConfiguration{SpanMetricsMode: &allSpans},
			AgentMetrics: &common.MetricsSourceAgentMetricsConfiguration{SpanMetrics: &common.MetricsSourceAgentSpanMetricsConfiguration{Enabled: enabled}},
		}}
	}
	run := func(c *common.OdigosConfiguration, d *distro.OtelDistro, nodeCG *odigosv1.CollectorsGroup) *DynamicContainerConfigs {
		t.Helper()
		configs, disabled := CalculateDynamicContainerConfig("app", &[]odigosv1.InstrumentationRule{}, c, runtimeDetails,
			&[]odigosv1.Action{}, &[]odigosv1.Sampling{}, nil, pw, d, signals.EnabledSignals{TracesEnabled: true, MetricsEnabled: true}, nodeCG, nil)
		require.Nil(t, disabled)
		return configs
	}

	configs := run(conf(false), recording, wantsSpanMetrics)
	assert.Nil(t, configs.AgentMetricsConfig.SpanMetrics, "a destination wanting span metrics leaves them to the collectors, as before")
	require.NotNil(t, configs.AgentTracesConfig.HeadSampling)
	assert.Equal(t, commonapisampling.SpanMetricsModeAllSpans, configs.AgentTracesConfig.HeadSampling.SpanMetricsMode)

	configs = run(conf(true), recording, wantsSpanMetrics)
	require.NotNil(t, configs.AgentMetricsConfig.SpanMetrics, "span metrics in the agents are enabled: the agent records them")
	assert.Equal(t, 60_000, configs.AgentMetricsConfig.SpanMetrics.IntervalMs)
	assert.Equal(t, commonapisampling.SpanMetricsModeAllSpans, configs.AgentTracesConfig.HeadSampling.SpanMetricsMode,
		"the span metrics mode is the configured one")

	configs = run(conf(true), &distro.OtelDistro{Traces: &distro.Traces{HeadSampling: &distro.HeadSampling{Supported: true}}}, wantsSpanMetrics)
	assert.Nil(t, configs.AgentMetricsConfig.SpanMetrics, "an agent that can't record leaves it to the collectors")
}
