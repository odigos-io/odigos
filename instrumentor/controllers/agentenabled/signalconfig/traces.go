package signalconfig

import (
	"fmt"
	"strconv"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
)

func CalculateTracesConfig(tracesEnabled bool, effectiveConfig *common.OdigosConfiguration, containerName string, templateRules *[]odigosv1.TemplateRuleConfig) (*odigosv1.AgentTracesConfig, *odigosv1.ContainerAgentConfig) {
	fmt.Printf("CalculateTracesConfig - tracesEnabled: %v, containerName: %s, templateRules count: %d\n", tracesEnabled, containerName, len(*templateRules))
	if !tracesEnabled {
		fmt.Printf("CalculateTracesConfig - traces disabled, returning nil\n")
		return nil, nil
	}

	tracesConfig := &odigosv1.AgentTracesConfig{}
	fmt.Printf("CalculateTracesConfig - created empty tracesConfig: %+v\n", tracesConfig)

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

	fmt.Printf("CalculateTracesConfig - processing %d template rules\n", len(*templateRules))
	for i, templateRule := range *templateRules {
		fmt.Printf("CalculateTracesConfig - adding template rule %d: %+v\n", i, templateRule)
		tracesConfig.TemplateRules = append(tracesConfig.TemplateRules, templateRule)
	}

	fmt.Printf("CalculateTracesConfig - final tracesConfig: %+v\n", tracesConfig)
	fmt.Printf("CalculateTracesConfig - final tracesConfig.TemplateRules: %+v\n", tracesConfig.TemplateRules)
	return tracesConfig, nil
}
