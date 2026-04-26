package category

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/matchers"
	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/common/collector"
	"github.com/odigos-io/odigos/common/odigosattributes"
)

// EvaluateHighlyRelevantOperations runs highly-relevant tail-sampling rules across all spans in the trace,
// sets per-span matching attributes, and returns whether a non-disabled deciding rule applies.
func EvaluateHighlyRelevantOperations(trace ptrace.Traces, configProvider collector.OdigosConfigExtension) (bool, *commonapisampling.HighlyRelevantOperation) {
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

				spanMostPercentageRule, matchedRules := processHighlyRelevantRulesForSingleSpan(span, highlyRelevantOperations)

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

// for a single span, evaluate all of the service highly relevant rules against the span.
// it will return the rule with the highest percentage that matched, and a list of all the rules that matched.
func processHighlyRelevantRulesForSingleSpan(span ptrace.Span, highlyRelevantOperations []commonapisampling.HighlyRelevantOperation) (*commonapisampling.HighlyRelevantOperation, []*commonapisampling.HighlyRelevantOperation) {
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

	if len(matchedRules) == 0 {
		return nil, nil
	}

	if len(matchedRules) == 1 {
		if matchedRules[0].Disabled {
			return nil, matchedRules
		}
		return matchedRules[0], matchedRules
	}

	var selectedRule *commonapisampling.HighlyRelevantOperation
	var selectedPercentage float64 = 101.0
	for _, rule := range matchedRules {
		if rule.Disabled {
			continue
		}
		percentage := GetPercentageOrDefault100(rule.PercentageAtLeast)
		if percentage < selectedPercentage {
			selectedRule = rule
			selectedPercentage = percentage
		}
	}
	return selectedRule, matchedRules
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

	if len(matchingRules) == 1 {
		for _, matchingRule := range matchingRules {
			if matchingRule.Disabled {
				return nil
			}
			return matchingRule
		}
	}

	var selectedRule *commonapisampling.HighlyRelevantOperation
	var selectedRulePercentage float64 = 0.0
	for _, matchingRule := range matchingRules {
		if matchingRule.Disabled {
			continue
		}
		percentage := GetPercentageOrDefault100(matchingRule.PercentageAtLeast)
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

func setHighlyRelevantRuleAttributesOnSpan(span ptrace.Span, rule *commonapisampling.HighlyRelevantOperation) {
	span.Attributes().PutStr(odigosattributes.SamplingSpanMatchingRuleId, rule.Id)
	span.Attributes().PutStr(odigosattributes.SamplingSpanMatchingRuleName, rule.Name)
	span.Attributes().PutDouble(odigosattributes.SamplingSpanMatchingRuleKeepPercentage, GetPercentageOrDefault100(rule.PercentageAtLeast))
}
