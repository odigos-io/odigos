package containers

import (
	corev1 "k8s.io/api/core/v1"
)

func GetContainerByName(containers []corev1.Container, name string) *corev1.Container {
	for i := range containers {
		if containers[i].Name == name {
			return &containers[i]
		}
	}
	return nil
}
