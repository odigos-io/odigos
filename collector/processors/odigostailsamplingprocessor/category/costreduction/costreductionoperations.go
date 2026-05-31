package costreduction

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/category"
	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/category/config"
	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/category/metrics"
	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/category/samplingspanattrs"
	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/matchers"
)

type CostReductionEvaluationResult struct {
	DecidingRule     *config.ComputedRule
	RulesEvalResults category.CategoryRulesEvaluationResults
}

// EvaluateCostReductionOperations matches cost-reduction rules on each span, sets per-span attributes,
// aggregates trace-level matches, and returns whether a non-disabled deciding rule applies.
func Evaluate(trace ptrace.Traces, configProvider config.TailSamplingConfigProvider) CostReductionEvaluationResult {
	matchingRules := map[string]*config.ComputedRule{}
	rulesEvalResults := category.CategoryRulesEvaluationResults{}

	rss := trace.ResourceSpans()
	for i := 0; i < rss.Len(); i++ {
		res := rss.At(i)

		costReductionRules := getCostReductionRulesConfig(configProvider, res.Resource())
		if costReductionRules == nil {
			continue
		}

		scopes := res.ScopeSpans()
		for j := 0; j < scopes.Len(); j++ {
			scope := scopes.At(j)
			spans := scope.Spans()
			for k := 0; k < spans.Len(); k++ {
				span := spans.At(k)

				matchedRules := matchCostReductionRulesForSingleSpan(rulesEvalResults, matchingRules, span, costReductionRules)
				spanLeastPercentageRule := selectCostReductionRuleFromMatches(matchedRules)

				if spanLeastPercentageRule != nil {
					samplingspanattrs.SetSpanMatchingRuleAttributesOnSpan(span, spanLeastPercentageRule)
				}
			}
		}
	}

	decidingRule := calculateCostReductionDecidingRule(matchingRules)
	return CostReductionEvaluationResult{
		DecidingRule:     decidingRule,
		RulesEvalResults: rulesEvalResults,
	}
}

// matchCostReductionRulesForSingleSpan returns every cost-reduction rule whose operation matcher passes for this span.
// it also updates the rulesEvalResults and matchingRules maps based on the matched rules.
func matchCostReductionRulesForSingleSpan(rulesEvalResults map[string]*category.RuleEvaluationResult, matchingRules map[string]*config.ComputedRule, span ptrace.Span, costReductionRules []config.ComputedCostReductionRule) []*config.ComputedCostReductionRule {
	matchedRules := []*config.ComputedCostReductionRule{}

	for _, rule := range costReductionRules {
		matched := matchers.TailSamplingOperationMatcher(rule.Rule.Operation, span)
		metrics.RecordEvalResultForSingleSpan(rulesEvalResults, rule.ComputedRule, matched)
		if matched {
			matchedRules = append(matchedRules, &rule)
			if _, found := matchingRules[rule.ComputedRule.RuleId]; !found {
				matchingRules[rule.ComputedRule.RuleId] = &rule.ComputedRule
			}
		}
	}

	return matchedRules
}

// selectCostReductionRuleFromMatches picks the span-level rule using the same logic as the trace deciding rule
// (see calculateCostReductionDecidingRule): smallest PercentageAtMost among enabled matches.
func selectCostReductionRuleFromMatches(matchedRules []*config.ComputedCostReductionRule) *config.ComputedRule {
	byID := make(map[string]*config.ComputedRule, len(matchedRules))
	for _, r := range matchedRules {
		byID[r.ComputedRule.RuleId] = &r.ComputedRule
	}
	return calculateCostReductionDecidingRule(byID)
}

func getCostReductionRulesConfig(configProvider config.TailSamplingConfigProvider, resource pcommon.Resource) []config.ComputedCostReductionRule {
	tailSampling, found := configProvider.GetTailSamplingConfig(resource)
	if !found || tailSampling == nil {
		return nil
	}
	if len(tailSampling.CostReductionRules) == 0 {
		return nil
	}
	return tailSampling.CostReductionRules
}

// calculateCostReductionDecidingRule returns the enabled rule with the lowest PercentageAtMost (most restrictive).
// Used for the trace-level deciding rule and for per-span attributes after a span's matches are keyed by rule id.
func calculateCostReductionDecidingRule(matchingRules map[string]*config.ComputedRule) *config.ComputedRule {
	if len(matchingRules) == 0 {
		return nil
	}

	// check for lowest percentage enabled sampling rule.
	var selectedRule *config.ComputedRule
	for _, r := range matchingRules {
		if r.Disabled {
			continue
		}
		// shortcut for when we hit the bottom and don't need to keep iterating.
		if r.Percentage == 0 {
			return r
		}
		// update if it's the first one, or if it's less than the current smallest one.
		if selectedRule == nil || r.Percentage < selectedRule.Percentage {
			selectedRule = r
		}
	}
	return selectedRule
}
