package samplingspanattrs

import (
	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/category/config"
	"github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/common/odigosattributes"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func SetSpanMatchingRuleAttributesOnSpan(span ptrace.Span, rule *config.ComputedRule) {
	span.Attributes().PutStr(odigosattributes.SamplingSpanMatchingRuleId, rule.RuleId)
	span.Attributes().PutStr(odigosattributes.SamplingSpanMatchingRuleName, rule.Name)
	span.Attributes().PutDouble(odigosattributes.SamplingSpanMatchingRuleKeepPercentage, rule.Percentage)
}

// add few span attributes to all spans in the trace to indicate the sampling info.
func SetTraceSamplingAttributesOnSpans(td ptrace.Traces, category consts.SamplingCategory, decidingRule *config.ComputedRule, dryRun bool, kept bool, spanSamplingAttributes *sampling.SpanSamplingAttributesConfiguration) {

	recordCategoryEnabled := spanSamplingAttributes == nil || spanSamplingAttributes.SamplingCategoryDisabled == nil || !*spanSamplingAttributes.SamplingCategoryDisabled
	recordTraceDecidingRuleEnabled := spanSamplingAttributes == nil || spanSamplingAttributes.TraceDecidingRuleDisabled == nil || !*spanSamplingAttributes.TraceDecidingRuleDisabled

	for i := 0; i < td.ResourceSpans().Len(); i++ {
		resourceSpan := td.ResourceSpans().At(i)
		scopeSpans := resourceSpan.ScopeSpans()
		for j := 0; j < scopeSpans.Len(); j++ {
			scopeSpan := scopeSpans.At(j)
			spans := scopeSpan.Spans()
			for k := 0; k < spans.Len(); k++ {
				span := spans.At(k)

				if recordCategoryEnabled {
					span.Attributes().PutStr(odigosattributes.SamplingCategory, string(category))
				}

				if recordTraceDecidingRuleEnabled {
					span.Attributes().PutStr(odigosattributes.SamplingTraceDecidingRuleId, decidingRule.RuleId)
					span.Attributes().PutDouble(odigosattributes.SamplingTraceDecidingRuleKeepPercentage, decidingRule.Percentage)

					if decidingRule.Name != "" {
						span.Attributes().PutStr(odigosattributes.SamplingTraceDecidingRuleName, decidingRule.Name)
					}
				}

				if dryRun {
					span.Attributes().PutBool(odigosattributes.SamplingDryRun, dryRun)
					span.Attributes().PutBool(odigosattributes.SamplingTraceKept, kept) // can be false to indicate this trace would have been dropped.
				}
			}
		}
	}
}
