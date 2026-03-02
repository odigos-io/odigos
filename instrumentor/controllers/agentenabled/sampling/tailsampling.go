package sampling

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/distros/distro"
	commonapi "github.com/odigos-io/odigos/common/api"
)

func FilterTailSamplingRulesForContainer(samplingRules *[]odigosv1.Sampling, language common.ProgrammingLanguage,
	pw k8sconsts.PodWorkload, containerName string, distro *distro.OtelDistro) ([]commonapi.WorkloadNoisyOperation, []commonapi.WorkloadHighlyRelevantOperation, []commonapi.WorkloadCostReductionRule) {

	var filteredNoisyOps []commonapi.WorkloadNoisyOperation
	var filteredRelevantOps []commonapi.WorkloadHighlyRelevantOperation
	var filteredCostRules []commonapi.WorkloadCostReductionRule

	for _, samplingRule := range *samplingRules {
		// Filter and convert NoisyOperations, HighlyRelevantOperations, CostReductionRules.
		// Exclude SourceScopes and Notes from the rules because we want the instrumentationConfig to be more lightweight.

		// If the distro not supports head sampling, we need the NoisyOperations to be applied at the collector level (tailsampling).
		if distro.Traces == nil || distro.Traces.HeadSampling == nil || !distro.Traces.HeadSampling.Supported {
			for _, noisyOp := range samplingRule.Spec.NoisyOperations {
				if IsServiceInRuleScope(noisyOp.SourceScopes, pw, containerName, language) {
					filteredNoisyOps = append(filteredNoisyOps, commonapi.WorkloadNoisyOperation{
						Id:               odigosv1.ComputeNoisyOperationHash(&noisyOp),
						Operation:        noisyOp.Operation,
						PercentageAtMost: noisyOp.PercentageAtMost,
					})
				}
			}
		}

		// Filter and convert HighlyRelevantOperations - exclude SourceScopes and Notes
		for _, relevantOp := range samplingRule.Spec.HighlyRelevantOperations {
			if IsServiceInRuleScope(relevantOp.SourceScopes, pw, containerName, language) {
				filteredRelevantOps = append(filteredRelevantOps, commonapi.WorkloadHighlyRelevantOperation{
					Id:                odigosv1.ComputeHighlyRelevantOperationHash(&relevantOp),
					Error:             relevantOp.Error,
					DurationAtLeastMs: relevantOp.DurationAtLeastMs,
					Operation:         relevantOp.Operation,
					PercentageAtLeast: relevantOp.PercentageAtLeast,
				})
			}
		}

		for _, costRule := range samplingRule.Spec.CostReductionRules {
			if IsServiceInRuleScope(costRule.SourceScopes, pw, containerName, language) {
				filteredCostRules = append(filteredCostRules, commonapi.WorkloadCostReductionRule{
					Id:               odigosv1.ComputeCostReductionRuleHash(&costRule),
					Operation:        costRule.Operation,
					PercentageAtMost: costRule.PercentageAtMost,
				})
			}
		}
	}

	return filteredNoisyOps, filteredRelevantOps, filteredCostRules
}
