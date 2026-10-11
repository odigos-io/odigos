package container

import (
	v1 "k8s.io/api/core/v1"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
)

// given an instrumentation config spec containers object,
// find and return the config for a specific container by name.
// return nil if not found.
func GetContainerConfigByName(containers []odigosv1.ContainerAgentConfig, containerName string) *odigosv1.ContainerAgentConfig {
	for i := range containers {
		if containers[i].ContainerName == containerName {
			return &containers[i]
		}
	}
	return nil
}

func IsContainerInCrashLoopBackOff(containerStatus *v1.ContainerStatus) bool {
	return containerStatus.State.Waiting != nil && containerStatus.State.Waiting.Reason == "CrashLoopBackOff"
}

func IsContainerInImagePullBackOff(containerStatus *v1.ContainerStatus) bool {
	return containerStatus.State.Waiting != nil && containerStatus.State.Waiting.Reason == "ImagePullBackOff"
}

// IsContainerInBackOff returns true if the container is in CrashLoopBackOff or ImagePullBackOff
func IsContainerInBackOff(containerStatus *v1.ContainerStatus) bool {
	return IsContainerInCrashLoopBackOff(containerStatus) || IsContainerInImagePullBackOff(containerStatus)
}
