package webhookenvinjector

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/envOverwrite"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
)

func InjectOdigosAgentEnvVars(ctx context.Context, p client.Client, logger logr.Logger, podWorkload workload.PodWorkload, container *corev1.Container,
	pl common.ProgrammingLanguage, otelsdk common.OtelSdk) {
	envVarsPerLanguage := getEnvVarsForLanguage(pl)
	if envVarsPerLanguage == nil {
		return
	}

	for _, envVarName := range envVarsPerLanguage {
		if handleManifestEnvVar(container, envVarName, otelsdk, logger) {
			continue
		}

		err := injectEnvVarsFromRuntime(ctx, p, logger, podWorkload, container, envVarName, otelsdk)
		if err != nil {
			logger.Error(err, "failed to inject environment variables for container", "container", container.Name)
		}
	}
}

func getEnvVarsForLanguage(pl common.ProgrammingLanguage) []string {
	// Check if the key exists in the map - for safety
	if envVars, exists := envOverwrite.EnvVarsForLanguage[pl]; exists {
		return envVars
	}

	// Return nil if the key doesn't exist
	return nil
}

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
		logger.Info("env var exists in the manifest and already includes odigos values", "envVarName", envVarName)
		return true // Skip further processing
	}

	updatedEnvValue := updatedGetPatchedEnvValue(envVarName, manifestEnvVar.Value, odigosValueForOtelSdk)
	if updatedEnvValue != nil {
		manifestEnvVar.Value = *updatedEnvValue
		logger.Info("updated manifest environment variable", "envVarName", envVarName, "value", *updatedEnvValue)
	}
	return true // Handled, no need for further processing
}

func injectEnvVarsFromRuntime(ctx context.Context, p client.Client, logger logr.Logger, podWorkload workload.PodWorkload,
	container *corev1.Container, envVarName string, otelsdk common.OtelSdk) error {

	var workloadInstrumentationConfig v1alpha1.InstrumentationConfig
	instrumentationConfigName := workload.CalculateWorkloadRuntimeObjectName(podWorkload.Name, podWorkload.Kind)

	if err := p.Get(ctx, client.ObjectKey{Namespace: podWorkload.Namespace, Name: instrumentationConfigName}, &workloadInstrumentationConfig); err != nil {
		return fmt.Errorf("failed to get instrumentationConfig: %w", err)
	}

	runtimeDetails := workloadInstrumentationConfig.Status.GetRuntimeDetailsForContainer(*container)
	if runtimeDetails == nil {
		logger.Error(nil, "failed to get runtime details for container", "container", container.Name)
		return nil
	}

	if !shouldInject(runtimeDetails, logger, container.Name) {
		return nil
	}

	envVarsToInject := prepareEnvVars(runtimeDetails, envVarName, otelsdk)
	container.Env = append(container.Env, envVarsToInject...)
	return nil
}

func prepareEnvVars(runtimeDetails *v1alpha1.RuntimeDetailsByContainer, envVarName string, otelsdk common.OtelSdk) []corev1.EnvVar {
	var envVars []corev1.EnvVar
	if runtimeDetails.EnvFromContainerRuntime == nil {
		odigosValueForOtelSdk := envOverwrite.GetPossibleValuesPerEnv(envVarName)
		if odigosValueForOtelSdk != nil {
			valueToInject := odigosValueForOtelSdk[otelsdk]
			patchedEnvVarValue := updatedGetPatchedEnvValue(envVarName, "", valueToInject) // empty observedValue
			envVars = append(envVars, corev1.EnvVar{Name: envVarName, Value: *patchedEnvVarValue})
		}

	} else {
		for _, envVar := range runtimeDetails.EnvFromContainerRuntime {
			// Get the relevant envVar that we're iterating over
			if envVar.Name != envVarName {
				continue
			}
			odigosValueForOtelSdk := envOverwrite.GetPossibleValuesPerEnv(envVarName)
			if odigosValueForOtelSdk != nil {
				valueToInject := odigosValueForOtelSdk[otelsdk]
				patchedEnvVarValue := updatedGetPatchedEnvValue(envVarName, envVar.Value, valueToInject)
				envVars = append(envVars, corev1.EnvVar{Name: envVarName, Value: *patchedEnvVarValue})
			}
		}
	}
	return envVars
}

func updatedGetPatchedEnvValue(envName string, observedValue string, desiredOdigosAddition string) *string {
	_, ok := envOverwrite.EnvValuesMap[envName]
	if !ok {
		// Odigos does not manipulate this environment variable, so ignore it
		return nil
	}

	// In case observedValue is exists but empty, we just need to set the desiredOdigosAddition without delim before
	if observedValue == "" {
		return &desiredOdigosAddition
	} else {
		// In case observedValue is not empty, we need to append the desiredOdigosAddition with the delim
		delim := envOverwrite.GetDelimPerEnv(envName)
		if delim == nil { // for safety
			return nil
		}

		mergedEnvValue := observedValue + *delim + desiredOdigosAddition
		return &mergedEnvValue
	}
}

func shouldInject(runtimeDetails *v1alpha1.RuntimeDetailsByContainer, logger logr.Logger, containerName string) bool {
	if runtimeDetails.CriErrorMessage != nil {
		logger.Info("CRI error message present, skipping environment variable injection", "container", containerName, "error", *runtimeDetails.CriErrorMessage)
		return false
	}

	if runtimeDetails.RuntimeUpdateState == nil {
		logger.Info("RuntimeUpdateState is nil, skipping environment variable injection", "container", containerName)
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
