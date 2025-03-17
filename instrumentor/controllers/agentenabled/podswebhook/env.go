package podswebhook

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
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

func InjectOdigosK8sEnvVars(existingEnvNames *map[string]struct{}, container *corev1.Container, distroName string, ns string) {
	injectEnvVarToPodContainer(existingEnvNames, container, k8sconsts.OdigosEnvVarContainerName, container.Name)
	injectEnvVarToPodContainer(existingEnvNames, container, k8sconsts.OdigosEnvVarDistroName, distroName)
	injectEnvVarObjectFieldRefToPodContainer(existingEnvNames, container, k8sconsts.OdigosEnvVarPodName, "metadata.name")
	injectEnvVarToPodContainer(existingEnvNames, container, k8sconsts.OdigosEnvVarNamespace, ns)
}

func InjectStaticEnvVar(existingEnvNames *map[string]struct{}, container *corev1.Container, envVarName string, envVarValue string) {
	injectEnvVarToPodContainer(existingEnvNames, container, envVarName, envVarValue)
}
