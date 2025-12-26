package signalconfig

import (
	"fmt"
	"strconv"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
)

func CalculateTracesConfig(tracesEnabled bool, effectiveConfig *common.OdigosConfiguration, containerName string, urlTemplatizationConfig *odigosv1.UrlTemplatizationConfig, irls *[]odigosv1.InstrumentationRule) (*odigosv1.AgentTracesConfig, *odigosv1.ContainerAgentConfig) {
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

	// http headers collection configuration
	headerKeysToCollectHttp := []string{}
	for _, irl := range *irls {
		if irl.Spec.HeadersCollection != nil {
			headerKeysToCollectHttp = append(headerKeysToCollectHttp, irl.Spec.HeadersCollection.HeaderKeys...)
		}
	}
	if len(headerKeysToCollectHttp) > 0 {
		tracesConfig.HeadersCollection = &odigosv1.HeadersCollectionConfig{
			HttpHeaderKeys: headerKeysToCollectHttp,
		}
	}

	return tracesConfig, nil
}
