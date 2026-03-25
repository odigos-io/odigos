package webhookenvinjector

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/go-logr/logr"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	commonconsts "github.com/odigos-io/odigos/common/consts"
	distroTypes "github.com/odigos-io/odigos/distros/distro"
	corev1 "k8s.io/api/core/v1"

	"github.com/odigos-io/odigos/api/k8sconsts"
)

func InjectOdigosAgentEnvVars(ctx context.Context, logger logr.Logger, container *corev1.Container,
	otelDistro *distroTypes.OtelDistro, runtimeDetails *odigosv1.RuntimeDetailsByContainer, config *common.OdigosConfiguration) error {

	appendEnvVars := otelDistro.EnvironmentVariables.AppendOdigosVariables

	if len(appendEnvVars) == 0 {
		// no env vars to inject for this language
		return nil
	}

	injectionMethod := config.AgentEnvVarsInjectionMethod
	if injectionMethod == nil {
		// we are reading the effective config which should already have the env injection method resolved or defaulted
		return errors.New("env injection method is not set in ODIGOS config")
	}

	// check if odigos loader should be used
	if *injectionMethod == common.LoaderEnvInjectionMethod || *injectionMethod == common.LoaderFallbackToPodManifestInjectionMethod {
		odigosLoaderPath := filepath.Join(k8sconsts.OdigosAgentsDirectory, commonconsts.OdigosLoaderDirName, commonconsts.OdigosLoaderName)

		manifestValExits := getContainerEnvVarPointer(&container.Env, commonconsts.LdPreloadEnvVarName) != nil
		runtimeDetailsVal, foundInInspection := getEnvVarFromRuntimeDetails(runtimeDetails, commonconsts.LdPreloadEnvVarName)
		ldPreloadUnsetOrExpected := !foundInInspection || strings.Contains(runtimeDetailsVal, odigosLoaderPath)
		secureExecution := runtimeDetails.SecureExecutionMode == nil || *runtimeDetails.SecureExecutionMode

		if !manifestValExits && ldPreloadUnsetOrExpected && !secureExecution {
			// adding to the pod manifest env var:
			// if the LD_PRELOAD env var is not already present in the manifest and the runtime details env var is not set or set to the odigos loader path.
			// the odigos loader path may be detected in the runtime details from previous installations or from terminating pods.
			container.Env = append(container.Env, corev1.EnvVar{
				Name:  commonconsts.LdPreloadEnvVarName,
				Value: odigosLoaderPath,
			})
			return nil
		}

		// the LD_PRELOAD env var is preset. for now, we don't attempt to append our value to the user defined one.
		if *injectionMethod == common.LoaderEnvInjectionMethod {
			// we're specifically requested to use the loader env var injection method
			// and the user defined LD_PRELOAD env var is already present or running in a secure execution mode.
			// so we avoid the fallback to pod manifest env var injection method
			return errors.New("loader env var injection method is requested but the LD_PRELOAD env var is already present or running in a secure execution mode")
		}

		switch {
		case manifestValExits:
			logger.Info("LD_PRELOAD env var already exists in the pod manifest, fallback to pod manifest env injection", "container", container.Name)
		case foundInInspection:
			logger.Info("LD_PRELOAD env var already exists in the runtime details, fallback to pod manifest env injection", "container", container.Name, "found value", runtimeDetailsVal)
		case secureExecution:
			logger.Info("Secure execution mode is enabled, fallback to pod manifest env injection", "container", container.Name)
		}
	}

	// from this point on, we are using the pod manifest env var injection method

	// Odigos appends necessary environment variables to enable its agent.
	// It handles this in the following ways:
	// 1. Appends Odigos-specific values to environment variables already defined by the user in the manifest.
	// 2. Appends Odigos-specific values to environment variables already defined by the user at runtime.
	// 3. Sets environment variables with Odigos defaults when they are not defined in either the manifest or the runtime.

	isOdigosAgentEnvAppended := false
	for i := range appendEnvVars {
		appendEnvVar := &appendEnvVars[i]

		// 1.
		if handleManifestEnvVar(container, appendEnvVar, logger) {
			isOdigosAgentEnvAppended = true
			continue
		}

		// 2.
		if injectEnvVarsFromRuntime(logger, container, appendEnvVar, runtimeDetails) {
			isOdigosAgentEnvAppended = true
			continue
		}
	}

	// 3.
	if !isOdigosAgentEnvAppended {
		applyOdigosEnvDefaults(container, appendEnvVars, logger)
	}

	return nil
}

func getEnvVarFromRuntimeDetails(runtimeDetails *odigosv1.RuntimeDetailsByContainer, envVarName string) (string, bool) {
	for _, envVar := range runtimeDetails.EnvVars {
		if envVar.Name == envVarName {
			return envVar.Value, true
		}
	}
	return "", false
}

// Return true if further processing should be skipped, either because it was already handled or due to a potential error (e.g., missing possible values)
// Return false if the env was not processed using the manifest value and requires further handling by other methods.
func handleManifestEnvVar(container *corev1.Container, appendEnvVar *distroTypes.AppendOdigosEnvironmentVariable, logger logr.Logger) bool {
	envVarName := appendEnvVar.EnvName
	manifestEnvVar := getContainerEnvVarPointer(&container.Env, envVarName)
	if manifestEnvVar == nil || (manifestEnvVar.ValueFrom == nil && manifestEnvVar.Value == "") {
		return false // Not found in manifest. further process it
	}

	// In case of env configured as ValueFrom [env[name].valueFrom.configMapKeyRef.key]
	// We are changing the user MY_ENV to ORIGINAL_{MY_ENV}
	// and setting MY_ENV to be ORIGINAL_MY_ENV value + Odigos additions
	if isValueFromConfigmap(manifestEnvVar) {
		handleValueFromEnvVar(container, manifestEnvVar, appendEnvVar, logger)
		return true // Handled, no need for further processing
	}

	if strings.Contains(manifestEnvVar.Value, "/var/odigos/") {
		logger.Info("env var exists in the manifest and already includes odigos values, skipping injection into manifest", "envVarName", envVarName,
			"container", container.Name)
		return true // Skip further processing
	}

	updatedEnvValue := distroTypes.EvaluateReplacePattern(appendEnvVar.ReplacePattern, manifestEnvVar.Value, k8sconsts.OdigosAgentsDirectory)
	manifestEnvVar.Value = updatedEnvValue
	return true // Handled, no need for further processing
}

func injectEnvVarsFromRuntime(logger logr.Logger, container *corev1.Container, appendEnvVar *distroTypes.AppendOdigosEnvironmentVariable,
	runtimeDetails *odigosv1.RuntimeDetailsByContainer) bool {

	if !shouldInject(runtimeDetails, logger, container.Name) {
		return false
	}

	envVarsToInject := processEnvVarsFromRuntimeDetails(runtimeDetails, appendEnvVar, logger)
	if len(envVarsToInject) > 0 {
		container.Env = append(container.Env, envVarsToInject...)
		return true
	}

	return false
}

func processEnvVarsFromRuntimeDetails(runtimeDetails *odigosv1.RuntimeDetailsByContainer, appendEnvVar *distroTypes.AppendOdigosEnvironmentVariable, logger logr.Logger) []corev1.EnvVar {
	var envVars []corev1.EnvVar
	envVarName := appendEnvVar.EnvName

	for _, envVar := range runtimeDetails.EnvFromContainerRuntime {
		if envVar.Name != envVarName {
			continue
		}

		if envVar.Value == "" {
			// if the value is empty, treat it as it's not existing.
			// from the env appending perspective, this is the same as not having it at all
			// we want to set it to the odigos value
			// this will be done at the last step
			logger.Info("[DEBUG] env var found in runtime but value is empty, will fall through to default", "envName", envVarName)
			continue
		}

		patchedEnvVarValue := distroTypes.EvaluateReplacePattern(appendEnvVar.ReplacePattern, envVar.Value, k8sconsts.OdigosAgentsDirectory)
		logger.Info("[DEBUG] patched env var from runtime", "envName", envVarName, "originalValue", envVar.Value, "patchedValue", patchedEnvVarValue)
		envVars = append(envVars, corev1.EnvVar{Name: envVarName, Value: patchedEnvVarValue})
	}

	return envVars
}

func applyOdigosEnvDefaults(container *corev1.Container, appendEnvVars []distroTypes.AppendOdigosEnvironmentVariable, logger logr.Logger) {
	for i := range appendEnvVars {
		appendEnvVar := &appendEnvVars[i]
		// EvaluateReplacePattern with empty originalValue strips the placeholder and any
		// adjacent delimiter, giving the pure odigos default value.
		valueToInject := distroTypes.EvaluateReplacePattern(appendEnvVar.ReplacePattern, "", k8sconsts.OdigosAgentsDirectory)

		existingEnv := getContainerEnvVarPointer(&container.Env, appendEnvVar.EnvName)
		if existingEnv != nil && existingEnv.ValueFrom == nil {
			if existingEnv.Value == "" {
				existingEnv.Value = valueToInject
			}
			continue
		}

		container.Env = append(container.Env, corev1.EnvVar{
			Name:  appendEnvVar.EnvName,
			Value: valueToInject,
		})
	}
}

func shouldInject(runtimeDetails *odigosv1.RuntimeDetailsByContainer, logger logr.Logger, containerName string) bool {

	// Skip injection if runtimeDetails.RuntimeUpdateState is nil.
	// This indicates that either the new runtime detection or the new runtime detection migrator did not run for this container.
	if runtimeDetails.RuntimeUpdateState == nil {
		logger.Info("RuntimeUpdateState is nil, skipping environment variable injection", "container", containerName)
		return false
	}

	if *runtimeDetails.RuntimeUpdateState == odigosv1.ProcessingStateFailed {
		var criErrorMessage string
		if runtimeDetails.CriErrorMessage != nil {
			criErrorMessage = *runtimeDetails.CriErrorMessage
		}
		logger.Info("CRI error message present, skipping environment variable injection", "container", containerName, "message", criErrorMessage)
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

func isValueFromConfigmap(envVar *corev1.EnvVar) bool {
	return envVar.ValueFrom != nil
}

// handleValueFromEnvVar deals with env vars that use valueFrom (e.g. ConfigMap/Secret reference).
// Since we can't append to a valueFrom directly, we rename the original env var to ORIGINAL_<name>
// (keeping its valueFrom intact) and create a new plain env var that combines the renamed reference
// with the odigos agent value via k8s variable expansion: $(ORIGINAL_<name>) + odigos addition.
func handleValueFromEnvVar(container *corev1.Container, envVar *corev1.EnvVar, appendEnvVar *distroTypes.AppendOdigosEnvironmentVariable, logger logr.Logger) {
	originalNewKey := "ORIGINAL_" + envVar.Name

	combinedValue := distroTypes.EvaluateReplacePattern(appendEnvVar.ReplacePattern, fmt.Sprintf("$(%s)", originalNewKey), k8sconsts.OdigosAgentsDirectory)
	envVar.Name = originalNewKey
	newEnv := corev1.EnvVar{Name: appendEnvVar.EnvName, Value: combinedValue}
	container.Env = append(container.Env, newEnv)
}
