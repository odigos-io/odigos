package container

import (
	"errors"
	"strings"

	v1 "k8s.io/api/core/v1"

	"github.com/odigos-io/odigos/common"
)

var (
	ErrDeviceNotDetected     = errors.New("device not detected")
	ErrContainerNotInPodSpec = errors.New("container not found in pod spec")
)

func LanguageAndSdk(pod *v1.Pod, containerName string, distroName string) (common.ProgrammingLanguage, common.OtelSdk, error) {
	if distroName != "" {
		// TODO: so we can remove the device slowly while having backward compatibility,
		// we map here the distroNames one by one.
		// this is temporary, and should be refactored once device is removed
		switch distroName {
		case "golang-community":
			return common.GoProgrammingLanguage, common.OtelSdkEbpfCommunity, nil
		case "golang-enterprise":
			return common.GoProgrammingLanguage, common.OtelSdkEbpfEnterprise, nil
		case "java-enterprise":
			return common.JavaProgrammingLanguage, common.OtelSdkNativeEnterprise, nil
		case "java-ebpf-instrumentations":
			return common.JavaProgrammingLanguage, common.OtelSdkEbpfEnterprise, nil
		case "python-enterprise":
			return common.PythonProgrammingLanguage, common.OtelSdkEbpfEnterprise, nil
		case "nodejs-enterprise":
			return common.JavascriptProgrammingLanguage, common.OtelSdkEbpfEnterprise, nil
		case "mysql-enterprise":
			return common.MySQLProgrammingLanguage, common.OtelSdkEbpfEnterprise, nil
		}
	}

	// TODO: this is fallback for migration from device (so that we can handle pods that have not been updated yet)
	// remove this once device is removed
	return LanguageSdkFromPodContainer(pod, containerName)
}

func LanguageSdkFromPodContainer(pod *v1.Pod, containerName string) (common.ProgrammingLanguage, common.OtelSdk, error) {
	for i := range pod.Spec.Containers {
		container := pod.Spec.Containers[i]
		if container.Name == containerName {
			language, sdk, found := GetLanguageAndOtelSdk(&container)
			if !found {
				return common.UnknownProgrammingLanguage, common.OtelSdk{}, ErrDeviceNotDetected
			}

			return language, sdk, nil
		}
	}

	return common.UnknownProgrammingLanguage, common.OtelSdk{}, ErrContainerNotInPodSpec
}

func GetLanguageAndOtelSdk(container *v1.Container) (common.ProgrammingLanguage, common.OtelSdk, bool) {
	deviceName := podContainerDeviceName(container)
	if deviceName == nil {
		return common.UnknownProgrammingLanguage, common.OtelSdk{}, false
	}

	language, sdk := common.InstrumentationDeviceNameToComponents(*deviceName)
	return language, sdk, true
}

func podContainerDeviceName(container *v1.Container) *string {
	if container.Resources.Limits == nil {
		return nil
	}

	for resourceName := range container.Resources.Limits {
		resourceNameStr := string(resourceName)
		if strings.HasPrefix(resourceNameStr, common.OdigosResourceNamespace) {
			return &resourceNameStr
		}
	}

	return nil
}

func AllContainersReady(pod *v1.Pod) bool {
	// If pod has no containers, return false as we can't determine readiness
	if len(pod.Status.ContainerStatuses) == 0 {
		return false
	}
	// Check if pod is in Running phase.
	if pod.Status.Phase != v1.PodRunning {
		return false
	}
	// Iterate over all containers in the pod
	// Return false if any container is:
	// 1. Not Ready
	// 2. Started is nil or false
	for i := range pod.Status.ContainerStatuses {
		containerStatus := &pod.Status.ContainerStatuses[i]
		if !containerStatus.Ready || containerStatus.Started == nil || !*containerStatus.Started {
			return false
		}
	}
	return true
}

func GetContainerEnvVarValue(container *v1.Container, envVarName string) *string {
	for _, env := range container.Env {
		if env.Name == envVarName {
			return &env.Value
		}
	}
	return nil
}
