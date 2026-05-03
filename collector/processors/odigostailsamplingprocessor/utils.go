package odigostailsamplingprocessor

import (
	"errors"
	"strings"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/common/odigosattributes"
)

const odigosTraceStateKey = "odigos"

// getRootSpan finds and returns the root span of the trace.
// the trace should be all spans belonging to a single trace id,
// as reported by a "groupbytraceid" processor.
// returns the root span if found, the resource of the root span, and a boolean indicating if the root span was found.
func getRootSpan(trace ptrace.Traces) (ptrace.Span, pcommon.Resource, bool) {
	resourceSpans := trace.ResourceSpans()
	for i := 0; i < resourceSpans.Len(); i++ {
		resourceSpan := resourceSpans.At(i)
		scopeSpans := resourceSpan.ScopeSpans()
		for j := 0; j < scopeSpans.Len(); j++ {
			scopeSpan := scopeSpans.At(j)
			spans := scopeSpan.Spans()
			for k := 0; k < spans.Len(); k++ {
				span := spans.At(k)
				if span.ParentSpanID().IsEmpty() {
					return span, resourceSpan.Resource(), true
				}
			}
		}
	}
	return ptrace.Span{}, pcommon.Resource{}, false
}

// add few span attributes to all spans in the trace to indicate the sampling info.
func enrichSpansWithSamplingAttributes(td ptrace.Traces, category consts.SamplingCategory, ruleId string, ruleName string, keepPercentage float64, dryRun bool, kept bool, spanSamplingAttributes *sampling.SpanSamplingAttributesConfiguration) {

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
					span.Attributes().PutStr(odigosattributes.SamplingTraceDecidingRuleId, ruleId)
					span.Attributes().PutDouble(odigosattributes.SamplingTraceDecidingRuleKeepPercentage, keepPercentage)

					if ruleName != "" {
						span.Attributes().PutStr(odigosattributes.SamplingTraceDecidingRuleName, ruleName)
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

// checkPrerequists decides whether tail sampling should run on td.
// It assumes this processor runs after groupbytraceid, so all spans should share one trace ID.
//
// Validation:
//   - If spans disagree on trace ID, returns a non-nil error (misconfiguration or upstream bug).
//   - If any span’s W3C tracestate contains the odigos vendor entry (see odigosTraceStateKey),
//     head sampling already ran; returns (_, false, nil) so tail sampling is skipped.
//   - If there are no spans, returns (_, false, nil) so the batch is skipped.
//
// Returns:
//   - traceID: the common trace ID when shouldProcess is true, or the zero value when skipping or on error.
//   - shouldProcess: true only when there is at least one span, all share traceID, and none carry odigos tracestate.
//   - err: non-nil only when multiple trace IDs appear in the same batch.
func checkPrerequists(td ptrace.Traces) (pcommon.TraceID, bool, error) {

	var traceId pcommon.TraceID
	found := false

	for i := 0; i < td.ResourceSpans().Len(); i++ {
		resourceSpan := td.ResourceSpans().At(i)
		for j := 0; j < resourceSpan.ScopeSpans().Len(); j++ {
			scopeSpan := resourceSpan.ScopeSpans().At(j)
			for k := 0; k < scopeSpan.Spans().Len(); k++ {
				span := scopeSpan.Spans().At(k)

				currTraceId := span.TraceID()
				if !found {
					traceId = currTraceId
					found = true
				} else if currTraceId != traceId {
					return pcommon.TraceID{}, false, errors.New("not all spans belong to the same trace")
				}

				// check if we have odigos entry in the trace state, which indicates head sampling was applied.
				odigosTraceState := extractOdigosTraceStateValue(span.TraceState().AsRaw())
				if odigosTraceState != "" {
					// trace has already been sampled by head sampling, no need to process it again.
					return pcommon.TraceID{}, false, nil
				}
			}
		}
	}

	// if no spans found, we will return false to indicate it should be skipped.
	return traceId, found, nil
}

// extractOdigosTraceStateValue extracts the value of the "odigos" key from a W3C tracestate string.
// Tracestate format: "key1=value1,key2=value2,..."
func extractOdigosTraceStateValue(traceState string) string {
	for _, entry := range strings.Split(traceState, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		k, v, found := strings.Cut(entry, "=")
		if !found {
			continue
		}
		if strings.TrimSpace(k) == odigosTraceStateKey {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
