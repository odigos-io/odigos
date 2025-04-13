package podswebhook

import (
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	commonconsts "github.com/odigos-io/odigos/common/consts"
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

func injectEnvVarToPodContainer(existingEnvNames EnvVarNamesMap, container *corev1.Container, envVarName, envVarValue string) EnvVarNamesMap {
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

func injectNodeIpEnvVar(existingEnvNames EnvVarNamesMap, container *corev1.Container) EnvVarNamesMap {
	return injectEnvVarObjectFieldRefToPodContainer(existingEnvNames, container, k8sconsts.NodeIPEnvVar, "status.hostIP")
}

func InjectOdigosK8sEnvVars(existingEnvNames EnvVarNamesMap, container *corev1.Container, distroName string, ns string) EnvVarNamesMap {
	existingEnvNames = injectEnvVarToPodContainer(existingEnvNames, container, k8sconsts.OdigosEnvVarContainerName, container.Name)
	existingEnvNames = injectEnvVarToPodContainer(existingEnvNames, container, k8sconsts.OdigosEnvVarDistroName, distroName)
	existingEnvNames = injectEnvVarObjectFieldRefToPodContainer(existingEnvNames, container, k8sconsts.OdigosEnvVarPodName, "metadata.name")
	existingEnvNames = injectEnvVarToPodContainer(existingEnvNames, container, k8sconsts.OdigosEnvVarNamespace, ns)
	return existingEnvNames
}

func InjectStaticEnvVar(existingEnvNames EnvVarNamesMap, container *corev1.Container, envVarName string, envVarValue string) EnvVarNamesMap {
	return injectEnvVarToPodContainer(existingEnvNames, container, envVarName, envVarValue)
}

func InjectOpampServerEnvVar(existingEnvNames EnvVarNamesMap, container *corev1.Container) EnvVarNamesMap {
	existingEnvNames = injectNodeIpEnvVar(existingEnvNames, container)
	opAmpServerHost := fmt.Sprintf("$(NODE_IP):%d", commonconsts.OpAMPPort)
	existingEnvNames = injectEnvVarToPodContainer(existingEnvNames, container, commonconsts.OpampServerHostEnvName, opAmpServerHost)
	return existingEnvNames
}

func InjectOtlpHttpEndpointEnvVar(existingEnvNames EnvVarNamesMap, container *corev1.Container) EnvVarNamesMap {
	existingEnvNames = injectNodeIpEnvVar(existingEnvNames, container)
	otlpHttpEndpoint := service.LocalTrafficOTLPHttpDataCollectionEndpoint("$(NODE_IP)")
	existingEnvNames = injectEnvVarToPodContainer(existingEnvNames, container, commonconsts.OtelExporterEndpointEnvName, otlpHttpEndpoint)
	return existingEnvNames
}

func InjectEnvsBasedOnLanguage(odigosConfig *common.OdigosConfiguration, pod *corev1.Pod, ic *odigosv1alpha1.InstrumentationConfig) {
	languageSpecificEnvs := odigosConfig.AdditionalInstrumentationEnvs.Languages

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
			existingEnvNames = injectEnvVarToPodContainer(
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
