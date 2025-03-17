package podswebhook

import (
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	commonconsts "github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/service"
	corev1 "k8s.io/api/core/v1"
)

func injectEnvVarObjectFieldRefToPodContainer(existingEnvNames *map[string]struct{}, container *corev1.Container, envVarName, envVarRef string) {
	if _, exists := (*existingEnvNames)[envVarName]; exists {
		return
	}

	container.Env = append(container.Env, corev1.EnvVar{
		Name: envVarName,
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: envVarRef,
			},
		},
	})

	(*existingEnvNames)[envVarName] = struct{}{}
}

func injectEnvVarToPodContainer(existingEnvNames *map[string]struct{}, container *corev1.Container, envVarName, envVarValue string) {
	if _, exists := (*existingEnvNames)[envVarName]; exists {
		return
	}

	container.Env = append(container.Env, corev1.EnvVar{
		Name:  envVarName,
		Value: envVarValue,
	})

	(*existingEnvNames)[envVarName] = struct{}{}
}

func injectNodeIpEnvVar(existingEnvNames *map[string]struct{}, container *corev1.Container) {
	injectEnvVarObjectFieldRefToPodContainer(existingEnvNames, container, k8sconsts.NodeIPEnvVar, "status.hostIP")
}

func InjectOdigosK8sEnvVars(existingEnvNames *map[string]struct{}, container *corev1.Container, distroName string, ns string) {
	injectEnvVarToPodContainer(existingEnvNames, container, k8sconsts.OdigosEnvVarContainerName, container.Name)
	injectEnvVarToPodContainer(existingEnvNames, container, k8sconsts.OdigosEnvVarDistroName, distroName)
	injectEnvVarObjectFieldRefToPodContainer(existingEnvNames, container, k8sconsts.OdigosEnvVarPodName, "metadata.name")
	injectEnvVarToPodContainer(existingEnvNames, container, k8sconsts.OdigosEnvVarNamespace, ns)
}

func InjectStaticEnvVar(existingEnvNames *map[string]struct{}, container *corev1.Container, envVarName string, envVarValue string) {
	injectEnvVarToPodContainer(existingEnvNames, container, envVarName, envVarValue)
}

func InjectOpampServerEnvVar(existingEnvNames *map[string]struct{}, container *corev1.Container) {
	injectNodeIpEnvVar(existingEnvNames, container)
	opAmpServerHost := fmt.Sprintf("$(NODE_IP):%d", commonconsts.OpAMPPort)
	injectEnvVarToPodContainer(existingEnvNames, container, commonconsts.OpampServerHostEnvName, opAmpServerHost)
}

func InjectOtlpHttpEndpointEnvVar(existingEnvNames *map[string]struct{}, container *corev1.Container) {
	injectNodeIpEnvVar(existingEnvNames, container)
	otlpHttpEndpoint := service.LocalTrafficOTLPHttpDataCollectionEndpoint("$(NODE_IP)")
	injectEnvVarToPodContainer(existingEnvNames, container, commonconsts.OtelExporterEndpointEnvName, otlpHttpEndpoint)
}
