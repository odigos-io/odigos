package samplingspanattrs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/category/config"
	"github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/common/odigosattributes"
)

func boolPtr(b bool) *bool { return &b }

func decidingRule() *config.ComputedRule {
	return &config.ComputedRule{RuleId: "cost-1", Name: "static assets", Percentage: 25}
}

// spanAttrsTestTrace builds a trace holding spanCount spans spread over more than one resource,
// more than one scope per resource and more than one span per scope, so a walk that only visits the
// first of any level is caught.
func spanAttrsTestTrace(spanCount int) ptrace.Traces {
	td := ptrace.NewTraces()
	for i := 0; i < spanCount; i++ {
		switch i % 3 {
		case 0:
			// a new resource with a new scope.
			rs := td.ResourceSpans().AppendEmpty()
			rs.ScopeSpans().AppendEmpty().Spans().AppendEmpty().SetName("span")
		case 1:
			// a second scope under the resource added above.
			rs := td.ResourceSpans().At(td.ResourceSpans().Len() - 1)
			rs.ScopeSpans().AppendEmpty().Spans().AppendEmpty().SetName("span")
		case 2:
			// a second span under the scope added above.
			rs := td.ResourceSpans().At(td.ResourceSpans().Len() - 1)
			scope := rs.ScopeSpans().At(rs.ScopeSpans().Len() - 1)
			scope.Spans().AppendEmpty().SetName("span")
		}
	}
	return td
}

func allSpanAttributes(td ptrace.Traces) []map[string]any {
	var out []map[string]any
	rss := td.ResourceSpans()
	for i := 0; i < rss.Len(); i++ {
		scopes := rss.At(i).ScopeSpans()
		for j := 0; j < scopes.Len(); j++ {
			spans := scopes.At(j).Spans()
			for k := 0; k < spans.Len(); k++ {
				out = append(out, spans.At(k).Attributes().AsRaw())
			}
		}
	}
	return out
}

func TestSetSpanMatchingRuleAttributesOnSpan(t *testing.T) {
	span := ptrace.NewSpan()
	SetSpanMatchingRuleAttributesOnSpan(span, decidingRule())

	assert.Equal(t, map[string]any{
		odigosattributes.SamplingSpanMatchingRuleId:             "cost-1",
		odigosattributes.SamplingSpanMatchingRuleName:           "static assets",
		odigosattributes.SamplingSpanMatchingRuleKeepPercentage: 25.0,
	}, span.Attributes().AsRaw())
}

// Every span of the trace has to carry the decision, not just the root or the first one: the
// attributes are what a user filters on to find the traces a rule affected.
func TestSetTraceSamplingAttributesOnSpansAnnotatesEverySpan(t *testing.T) {
	td := spanAttrsTestTrace(6)

	SetTraceSamplingAttributesOnSpans(td, consts.SamplingCategoryCostReduction, decidingRule(), false, true, nil)

	want := map[string]any{
		odigosattributes.SamplingCategory:                        "cost reduction",
		odigosattributes.SamplingTraceDecidingRuleId:             "cost-1",
		odigosattributes.SamplingTraceDecidingRuleName:           "static assets",
		odigosattributes.SamplingTraceDecidingRuleKeepPercentage: 25.0,
	}
	attrs := allSpanAttributes(td)
	require.Len(t, attrs, 6)
	for _, got := range attrs {
		assert.Equal(t, want, got)
	}
}

// Outside dry run the decision is not recorded on the span: a span that reached the exporter was
// kept by definition. The dry run attributes are the only way to see a would-be drop.
func TestSetTraceSamplingAttributesOnSpansDryRun(t *testing.T) {
	tests := []struct {
		name string
		kept bool
	}{
		{name: "trace that would have been kept", kept: true},
		{name: "trace that would have been dropped", kept: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			td := spanAttrsTestTrace(1)
			SetTraceSamplingAttributesOnSpans(td, consts.SamplingCategoryNoise, decidingRule(), true, tt.kept, nil)

			assert.Equal(t, map[string]any{
				odigosattributes.SamplingCategory:                        "noise",
				odigosattributes.SamplingTraceDecidingRuleId:             "cost-1",
				odigosattributes.SamplingTraceDecidingRuleName:           "static assets",
				odigosattributes.SamplingTraceDecidingRuleKeepPercentage: 25.0,
				odigosattributes.SamplingDryRun:                          true,
				odigosattributes.SamplingTraceKept:                       tt.kept,
			}, allSpanAttributes(td)[0])
		})
	}
}

// These toggles exist to cap the cost of the sampling attributes on high volume pipelines, so each
// one has to switch off exactly its own attributes.
func TestSetTraceSamplingAttributesOnSpansRespectsTheAttributeToggles(t *testing.T) {
	tests := []struct {
		name   string
		config *sampling.SpanSamplingAttributesConfiguration
		want   map[string]any
	}{
		{
			name:   "no configuration records everything",
			config: nil,
			want: map[string]any{
				odigosattributes.SamplingCategory:                        "cost reduction",
				odigosattributes.SamplingTraceDecidingRuleId:             "cost-1",
				odigosattributes.SamplingTraceDecidingRuleName:           "static assets",
				odigosattributes.SamplingTraceDecidingRuleKeepPercentage: 25.0,
			},
		},
		{
			name:   "empty configuration records everything",
			config: &sampling.SpanSamplingAttributesConfiguration{},
			want: map[string]any{
				odigosattributes.SamplingCategory:                        "cost reduction",
				odigosattributes.SamplingTraceDecidingRuleId:             "cost-1",
				odigosattributes.SamplingTraceDecidingRuleName:           "static assets",
				odigosattributes.SamplingTraceDecidingRuleKeepPercentage: 25.0,
			},
		},
		{
			name: "explicitly enabled toggles record everything",
			config: &sampling.SpanSamplingAttributesConfiguration{
				SamplingCategoryDisabled:  boolPtr(false),
				TraceDecidingRuleDisabled: boolPtr(false),
			},
			want: map[string]any{
				odigosattributes.SamplingCategory:                        "cost reduction",
				odigosattributes.SamplingTraceDecidingRuleId:             "cost-1",
				odigosattributes.SamplingTraceDecidingRuleName:           "static assets",
				odigosattributes.SamplingTraceDecidingRuleKeepPercentage: 25.0,
			},
		},
		{
			name: "disabling the category keeps the deciding rule",
			config: &sampling.SpanSamplingAttributesConfiguration{
				SamplingCategoryDisabled: boolPtr(true),
			},
			want: map[string]any{
				odigosattributes.SamplingTraceDecidingRuleId:             "cost-1",
				odigosattributes.SamplingTraceDecidingRuleName:           "static assets",
				odigosattributes.SamplingTraceDecidingRuleKeepPercentage: 25.0,
			},
		},
		{
			name: "disabling the deciding rule keeps the category",
			config: &sampling.SpanSamplingAttributesConfiguration{
				TraceDecidingRuleDisabled: boolPtr(true),
			},
			want: map[string]any{
				odigosattributes.SamplingCategory: "cost reduction",
			},
		},
		{
			name: "disabling both records nothing",
			config: &sampling.SpanSamplingAttributesConfiguration{
				SamplingCategoryDisabled:  boolPtr(true),
				TraceDecidingRuleDisabled: boolPtr(true),
			},
			want: map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			td := spanAttrsTestTrace(1)
			SetTraceSamplingAttributesOnSpans(td, consts.SamplingCategoryCostReduction, decidingRule(), false, true, tt.config)
			assert.Equal(t, tt.want, allSpanAttributes(td)[0])
		})
	}
}

// The dry run attributes are diagnostics that do not depend on the attribute toggles: without them
// a dry run with the toggles off would produce no evidence of what sampling would have done.
func TestSetTraceSamplingAttributesOnSpansDryRunIgnoresTheAttributeToggles(t *testing.T) {
	td := spanAttrsTestTrace(1)
	SetTraceSamplingAttributesOnSpans(td, consts.SamplingCategoryNoise, decidingRule(), true, false, &sampling.SpanSamplingAttributesConfiguration{
		SamplingCategoryDisabled:  boolPtr(true),
		TraceDecidingRuleDisabled: boolPtr(true),
	})

	assert.Equal(t, map[string]any{
		odigosattributes.SamplingDryRun:    true,
		odigosattributes.SamplingTraceKept: false,
	}, allSpanAttributes(td)[0])
}

// A rule name is optional, and an empty one must be omitted rather than recorded as an empty
// string, so queries on the attribute do not have to special case it.
func TestSetTraceSamplingAttributesOnSpansOmitsAnEmptyRuleName(t *testing.T) {
	td := spanAttrsTestTrace(1)
	SetTraceSamplingAttributesOnSpans(td, consts.SamplingCategoryNoise, &config.ComputedRule{RuleId: "noise-1", Percentage: 0}, false, true, nil)

	assert.Equal(t, map[string]any{
		odigosattributes.SamplingCategory:                        "noise",
		odigosattributes.SamplingTraceDecidingRuleId:             "noise-1",
		odigosattributes.SamplingTraceDecidingRuleKeepPercentage: 0.0,
	}, allSpanAttributes(td)[0])
}
