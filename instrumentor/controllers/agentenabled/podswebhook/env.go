package podswebhook

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	corev1 "k8s.io/api/core/v1"
)

func InjectK8sEnvVars(container *corev1.Container, distroName string, pw k8sconsts.PodWorkload) {

	// check for existing env vars so we don't introduce them again
	existingEnvNames := make(map[string]struct{})
	for _, envVar := range container.Env {
		existingEnvNames[envVar.Name] = struct{}{}
	}

	injectEnvVarToPodContainer(&existingEnvNames, container, k8sconsts.OdigosEnvVarContainerName, container.Name)
	injectEnvVarToPodContainer(&existingEnvNames, container, k8sconsts.OdigosEnvVarDistroName, distroName)
	injectEnvVarToPodContainer(&existingEnvNames, container, k8sconsts.OdigosEnvVarPodName, pw.Name)
	injectEnvVarToPodContainer(&existingEnvNames, container, k8sconsts.OdigosEnvVarNamespace, pw.Namespace)
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
