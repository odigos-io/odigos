package signalconfig

import (
	"fmt"
	"strconv"

	actionsv1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	actions "github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
)

func CalculateTracesConfig(
	tracesEnabled bool,
	effectiveConfig *common.OdigosConfiguration,
	containerName string,
	programmingLanguage common.ProgrammingLanguage,
	urlTemplatizationConfig *odigosv1.UrlTemplatizationConfig,
	ignoreHealthChecks []actionsv1.IgnoreHealthChecksConfig,
	irls *[]odigosv1.InstrumentationRule,
	agentLevelActions *[]odigosv1.Action,
	workloadObj workload.Workload,
	distro *distro.OtelDistro) (*odigosv1.AgentTracesConfig, *odigosv1.ContainerAgentConfig) {

	if !tracesEnabled {
		return nil, nil
	}

	tracesConfig := &odigosv1.AgentTracesConfig{}

	// for traces, also allow to configure the id generator as "timedwall",
	// if trace id suffix is provided.
	if effectiveConfig.TraceIdSuffix != "" {
		sourceId, err := strconv.ParseUint(effectiveConfig.TraceIdSuffix, 16, 8)
		if err != nil {
			return nil, &odigosv1.ContainerAgentConfig{
				ContainerName:       containerName,
				AgentEnabled:        false,
				AgentEnabledReason:  odigosv1.AgentEnabledReasonInjectionConflict,
				AgentEnabledMessage: fmt.Sprintf("failed to parse trace id suffix: %s. trace id suffix must be a single byte hex value (for example 'A3')", err),
			}
		}
		tracesConfig.IdGenerator = &odigosv1.IdGeneratorConfig{
			TimedWall: &odigosv1.IdGeneratorTimedWallConfig{
				SourceId: uint8(sourceId),
			},
		}
	}

	tracesConfig.UrlTemplatization = urlTemplatizationConfig
	tracesConfig.HeadersCollection = calculateHeaderCollectionConfig(distro, irls)
	tracesConfig.HeadSampling = calculateHeadSamplingConfig(distro, workloadObj, containerName, irls, ignoreHealthChecks)
	tracesConfig.SpanRenamer = filterSpanRenamerForContainer(agentLevelActions, programmingLanguage)

	return tracesConfig, nil
}

func calculateHeaderCollectionConfig(distro *distro.OtelDistro, irls *[]odigosv1.InstrumentationRule) *odigosv1.HeadersCollectionConfig {
	// only calculate header collection config if the distro supports it
	if distro.Traces == nil || distro.Traces.HeadersCollection == nil || !distro.Traces.HeadersCollection.Supported {
		return nil
	}

	// http headers collection configuration
	headerKeysToCollectHttp := []string{}
	for _, irl := range *irls {
		if irl.Spec.HeadersCollection != nil {
			headerKeysToCollectHttp = append(headerKeysToCollectHttp, irl.Spec.HeadersCollection.HeaderKeys...)
		}
	}
	if len(headerKeysToCollectHttp) == 0 {
		return nil
	}

	return &odigosv1.HeadersCollectionConfig{
		HttpHeaderKeys: headerKeysToCollectHttp,
	}
}

func calculateHeadSamplingConfig(distro *distro.OtelDistro, workloadObj workload.Workload, containerName string, irls *[]odigosv1.InstrumentationRule, ignoreHealthChecks []actionsv1.IgnoreHealthChecksConfig) *odigosv1.HeadSamplingConfig {

	// only calculate head sampling config if the distro supports it
	if distro.Traces == nil || distro.Traces.HeadSampling == nil || !distro.Traces.HeadSampling.Supported {
		return nil
	}

	// only calculate head sampling config if the workload object is available
	// since we need to scrape the probes paths from the workload object
	if workloadObj == nil {
		return nil
	}

	// check if there are any rules to ignore health checks
	healthCheckFraction, headSamplingFallbackFraction := calculateHeadSamplingFractions(irls, ignoreHealthChecks)
	fallbackFractionSet := headSamplingFallbackFraction != nil && *headSamplingFallbackFraction != 1
	if healthCheckFraction == nil && !fallbackFractionSet {
		return nil
	}

	// find the probes path for this container
	// use map to avoid duplicates
	headSamplingRules := []odigosv1.AttributesAndSamplerRule{}
	if healthCheckFraction != nil {
		healthCheckPathsHttpGet := map[string]struct{}{}
		for _, container := range workloadObj.PodSpec().Containers {
			if container.Name == containerName {
				if container.LivenessProbe != nil && container.LivenessProbe.HTTPGet != nil {
					healthCheckPathsHttpGet[container.LivenessProbe.HTTPGet.Path] = struct{}{}
				}
				if container.ReadinessProbe != nil && container.ReadinessProbe.HTTPGet != nil {
					healthCheckPathsHttpGet[container.ReadinessProbe.HTTPGet.Path] = struct{}{}
				}
			}
		}

		// if there are no health check paths and fallback is not set, we have nothing to use for the head sampler
		if len(healthCheckPathsHttpGet) == 0 && !fallbackFractionSet {
			return nil
		}

		// support for old http semantic conventions in nodejs.
		// TODO: remove this once all migrates to new semantic conventions.
		urlPathAttributeKey := "url.path"
		if distro.Traces.HeadSampling.UrlPathAttributeKey != "" {
			urlPathAttributeKey = distro.Traces.HeadSampling.UrlPathAttributeKey
		}
		httpRequestMethodAttributeKey := "http.request.method"
		if distro.Traces.HeadSampling.HttpRequestMethodAttributeKey != "" {
			httpRequestMethodAttributeKey = distro.Traces.HeadSampling.HttpRequestMethodAttributeKey
		}

		for path := range healthCheckPathsHttpGet {
			headSamplingRules = append(headSamplingRules, odigosv1.AttributesAndSamplerRule{
				AttributeConditions: []odigosv1.AttributeCondition{
					{
						Key:      urlPathAttributeKey,
						Val:      path,
						Operator: odigosv1.Equals,
					},
					{
						Key:      httpRequestMethodAttributeKey,
						Val:      "GET",
						Operator: odigosv1.Equals,
					},
				},
				Fraction: *healthCheckFraction,
			})
		}
	}

	// use fallback fraction if set, otherwise use 1.0 (take all traces)
	fallbackFraction := 1.0
	if headSamplingFallbackFraction != nil {
		fallbackFraction = *headSamplingFallbackFraction
	}

	return &odigosv1.HeadSamplingConfig{
		AttributesAndSamplerRules: headSamplingRules,
		FallbackFraction:          fallbackFraction,
	}
}

// calculate the max fraction to record for health checks and the max fraction to keep for head sampling fallback from all the rules
// return nil if no rules to ignore health checks
func calculateHeadSamplingFractions(irls *[]odigosv1.InstrumentationRule, ignoreHealthChecks []actionsv1.IgnoreHealthChecksConfig) (*float64, *float64) {

	// take the max fraction to record for health checks
	var healthCheckFraction *float64
	for _, ignoreHealthCheck := range ignoreHealthChecks {
		if healthCheckFraction == nil {
			healthCheckFraction = &ignoreHealthCheck.FractionToRecord
		} else if *healthCheckFraction < ignoreHealthCheck.FractionToRecord {
			healthCheckFraction = &ignoreHealthCheck.FractionToRecord
		}
	}
	if healthCheckFraction != nil {
		*healthCheckFraction = limitFractionToRange(*healthCheckFraction)
	}

	// take the max fraction to keep for head sampling fallback
	var headSamplingFallbackFraction *float64
	for _, irl := range *irls {
		if irl.Spec.HeadSamplingFallbackFraction != nil {
			if headSamplingFallbackFraction == nil {
				headSamplingFallbackFraction = &irl.Spec.HeadSamplingFallbackFraction.FractionToKeep
			} else if *headSamplingFallbackFraction < irl.Spec.HeadSamplingFallbackFraction.FractionToKeep {
				headSamplingFallbackFraction = &irl.Spec.HeadSamplingFallbackFraction.FractionToKeep
			}
		}
	}
	if headSamplingFallbackFraction != nil {
		*headSamplingFallbackFraction = limitFractionToRange(*headSamplingFallbackFraction)
	}

	return healthCheckFraction, headSamplingFallbackFraction
}

func limitFractionToRange(fraction float64) float64 {
	if fraction < 0 {
		return 0
	} else if fraction > 1 {
		return 1
	}
	return fraction
}

func filterSpanRenamerForContainer(agentLevelActions *[]odigosv1.Action, language common.ProgrammingLanguage) *odigosv1.SpanRenamerConfig {

	spanRenamerScopeConfigs := []odigosv1.SpanRenamerScopeConfig{}
	var javaQuartzSpanRenamer *actions.SpanRenamerJavaQuartz

	for _, action := range *agentLevelActions {
		if action.Spec.SpanRenamer != nil {
			if action.Spec.SpanRenamer.Generic != nil {
				if action.Spec.SpanRenamer.Generic.ProgrammingLanguage == language {
					// there can be conflict here, where the scope name can be added multiple times
					// with same or different value.
					// currently ignored, but should be handled sometimes.
					spanRenamerScopeConfigs = append(spanRenamerScopeConfigs, odigosv1.SpanRenamerScopeConfig{
						ScopeName:        action.Spec.SpanRenamer.Generic.ScopeName,
						ConstantSpanName: action.Spec.SpanRenamer.Generic.ConstantSpanName,
					})
				}
			}
			if action.Spec.SpanRenamer.JavaQuartz != nil && language == common.JavaProgrammingLanguage {
				// notice: there can be multiple java quarts span renamer configs,
				// but we only take the last one.
				javaQuartzSpanRenamer = action.Spec.SpanRenamer.JavaQuartz
			}
		}
	}

	if len(spanRenamerScopeConfigs) == 0 && javaQuartzSpanRenamer == nil {
		return nil
	}

	return &odigosv1.SpanRenamerConfig{
		ConstantSpanNameConfigs: spanRenamerScopeConfigs,
		JavaQuartz:              javaQuartzSpanRenamer,
	}
}
