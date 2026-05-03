package highlyrelevant

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/category"
	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/matchers"
	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/common/collector"
	"github.com/odigos-io/odigos/common/odigosattributes"
)

// Evaluate:
// - checks all highly-relevant tail-sampling rules across all spans in the trace for matches,
// - compute a deciding rule based on the rules that matched,
// - returns wether this category matched, and the deciding rule if it did.
func Evaluate(trace ptrace.Traces, configProvider collector.OdigosConfigExtension) (bool, *commonapisampling.HighlyRelevantOperation) {
	matchingRules := map[string]*commonapisampling.HighlyRelevantOperation{}

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

				matchedRules := matchHighlyRelevantRulesForSingleSpan(span, highlyRelevantOperations)
				spanMostPercentageRule := selectHighlyRelevantRuleFromMatches(matchedRules)

				if spanMostPercentageRule != nil {
					setHighlyRelevantRuleAttributesOnSpan(span, spanMostPercentageRule)
				}
				if len(matchedRules) > 0 {
					for _, matchedRule := range matchedRules {
						matchingRules[matchedRule.Id] = matchedRule
					}
				}
			}
		}
	}

	decidingRule := calculateDecidingRule(matchingRules)
	return decidingRule != nil, decidingRule
}

// matchHighlyRelevantRulesForSingleSpan returns every highly-relevant rule whose matchers all pass for this span.
func matchHighlyRelevantRulesForSingleSpan(span ptrace.Span, highlyRelevantOperations []commonapisampling.HighlyRelevantOperation) []*commonapisampling.HighlyRelevantOperation {
	matchedRules := []*commonapisampling.HighlyRelevantOperation{}

	for _, highlyRelevantOperation := range highlyRelevantOperations {
		matched := true
		matched = matched && matchers.TailSamplingOperationMatcher(highlyRelevantOperation.Operation, span)
		matched = matched && matchers.SpanErrorMatcher(span, highlyRelevantOperation.Error)
		matched = matched && matchers.SpanDurationMatcher(span, highlyRelevantOperation.DurationAtLeastMs)

		if matched {
			matchedRules = append(matchedRules, &highlyRelevantOperation)
		}
	}

	return matchedRules
}

// selectHighlyRelevantRuleFromMatches picks the span-level rule using the same logic as the trace deciding rule
// (see calculateDecidingRule): highest PercentageAtLeast among enabled matches.
func selectHighlyRelevantRuleFromMatches(matchedRules []*commonapisampling.HighlyRelevantOperation) *commonapisampling.HighlyRelevantOperation {
	byID := make(map[string]*commonapisampling.HighlyRelevantOperation, len(matchedRules))
	for _, r := range matchedRules {
		byID[r.Id] = r
	}
	return calculateDecidingRule(byID)
}

func getHighlyRelevantOperationsConfig(configProvider collector.OdigosConfigExtension, resource pcommon.Resource) []commonapisampling.HighlyRelevantOperation {
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
func calculateDecidingRule(matchingRules map[string]*commonapisampling.HighlyRelevantOperation) *commonapisampling.HighlyRelevantOperation {
	if len(matchingRules) == 0 {
		return nil
	}

	var selectedRule *commonapisampling.HighlyRelevantOperation
	var selectedPercentage float64 = 0.0
	for _, matchingRule := range matchingRules {
		if matchingRule.Disabled {
			continue
		}
		percentage := category.GetPercentageOrDefault100(matchingRule.PercentageAtLeast)

		// once we hit maximum, no point in keeping iterating.
		if percentage == 100.0 {
			return matchingRule
		}

		// update if it's the first one, or if it's greater than the current largest one.
		if selectedRule == nil || percentage > selectedPercentage {
			selectedRule = matchingRule
			selectedPercentage = percentage
		}
	}
	return selectedRule // can be nil if all rules are disabled.
}

func setHighlyRelevantRuleAttributesOnSpan(span ptrace.Span, rule *commonapisampling.HighlyRelevantOperation) {
	span.Attributes().PutStr(odigosattributes.SamplingSpanMatchingRuleId, rule.Id)
	span.Attributes().PutStr(odigosattributes.SamplingSpanMatchingRuleName, rule.Name)
	span.Attributes().PutDouble(odigosattributes.SamplingSpanMatchingRuleKeepPercentage, category.GetPercentageOrDefault100(rule.PercentageAtLeast))
}
