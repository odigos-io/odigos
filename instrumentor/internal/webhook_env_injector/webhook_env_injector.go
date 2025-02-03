package webhookenvinjector

import (
	"strings"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/envOverwrite"
	corev1 "k8s.io/api/core/v1"

	"github.com/odigos-io/odigos/api/k8sconsts"
	v1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
)

func InjectOdigosAgentEnvVars(logger logr.Logger, podWorkload k8sconsts.PodWorkload, container *corev1.Container,
	otelsdk common.OtelSdk, runtimeDetails *v1alpha1.RuntimeDetailsByContainer) {
	envVarsPerLanguage := getEnvVarNamesForLanguage(runtimeDetails.Language)
	if envVarsPerLanguage == nil {
		return
	}

	for _, envVarName := range envVarsPerLanguage {
		if handleManifestEnvVar(container, envVarName, otelsdk, logger) {
			continue
		}

		err := injectEnvVarsFromRuntime(logger, container, envVarName, otelsdk, runtimeDetails)
		if err != nil {
			logger.Error(err, "failed to inject environment variables for container", "container", container.Name)
		}
	}
}

func getEnvVarNamesForLanguage(pl common.ProgrammingLanguage) []string {
	return envOverwrite.EnvVarsForLanguage[pl]
}

// Return true if further processing should be skipped, either because it was already handled or due to a potential error (e.g., missing possible values)
// Return false if the env was not processed using the manifest value and requires further handling by other methods.
func handleManifestEnvVar(container *corev1.Container, envVarName string, otelsdk common.OtelSdk, logger logr.Logger) bool {
	manifestEnvVar := getContainerEnvVarPointer(&container.Env, envVarName)
	if manifestEnvVar == nil {
		return false // Not found in manifest. further process it
	}

	possibleValues := envOverwrite.GetPossibleValuesPerEnv(manifestEnvVar.Name)
	if possibleValues == nil {
		return true // Skip further processing
	}

	odigosValueForOtelSdk := possibleValues[otelsdk]
	if strings.Contains(manifestEnvVar.Value, "/var/odigos/") {
		logger.Info("env var exists in the manifest and already includes odigos values, skipping injection into manifest", "envVarName", envVarName,
			"container", container.Name)
		return true // Skip further processing
	}

	updatedEnvValue := envOverwrite.AppendOdigosAdditionsToEnvVar(envVarName, manifestEnvVar.Value, odigosValueForOtelSdk)
	if updatedEnvValue != nil {
		manifestEnvVar.Value = *updatedEnvValue
		logger.Info("updated manifest environment variable", "envVarName", envVarName, "value", *updatedEnvValue)
	}
	return true // Handled, no need for further processing
}

func injectEnvVarsFromRuntime(logger logr.Logger, container *corev1.Container, envVarName string,
	otelsdk common.OtelSdk, runtimeDetails *v1alpha1.RuntimeDetailsByContainer) error {
	logger.Info("Inject Odigos values based on runtime details", "envVarName", envVarName, "container", container.Name)

	if !shouldInject(runtimeDetails, logger, container.Name) {
		return nil
	}

	envVarsToInject := processEnvVarsFromRuntimeDetails(runtimeDetails, envVarName, otelsdk)
	container.Env = append(container.Env, envVarsToInject...)
	return nil
}

func processEnvVarsFromRuntimeDetails(runtimeDetails *v1alpha1.RuntimeDetailsByContainer, envVarName string, otelsdk common.OtelSdk) []corev1.EnvVar {
	var envVars []corev1.EnvVar

	odigosValueForOtelSdk := envOverwrite.GetPossibleValuesPerEnv(envVarName)
	if odigosValueForOtelSdk == nil { // No odigos values for this env var
		return envVars
	}
	valueToInject, ok := odigosValueForOtelSdk[otelsdk]
	if !ok { // No odigos value for this SDK
		return envVars
	}

	if runtimeDetails.EnvFromContainerRuntime == nil {
		envVars = append(envVars, corev1.EnvVar{Name: envVarName, Value: valueToInject})
	} else {
		for _, envVar := range runtimeDetails.EnvFromContainerRuntime {

			// Get the relevant envVar that we're iterating over
			if envVar.Name != envVarName {
				continue
			}

			patchedEnvVarValue := envOverwrite.AppendOdigosAdditionsToEnvVar(envVarName, envVar.Value, valueToInject)
			envVars = append(envVars, corev1.EnvVar{Name: envVarName, Value: *patchedEnvVarValue})
		}
		// If EnvFromContainerRuntime does not include the relevant envVar (e.g., JAVA_OPTS), it should still be added with the Odigos value.
		if len(envVars) == 0 {
			envVars = append(envVars, corev1.EnvVar{Name: envVarName, Value: valueToInject})
		}
	}
	return envVars
}

func shouldInject(runtimeDetails *v1alpha1.RuntimeDetailsByContainer, logger logr.Logger, containerName string) bool {

	// Skip injection if runtimeDetails.RuntimeUpdateState is nil.
	// This indicates that either the new runtime detection or the new runtime detection migrator did not run for this container.
	if runtimeDetails.RuntimeUpdateState == nil {
		logger.Info("RuntimeUpdateState is nil, skipping environment variable injection", "container", containerName)
		return false
	}

	if *runtimeDetails.RuntimeUpdateState == v1alpha1.ProcessingStateFailed {
		var criErrorMessage string
		if runtimeDetails.CriErrorMessage != nil {
			criErrorMessage = *runtimeDetails.CriErrorMessage
		}
		logger.Info("CRI error message present, skipping environment variable injection", "container", containerName, "error", criErrorMessage)
		return false
	}

	// All conditions are satisfied
	return true
}

func getContainerEnvVarPointer(containerEnv *[]corev1.EnvVar, envVarName string) *corev1.EnvVar {
	for i := range *containerEnv { // Use the index to avoid creating a copy
		if (*containerEnv)[i].Name == envVarName {
			return &(*containerEnv)[i]
		}
	}
	return nil
}
