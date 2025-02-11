package podswebhook

import (
	"strings"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/distros/distro"
	corev1 "k8s.io/api/core/v1"
)

func checkIfMountDirectoryExists(containerSpec *corev1.Container, dir string) bool {
	for _, volumeMount := range containerSpec.VolumeMounts {
		if volumeMount.SubPath == dir {
			return true
		}
	}
	return false
}

func MountDirectory(containerSpec *corev1.Container, dir string) {
	// TODO: assuming the directory always starts with {{ODIGOS_AGENTS_DIR}}. This should be validated.
	// Should we return errors here to validate static values?
	relativePath := strings.TrimPrefix(dir, distro.AgentPlaceholderDirectory+"/")
	if checkIfMountDirectoryExists(containerSpec, dir) {
		// avoid adding the directory volume twice to the container
		return
	}

	absolutePath := strings.ReplaceAll(dir, distro.AgentPlaceholderDirectory, k8sconsts.OdigosAgentsDirectory)
	containerSpec.VolumeMounts = append(containerSpec.VolumeMounts, corev1.VolumeMount{
		Name:      k8sconsts.OdigosAgentMountVolumeName,
		SubPath:   relativePath,
		MountPath: absolutePath,
		ReadOnly:  true,
	})
}

func checkIfVolumExists(pod *corev1.Pod) bool {
	for _, volume := range pod.Spec.Volumes {
		if volume.Name == k8sconsts.OdigosAgentMountVolumeName {
			return true
		}
	}
	return false
}

func MountPodVolume(pod *corev1.Pod) {

	if checkIfVolumExists(pod) {
		// avoid adding the volume twice to the pod
		return
	}

	pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
		Name: k8sconsts.OdigosAgentMountVolumeName,
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: k8sconsts.OdigosAgentsDirectory,
			},
		},
	})
}
