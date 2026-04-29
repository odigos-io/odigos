package dynamicconfig

import (
	"slices"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	commonapi "github.com/odigos-io/odigos/common/api"
	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/odigos-io/odigos/instrumentor/controllers/agentenabled/dynamicconfig/metrics"
	"github.com/odigos-io/odigos/instrumentor/controllers/agentenabled/dynamicconfig/traces"
	"github.com/odigos-io/odigos/instrumentor/controllers/agentenabled/signals"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
)

// return type from the function that computes the dynamic container configs.
// dynamic config contains agent signals configs, and collector config (if any).
type DynamicContainerConfigs struct {
	// Agent
	AgentTracesConfig  *odigosv1.AgentTracesConfig
	AgentMetricsConfig *odigosv1.AgentMetricsConfig
	AgentLogsConfig    *odigosv1.AgentLogsConfig

	// Collector
	CollectorConfig *commonapi.ContainerCollectorConfig
}

func calculateTracesConfig(
	agentLevelActions *[]odigosv1.Action,
	containerName string,
	runtimeDetails *odigosv1.RuntimeDetailsByContainer,
	pw k8sconsts.PodWorkload,
	d *distro.OtelDistro,
	workloadObj workload.Workload,
	effectiveConfig *common.OdigosConfiguration,
	samplingRules *[]odigosv1.Sampling,
	irls *[]odigosv1.InstrumentationRule,
	nodeCollectorsGroup *odigosv1.CollectorsGroup,
) (*odigosv1.AgentTracesConfig, *commonapi.ContainerCollectorConfig, *odigosv1.AgentDisabledInfo) {
	agentConfig := &odigosv1.AgentTracesConfig{}
	var collectorConfig *commonapi.ContainerCollectorConfig

	// Id Generator
	idGeneratorConfig, err := traces.CalculateIdGeneratorConfig(effectiveConfig)
	if err != nil {
		return nil, nil, err
	}
	agentConfig.IdGenerator = idGeneratorConfig // can be nil

	// Url Templatization
	urlTemplatizationConfig := traces.CalculateUrlTemplatizationConfig(agentLevelActions, containerName, runtimeDetails.Language, pw)
	if urlTemplatizationConfig != nil {
		agentSpanMetricsEnabled := metrics.AgentSpanMetricsEnabled(effectiveConfig)
		if traces.DistroSupportsTracesUrlTemplatization(d) && agentSpanMetricsEnabled {
			agentConfig.UrlTemplatization = urlTemplatizationConfig
		} else {
			collectorConfig = &commonapi.ContainerCollectorConfig{
				UrlTemplatization: urlTemplatizationConfig,
			}
		}
	}

	// Sampling
	noisyOps, relevantOps, costRules := traces.CalculateSamplingCategoryRulesForContainer(samplingRules, runtimeDetails.Language, pw, containerName, d, workloadObj, effectiveConfig)
	// if we have any noisy operations, we need to add them to the traces config.
	// use head/tail sampling based on the distro support.
	if len(noisyOps) > 0 {
		if traces.DistroSupportsHeadSampling(d) {
			dryRun := false
			spanMetricsMode := commonapisampling.SpanMetricsModeSampledSpansOnly
			if effectiveConfig.Sampling != nil && effectiveConfig.Sampling.DryRun != nil {
				dryRun = *effectiveConfig.Sampling.DryRun
			}

			spanMetricsEnabled := nodeCollectorsGroup.Spec.Metrics.SpanMetrics == nil || nodeCollectorsGroup.Spec.Metrics.SpanMetrics.Disabled == nil || !*nodeCollectorsGroup.Spec.Metrics.SpanMetrics.Disabled
			metricsSignalEnabled := slices.Contains(nodeCollectorsGroup.Status.ReceiverSignals, common.MetricsObservabilitySignal)
			configuredMode := effectiveConfig.MetricsSources != nil &&
				effectiveConfig.MetricsSources.SpanMetrics != nil &&
				effectiveConfig.MetricsSources.SpanMetrics.SpanMetricsMode != nil
			if spanMetricsEnabled && metricsSignalEnabled && configuredMode {
				spanMetricsMode = *effectiveConfig.MetricsSources.SpanMetrics.SpanMetricsMode
			}
			agentConfig.HeadSampling = &odigosv1.HeadSamplingConfig{
				DryRun:          dryRun,
				SpanMetricsMode: spanMetricsMode,
				NoisyOperations: noisyOps,
			}
		} else {
			if collectorConfig == nil {
				collectorConfig = &commonapi.ContainerCollectorConfig{}
			}
			collectorConfig.TailSampling = &commonapisampling.TailSamplingSourceConfig{
				NoisyOperations: noisyOps,
			}
		}
	}
	// if we have any highly-relevant or cost-reduction rules, we need to add them to the collector config.
	// create tail sampling for this source if not already created.
	if len(relevantOps) > 0 || len(costRules) > 0 {
		if collectorConfig == nil {
			collectorConfig = &commonapi.ContainerCollectorConfig{}
		}
		if collectorConfig.TailSampling == nil {
			collectorConfig.TailSampling = &commonapisampling.TailSamplingSourceConfig{}
		}
		collectorConfig.TailSampling.HighlyRelevantOperations = relevantOps
		collectorConfig.TailSampling.CostReductionRules = costRules
	}

	// Headers Collection - Agent only (not applicable to collector)
	agentConfig.HeadersCollection = traces.CalculateHeaderCollectionConfig(d, irls)

	// Span Renamer
	// TODO: add support to do it in the collector
	agentConfig.SpanRenamer = traces.CalculateSpanRenamerConfig(d, agentLevelActions, runtimeDetails.Language)

	// Payload Collection - Agent only (not applicable to collector)
	agentConfig.PayloadCollection = traces.CalculatePayloadCollectionConfig(d, irls)

	// Code Attributes - Agent only (not applicable to collector)
	agentConfig.CodeAttributes = traces.CalculateCodeAttributesConfig(d, irls)

	// Custom Instrumentations - Agent only (not applicable to collector)
	agentConfig.CustomInstrumentations = traces.CalculateCustomInstrumentationsConfig(d, irls)

	return agentConfig, collectorConfig, nil
}

func calculateMetricsConfig(
	effectiveConfig *common.OdigosConfiguration,
	d *distro.OtelDistro,
) (*odigosv1.AgentMetricsConfig, *odigosv1.AgentDisabledInfo) {
	metricsConfig := &odigosv1.AgentMetricsConfig{}

	if metrics.DistroSupportsAgentSpanMetrics(d) && metrics.AgentSpanMetricsEnabled(effectiveConfig) {
		// for distros that supports recording span metrics directly in the agent.
		// this is useful for acurate metrics collection, as it see the data as it is collected,
		// before it has chance to be sampled out or dropped in the pipeline.
		agentSpanMetricsConfig, err := metrics.CalculateAgentSpanMetricsConfig(effectiveConfig, d)
		if err != nil {
			return nil, err
		}
		metricsConfig.SpanMetrics = agentSpanMetricsConfig
	}

	// Runtime Metrics for supported distros
	metricsConfig.RuntimeMetrics = metrics.CalculateAgentRuntimeMetricsConfig(d, effectiveConfig)

	return metricsConfig, nil
}

// Calculate the dynamic container configs for a given container.
// the dynamic config contains two parts:
// - agent (per signal config that is consumed by the selected instrumentation agent)
// - collector (config for this container used by collector components)
//
// this function computes the dynamic configs and returns them in a single struct.
// if there is any conflict or issue, and the container should be disabled, return the disabled info.
func CalculateDynamicContainerConfig(
	containerName string,
	irls *[]odigosv1.InstrumentationRule,
	effectiveConfig *common.OdigosConfiguration,
	runtimeDetails *odigosv1.RuntimeDetailsByContainer,
	agentLevelActions *[]odigosv1.Action,
	samplingRules *[]odigosv1.Sampling,
	workloadObj workload.Workload,
	pw k8sconsts.PodWorkload,
	d *distro.OtelDistro,
	enabledSignals signals.EnabledSignals,
	nodeCollectorsGroup *odigosv1.CollectorsGroup,
) (*DynamicContainerConfigs, *odigosv1.AgentDisabledInfo) {

	var collectorConfig *commonapi.ContainerCollectorConfig

	var tracesConfig *odigosv1.AgentTracesConfig
	if enabledSignals.TracesEnabled {
		agentTracesConfig, collectorTracesConfig, err := calculateTracesConfig(agentLevelActions, containerName, runtimeDetails, pw, d, workloadObj, effectiveConfig, samplingRules, irls, nodeCollectorsGroup)
		if err != nil {
			return nil, err
		}
		tracesConfig = agentTracesConfig
		collectorConfig = collectorTracesConfig
	}

	var metricsConfig *odigosv1.AgentMetricsConfig
	if enabledSignals.MetricsEnabled {
		agentMetricsConfig, err := calculateMetricsConfig(effectiveConfig, d)
		if err != nil {
			return nil, err
		}
		metricsConfig = agentMetricsConfig
	}

	var logsConfig *odigosv1.AgentLogsConfig
	if enabledSignals.LogsEnabled {
		logsConfig = &odigosv1.AgentLogsConfig{}
	}

	return &DynamicContainerConfigs{
		AgentTracesConfig:  tracesConfig,
		AgentMetricsConfig: metricsConfig,
		AgentLogsConfig:    logsConfig,
		CollectorConfig:    collectorConfig,
	}, nil
}
