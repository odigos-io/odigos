package signalconfig

import (
	"fmt"
	"strconv"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
	"github.com/odigos-io/odigos/common"
	commonapi "github.com/odigos-io/odigos/common/api"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/odigos-io/odigos/instrumentor/controllers/agentenabled/sampling"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
)

func CalculateTracesConfig(
	tracesEnabled bool,
	effectiveConfig *common.OdigosConfiguration,
	containerName string,
	programmingLanguage common.ProgrammingLanguage,
	urlTemplatizationConfig *commonapi.UrlTemplatizationConfig,
	irls *[]odigosv1.InstrumentationRule,
	agentLevelActions *[]odigosv1.Action,
	samplingRules *[]odigosv1.Sampling,
	workloadObj workload.Workload,
	pw k8sconsts.PodWorkload,
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
	tracesConfig.HeadSampling = sampling.CalculateHeadSamplingConfig(distro, workloadObj, containerName, effectiveConfig, samplingRules, pw)
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

func filterSpanRenamerForContainer(agentLevelActions *[]odigosv1.Action, language common.ProgrammingLanguage) *odigosv1.SpanRenamerConfig {

	gotRenamingConfig := false
	scopeRulesMap := map[string]odigosv1.SpanRenamerScopeRules{}

	for _, action := range *agentLevelActions {
		if action.Spec.SpanRenamer != nil {
			if action.Spec.SpanRenamer.ProgrammingLanguage != language {
				continue
			}
			scopeName := action.Spec.SpanRenamer.ScopeName
			for _, scopeRule := range action.Spec.SpanRenamer.RegexReplacements {
				if existing, ok := scopeRulesMap[scopeName]; ok {
					existing.RegexReplacements = append(existing.RegexReplacements, scopeRule)
					scopeRulesMap[scopeName] = existing
				} else {
					scopeRulesMap[scopeName] = odigosv1.SpanRenamerScopeRules{
						ScopeName:         scopeName,
						RegexReplacements: []actions.SpanRenamerRegexReplacement{scopeRule},
					}
				}
				gotRenamingConfig = true
			}
		}
	}

	if !gotRenamingConfig {
		return nil
	}

	scopeRules := []odigosv1.SpanRenamerScopeRules{}
	for _, scopeRule := range scopeRulesMap {
		scopeRules = append(scopeRules, scopeRule)
	}
	return &odigosv1.SpanRenamerConfig{
		ScopeRules: scopeRules,
	}
}
