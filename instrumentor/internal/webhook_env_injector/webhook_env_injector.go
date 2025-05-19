package webhookenvinjector

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/go-logr/logr"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	commonconsts "github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/common/envOverwrite"
	"github.com/odigos-io/odigos/instrumentor/controllers/agentenabled/podswebhook"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/odigos-io/odigos/api/k8sconsts"
)

func InjectOdigosAgentEnvVars(ctx context.Context, logger logr.Logger, podWorkload k8sconsts.PodWorkload, container *corev1.Container,
	otelsdk common.OtelSdk, runtimeDetails *odigosv1.RuntimeDetailsByContainer, client client.Client, config *common.OdigosConfiguration) error {

	otelSignalExporterLanguages := []common.ProgrammingLanguage{
		common.JavaProgrammingLanguage,
		common.PhpProgrammingLanguage,
	}

	if slices.Contains(otelSignalExporterLanguages, runtimeDetails.Language) && otelsdk == common.OtelSdkNativeCommunity {
		// Set the OTEL signals exporter env vars
		setOtelSignalsExporterEnvVars(ctx, logger, container, client)
	}

	envVarsPerLanguage := getEnvVarNamesForLanguage(runtimeDetails.Language)
	if envVarsPerLanguage == nil {
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
		odigosLoaderPath := filepath.Join(k8sconsts.OdigosAgentsDirectory, commonconsts.OdigosLoaderName)

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

	avoidAddingJavaOpts := config != nil && config.AvoidInjectingJavaOptsEnvVar != nil && *config.AvoidInjectingJavaOptsEnvVar

	// Odigos appends necessary environment variables to enable its agent.
	// It handles this in the following ways:
	// 1. Appends Odigos-specific values to environment variables already defined by the user in the manifest.
	// 2. Appends Odigos-specific values to environment variables already defined by the user at runtime.
	// 3. Sets environment variables with Odigos defaults when they are not defined in either the manifest or the runtime.

	isOdigosAgentEnvAppended := false
	for _, envVarName := range envVarsPerLanguage {
		// Skip JAVA_OPTS env var if avoidAddingJavaOpts is true
		// this is a migration path - we should eventually remove this and the avoidAddingJavaOpts config
		// and never add the JAVA_OPTS env var
		if avoidAddingJavaOpts && envVarName == "JAVA_OPTS" {
			continue
		}

		// 1.
		if handleManifestEnvVar(container, envVarName, otelsdk, logger) {
			isOdigosAgentEnvAppended = true
			continue
		}

		// 2.
		if injectEnvVarsFromRuntime(logger, container, envVarName, otelsdk, runtimeDetails) {
			isOdigosAgentEnvAppended = true
			continue
		}
	}

	// 3.
	if !isOdigosAgentEnvAppended {
		applyOdigosEnvDefaults(container, envVarsPerLanguage, otelsdk, avoidAddingJavaOpts)
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

func getEnvVarNamesForLanguage(pl common.ProgrammingLanguage) []string {
	return envOverwrite.EnvVarsForLanguage[pl]
}

// Return true if further processing should be skipped, either because it was already handled or due to a potential error (e.g., missing possible values)
// Return false if the env was not processed using the manifest value and requires further handling by other methods.
func handleManifestEnvVar(container *corev1.Container, envVarName string, otelsdk common.OtelSdk, logger logr.Logger) bool {
	manifestEnvVar := getContainerEnvVarPointer(&container.Env, envVarName)
	if manifestEnvVar == nil || (manifestEnvVar.ValueFrom == nil && manifestEnvVar.Value == "") {
		return false // Not found in manifest. further process it
	}

	possibleValues := envOverwrite.GetPossibleValuesPerEnv(manifestEnvVar.Name)
	if possibleValues == nil {
		return true // Skip further processing
	}

	odigosValueForOtelSdk := possibleValues[otelsdk]

	// In case of env configured as ValueFrom [env[name].valueFrom.configMapKeyRef.key]
	// We are changing the user MY_ENV to ORIGINAL_{MY_ENV}
	// and setting MY_ENV to be ORIGINAL_MY_ENV value + Odigos additions
	if isValueFromConfigmap(manifestEnvVar) {
		handleValueFromEnvVar(container, manifestEnvVar, envVarName, odigosValueForOtelSdk)
		return true // Handled, no need for further processing
	}

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
	otelsdk common.OtelSdk, runtimeDetails *odigosv1.RuntimeDetailsByContainer) bool {
	logger.Info("Inject Odigos values based on runtime details", "envVarName", envVarName, "container", container.Name)

	if !shouldInject(runtimeDetails, logger, container.Name) {
		return false
	}

	envVarsToInject := processEnvVarsFromRuntimeDetails(runtimeDetails, envVarName, otelsdk)
	if len(envVarsToInject) > 0 {
		container.Env = append(container.Env, envVarsToInject...)
		return true
	}
	return false
}

func processEnvVarsFromRuntimeDetails(runtimeDetails *odigosv1.RuntimeDetailsByContainer, envVarName string, otelsdk common.OtelSdk) []corev1.EnvVar {
	var envVars []corev1.EnvVar

	odigosValueForOtelSdk := envOverwrite.GetPossibleValuesPerEnv(envVarName)
	if odigosValueForOtelSdk == nil { // No odigos values for this env var
		return envVars
	}
	valueToInject, ok := odigosValueForOtelSdk[otelsdk]
	if !ok { // No odigos value for this SDK
		return envVars
	}
	for _, envVar := range runtimeDetails.EnvFromContainerRuntime {

		// Get the relevant envVar that we're iterating over
		if envVar.Name != envVarName {
			continue
		}

		if envVar.Value == "" {
			// if the value is empty, treat it as it's not existing.
			// from the env appending perspective, this is the same as not having it at all
			// we want to set it to the odigos value
			// this will be done at the last step
			continue
		}

		patchedEnvVarValue := envOverwrite.AppendOdigosAdditionsToEnvVar(envVarName, envVar.Value, valueToInject)
		envVars = append(envVars, corev1.EnvVar{Name: envVarName, Value: *patchedEnvVarValue})
	}

	return envVars
}

func applyOdigosEnvDefaults(container *corev1.Container, envVarsPerLanguage []string, otelsdk common.OtelSdk, avoidAddingJavaOpts bool) {
	for _, envVarName := range envVarsPerLanguage {
		if avoidAddingJavaOpts && envVarName == "JAVA_OPTS" {
			continue
		}

		odigosValueForOtelSdk := envOverwrite.GetPossibleValuesPerEnv(envVarName)
		if odigosValueForOtelSdk == nil { // No Odigos values for this env var
			continue
		}

		valueToInject, ok := odigosValueForOtelSdk[otelsdk]
		if !ok { // No Odigos value for this SDK
			continue
		}

		existingEnv := getContainerEnvVarPointer(&container.Env, envVarName)
		if existingEnv != nil && existingEnv.ValueFrom == nil {
			if existingEnv.Value == "" {
				existingEnv.Value = valueToInject
			}
			continue
		}

		container.Env = append(container.Env, corev1.EnvVar{
			Name:  envVarName,
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

func setOtelSignalsExporterEnvVars(ctx context.Context, logger logr.Logger, container *corev1.Container, client client.Client) {
	odigosNamespace := env.GetCurrentNamespace()

	var nodeCollectorGroup odigosv1.CollectorsGroup
	err := client.Get(ctx, types.NamespacedName{
		Namespace: odigosNamespace,
		Name:      k8sconsts.OdigosNodeCollectorDaemonSetName,
	}, &nodeCollectorGroup)
	if err != nil {
		// Uses OTEL's default settings by omitting these environment variables.
		// Although the current default is "otlp," it's safer to set them explicitly
		// to avoid potential future changes and improve clarity.
		logger.Error(err, "Failed to get nodeCollectorGroup using default OTEL settings")
		return
	}

	signals := nodeCollectorGroup.Status.ReceiverSignals

	// Default values
	logsExporter := "none"
	metricsExporter := "none"
	tracesExporter := "none"

	for _, signal := range signals {
		switch signal {
		case common.LogsObservabilitySignal:
			logsExporter = "otlp"
		case common.MetricsObservabilitySignal:
			metricsExporter = "otlp"
		case common.TracesObservabilitySignal:
			tracesExporter = "otlp"
		}
	}

	// check for existing env vars so we don't introduce them again
	existingEnvNames := podswebhook.GetEnvVarNamesSet(container)
	existingEnvNames = podswebhook.InjectEnvVarToPodContainer(existingEnvNames, container, commonconsts.OtelLogsExporter, logsExporter, nil)
	existingEnvNames = podswebhook.InjectEnvVarToPodContainer(existingEnvNames, container, commonconsts.OtelMetricsExporter, metricsExporter, nil)
	podswebhook.InjectEnvVarToPodContainer(existingEnvNames, container, commonconsts.OtelTracesExporter, tracesExporter, nil)
}

func isValueFromConfigmap(envVar *corev1.EnvVar) bool {
	return envVar.ValueFrom != nil
}

func handleValueFromEnvVar(container *corev1.Container, envVar *corev1.EnvVar, originalName, odigosValue string) {
	originalNewKey := "ORIGINAL_" + envVar.Name

	combinedValue := envOverwrite.AppendOdigosAdditionsToEnvVar(originalName, fmt.Sprintf("$(%s)", originalNewKey), odigosValue)
	if combinedValue != nil {
		envVar.Name = originalNewKey
		newEnv := corev1.EnvVar{Name: originalName, Value: *combinedValue}
		container.Env = append(container.Env, newEnv)
	}
}
