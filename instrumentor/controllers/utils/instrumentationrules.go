package utils

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
)

// naive implementation, can be optimized.
// assumption is that the list of workloads is small
func IsWorkloadParticipatingInRule(workload k8sconsts.PodWorkload, rule *odigosv1alpha1.InstrumentationRule) bool {

	// first check if the rule is disabled
	if rule.Spec.Disabled {
		return false
	}

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
