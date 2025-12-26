package signalconfig

import (
	"fmt"
	"strconv"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
)

func CalculateTracesConfig(tracesEnabled bool, effectiveConfig *common.OdigosConfiguration, containerName string, urlTemplatizationConfig *odigosv1.UrlTemplatizationConfig, irls *[]odigosv1.InstrumentationRule, workloadObj workload.Workload) (*odigosv1.AgentTracesConfig, *odigosv1.ContainerAgentConfig) {
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

	healthCheckFraction := calculateAvoidHttpFraction(irls)
	if healthCheckFraction != nil && workloadObj != nil {
		// find the probes path for this container
		// use map to avoid duplicates
		healthCheckPathsHttpGet := map[string]struct{}{}
		for _, container := range workloadObj.PodTemplateSpec().Spec.Containers {
			if container.Name == containerName {
				if container.LivenessProbe != nil && container.LivenessProbe.HTTPGet != nil {
					healthCheckPathsHttpGet[container.LivenessProbe.HTTPGet.Path] = struct{}{}
				}
				if container.ReadinessProbe != nil && container.ReadinessProbe.HTTPGet != nil {
					healthCheckPathsHttpGet[container.ReadinessProbe.HTTPGet.Path] = struct{}{}
				}
			}
		}

		headSamplingRules := make([]odigosv1.AttributesAndSamplerRule, 0, len(healthCheckPathsHttpGet))
		for path := range healthCheckPathsHttpGet {
			headSamplingRules = append(headSamplingRules, odigosv1.AttributesAndSamplerRule{
				AttributeConditions: []odigosv1.AttributeCondition{
					{
						Key:      "http.target", // this works for nodejs, if extended, need to check compatibility
						Val:      path,
						Operator: odigosv1.Equals,
					},
					{
						Key:      "http.method",
						Val:      "GET",
						Operator: odigosv1.Equals,
					},
				},
				Fraction: *healthCheckFraction,
			})
		}

		tracesConfig.HeadSampling = &odigosv1.HeadSamplingConfig{
			AttributesAndSamplerRules: headSamplingRules,
			FallbackFraction:          1.0,
		}
	}

	return tracesConfig, nil
}

// calculate the max fraction to record for health checks from all the rules
// return nil if no rules to avoid health checks
func calculateAvoidHttpFraction(irls *[]odigosv1.InstrumentationRule) *float64 {
	var healthCheckFraction *float64
	for _, irl := range *irls {
		if irl.Spec.AvoidHealthChecks != nil {
			if healthCheckFraction == nil {
				healthCheckFraction = &irl.Spec.AvoidHealthChecks.FractionToRecord
			} else if *healthCheckFraction < irl.Spec.AvoidHealthChecks.FractionToRecord {
				healthCheckFraction = &irl.Spec.AvoidHealthChecks.FractionToRecord
			}
		}
	}
	// make sure the health check fraction is in range [0, 1]
	if healthCheckFraction != nil {
		if *healthCheckFraction < 0 {
			*healthCheckFraction = 0
		} else if *healthCheckFraction > 1 {
			*healthCheckFraction = 1
		}
	}
	return healthCheckFraction
}
