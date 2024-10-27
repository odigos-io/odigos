package source

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type K8sSourceObject struct {
	metav1.ObjectMeta
	Kind            string
	PodTemplateSpec *corev1.PodTemplateSpec
	LabelSelector   *metav1.LabelSelector
}
