package noisy

import (
	"go.opentelemetry.io/collector/pdata/ptrace"

	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/category"
	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/category/config"
)

type NoisyOperationsEvaluationResult struct {
	DecidingRule     *config.ComputedRule
	RulesEvalResults category.CategoryRulesEvaluationResults
}

// givin a root span for a trace, and a list of noisy operation sampling rules,
// evaluate if the trace belongs to the noisy operations category,
// and return the "matching rule" - e.g. the rule with the least percentage.
func Evaluate(span ptrace.Span, noisyOperations []config.ComputedNoisyOperation) NoisyOperationsEvaluationResult {

	rulesEvalResults := category.CategoryRulesEvaluationResults{}

	// aggregate the matching rules in a list.
	// there should be very few, so the length is expected to be 0 almost always,
	// 1 occassionally, and more very rarely.
	var leastPercentageRule *config.ComputedRule

	for _, noisyOperation := range noisyOperations {

		currentPercentage := noisyOperation.Percentage

		if noisyOperation.Rule.Disabled {
			continue
		}

		// shortcut - we are only interested in the least percentage rule,
		// so avoid checking when unnecessary.
		// percentageAtMost as nil, means that it's the default 0%, so it's already the smallest possible.
		if leastPercentageRule != nil && (leastPercentageRule.Percentage == 0 || currentPercentage >= leastPercentageRule.Percentage) {
			continue
		}

		// check if the operation matches the span.
		matched := noisyOperation.Matcher.Match(span)

		if _, found := rulesEvalResults[noisyOperation.Rule.Id]; !found {
			rulesEvalResults[noisyOperation.Rule.Id] = &category.RuleEvaluationResult{
				ComputedRule: noisyOperation.ComputedRule,
			}
		}
		res := rulesEvalResults[noisyOperation.Rule.Id]
		res.SpanCheckedCount++

		// at this point, we already know the current percentage is least than the one seen so far,
		// so if we have a match, we update.
		if matched {
			leastPercentageRule = &noisyOperation.ComputedRule
			res.SpanMatchedCount++
		}
	}

	return NoisyOperationsEvaluationResult{
		DecidingRule:     leastPercentageRule,
		RulesEvalResults: rulesEvalResults,
	}
}
