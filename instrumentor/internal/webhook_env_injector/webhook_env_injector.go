package webhookenvinjector

import (
	"context"
	"strings"

	"github.com/go-logr/logr"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	commonconsts "github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/common/envOverwrite"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/odigos-io/odigos/api/k8sconsts"
)

func InjectOdigosAgentEnvVars(ctx context.Context, logger logr.Logger, podWorkload k8sconsts.PodWorkload, container *corev1.Container,
	otelsdk common.OtelSdk, runtimeDetails *odigosv1.RuntimeDetailsByContainer, client client.Client) {

	if runtimeDetails.Language == common.JavaProgrammingLanguage && otelsdk == common.OtelSdkNativeCommunity {
		injectJavaCommunityEnvVars(ctx, logger, container, client)
	}

	envVarsPerLanguage := getEnvVarNamesForLanguage(runtimeDetails.Language)
	if envVarsPerLanguage == nil {
		return
	}

	// Odigos appends necessary environment variables to enable its agent.
	// It handles this in the following ways:
	// 1. Appends Odigos-specific values to environment variables already defined by the user in the manifest.
	// 2. Appends Odigos-specific values to environment variables already defined by the user at runtime.
	// 3. Sets environment variables with Odigos defaults when they are not defined in either the manifest or the runtime.

	isOdigosAgentEnvAppended := false
	for _, envVarName := range envVarsPerLanguage {
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
		applyOdigosEnvDefaults(container, envVarsPerLanguage, otelsdk)
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

		patchedEnvVarValue := envOverwrite.AppendOdigosAdditionsToEnvVar(envVarName, envVar.Value, valueToInject)
		envVars = append(envVars, corev1.EnvVar{Name: envVarName, Value: *patchedEnvVarValue})
	}

	return envVars
}

func applyOdigosEnvDefaults(container *corev1.Container, envVarsPerLanguage []string, otelsdk common.OtelSdk) {
	for _, envVarName := range envVarsPerLanguage {
		odigosValueForOtelSdk := envOverwrite.GetPossibleValuesPerEnv(envVarName)
		if odigosValueForOtelSdk == nil { // No Odigos values for this env var
			continue
		}

		valueToInject, ok := odigosValueForOtelSdk[otelsdk]
		if !ok { // No Odigos value for this SDK
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

func injectJavaCommunityEnvVars(ctx context.Context, logger logr.Logger,
	container *corev1.Container, client client.Client) {

	// Set the OTEL signals exporter env vars
	setOtelSignalsExporterEnvVars(ctx, logger, container, client)
}

func setOtelSignalsExporterEnvVars(ctx context.Context, logger logr.Logger,
	container *corev1.Container, client client.Client) {

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

	container.Env = append(container.Env,
		corev1.EnvVar{Name: commonconsts.OtelLogsExporter, Value: logsExporter},
		corev1.EnvVar{Name: commonconsts.OtelMetricsExporter, Value: metricsExporter},
		corev1.EnvVar{Name: commonconsts.OtelTracesExporter, Value: tracesExporter},
	)
}
