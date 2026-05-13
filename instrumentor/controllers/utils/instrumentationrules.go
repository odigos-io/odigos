package utils

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
)

// Resolves whether the rule applies to the workload for at least one container on the InstrumentationConfig.
func IsInstrumentationConfigParticipatingInRule(
	workload k8sconsts.PodWorkload,
	ic *odigosv1alpha1.InstrumentationConfig,
	rule *odigosv1alpha1.InstrumentationRule,
) bool {
	if rule.Spec.Disabled {
		return false
	}

	// this function is used for the "instrumentationconfig" controllers which are about to get deprecated soon,
	// and are unused in odigos.
	// the rules were always applied to the entire cluster, so we can relax the check here for short while
	// until the controllers are removed.
	return true
}
