package category

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/matchers"
	commonapi "github.com/odigos-io/odigos/common/api"
	"github.com/odigos-io/odigos/common/collector"
)

type RuleMetrics struct {

	// number of spans on which we invoked this rule.
	SpanInvocationCount int

	// number of spans on which we matched this rule.
	SpanMatchingCount int

	// number of traces on which we invoked this rule at least once.
	TraceInvocationCount int

	// number of traces on which we matched this rule at least once.
	TraceMatchingCount int

	// for enabled rules, out of those that matched, how many traces/total-spans were kept?
	RuleTracesKeptCount     int
	RuleTotalSpansKeptCount int
}

func EvaluateHighlyRelevantOperations(td ptrace.Traces, configProvider collector.OdigosConfigExtension, tracePercentage float64) (bool, *commonapi.WorkloadHighlyRelevantOperation) {

	var mostPercentageRule *commonapi.WorkloadHighlyRelevantOperation
	var mostPercentagePercent float64 = 0.0

	rulesMetrics := make(map[string]RuleMetrics)
	totalSpansCount := 0

	// kept a count of all matching rules that were invoked.
	matchingRules := map[string]*commonapi.WorkloadHighlyRelevantOperation{}
	invoctedRules := map[string]struct{}{}

	rss := td.ResourceSpans()
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

				spanMostPercentageRule, matchedRules := processSpanHighlyRelevantRules(span, highlyRelevantOperations)
				if spanMostPercentageRule != nil {
					// rule matched, update the global one if needed
					currentPercentage := GetPercentageOrDefault100(spanMostPercentageRule.PercentageAtLeast)
					if mostPercentageRule == nil || currentPercentage > mostPercentagePercent {
						mostPercentagePercent = currentPercentage
						mostPercentageRule = spanMostPercentageRule
					}

					// TODO: check if span attributes annotations are enabled
					span.Attributes().PutStr("odigos.sampling.span.matching_rule.id", spanMostPercentageRule.Id)
					span.Attributes().PutDouble("odigos.sampling.span.matching_rule.percentage_at_least", GetPercentageOrDefault100(spanMostPercentageRule.PercentageAtLeast))
				}

				// capture metrics for all the invocations of theses rules
				for _, rule := range highlyRelevantOperations {

					if _, found := rulesMetrics[rule.Id]; !found {
						rulesMetrics[rule.Id] = RuleMetrics{}
					}
					metrics := rulesMetrics[rule.Id]
					metrics.SpanInvocationCount++

					invoctedRules[rule.Id] = struct{}{}
				}
				for _, matchedRule := range matchedRules {

					// rule id has already been created in the map already
					metrics := rulesMetrics[matchedRule.Id]
					metrics.SpanMatchingCount++

					matchingRules[matchedRule.Id] = matchedRule
				}
			}
		}
	}

	for ruleId := range invoctedRules {
		rulesMetrics[ruleId] = RuleMetrics{
			// marked it as invoked for metrics
			TraceInvocationCount: 1,
		}
	}

	var selectedRule *commonapi.WorkloadHighlyRelevantOperation
	var selectedRulePercentage float64 = 0.0
	for _, matchingRule := range matchingRules {
		percentage := GetPercentageOrDefault100(matchingRule.PercentageAtLeast)
		if selectedRule == nil || percentage > selectedRulePercentage {
			selectedRule = matchingRule
			selectedRulePercentage = percentage
		}
	}

	for _, matchingRule := range matchingRules {

		kept := tracePercentage >= GetPercentageOrDefault100(matchingRule.PercentageAtLeast)

		metrics := rulesMetrics[matchingRule.Id]
		metrics.TraceMatchingCount++
		if kept {
			metrics.RuleTracesKeptCount++
			metrics.RuleTotalSpansKeptCount += totalSpansCount
		}
	}

	return mostPercentageRule != nil, mostPercentageRule
}

func processSpanHighlyRelevantRules(span ptrace.Span, highlyRelevantOperations []commonapi.WorkloadHighlyRelevantOperation) (*commonapi.WorkloadHighlyRelevantOperation, []*commonapi.WorkloadHighlyRelevantOperation) {

	matchedRules := []*commonapi.WorkloadHighlyRelevantOperation{}

	for _, highlyRelevantOperation := range highlyRelevantOperations {

		// operation matching
		matched := matchers.TailSamplingOperationMatcher(highlyRelevantOperation.Operation, span)

		// error matching
		if highlyRelevantOperation.Error {
			errorMatched := span.Status().Code() == ptrace.StatusCodeError
			if !errorMatched {
				matched = false
			}
		}

		// duration matching
		if highlyRelevantOperation.DurationAtLeastMs != nil {
			currentSpanDurationNano := uint64(span.EndTimestamp() - span.StartTimestamp())
			ruleDurationNano := uint64(*highlyRelevantOperation.DurationAtLeastMs) * 1e6

			aboveThreshold := currentSpanDurationNano >= ruleDurationNano
			if !aboveThreshold {
				matched = false
			}
		}

		if matched {
			matchedRules = append(matchedRules, &highlyRelevantOperation)
		}
	}

	if len(matchedRules) == 0 {
		return nil, nil
	}
	if len(matchedRules) == 1 { // shortcut for common case
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
