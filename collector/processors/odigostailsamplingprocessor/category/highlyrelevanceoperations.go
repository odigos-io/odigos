package category

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/matchers"
	commonapi "github.com/odigos-io/odigos/common/api"
	"github.com/odigos-io/odigos/common/collector"
)

type RuleMetrics struct {
	CommonRuleMetrics
	// for enabled rules, out of those that matched, how many traces/total-spans were kept?
	RuleTracesKeptCount     int
	RuleTotalSpansKeptCount int
}

func EvaluateHighlyRelevantOperations(trace ptrace.Traces, configProvider collector.OdigosConfigExtension, tracePercentage float64) (bool, *commonapi.WorkloadHighlyRelevantOperation, map[string]*RuleMetrics) {

	// keep a trace for metrics for running rules on this trace.
	// this map is not expected to be very large,
	// as each source should (ideally) have zero, or only few rules, and there is a lot of overlap.
	rulesMetrics := make(map[string]*RuleMetrics)
	totalSpansCount := 0

	// keep all the rules that matched here as they are evaluated in all the spans of the given trace.
	matchingRules := map[string]*commonapi.WorkloadHighlyRelevantOperation{}

	rss := trace.ResourceSpans()
	for i := 0; i < rss.Len(); i++ {
		res := rss.At(i)

		highlyRelevantOperations := getHighlyRelevantOperationsConfig(configProvider, res.Resource())
		if highlyRelevantOperations == nil {
			continue
		}

		scopes := res.ScopeSpans()
		for j := 0; j < scopes.Len(); j++ {
			scope := scopes.At(j)
			spans := scope.Spans()
			for k := 0; k < spans.Len(); k++ {

				span := spans.At(k)
				totalSpansCount++

				spanMostPercentageRule, matchedRules := processHighlyRelevantRulesForSingleSpan(span, highlyRelevantOperations)

				// capture metrics for invocations of theses rules
				recordMetricsInvocationsForSingleSpan(rulesMetrics, highlyRelevantOperations)

				if spanMostPercentageRule != nil {
					setMatchingRuleAttributesOnSpan(span, spanMostPercentageRule.Id, GetPercentageOrDefault100(spanMostPercentageRule.PercentageAtLeast))
					recordMetricsMatchingForSingleSpan(rulesMetrics, matchedRules)

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
	rulesMetrics = recordMetricsMatchingAndKept(rulesMetrics, matchingRules, tracePercentage, totalSpansCount)
	decidingRule := calculateDecidingRule(matchingRules)

	return decidingRule != nil, decidingRule, rulesMetrics
}

// record all the rules that invoked on a single span into the givin metrics.
// this function will update the rulesMetrics map in place.
func recordMetricsInvocationsForSingleSpan(rulesMetrics map[string]*RuleMetrics, highlyRelevantOperations []commonapi.WorkloadHighlyRelevantOperation) {
	for _, rule := range highlyRelevantOperations {
		metrics, found := rulesMetrics[rule.Id]
		if !found {
			metrics = &RuleMetrics{}
			rulesMetrics[rule.Id] = metrics
		}
		metrics.SpanCheckedCount++
	}
}

// record all the rules that matched on a single span into the givin metrics.
// this function will update the rulesMetrics map in place.
func recordMetricsMatchingForSingleSpan(rulesMetrics map[string]*RuleMetrics, matchedRules []*commonapi.WorkloadHighlyRelevantOperation) {
	for _, rule := range matchedRules {
		metrics := rulesMetrics[rule.Id]
		metrics.SpanMatchingCount++
	}
}

// for a single span, evaluate all of the service highly relevant rules against the span.
// it will return the rule with the highest percentage that matched, and a list of all the rules that matched.
func processHighlyRelevantRulesForSingleSpan(span ptrace.Span, highlyRelevantOperations []commonapi.WorkloadHighlyRelevantOperation) (*commonapi.WorkloadHighlyRelevantOperation, []*commonapi.WorkloadHighlyRelevantOperation) {

	// keep all the rules that matched, it will most likely contains 0 entries,
	// but occasionally 1 (when this span interacted with sampling), or a few values.
	matchedRules := []*commonapi.WorkloadHighlyRelevantOperation{}

	for _, highlyRelevantOperation := range highlyRelevantOperations {

		// try all the matchers in a single pass.
		// all of them must return true (AND logic) for the rule to match.
		// if they are not specified, they will default to true.
		matched := true
		matched = matched && matchers.TailSamplingOperationMatcher(highlyRelevantOperation.Operation, span)
		matched = matched && matchers.SpanErrorMatcher(span, highlyRelevantOperation.Error)
		matched = matched && matchers.SpanDurationMatcher(span, highlyRelevantOperation.DurationAtLeastMs)

		if matched {
			matchedRules = append(matchedRules, &highlyRelevantOperation)
		}
	}

	if len(matchedRules) == 0 {
		return nil, nil
	}
	if len(matchedRules) == 1 { // shortcut for common easy case
		return matchedRules[0], matchedRules
	}

	// find the rule with the highest percentage
	highestPercentageRule := matchedRules[0]
	highestPercentage := GetPercentageOrDefault100(highestPercentageRule.PercentageAtLeast)
	for _, rule := range matchedRules {
		rulePercentage := GetPercentageOrDefault100(rule.PercentageAtLeast)
		if rulePercentage > highestPercentage {
			highestPercentage = rulePercentage
			highestPercentageRule = rule
		}
	}

	return highestPercentageRule, matchedRules
}

func getHighlyRelevantOperationsConfig(configProvider collector.OdigosConfigExtension, resource pcommon.Resource) []commonapi.WorkloadHighlyRelevantOperation {
	cfg, found := configProvider.GetFromResource(resource)
	if !found {
		return nil
	}
	if cfg.TailSampling == nil {
		return nil
	}
	if len(cfg.TailSampling.HighlyRelevantOperations) == 0 {
		return nil
	}
	return cfg.TailSampling.HighlyRelevantOperations
}

// based on all the matching rules, find the one with the highest percentage.
// if multiple rules have the same highest percentage, one of them will be selected arbitrarily.
// this is used to mark spans with a single rule (most allowing) for sampling traceability.
func calculateDecidingRule(matchingRules map[string]*commonapi.WorkloadHighlyRelevantOperation) *commonapi.WorkloadHighlyRelevantOperation {
	if len(matchingRules) == 0 {
		return nil
	}

	// shortcut for common and easy case
	if len(matchingRules) == 1 {
		for _, matchingRule := range matchingRules {
			return matchingRule
		}
	}

	// pick the rule with the highest percentage.
	var selectedRule *commonapi.WorkloadHighlyRelevantOperation
	var selectedRulePercentage float64 = 0.0
	for _, matchingRule := range matchingRules {
		percentage := GetPercentageOrDefault100(matchingRule.PercentageAtLeast)
		// we don't need to continue once we found the first rule which is 100% (most permissive rule).
		if percentage == 100.0 {
			return matchingRule
		}
		if selectedRule == nil || percentage > selectedRulePercentage {
			selectedRule = matchingRule
			selectedRulePercentage = percentage
		}
	}
	return selectedRule
}

// recordMetricsMatchingAndKept updates rulesMetrics for each matching rule:
// - the trace is counted once for being matched by this rule.
// - if the rules decision for this trace is "keep", we count the trace once and number of spans in the "kept" metrics.
func recordMetricsMatchingAndKept(rulesMetrics map[string]*RuleMetrics, matchingRules map[string]*commonapi.WorkloadHighlyRelevantOperation, tracePercentage float64, totalSpansCount int) map[string]*RuleMetrics {
	for _, matchingRule := range matchingRules {
		kept := tracePercentage >= GetPercentageOrDefault100(matchingRule.PercentageAtLeast)
		metrics := rulesMetrics[matchingRule.Id] // rule has already been added when we marked the trace as matched by this rule.
		metrics.TraceMatchingCount++
		if kept {
			metrics.RuleTracesKeptCount++
			metrics.RuleTotalSpansKeptCount += totalSpansCount
		}
	}
	return rulesMetrics
}
