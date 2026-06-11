package dynamicconfig

import (
	"slices"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	commonapi "github.com/odigos-io/odigos/common/api"
	"github.com/odigos-io/odigos/common/api/agentsignalconfig"
	"github.com/odigos-io/odigos/common/api/instrumentationrules"
	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/odigos-io/odigos/instrumentor/controllers/agentenabled/dynamicconfig/logs"
	"github.com/odigos-io/odigos/instrumentor/controllers/agentenabled/dynamicconfig/metrics"
	"github.com/odigos-io/odigos/instrumentor/controllers/agentenabled/dynamicconfig/traces"
	"github.com/odigos-io/odigos/instrumentor/controllers/agentenabled/signals"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
)

// return type from the function that computes the dynamic container configs.
// dynamic config contains agent signals configs, and collector config (if any).
type DynamicContainerConfigs struct {
	// Agent
	AgentTracesConfig  *agentsignalconfig.AgentTracesConfig
	AgentMetricsConfig *agentsignalconfig.AgentMetricsConfig
	AgentLogsConfig    *agentsignalconfig.AgentLogsConfig

	// Collector
	CollectorConfig *commonapi.ContainerCollectorConfig

	// Odigos Agent self logger
	AgentDiagnostics *instrumentationrules.AgentDiagnostics
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
) (*agentsignalconfig.AgentTracesConfig, *commonapi.ContainerCollectorConfig, *odigosv1.AgentDisabledInfo) {
	agentConfig := &agentsignalconfig.AgentTracesConfig{}
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

	// use head/tail sampling based on the distro support.
	// we need to set the span metrics mode even if no noisy operations are present,
	// since the decision can be made at other service and propagate to this one.
	distroSupportsHeadSampling := traces.DistroSupportsHeadSampling(d)
	if distroSupportsHeadSampling {

		spanMetricsMode := metrics.CalculateSpanMetricsMode(effectiveConfig, nodeCollectorsGroup)

		// write the head sampling only if needed, e.g. if there are any noisy operations or non-default configuration.
		if len(noisyOps) > 0 || spanMetricsMode != commonapisampling.SpanMetricsModeSampledSpansOnly {

			dryRun := metrics.CalculateDryRun(effectiveConfig)

			agentConfig.HeadSampling = &commonapisampling.HeadSamplingConfig{
				DryRun:          dryRun,
				SpanMetricsMode: spanMetricsMode,
				NoisyOperations: noisyOps,
			}
		}
	} else if len(noisyOps) > 0 {
		if collectorConfig == nil {
			collectorConfig = &commonapi.ContainerCollectorConfig{}
		}
		collectorConfig.TailSampling = &commonapisampling.TailSamplingSourceConfig{
			NoisyOperations: noisyOps,
		}
	}

	if len(noisyOps) > 0 {
		if traces.DistroSupportsHeadSampling(d) {

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

	// Trace Verbosity - Agent only (not applicable to collector)
	agentConfig.TraceVerbosity = traces.CalculateTraceVerbosityConfig(d, irls)

	// Custom Instrumentations - Agent only (not applicable to collector)
	agentConfig.CustomInstrumentations = traces.CalculateCustomInstrumentationsConfig(d, irls)

	return agentConfig, collectorConfig, nil
}

func calculateMetricsConfig(
	effectiveConfig *common.OdigosConfiguration,
	d *distro.OtelDistro,
) (*agentsignalconfig.AgentMetricsConfig, *odigosv1.AgentDisabledInfo) {
	metricsConfig := &agentsignalconfig.AgentMetricsConfig{}

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
	clusterCollectorsGroup *odigosv1.CollectorsGroup,
) (*DynamicContainerConfigs, *odigosv1.AgentDisabledInfo) {

	var collectorConfig *commonapi.ContainerCollectorConfig

	var tracesConfig *agentsignalconfig.AgentTracesConfig
	if enabledSignals.TracesEnabled {
		agentTracesConfig, collectorTracesConfig, err := calculateTracesConfig(agentLevelActions, containerName, runtimeDetails, pw, d, workloadObj, effectiveConfig, samplingRules, irls, nodeCollectorsGroup)
		if err != nil {
			return nil, err
		}
		tracesConfig = agentTracesConfig
		collectorConfig = collectorTracesConfig
	}

	var metricsConfig *agentsignalconfig.AgentMetricsConfig
	if enabledSignals.MetricsEnabled {
		agentMetricsConfig, err := calculateMetricsConfig(effectiveConfig, d)
		if err != nil {
			return nil, err
		}
		metricsConfig = agentMetricsConfig
	}

	// To determine if logs are enabled, we check the gateway collector group receiver signals
	// because logs won't be present in the data collection (node) collector group.
	logsEnabled := clusterCollectorsGroup != nil && slices.Contains(clusterCollectorsGroup.Status.ReceiverSignals, common.LogsObservabilitySignal)
	var logsConfig *agentsignalconfig.AgentLogsConfig
	ebpfLogCaptureConfig := logs.CalculateEbpfLogCaptureConfig(d, irls)

	if logsEnabled && ebpfLogCaptureConfig != nil {
		logsConfig = &agentsignalconfig.AgentLogsConfig{}
		logsConfig.EbpfLogCapture = ebpfLogCaptureConfig
	}

	odigosAgentDiagnostics := CalculateAgentDiagnostics(irls, d)

	return &DynamicContainerConfigs{
		AgentTracesConfig:  tracesConfig,
		AgentMetricsConfig: metricsConfig,
		AgentLogsConfig:    logsConfig,
		CollectorConfig:    collectorConfig,
		AgentDiagnostics:   odigosAgentDiagnostics,
	}, nil
}
