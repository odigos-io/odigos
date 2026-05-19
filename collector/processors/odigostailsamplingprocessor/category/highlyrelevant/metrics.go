package highlyrelevant

import (
	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/category"
	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
)

func recordEvalResultForSingleSpan(aggregatedResults map[string]*category.RuleEvaluationResult, rules []commonapisampling.HighlyRelevantOperation) {
	for _, rule := range rules {
		currResult, found := aggregatedResults[rule.Id]
		if !found {
			currResult = &category.RuleEvaluationResult{
				RuleId:         rule.Id,
				RuleName:       rule.Name,
				RulePercentage: category.GetPercentageOrDefault100(rule.PercentageAtLeast),
				RuleDisabled:   rule.Disabled,
			}
			aggregatedResults[rule.Id] = currResult
		}
		currResult.SpanCheckedCount++
	}
}
