package category

import (
	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/category/config"
)

type RuleEvaluationResult struct {
	ComputedRule config.ComputedRule

	// number of spans on which we evaluated this rule
	SpanCheckedCount int

	// number of spans on which we matched this rule
	SpanMatchedCount int
}

type CategoryRulesEvaluationResults map[string]*RuleEvaluationResult
