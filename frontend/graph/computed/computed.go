package computed

import (
	"github.com/odigos-io/odigos/frontend/graph/model"
	corev1 "k8s.io/api/core/v1"
)

type ComputedPodValues struct {
	AgentInjected       bool
	AgentInjectedStatus *model.DesiredConditionStatus
}

type CachedPod struct {
	Pod               *corev1.Pod
	ComputedPodValues *ComputedPodValues
}
