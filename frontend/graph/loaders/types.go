package loaders

import (
	corev1 "k8s.io/api/core/v1"
)

type WorkloadManifest struct {
	AvailableReplicas int32
	PodTemplateSpec   *corev1.PodTemplateSpec
}
