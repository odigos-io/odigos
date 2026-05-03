package costreduction

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/matchers"
	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/common/collector"
	"github.com/odigos-io/odigos/common/odigosattributes"
)

// EvaluateCostReductionOperations matches cost-reduction rules on each span, sets per-span attributes,
// aggregates trace-level matches, and returns whether a non-disabled deciding rule applies.
func Evaluate(trace ptrace.Traces, configProvider collector.OdigosConfigExtension) (bool, *commonapisampling.CostReductionRule) {
	matchingRules := map[string]*commonapisampling.CostReductionRule{}

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

				matchedRules := matchCostReductionRulesForSingleSpan(span, costReductionRules)
				spanLeastPercentageRule := selectCostReductionRuleFromMatches(matchedRules)

				if spanLeastPercentageRule != nil {
					setCostReductionRuleAttributesOnSpan(span, spanLeastPercentageRule)
				}
				if len(matchedRules) > 0 {
					for _, matchedRule := range matchedRules {
						matchingRules[matchedRule.Id] = matchedRule
					}
				}
			}
		}
	}

	decidingRule := calculateCostReductionDecidingRule(matchingRules)
	return decidingRule != nil, decidingRule
}

// matchCostReductionRulesForSingleSpan returns every cost-reduction rule whose operation matcher passes for this span.
func matchCostReductionRulesForSingleSpan(span ptrace.Span, costReductionRules []commonapisampling.CostReductionRule) []*commonapisampling.CostReductionRule {
	matchedRules := []*commonapisampling.CostReductionRule{}

	for _, rule := range costReductionRules {
		matched := matchers.TailSamplingOperationMatcher(rule.Operation, span)
		if matched {
			matchedRules = append(matchedRules, &rule)
		}
	}

	return matchedRules
}

// selectCostReductionRuleFromMatches picks the span-level rule using the same logic as the trace deciding rule
// (see calculateCostReductionDecidingRule): smallest PercentageAtMost among enabled matches.
func selectCostReductionRuleFromMatches(matchedRules []*commonapisampling.CostReductionRule) *commonapisampling.CostReductionRule {
	byID := make(map[string]*commonapisampling.CostReductionRule, len(matchedRules))
	for _, r := range matchedRules {
		byID[r.Id] = r
	}
	return calculateCostReductionDecidingRule(byID)
}

func getCostReductionRulesConfig(configProvider collector.OdigosConfigExtension, resource pcommon.Resource) []commonapisampling.CostReductionRule {
	cfg, found := configProvider.GetFromResource(resource)
	if !found {
		return nil
	}
	if cfg.TailSampling == nil {
		return nil
	}
	if len(cfg.TailSampling.CostReductionRules) == 0 {
		return nil
	}
	return cfg.TailSampling.CostReductionRules
}

// calculateCostReductionDecidingRule returns the enabled rule with the lowest PercentageAtMost (most restrictive).
// Used for the trace-level deciding rule and for per-span attributes after a span's matches are keyed by rule id.
func calculateCostReductionDecidingRule(matchingRules map[string]*commonapisampling.CostReductionRule) *commonapisampling.CostReductionRule {
	if len(matchingRules) == 0 {
		return nil
	}

	// check for lowest percentage enabled sampling rule.
	var selectedRule *commonapisampling.CostReductionRule
	for _, r := range matchingRules {
		if r.Disabled {
			continue
		}
		// shortcut for when we hit the bottom and don't need to keep iterating.
		if r.PercentageAtMost == 0 {
			return r
		}
		// update if it's the first one, or if it's less than the current smallest one.
		if selectedRule == nil || r.PercentageAtMost < selectedRule.PercentageAtMost {
			selectedRule = r
		}
	}
	return selectedRule
}

func setCostReductionRuleAttributesOnSpan(span ptrace.Span, rule *commonapisampling.CostReductionRule) {
	span.Attributes().PutStr(odigosattributes.SamplingSpanMatchingRuleId, rule.Id)
	span.Attributes().PutStr(odigosattributes.SamplingSpanMatchingRuleName, rule.Name)
	span.Attributes().PutDouble(odigosattributes.SamplingSpanMatchingRuleKeepPercentage, rule.PercentageAtMost)
}
