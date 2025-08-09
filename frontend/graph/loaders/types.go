package loaders

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type WorkloadManifest struct {
	AvailableReplicas int32
	Selector          *metav1.LabelSelector
	PodTemplateSpec   *corev1.PodTemplateSpec
}
