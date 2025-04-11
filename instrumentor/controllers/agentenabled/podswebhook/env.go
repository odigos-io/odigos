package podswebhook

import (
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
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

func InjectEnvVarToPodContainer(existingEnvNames EnvVarNamesMap, container *corev1.Container, envVarName, envVarValue string) EnvVarNamesMap {
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
	existingEnvNames = InjectEnvVarToPodContainer(existingEnvNames, container, k8sconsts.OdigosEnvVarContainerName, container.Name)
	existingEnvNames = InjectEnvVarToPodContainer(existingEnvNames, container, k8sconsts.OdigosEnvVarDistroName, distroName)
	existingEnvNames = injectEnvVarObjectFieldRefToPodContainer(existingEnvNames, container, k8sconsts.OdigosEnvVarPodName, "metadata.name")
	existingEnvNames = InjectEnvVarToPodContainer(existingEnvNames, container, k8sconsts.OdigosEnvVarNamespace, ns)
	return existingEnvNames
}

func InjectStaticEnvVar(existingEnvNames EnvVarNamesMap, container *corev1.Container, envVarName string, envVarValue string) EnvVarNamesMap {
	return InjectEnvVarToPodContainer(existingEnvNames, container, envVarName, envVarValue)
}

func InjectOpampServerEnvVar(existingEnvNames EnvVarNamesMap, container *corev1.Container) EnvVarNamesMap {
	existingEnvNames = injectNodeIpEnvVar(existingEnvNames, container)
	opAmpServerHost := fmt.Sprintf("$(NODE_IP):%d", commonconsts.OpAMPPort)
	existingEnvNames = InjectEnvVarToPodContainer(existingEnvNames, container, commonconsts.OpampServerHostEnvName, opAmpServerHost)
	return existingEnvNames
}

func InjectOtlpHttpEndpointEnvVar(existingEnvNames EnvVarNamesMap, container *corev1.Container) EnvVarNamesMap {
	existingEnvNames = injectNodeIpEnvVar(existingEnvNames, container)
	otlpHttpEndpoint := service.LocalTrafficOTLPHttpDataCollectionEndpoint("$(NODE_IP)")
	existingEnvNames = InjectEnvVarToPodContainer(existingEnvNames, container, commonconsts.OtelExporterEndpointEnvName, otlpHttpEndpoint)
	return existingEnvNames
}
