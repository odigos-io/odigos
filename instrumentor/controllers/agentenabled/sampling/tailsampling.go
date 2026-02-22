package sampling

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/distros/distro"
)

func FilterTailSamplingRulesForContainer(samplingRules *[]odigosv1.Sampling, language common.ProgrammingLanguage,
	pw k8sconsts.PodWorkload, containerName string, distro *distro.OtelDistro) ([]odigosv1.NoisyOperations, []odigosv1.HighlyRelevantOperation, []odigosv1.CostReductionRule) {

	var filteredNoisyOps []odigosv1.NoisyOperations
	var filteredRelevantOps []odigosv1.HighlyRelevantOperation
	var filteredCostRules []odigosv1.CostReductionRule

	for _, samplingRule := range *samplingRules {
		// Filter and convert NoisyOperations, HighlyRelevantOperations, CostReductionRules.
		// Exclude SourceScopes and Notes from the rules because we want the instrumentationConfig to be more lightweight.

		// If the distro not supports head sampling, we need the NoisyOperations to be applied at the collector level (tailsampling).
		if distro.Traces == nil || distro.Traces.HeadSampling == nil || !distro.Traces.HeadSampling.Supported {
			for _, noisyOp := range samplingRule.Spec.NoisyOperations {
				if IsServiceInRuleScope(noisyOp.SourceScopes, pw, containerName, language) {
					filteredNoisyOps = append(filteredNoisyOps, odigosv1.NoisyOperations{
						Operation:        noisyOp.Operation,
						PercentageAtMost: noisyOp.PercentageAtMost,
					})
				}
			}
		}

		// Filter and convert HighlyRelevantOperations - exclude SourceScopes and Notes
		for _, relevantOp := range samplingRule.Spec.HighlyRelevantOperations {
			if IsServiceInRuleScope(relevantOp.SourceScopes, pw, containerName, language) {
				filteredRelevantOps = append(filteredRelevantOps, odigosv1.HighlyRelevantOperation{
					Error:             relevantOp.Error,
					DurationAtLeastMs: relevantOp.DurationAtLeastMs,
					Operation:         relevantOp.Operation,
					PercentageAtLeast: relevantOp.PercentageAtLeast,
				})
			}
		}

		for _, costRule := range samplingRule.Spec.CostReductionRules {
			if IsServiceInRuleScope(costRule.SourceScopes, pw, containerName, language) {
				filteredCostRules = append(filteredCostRules, odigosv1.CostReductionRule{
					Operation:        costRule.Operation,
					PercentageAtMost: costRule.PercentageAtMost,
				})
			}
		}
	}

	return filteredNoisyOps, filteredRelevantOps, filteredCostRules
}
