package computed

import (
	"github.com/odigos-io/odigos/frontend/graph/model"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CachedWorkloadManifest struct {
	AvailableReplicas    int32
	Selector             *metav1.LabelSelector
	WorkloadHealthStatus *model.DesiredConditionStatus
}
