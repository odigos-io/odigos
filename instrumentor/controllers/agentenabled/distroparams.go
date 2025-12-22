package agentenabled

import (
	"fmt"

	"github.com/hashicorp/go-version"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	distroTypes "github.com/odigos-io/odigos/distros/distro"
)

// DistroParams is a map of string keys to string values,
// Which allows the reconcile to pass parameters to the distro,
// used in webhook to make the injection as a parameterized operation.
//
// They capture info from runtime detection, verify it's validation and existence
// and then populate it in the container agent config.
// It allows the webhook to be simpler, only relying on processed, transactional data,
// that can be directly used in the injection logic.

type DistroParam = map[string]string

func addLibcDistroParamFromRuntimeDetails(params DistroParam, distroName string, runtimeDetails *odigosv1.RuntimeDetailsByContainer) (err *odigosv1.ContainerAgentConfig) {
	if runtimeDetails.LibCType == nil {
		return &odigosv1.ContainerAgentConfig{
			ContainerName:       runtimeDetails.ContainerName,
			AgentEnabled:        false,
			AgentEnabledReason:  odigosv1.AgentEnabledReasonMissingDistroParameter,
			AgentEnabledMessage: fmt.Sprintf("failed to detect libc type for opentelemetry distribution '%s' which is required for instrumentation", distroName),
		}
	}
	params[common.LibcTypeDistroParameterName] = string(*runtimeDetails.LibCType)

	return nil
}

func addRuntimeVersionMajorMinorDistroParamFromRuntimeDetails(params DistroParam, distroName string, runtimeDetails *odigosv1.RuntimeDetailsByContainer) *odigosv1.ContainerAgentConfig {
	if runtimeDetails.RuntimeVersion == "" {
		return &odigosv1.ContainerAgentConfig{
			ContainerName:       runtimeDetails.ContainerName,
			AgentEnabled:        false,
			AgentEnabledReason:  odigosv1.AgentEnabledReasonMissingDistroParameter,
			AgentEnabledMessage: fmt.Sprintf("failed to detect runtime version for opentelemetry distribution '%s' which is required for instrumentation", distroName),
		}
	}
	version, err := version.NewVersion(runtimeDetails.RuntimeVersion)
	if err != nil {
		return &odigosv1.ContainerAgentConfig{
			ContainerName:       runtimeDetails.ContainerName,
			AgentEnabled:        false,
			AgentEnabledReason:  odigosv1.AgentEnabledReasonMissingDistroParameter,
			AgentEnabledMessage: fmt.Sprintf("failed to parse runtime version from detection: %s", runtimeDetails.RuntimeVersion),
		}
	}
	versionAsMajorMinor, err := common.MajorMinorStringOnly(version)
	if err != nil {
		return &odigosv1.ContainerAgentConfig{
			ContainerName:       runtimeDetails.ContainerName,
			AgentEnabled:        false,
			AgentEnabledReason:  odigosv1.AgentEnabledReasonMissingDistroParameter,
			AgentEnabledMessage: fmt.Sprintf("failed to parse runtime version as major.minor: %s", runtimeDetails.RuntimeVersion),
		}
	}
	params[distroTypes.RuntimeVersionMajorMinorDistroParameterName] = versionAsMajorMinor

	return nil
}

func processSingleRequiredParameter(existingParams DistroParam, distro *distroTypes.OtelDistro, runtimeDetails *odigosv1.RuntimeDetailsByContainer, parameterName string) (err *odigosv1.ContainerAgentConfig) {
	switch parameterName {
	case common.LibcTypeDistroParameterName:
		return addLibcDistroParamFromRuntimeDetails(existingParams, distro.Name, runtimeDetails)
	case distroTypes.RuntimeVersionMajorMinorDistroParameterName:
		return addRuntimeVersionMajorMinorDistroParamFromRuntimeDetails(existingParams, distro.Name, runtimeDetails)
	default:
		return &odigosv1.ContainerAgentConfig{
			ContainerName:       runtimeDetails.ContainerName,
			AgentEnabled:        false,
			AgentEnabledReason:  odigosv1.AgentEnabledReasonMissingDistroParameter,
			AgentEnabledMessage: fmt.Sprintf("unsupported parameter '%s' for distro '%s'", parameterName, distro.Name),
		}
	}
}

func calculateRequiredParameters(distro *distroTypes.OtelDistro, runtimeDetails *odigosv1.RuntimeDetailsByContainer) (requiredParams DistroParam, err *odigosv1.ContainerAgentConfig) {
	requiredParams = DistroParam{}
	for _, parameterName := range distro.RequireParameters {
		err := processSingleRequiredParameter(requiredParams, distro, runtimeDetails, parameterName)
		if err != nil {
			return requiredParams, err
		}
	}
	return requiredParams, nil
}

func calculateAppendEnvParameters(distro *distroTypes.OtelDistro, runtimeDetails *odigosv1.RuntimeDetailsByContainer) (appendEnvParams DistroParam, err *odigosv1.ContainerAgentConfig) {
	if runtimeDetails.CriErrorMessage != nil {
		return appendEnvParams, &odigosv1.ContainerAgentConfig{
			ContainerName:       runtimeDetails.ContainerName,
			AgentEnabled:        false,
			AgentEnabledReason:  odigosv1.AgentEnabledReasonMissingDistroParameter,
			AgentEnabledMessage: fmt.Sprintf("failed to detect environment variables from container runtime: %s", *runtimeDetails.CriErrorMessage),
		}
	}

	// any "append env var" from distro manifest that is found in the runtime details,
	// is added here as a distro parameter with the same name and the value from the container runtime env vars
	appendEnvParams = DistroParam{}
	for _, envVar := range distro.EnvironmentVariables.AppendOdigosVariables {
		envName := envVar.EnvName
		criValue, ok := getEnvVarFromList(runtimeDetails.EnvFromContainerRuntime, envName)
		if ok && criValue != "" {
			appendEnvParams[envName] = criValue
		}
	}
	return appendEnvParams, nil
}

func calculateDistroParams(distro *distroTypes.OtelDistro, runtimeDetails *odigosv1.RuntimeDetailsByContainer, envInjectionMethod *common.EnvInjectionDecision) (distroParams DistroParam, err *odigosv1.ContainerAgentConfig) {
	distroParams = DistroParam{}

	if len(distro.RequireParameters) > 0 {
		distroParams, err = calculateRequiredParameters(distro, runtimeDetails)
		if err != nil {
			return distroParams, err
		}
	}

	envInjectionMethodIsPodManifest := envInjectionMethod != nil && *envInjectionMethod == common.EnvInjectionDecisionPodManifest
	if envInjectionMethodIsPodManifest && len(distro.EnvironmentVariables.AppendOdigosVariables) > 0 {
		appendEnvParams, err := calculateAppendEnvParameters(distro, runtimeDetails)
		if err != nil {
			return distroParams, err
		}
		distroParams = mergeMaps(distroParams, appendEnvParams)
	}

	// If result is empty, preserve the existing value's nil/empty state to avoid unnecessary diffs
	if len(distroParams) == 0 {
		return nil, nil
	}

	return distroParams, nil
}
