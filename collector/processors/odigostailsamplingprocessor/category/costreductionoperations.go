package category

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/matchers"
	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/common/collector"
	"github.com/odigos-io/odigos/common/odigosattributes"
)

// EvaluateCostReductionOperations runs cost-reduction tail-sampling rules across all spans in the trace,
// sets per-span matching attributes, and returns whether a non-disabled deciding rule applies.
func EvaluateCostReductionOperations(trace ptrace.Traces, configProvider collector.OdigosConfigExtension) (bool, *commonapisampling.CostReductionRule) {
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

				spanLeastPercentageRule, matchedRules := processCostReductionRulesForSingleSpan(span, costReductionRules)

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

// for a single span, evaluate all of the cost reduction rules against the span.
// it will return the rule with the smallest percentage (at most semantics) that matched, and a list of all the rules that matched.
func processCostReductionRulesForSingleSpan(span ptrace.Span, costReductionRules []commonapisampling.CostReductionRule) (*commonapisampling.CostReductionRule, []*commonapisampling.CostReductionRule) {
	matchedRules := []*commonapisampling.CostReductionRule{}

	for _, rule := range costReductionRules {
		matched := matchers.TailSamplingOperationMatcher(rule.Operation, span)
		if matched {
			matchedRules = append(matchedRules, &rule)
		}
	}

	if len(matchedRules) == 0 {
		return nil, nil
	}

	if len(matchedRules) == 1 {
		if matchedRules[0].Disabled {
			return nil, matchedRules
		}
		return matchedRules[0], matchedRules
	}

	var selectedRule *commonapisampling.CostReductionRule
	var selectedPercentage float64 = 101.0
	for _, rule := range matchedRules {
		if rule.Disabled {
			continue
		}
		if rule.PercentageAtMost < selectedPercentage {
			selectedRule = rule
			selectedPercentage = rule.PercentageAtMost
		}
	}
	return selectedRule, matchedRules
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

// calculateCostReductionDecidingRule returns the rule with the lowest percentage (most restrictive).
func calculateCostReductionDecidingRule(matchingRules map[string]*commonapisampling.CostReductionRule) *commonapisampling.CostReductionRule {
	if len(matchingRules) == 0 {
		return nil
	}
	if len(matchingRules) == 1 {
		for _, r := range matchingRules {
			if r.Disabled {
				return nil
			}
			return r
		}
	}

	var selectedRule *commonapisampling.CostReductionRule
	var selectedPercentage float64 = 101.0
	for _, r := range matchingRules {
		if r.Disabled {
			continue
		}
		if r.PercentageAtMost < selectedPercentage {
			selectedRule = r
			selectedPercentage = r.PercentageAtMost
		}
	}
	return selectedRule
}

func setCostReductionRuleAttributesOnSpan(span ptrace.Span, rule *commonapisampling.CostReductionRule) {
	span.Attributes().PutStr(odigosattributes.SamplingSpanMatchingRuleId, rule.Id)
	span.Attributes().PutStr(odigosattributes.SamplingSpanMatchingRuleName, rule.Name)
	span.Attributes().PutDouble(odigosattributes.SamplingSpanMatchingRuleKeepPercentage, rule.PercentageAtMost)
}
