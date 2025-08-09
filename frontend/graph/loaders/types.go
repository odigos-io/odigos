package loaders

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CachedPod struct {
	Pod               *corev1.Pod
	ComputedPodValues *ComputedPodValues
}

type WorkloadManifest struct {
	AvailableReplicas int32
	Selector          *metav1.LabelSelector
	PodTemplateSpec   *corev1.PodTemplateSpec
}
