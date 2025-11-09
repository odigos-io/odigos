package containers

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/odigos-io/odigos/api/k8sconsts"
)

func GetContainerByName(containers []corev1.Container, name string) *corev1.Container {
	for i := range containers {
		if containers[i].Name == name {
			return &containers[i]
		}
	}
	return nil
}

func GetCollectorContainerName(pod *corev1.Pod) string {
	role := pod.Labels[k8sconsts.OdigosCollectorRoleLabel]
	switch k8sconsts.CollectorRole(role) {
	case k8sconsts.CollectorsRoleClusterGateway:
		return k8sconsts.OdigosClusterCollectorContainerName
	case k8sconsts.CollectorsRoleNodeCollector:
		return k8sconsts.OdigosNodeCollectorContainerName
	default:
		return ""
	}
}
