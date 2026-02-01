package podswebhook

import (
	"path/filepath"
	"strings"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/distros/distro"
	corev1 "k8s.io/api/core/v1"
)

func MountDirectory(containerSpec *corev1.Container, dir string) {
	// TODO: assuming the directory always starts with {{ODIGOS_AGENTS_DIR}}. This should be validated.
	// Should we return errors here to validate static values?
	absolutePath := strings.ReplaceAll(dir, distro.AgentPlaceholderDirectory, k8sconsts.OdigosAgentsDirectory)
	relativePath := filepath.Base(absolutePath)

	// make sure we are idempotent, not adding ourselves multiple times
	for _, volumeMount := range containerSpec.VolumeMounts {
		if volumeMount.MountPath == absolutePath {
			// the volume is already mounted, do not add it again
			return
		}
	}

	containerSpec.VolumeMounts = append(containerSpec.VolumeMounts, corev1.VolumeMount{
		Name:      k8sconsts.OdigosAgentMountVolumeName,
		SubPath:   relativePath,
		MountPath: absolutePath,
		ReadOnly:  true,
	})
}

func mountPodVolumeIfNotExists(pod *corev1.Pod, volumeSource corev1.VolumeSource) {
	// make sure we are idempotent, not adding ourselves multiple times
	for _, volume := range pod.Spec.Volumes {
		if volume.Name == k8sconsts.OdigosAgentMountVolumeName {
			// the volume is already mounted, do not add it again
			return
		}
	}

	pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
		Name:         k8sconsts.OdigosAgentMountVolumeName,
		VolumeSource: volumeSource,
	})
}

func MountPodVolumeToHostPath(pod *corev1.Pod) {
	mountPodVolumeIfNotExists(pod, corev1.VolumeSource{
		HostPath: &corev1.HostPathVolumeSource{
			Path: k8sconsts.OdigosAgentsDirectory,
		},
	})
}

func MountPodVolumeToEmptyDir(pod *corev1.Pod) {
	mountPodVolumeIfNotExists(pod, corev1.VolumeSource{
		EmptyDir: &corev1.EmptyDirVolumeSource{},
	})
}

func MountPodVolumeToCSI(pod *corev1.Pod) {
	mountPodVolumeIfNotExists(pod, corev1.VolumeSource{
		CSI: &corev1.CSIVolumeSource{
			Driver: k8sconsts.OdigletCSIDriverName,
			VolumeAttributes: map[string]string{
				"type": "instrumentation",
			},
		},
	})
}
