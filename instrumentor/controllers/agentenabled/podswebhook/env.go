package podswebhook

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	commonconsts "github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/odigos-io/odigos/k8sutils/pkg/service"
	corev1 "k8s.io/api/core/v1"
)

type EnvVarNamesMap map[string]struct{}

func injectEnvVarObjectFieldRefToPodContainer(existingEnvNames EnvVarNamesMap, container *corev1.Container, envVarName, envVarRef string) EnvVarNamesMap {
	if _, exists := (existingEnvNames)[envVarName]; exists {
		return existingEnvNames
	}

	container.Env = append(container.Env, corev1.EnvVar{
		Name: envVarName,
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: envVarRef,
			},
		},
	})
	existingEnvNames[envVarName] = struct{}{}
	return existingEnvNames
}

func InjectConstEnvVarToPodContainer(existingEnvNames EnvVarNamesMap, container *corev1.Container, envVarName, envVarValue string) EnvVarNamesMap {
	if _, exists := existingEnvNames[envVarName]; exists {
		return existingEnvNames
	}
	container.Env = append(container.Env, corev1.EnvVar{
		Name:  envVarName,
		Value: envVarValue,
	})
	existingEnvNames[envVarName] = struct{}{}
	return existingEnvNames
}

func InjectTemplatedEnvVarToPodContainer(existingEnvNames EnvVarNamesMap, container *corev1.Container, envVarName string, envVarValueTemplate *template.Template, distroParams map[string]string) (EnvVarNamesMap, error) {
	if _, exists := existingEnvNames[envVarName]; exists {
		return existingEnvNames, nil
	}

	var buf bytes.Buffer
	err := envVarValueTemplate.Execute(&buf, distroParams)
	if err != nil {
		// Should not happen. values are statically used from distro manifest which should be tested.
		return existingEnvNames, err
	}
	templatedEnvVarValue := buf.String()

	container.Env = append(container.Env, corev1.EnvVar{
		Name:  envVarName,
		Value: templatedEnvVarValue,
	})

	existingEnvNames[envVarName] = struct{}{}
	return existingEnvNames, nil
}

func injectNodeIpEnvVar(existingEnvNames EnvVarNamesMap, container *corev1.Container) EnvVarNamesMap {
	return injectEnvVarObjectFieldRefToPodContainer(existingEnvNames, container, k8sconsts.NodeIPEnvVar, "status.hostIP")
}

func InjectOdigosK8sEnvVars(existingEnvNames EnvVarNamesMap, container *corev1.Container, distroName string, ns string) EnvVarNamesMap {
	existingEnvNames = InjectConstEnvVarToPodContainer(existingEnvNames, container, k8sconsts.OdigosEnvVarContainerName, container.Name)
	existingEnvNames = InjectConstEnvVarToPodContainer(existingEnvNames, container, k8sconsts.OdigosEnvVarDistroName, distroName)
	existingEnvNames = injectEnvVarObjectFieldRefToPodContainer(existingEnvNames, container, k8sconsts.OdigosEnvVarPodName, "metadata.name")
	existingEnvNames = InjectConstEnvVarToPodContainer(existingEnvNames, container, k8sconsts.OdigosEnvVarNamespace, ns)
	return existingEnvNames
}

func InjectOpampServerEnvVar(existingEnvNames EnvVarNamesMap, container *corev1.Container) EnvVarNamesMap {
	existingEnvNames = injectNodeIpEnvVar(existingEnvNames, container)
	opAmpServerHost := service.LocalTrafficOpAmpOdigletEndpoint("$(NODE_IP)")
	existingEnvNames = InjectConstEnvVarToPodContainer(existingEnvNames, container, commonconsts.OpampServerHostEnvName, opAmpServerHost)
	return existingEnvNames
}

func InjectOtlpHttpEndpointEnvVar(existingEnvNames EnvVarNamesMap, container *corev1.Container) EnvVarNamesMap {
	existingEnvNames = injectNodeIpEnvVar(existingEnvNames, container)
	otlpHttpEndpoint := service.LocalTrafficOTLPHttpDataCollectionEndpoint("$(NODE_IP)")
	existingEnvNames = InjectConstEnvVarToPodContainer(existingEnvNames, container, commonconsts.OtelExporterEndpointEnvName, otlpHttpEndpoint)
	return existingEnvNames
}

func InjectStaticEnvVarsToPodContainer(existingEnvNames EnvVarNamesMap, container *corev1.Container, envVars []distro.StaticEnvironmentVariable, distroParams map[string]string) (EnvVarNamesMap, error) {
	for _, envVar := range envVars {
		if envVar.Template == nil {
			existingEnvNames = InjectConstEnvVarToPodContainer(existingEnvNames, container, envVar.EnvName, envVar.EnvValue)
		} else {
			var err error // make sure we don't shadow the error or the existingEnvNames
			existingEnvNames, err = InjectTemplatedEnvVarToPodContainer(existingEnvNames, container, envVar.EnvName, envVar.Template, distroParams)
			if err != nil {
				return existingEnvNames, fmt.Errorf("failed to inject static environment variable %s: %w", envVar.EnvName, err)
			}
		}
	}
	return existingEnvNames, nil
}

func signalOtlpExporterEnvValue(enabled bool) string {
	if enabled {
		return "otlp"
	}
	return "none"
}

func InjectSignalsAsStaticOtelEnvVars(existingEnvNames EnvVarNamesMap, container *corev1.Container, tracesEnabled bool, metricsEnabled bool, logsEnabled bool) EnvVarNamesMap {

	logsExporter := signalOtlpExporterEnvValue(logsEnabled)
	existingEnvNames = InjectConstEnvVarToPodContainer(existingEnvNames, container, commonconsts.OtelLogsExporter, logsExporter)

	metricsExporter := signalOtlpExporterEnvValue(metricsEnabled)
	existingEnvNames = InjectConstEnvVarToPodContainer(existingEnvNames, container, commonconsts.OtelMetricsExporter, metricsExporter)

	tracesExporter := signalOtlpExporterEnvValue(tracesEnabled)
	existingEnvNames = InjectConstEnvVarToPodContainer(existingEnvNames, container, commonconsts.OtelTracesExporter, tracesExporter)

	return existingEnvNames
}

func InjectUserEnvForLang(odigosConfiguration *common.OdigosConfiguration, pod *corev1.Pod, ic *odigosv1.InstrumentationConfig) {
	languageSpecificEnvs := odigosConfiguration.UserInstrumentationEnvs.Languages

	// Check for conatiner language and inject env vars if they not exists
	for _, containerDetailes := range ic.Status.RuntimeDetailsByContainer {
		langConfig, exists := languageSpecificEnvs[containerDetailes.Language]
		if !exists || !langConfig.Enabled {
			continue
		}

		container := getContainerByName(pod, containerDetailes.ContainerName)
		if container == nil {
			continue
		}
		existingEnvNames := GetEnvVarNamesSet(container)

		for envName, envValue := range langConfig.EnvVars {
			existingEnvNames = InjectConstEnvVarToPodContainer(
				existingEnvNames,
				container,
				envName,
				envValue,
			)
		}
	}
}

func getContainerByName(pod *corev1.Pod, name string) *corev1.Container {
	for i := range pod.Spec.Containers {
		if pod.Spec.Containers[i].Name == name {
			return &pod.Spec.Containers[i]
		}
	}
	return nil
}

// Create a set of existing environment variable names
// to avoid duplicates when injecting new environment variables
// into the container.
func GetEnvVarNamesSet(container *corev1.Container) EnvVarNamesMap {
	envSet := make(EnvVarNamesMap, len(container.Env))
	for _, envVar := range container.Env {
		envSet[envVar.Name] = struct{}{}
	}
	return envSet
}
