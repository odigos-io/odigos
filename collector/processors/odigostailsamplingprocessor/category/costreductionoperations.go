package category

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/matchers"
	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/common/collector"
)

// CostReductionRuleMetrics holds per-rule metrics for cost reduction (percentage-at-most semantics).
// "Dropped" means the trace matched the rule but was dropped (trace percentage above rule threshold).
type CostReductionRuleMetrics struct {
	CommonRuleMetrics
	// for rules that matched, how many traces/total-spans were dropped (trace percentage > rule threshold)?
	RuleTracesDroppedCount     int
	RuleTotalSpansDroppedCount int
}

func EvaluateCostReductionOperations(trace ptrace.Traces, configProvider collector.OdigosConfigExtension, tracePercentage float64) (bool, *commonapisampling.CostReductionRule, map[string]*CostReductionRuleMetrics) {

	rulesMetrics := make(map[string]*CostReductionRuleMetrics)
	totalSpansCount := 0

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
				totalSpansCount++

				spanLeastPercentageRule, matchedRules := processCostReductionRulesForSingleSpan(span, costReductionRules)

				recordCostReductionMetricsInvocationsForSingleSpan(rulesMetrics, costReductionRules)

				if spanLeastPercentageRule != nil {
					setCostReductionRuleAttributesOnSpan(span, spanLeastPercentageRule)
				}
				if len(matchedRules) > 0 {
					recordCostReductionMetricsMatchingForSingleSpan(rulesMetrics, matchedRules)

					// update the map that tracks all the rules that matched for this trace,
					// so we can calculate combined result.
					for _, matchedRule := range matchedRules {
						matchingRules[matchedRule.Id] = matchedRule
					}
				}
			}
		}
	}

	for _, metrics := range rulesMetrics {
		metrics.TraceCheckedCount = 1
	}

	rulesMetrics = recordCostReductionMetricsMatchingAndDropped(rulesMetrics, matchingRules, tracePercentage, totalSpansCount)
	decidingRule := calculateCostReductionDecidingRule(matchingRules)

	return decidingRule != nil, decidingRule, rulesMetrics
}

func recordCostReductionMetricsInvocationsForSingleSpan(rulesMetrics map[string]*CostReductionRuleMetrics, costReductionRules []commonapisampling.CostReductionRule) {
	for _, rule := range costReductionRules {
		metrics, found := rulesMetrics[rule.Id]
		if !found {
			metrics = &CostReductionRuleMetrics{}
			rulesMetrics[rule.Id] = metrics
		}
		metrics.SpanCheckedCount++
	}
}

func recordCostReductionMetricsMatchingForSingleSpan(rulesMetrics map[string]*CostReductionRuleMetrics, matchedRules []*commonapisampling.CostReductionRule) {
	for _, rule := range matchedRules {
		metrics := rulesMetrics[rule.Id]
		metrics.SpanMatchingCount++
	}
}

// for a single span, evaluate all of the cost reduction rules against the span.
// it will return the rule with the smallest percentage (at most semantics) that matched, and a list of all the rules that matched.
// matching rules can also be disabled (and should be ignored or used where appropriate)
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

	// ignore disabled rules for the "decision" rule selection.
	if len(matchedRules) == 1 {
		if matchedRules[0].Disabled {
			return nil, matchedRules
		} else {
			return matchedRules[0], matchedRules
		}
	}

	// percentage at most: lowest percentage wins (most restrictive)
	var selectedRule *commonapisampling.CostReductionRule
	var selectedPercentage float64 = 101.0
	for _, rule := range matchedRules {
		if rule.Disabled {
			continue
		}
		// although it is possible, we are not expecting 0 here, as the intent is cost reduction, not noisy reduction.
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
			} else {
				return r
			}
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

// recordCostReductionMetricsMatchingAndDropped updates rulesMetrics for each matching rule:
// - the trace is counted once for being matched by this rule.
// - if the rule's decision is "drop" (trace percentage > rule threshold), we count trace and spans in dropped metrics.
func recordCostReductionMetricsMatchingAndDropped(rulesMetrics map[string]*CostReductionRuleMetrics, matchingRules map[string]*commonapisampling.CostReductionRule, tracePercentage float64, totalSpansCount int) map[string]*CostReductionRuleMetrics {
	for _, rule := range matchingRules {
		dropped := tracePercentage > rule.PercentageAtMost
		metrics := rulesMetrics[rule.Id]
		metrics.TraceMatchingCount++
		if dropped {
			metrics.RuleTracesDroppedCount++
			metrics.RuleTotalSpansDroppedCount += totalSpansCount
		}
	}
	return rulesMetrics
}

func setCostReductionRuleAttributesOnSpan(span ptrace.Span, rule *commonapisampling.CostReductionRule) {
	span.Attributes().PutStr("odigos.sampling.span.matching_rule.id", rule.Id)
	span.Attributes().PutStr("odigos.sampling.span.matching_rule.name", rule.Name)
	span.Attributes().PutDouble("odigos.sampling.span.matching_rule.percentage_at_most", rule.PercentageAtMost)
}
