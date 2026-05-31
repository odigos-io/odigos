package highlyrelevant

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/category"
	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/category/config"
	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/category/metrics"
	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/category/samplingspanattrs"
	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/matchers"
)

type HighlyRelevantEvaluationResult struct {
	DecidingRule     *config.ComputedRule
	RulesEvalResults category.CategoryRulesEvaluationResults
}

// Evaluate:
// - checks all highly-relevant tail-sampling rules across all spans in the trace for matches,
// - compute a deciding rule based on the rules that matched,
// - returns wether this category matched, and the deciding rule if it did.
func Evaluate(trace ptrace.Traces, configProvider config.TailSamplingConfigProvider) HighlyRelevantEvaluationResult {
	matchingRules := map[string]*config.ComputedRule{}
	rulesEvalResults := map[string]*category.RuleEvaluationResult{}

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

				matchedRules := matchHighlyRelevantRulesForSingleSpan(rulesEvalResults, matchingRules, span, highlyRelevantOperations)
				spanMostPercentageRule := selectHighlyRelevantRuleFromMatches(matchedRules)

				if spanMostPercentageRule != nil {
					samplingspanattrs.SetSpanMatchingRuleAttributesOnSpan(span, spanMostPercentageRule)
				}
			}
		}
	}

	decidingRule := calculateDecidingRule(matchingRules)
	return HighlyRelevantEvaluationResult{
		DecidingRule:     decidingRule,
		RulesEvalResults: rulesEvalResults,
	}
}

// matchHighlyRelevantRulesForSingleSpan returns every highly-relevant rule whose matchers all pass for this span.
// it also updates the rulesEvalResults and matchingRules maps based on the matched rules.
func matchHighlyRelevantRulesForSingleSpan(rulesEvalResults map[string]*category.RuleEvaluationResult, matchingRules map[string]*config.ComputedRule, span ptrace.Span, highlyRelevantOperations []config.ComputedHighlyRelevantOperation) []*config.ComputedRule {
	matchedRules := []*config.ComputedRule{}

	for _, highlyRelevantOperation := range highlyRelevantOperations {
		matched := true
		matched = matched && matchers.TailSamplingOperationMatcher(highlyRelevantOperation.Rule.Operation, span)
		matched = matched && matchers.SpanErrorMatcher(span, highlyRelevantOperation.Rule.Error)
		matched = matched && matchers.SpanDurationMatcher(span, highlyRelevantOperation.Rule.DurationAtLeastMs)

		metrics.RecordEvalResultForSingleSpan(rulesEvalResults, highlyRelevantOperation.ComputedRule, matched)

		if matched {
			matchedRules = append(matchedRules, &highlyRelevantOperation.ComputedRule)
			if _, found := matchingRules[highlyRelevantOperation.ComputedRule.RuleId]; !found {
				matchingRules[highlyRelevantOperation.ComputedRule.RuleId] = &highlyRelevantOperation.ComputedRule
			}
		}
	}

	return matchedRules
}

// selectHighlyRelevantRuleFromMatches picks the span-level rule using the same logic as the trace deciding rule
// (see calculateDecidingRule): highest PercentageAtLeast among enabled matches.
func selectHighlyRelevantRuleFromMatches(matchedRules []*config.ComputedRule) *config.ComputedRule {
	byID := make(map[string]*config.ComputedRule, len(matchedRules))
	for _, r := range matchedRules {
		byID[r.RuleId] = r
	}
	return calculateDecidingRule(byID)
}

func getHighlyRelevantOperationsConfig(configProvider config.TailSamplingConfigProvider, resource pcommon.Resource) []config.ComputedHighlyRelevantOperation {
	tailSampling, found := configProvider.GetTailSamplingConfig(resource)
	if !found || tailSampling == nil {
		return nil
	}
	if len(tailSampling.HighlyRelevantOperations) == 0 {
		return nil
	}
	return tailSampling.HighlyRelevantOperations
}

// based on all the matching rules, find the one with the highest percentage.
// if multiple rules have the same highest percentage, one of them will be selected arbitrarily.
// this is used to mark spans with a single rule (most allowing) for sampling traceability.
func calculateDecidingRule(matchingRules map[string]*config.ComputedRule) *config.ComputedRule {
	if len(matchingRules) == 0 {
		return nil
	}

	var selectedRule *config.ComputedRule
	for _, matchingRule := range matchingRules {
		if matchingRule.Disabled {
			continue
		}
		percentage := matchingRule.Percentage

		// once we hit maximum, no point in keeping iterating.
		if percentage == 100.0 {
			return matchingRule
		}

		// update if it's the first one, or if it's greater than the current largest one.
		if selectedRule == nil || percentage > selectedRule.Percentage {
			selectedRule = matchingRule
		}
	}
	return selectedRule // can be nil if all rules are disabled.
}
