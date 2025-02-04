package source

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/odigos-io/odigos/api/k8sconsts"
)

type K8sSourceObject struct {
	metav1.ObjectMeta
	Kind            k8sconsts.WorkloadKind
	PodTemplateSpec *corev1.PodTemplateSpec
	LabelSelector   *metav1.LabelSelector
}
