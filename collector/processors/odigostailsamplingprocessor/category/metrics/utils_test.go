package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"

	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/category"
	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/category/config"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/common/odigosattributes"
)

// The checked and matched counters are what the sampling dashboards divide to show a rule's hit
// rate, so a span that was evaluated must be counted exactly once whether or not it matched.
func TestRecordEvalResultForSingleSpan(t *testing.T) {
	results := map[string]*category.RuleEvaluationResult{}
	rule := config.ComputedRule{RuleId: "cost-1", Name: "static assets", Percentage: 25}

	RecordEvalResultForSingleSpan(results, rule, true)
	RecordEvalResultForSingleSpan(results, rule, false)
	RecordEvalResultForSingleSpan(results, rule, true)

	require.Contains(t, results, "cost-1")
	assert.Equal(t, 3, results["cost-1"].SpanCheckedCount)
	assert.Equal(t, 2, results["cost-1"].SpanMatchedCount)
	assert.Equal(t, rule, results["cost-1"].ComputedRule)
}

func TestRecordEvalResultForSingleSpanKeepsRulesApart(t *testing.T) {
	results := map[string]*category.RuleEvaluationResult{}

	RecordEvalResultForSingleSpan(results, config.ComputedRule{RuleId: "cost-1"}, true)
	RecordEvalResultForSingleSpan(results, config.ComputedRule{RuleId: "cost-2"}, false)

	assert.Equal(t, 1, results["cost-1"].SpanMatchedCount)
	assert.Zero(t, results["cost-2"].SpanMatchedCount)
	assert.Len(t, results, 2)
}

func TestCategoryMetricsAttributeSet(t *testing.T) {
	tests := []struct {
		name     string
		category consts.SamplingCategory
		dryRun   bool
		want     map[string]any
	}{
		{
			name:     "noise outside dry run",
			category: consts.SamplingCategoryNoise,
			want:     map[string]any{odigosattributes.SamplingCategory: "noise"},
		},
		{
			name:     "highly relevant in dry run",
			category: consts.SamplingCategoryHighlyRelevant,
			dryRun:   true,
			want: map[string]any{
				odigosattributes.SamplingCategory: "highly relevant",
				odigosattributes.SamplingDryRun:   true,
			},
		},
		{
			name:     "cost reduction outside dry run",
			category: consts.SamplingCategoryCostReduction,
			want:     map[string]any{odigosattributes.SamplingCategory: "cost reduction"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := map[string]any{}
			set := CategoryMetricsAttributeSet(tt.category, tt.dryRun)
			for _, kv := range set.ToSlice() {
				if kv.Value.Type() == attribute.BOOL {
					got[string(kv.Key)] = kv.Value.AsBool()
					continue
				}
				got[string(kv.Key)] = kv.Value.AsString()
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
