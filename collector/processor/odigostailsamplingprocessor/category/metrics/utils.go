package metrics

import (
	"github.com/odigos-io/odigos/collector/processor/odigostailsamplingprocessor/category"
	"github.com/odigos-io/odigos/collector/processor/odigostailsamplingprocessor/category/config"
)

func RecordEvalResultForSingleSpan(aggregatedResults map[string]*category.RuleEvaluationResult, rule config.ComputedRule, matched bool) {
	if _, found := aggregatedResults[rule.RuleId]; !found {
		aggregatedResults[rule.RuleId] = &category.RuleEvaluationResult{
			ComputedRule: rule,
		}
	}

	currResult := aggregatedResults[rule.RuleId]

	currResult.SpanCheckedCount++
	if matched {
		currResult.SpanMatchedCount++
	}
}
