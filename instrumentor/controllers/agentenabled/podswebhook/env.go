package podswebhook

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	commonconsts "github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/distros/distro"
	containersutil "github.com/odigos-io/odigos/k8sutils/pkg/containers"
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
		if envVar.AppendToExisting {
			var err error
			existingEnvNames, err = appendEnvVarToPodContainer(existingEnvNames, container, envVar, distroParams)
			if err != nil {
				return existingEnvNames, fmt.Errorf("failed to inject static environment variable %s: %w", envVar.EnvName, err)
			}
		} else if envVar.Template == nil {
			existingEnvNames = InjectConstEnvVarToPodContainer(existingEnvNames, container, envVar.EnvName, envVar.EnvValue)
		} else {
			var err error
			existingEnvNames, err = InjectTemplatedEnvVarToPodContainer(existingEnvNames, container, envVar.EnvName, envVar.Template, distroParams)
			if err != nil {
				return existingEnvNames, fmt.Errorf("failed to inject static environment variable %s: %w", envVar.EnvName, err)
			}
		}
	}
	return existingEnvNames, nil
}

// appendEnvVarToPodContainer handles env vars with AppendToExisting semantics.
// If the env var already exists in the container manifest, the rendered value is
// appended to it (e.g. "/user/path" + ":/odigos/path" = "/user/path:/odigos/path").
// If the env var is absent, but a CRI-detected runtime value exists in distroParams
// (keyed by the env var name), that value is prepended to the rendered value.
// Otherwise the rendered value is set directly (preserving any leading delimiter
// like the ":" prefix that PHP_INI_SCAN_DIR uses to retain the default scan dir).
func appendEnvVarToPodContainer(existingEnvNames EnvVarNamesMap, container *corev1.Container, envVar distro.StaticEnvironmentVariable, distroParams map[string]string) (EnvVarNamesMap, error) {
	var resolvedValue string
	if envVar.Template != nil {
		var buf bytes.Buffer
		if err := envVar.Template.Execute(&buf, distroParams); err != nil {
			return existingEnvNames, err
		}
		resolvedValue = buf.String()
	} else {
		resolvedValue = envVar.EnvValue
	}

	for i := range container.Env {
		if container.Env[i].Name != envVar.EnvName {
			continue
		}
		if strings.Contains(container.Env[i].Value, resolvedValue) {
			return existingEnvNames, nil
		}
		container.Env[i].Value += resolvedValue
		return existingEnvNames, nil
	}

	if criValue, ok := distroParams[envVar.EnvName]; ok && criValue != "" {
		if !strings.Contains(criValue, resolvedValue) {
			resolvedValue = criValue + resolvedValue
		}
	}

	container.Env = append(container.Env, corev1.EnvVar{
		Name:  envVar.EnvName,
		Value: resolvedValue,
	})
	existingEnvNames[envVar.EnvName] = struct{}{}
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

		container := containersutil.GetContainerByName(pod.Spec.Containers, containerDetailes.ContainerName)
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
