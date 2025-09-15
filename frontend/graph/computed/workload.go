package computed

import (
	"github.com/odigos-io/odigos/frontend/graph/model"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CachedWorkloadManifest struct {
	AvailableReplicas    int32
	Selector             *metav1.LabelSelector
	PodTemplateSpec      *corev1.PodTemplateSpec
	WorkloadHealthStatus *model.DesiredConditionStatus
}
