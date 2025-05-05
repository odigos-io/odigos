package podswebhook

import (
	"fmt"
	"strings"

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

func InjectEnvVarToPodContainer(existingEnvNames EnvVarNamesMap, container *corev1.Container, envVarName, envVarValue string, runtimeDetails *odigosv1.RuntimeDetailsByContainer) EnvVarNamesMap {
	if _, exists := existingEnvNames[envVarName]; exists {
		return existingEnvNames
	}

	if strings.Contains(envVarValue, distro.RuntimeVersionPlaceholderMajorMinor) {
		// This is a placeholder for the runtime version, we need to replace it with the actual runtime version
		if runtimeDetails != nil {
			majorMinor := common.MajorMinorStringOnly(common.GetVersion(runtimeDetails.RuntimeVersion))
			envVarValue = strings.ReplaceAll(envVarValue, distro.RuntimeVersionPlaceholderMajorMinor, majorMinor)
		} else {
			// If we don't have runtime details, we can't replace the placeholder
			return existingEnvNames
		}
	}

	container.Env = append(container.Env, corev1.EnvVar{
		Name:  envVarName,
		Value: envVarValue,
	})

	existingEnvNames[envVarName] = struct{}{}
	return existingEnvNames
}

func injectNodeIpEnvVar(existingEnvNames EnvVarNamesMap, container *corev1.Container) EnvVarNamesMap {
	return injectEnvVarObjectFieldRefToPodContainer(existingEnvNames, container, k8sconsts.NodeIPEnvVar, "status.hostIP")
}

func InjectOdigosK8sEnvVars(existingEnvNames EnvVarNamesMap, container *corev1.Container, distroName string, ns string) EnvVarNamesMap {
	existingEnvNames = InjectEnvVarToPodContainer(existingEnvNames, container, k8sconsts.OdigosEnvVarContainerName, container.Name, nil)
	existingEnvNames = InjectEnvVarToPodContainer(existingEnvNames, container, k8sconsts.OdigosEnvVarDistroName, distroName, nil)
	existingEnvNames = injectEnvVarObjectFieldRefToPodContainer(existingEnvNames, container, k8sconsts.OdigosEnvVarPodName, "metadata.name")
	existingEnvNames = InjectEnvVarToPodContainer(existingEnvNames, container, k8sconsts.OdigosEnvVarNamespace, ns, nil)
	return existingEnvNames
}

func InjectStaticEnvVar(existingEnvNames EnvVarNamesMap, container *corev1.Container, envVarName string, envVarValue string, runtimeDetails *odigosv1.RuntimeDetailsByContainer) EnvVarNamesMap {
	return InjectEnvVarToPodContainer(existingEnvNames, container, envVarName, envVarValue, runtimeDetails)
}

func InjectOpampServerEnvVar(existingEnvNames EnvVarNamesMap, container *corev1.Container) EnvVarNamesMap {
	existingEnvNames = injectNodeIpEnvVar(existingEnvNames, container)
	opAmpServerHost := fmt.Sprintf("$(NODE_IP):%d", commonconsts.OpAMPPort)
	existingEnvNames = InjectEnvVarToPodContainer(existingEnvNames, container, commonconsts.OpampServerHostEnvName, opAmpServerHost, nil)
	return existingEnvNames
}

func InjectOtlpHttpEndpointEnvVar(existingEnvNames EnvVarNamesMap, container *corev1.Container) EnvVarNamesMap {
	existingEnvNames = injectNodeIpEnvVar(existingEnvNames, container)
	otlpHttpEndpoint := service.LocalTrafficOTLPHttpDataCollectionEndpoint("$(NODE_IP)")
	existingEnvNames = InjectEnvVarToPodContainer(existingEnvNames, container, commonconsts.OtelExporterEndpointEnvName, otlpHttpEndpoint, nil)
	return existingEnvNames
}

func InjectUserEnvForLang(odigosConfig *common.OdigosConfiguration, pod *corev1.Pod, ic *odigosv1.InstrumentationConfig) {
	languageSpecificEnvs := odigosConfig.UserInstrumentationEnvs.Languages

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
			existingEnvNames = InjectEnvVarToPodContainer(
				existingEnvNames,
				container,
				envName,
				envValue,
				nil,
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
