package noisy

import (
	"go.opentelemetry.io/collector/pdata/ptrace"

	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/category"
	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/matchers"
	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
)

type NoisyOperationsEvaluationResult struct {
	DecidingRule     *commonapisampling.NoisyOperation
	RulesEvalResults category.CategoryRulesEvaluationResults
}

// givin a root span for a trace, and a list of noisy operation sampling rules,
// evaluate if the trace belongs to the noisy operations category,
// and return the "matching rule" - e.g. the rule with the least percentage.
func Evaluate(span ptrace.Span, noisyOperations []commonapisampling.NoisyOperation) NoisyOperationsEvaluationResult {

	rulesEvalResults := category.CategoryRulesEvaluationResults{}

	// aggregate the matching rules in a list.
	// there should be very few, so the length is expected to be 0 almost always,
	// 1 occassionally, and more very rarely.
	var leastPercentageRule *commonapisampling.NoisyOperation

	for _, noisyOperation := range noisyOperations {

		currentPercentage := category.GetPercentageOrDefault0(noisyOperation.PercentageAtMost)

		// shortcut - we are only interested in the least percentage rule,
		// so avoid checking when unnecessary.
		// percentageAtMost as nil, means that it's the default 0%, so it's already the smallest possible.
		if leastPercentageRule != nil && (leastPercentageRule.PercentageAtMost == nil || currentPercentage >= *(leastPercentageRule.PercentageAtMost)) {
			continue
		}

		// check if the operation matches the span.
		matched := matchers.HeadSamplingOperationMatcher(noisyOperation.Operation, span)

		if _, found := rulesEvalResults[noisyOperation.Id]; !found {
			rulesEvalResults[noisyOperation.Id] = &category.RuleEvaluationResult{
				RuleId:         noisyOperation.Id,
				RuleName:       noisyOperation.Name,
				RulePercentage: currentPercentage,
				RuleDisabled:   noisyOperation.Disabled,
			}
		}
		res := rulesEvalResults[noisyOperation.Id]
		res.SpanCheckedCount++

		// at this point, we already know the current percentage is least than the one seen so far,
		// so if we have a match, we update.
		if matched {
			if !noisyOperation.Disabled {
				leastPercentageRule = &noisyOperation
			}
			res.SpanMatchedCount++
		}
	}

	return NoisyOperationsEvaluationResult{
		DecidingRule:     leastPercentageRule,
		RulesEvalResults: rulesEvalResults,
	}
}
