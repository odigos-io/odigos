package sampling

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func percentageToFractionOrZero(percentage *float64) float64 {
	if percentage == nil {
		return 0
	}
	return percentageToFraction(*percentage)
}

func percentageToFraction(percentage float64) float64 {
	if percentage < 0 {
		return 0
	} else if percentage > 100 {
		return 1
	}
	return percentage / 100.0
}

func calculateHeadSamplingFraction(effectiveConfig *common.OdigosConfiguration) float64 {
	if effectiveConfig.Sampling == nil {
		return 0.0 // default if unset.
	} else if effectiveConfig.Sampling.K8sHealthProbesSampling == nil {
		return 0.0 // default if unset.
	} else if effectiveConfig.Sampling.K8sHealthProbesSampling.KeepPercentage == nil {
		return 0.0 // default if unset.
	}
	return percentageToFraction(*effectiveConfig.Sampling.K8sHealthProbesSampling.KeepPercentage)
}

func calculateKubeletHttpGetProbePaths(workloadObj workload.Workload, containerName string) map[string]struct{} {
	healthCheckPathsHttpGet := map[string]struct{}{}
	for _, container := range workloadObj.PodSpec().Containers {
		if container.Name == containerName {
			if container.StartupProbe != nil && container.StartupProbe.HTTPGet != nil {
				healthCheckPathsHttpGet[container.StartupProbe.HTTPGet.Path] = struct{}{}
			}
			if container.LivenessProbe != nil && container.LivenessProbe.HTTPGet != nil {
				healthCheckPathsHttpGet[container.LivenessProbe.HTTPGet.Path] = struct{}{}
			}
			if container.ReadinessProbe != nil && container.ReadinessProbe.HTTPGet != nil {
				healthCheckPathsHttpGet[container.ReadinessProbe.HTTPGet.Path] = struct{}{}
			}
		}
	}
	return healthCheckPathsHttpGet
}

func getDistroPathAndMethodAttributeKeys(distro *distro.OtelDistro) (attribute.Key, attribute.Key, attribute.Key) {
	urlPathAttributeKey := semconv.URLPathKey
	if distro.Traces.HeadSampling.UrlPathAttributeKey != "" {
		urlPathAttributeKey = attribute.Key(distro.Traces.HeadSampling.UrlPathAttributeKey)
	}
	httpRequestMethodAttributeKey := semconv.HTTPRequestMethodKey
	if distro.Traces.HeadSampling.HttpRequestMethodAttributeKey != "" {
		httpRequestMethodAttributeKey = attribute.Key(distro.Traces.HeadSampling.HttpRequestMethodAttributeKey)
	}
	serverAddressAttributeKey := semconv.ServerAddressKey
	if distro.Traces.HeadSampling.ServerAddressAttributeKey != "" {
		serverAddressAttributeKey = attribute.Key(distro.Traces.HeadSampling.ServerAddressAttributeKey)
	}
	return urlPathAttributeKey, httpRequestMethodAttributeKey, serverAddressAttributeKey
}

func calculateKubeletHttpGetProbeAttributeConditions(distro *distro.OtelDistro, healthCheckPathsHttpGet map[string]struct{}, fraction float64) []odigosv1.AttributesAndSamplerRule {
	urlPathAttributeKey, httpRequestMethodAttributeKey, _ := getDistroPathAndMethodAttributeKeys(distro)
	attributesAndSamplerRules := make([]odigosv1.AttributesAndSamplerRule, 0, len(healthCheckPathsHttpGet))
	for path := range healthCheckPathsHttpGet {
		attributesAndSamplerRules = append(attributesAndSamplerRules, odigosv1.AttributesAndSamplerRule{
			AttributeConditions: []odigosv1.AttributeCondition{
				{
					Key:      string(urlPathAttributeKey),
					Val:      path,
					Operator: odigosv1.Equals,
				},
				{
					Key:      string(httpRequestMethodAttributeKey),
					Val:      "GET", // Since this is comming from the HTTPGet probe config, it will be invoked as GET.
					Operator: odigosv1.Equals,
				},
			},
			Fraction: fraction,
		})
	}
	return attributesAndSamplerRules
}

func calculateKubeletHealthProbesSamplingRules(effectiveConfig *common.OdigosConfiguration, distro *distro.OtelDistro, workloadObj workload.Workload, containerName string) []odigosv1.AttributesAndSamplerRule {

	// only add health probe sampling rules when explicitly enabled
	if effectiveConfig.Sampling == nil || effectiveConfig.Sampling.K8sHealthProbesSampling == nil || effectiveConfig.Sampling.K8sHealthProbesSampling.Enabled == nil || !*effectiveConfig.Sampling.K8sHealthProbesSampling.Enabled {
		return nil
	}

	if workloadObj == nil {
		return nil
	}

	healthCheckPathsHttpGet := calculateKubeletHttpGetProbePaths(workloadObj, containerName)
	if len(healthCheckPathsHttpGet) == 0 {
		return nil
	}

	fraction := calculateHeadSamplingFraction(effectiveConfig)
	kubeletealthProbesConditions := calculateKubeletHttpGetProbeAttributeConditions(distro, healthCheckPathsHttpGet, fraction)

	return kubeletealthProbesConditions
}

// givin a specific container in a workload, matched to a distro, calculate it's head sampling based on odigos config and sampling rules.
func CalculateHeadSamplingConfig(distro *distro.OtelDistro, workloadObj workload.Workload, containerName string, effectiveConfig *common.OdigosConfiguration, samplingRules *[]odigosv1.Sampling, pw k8sconsts.PodWorkload) *odigosv1.HeadSamplingConfig {

	// only calculate head sampling config if the distro supports it
	if distro.Traces == nil || distro.Traces.HeadSampling == nil || !distro.Traces.HeadSampling.Supported {
		return nil
	}

	kubeletHealthProbesRules := calculateKubeletHealthProbesSamplingRules(effectiveConfig, distro, workloadObj, containerName)
	customSamplingRules, ambientFraction := convertSamplingRulesToHeadSamplingConfig(samplingRules, pw, containerName, distro)

	// if no rules are found, disable the head sampling (unused)
	if len(customSamplingRules) == 0 && len(kubeletHealthProbesRules) == 0 {
		return nil
	}

	attributesAndSamplerRules := append(kubeletHealthProbesRules, customSamplingRules...)

	return &odigosv1.HeadSamplingConfig{
		AttributesAndSamplerRules: attributesAndSamplerRules,
		FallbackFraction:          ambientFraction,
	}
}

func convertNoisyOperationToAttributeConditions(noisyOperation odigosv1.NoisyOperations, distro *distro.OtelDistro) []odigosv1.AttributeCondition {
	if noisyOperation.Operation != nil && noisyOperation.Operation.HttpServer != nil {
		return convertNoisyOperationHttpServerToAttributeConditions(noisyOperation.Operation.HttpServer, distro)
	}
	if noisyOperation.Operation != nil && noisyOperation.Operation.HttpClient != nil {
		return convertNoisyOperationHttpClientToAttributeConditions(noisyOperation.Operation.HttpClient, distro)
	}
	return nil
}

func convertNoisyOperationHttpServerToAttributeConditions(httpServer *odigosv1.HeadSamplingHttpServerOperationMatcher, distro *distro.OtelDistro) []odigosv1.AttributeCondition {
	urlPathAttributeKey, httpRequestMethodAttributeKey, _ := getDistroPathAndMethodAttributeKeys(distro)
	andConditions := []odigosv1.AttributeCondition{}
	if httpServer.Route != "" {
		andConditions = append(andConditions, odigosv1.AttributeCondition{
			Key:      string(urlPathAttributeKey),
			Val:      httpServer.Route,
			Operator: odigosv1.Equals,
		})
	}
	if httpServer.RoutePrefix != "" {
		andConditions = append(andConditions, odigosv1.AttributeCondition{
			Key:      string(urlPathAttributeKey),
			Val:      httpServer.RoutePrefix,
			Operator: odigosv1.StartWith,
		})
	}
	if httpServer.Method != "" {
		andConditions = append(andConditions, odigosv1.AttributeCondition{
			Key:      string(httpRequestMethodAttributeKey),
			Val:      httpServer.Method,
			Operator: odigosv1.Equals,
		})
	}
	return andConditions
}

func convertNoisyOperationHttpClientToAttributeConditions(httpClient *odigosv1.HeadSamplingHttpClientOperationMatcher, distro *distro.OtelDistro) []odigosv1.AttributeCondition {

	urlPathAttributeKey, httpRequestMethodAttributeKey, serverAddressAttributeKey := getDistroPathAndMethodAttributeKeys(distro)

	andConditions := []odigosv1.AttributeCondition{}
	if httpClient.ServerAddress != "" {
		andConditions = append(andConditions, odigosv1.AttributeCondition{
			Key:      string(serverAddressAttributeKey),
			Val:      httpClient.ServerAddress,
			Operator: odigosv1.Equals,
		})
	}
	if httpClient.UrlPath != "" {
		andConditions = append(andConditions, odigosv1.AttributeCondition{
			Key:      string(urlPathAttributeKey),
			Val:      httpClient.UrlPath,
			Operator: odigosv1.Equals,
		})
	}
	if httpClient.Method != "" {
		andConditions = append(andConditions, odigosv1.AttributeCondition{
			Key:      string(httpRequestMethodAttributeKey),
			Val:      httpClient.Method,
			Operator: odigosv1.Equals,
		})
	}
	return andConditions
}

func convertSamplingRulesToHeadSamplingConfig(samplingRules *[]odigosv1.Sampling, pw k8sconsts.PodWorkload, containerName string, distro *distro.OtelDistro) ([]odigosv1.AttributesAndSamplerRule, float64) {
	headSamplingRules := []odigosv1.AttributesAndSamplerRule{}

	// ambient fraction is the fraction of spans that are dropped by head sampling regardless of any span properties.
	// users can set a rule for noisy operations and not specify any attribute for matching, in this case,
	// the fraction is used as ambient and applied to all spans.
	// while not encouraged, this behaviour is consistent with the matching semantics of the attributes and sampler rules
	// and useful if a service goes wild and needs to be limited aggressively.
	ambientFraction := 1.0
	for _, samplingRule := range *samplingRules {

		if samplingRule.Spec.Disabled {
			continue
		}

		for _, noisyOperation := range samplingRule.Spec.NoisyOperations {

			// only take into account operations that are relevant to the current source and container
			if !IsServiceInRuleScope(noisyOperation.SourceScopes, pw, containerName, distro.Language) {
				continue
			}

			fraction := percentageToFractionOrZero(noisyOperation.PercentageAtMost)
			attributeConditions := convertNoisyOperationToAttributeConditions(noisyOperation, distro)
			if len(attributeConditions) == 0 {
				if fraction < ambientFraction {
					ambientFraction = fraction
				}
			} else {
				headSamplingRules = append(headSamplingRules, odigosv1.AttributesAndSamplerRule{
					AttributeConditions: attributeConditions,
					Fraction:            fraction,
				})
			}
		}
	}
	return headSamplingRules, ambientFraction
}
