package utils

import (
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
)

// naive implementation, can be optimized.
// assumption is that the list of workloads is small
func IsWorkloadParticipatingInRule(workload workload.PodWorkload, rule *odigosv1alpha1.InstrumentationRule) bool {
	// nil means all workloads are participating
	if rule.Spec.Workloads == nil {
		return true
	}
	for _, allowedWorkload := range *rule.Spec.Workloads {
		if allowedWorkload == workload {
			return true
		}
	}
	return false
}
