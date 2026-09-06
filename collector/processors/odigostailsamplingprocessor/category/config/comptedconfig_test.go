package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/otel/attribute"

	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/common/odigosattributes"
)

func percentagePtr(f float64) *float64 { return &f }

func intPointer(i int) *int { return &i }

func computedRuleTestSpan() ptrace.Span {
	span := ptrace.NewSpan()
	span.SetKind(ptrace.SpanKindServer)
	span.Attributes().PutStr("http.request.method", "GET")
	span.Attributes().PutStr("http.route", "/checkout")
	return span
}

func TestGetPercentageOrDefault(t *testing.T) {
	assert.Equal(t, 42.0, GetPercentageOrDefault(percentagePtr(42), 7))
	assert.Equal(t, 7.0, GetPercentageOrDefault(nil, 7))

	// an explicit zero must survive; falling back to the default here would silently keep traces
	// the user asked to drop entirely.
	assert.Equal(t, 0.0, GetPercentageOrDefault(percentagePtr(0), 7))
	assert.Equal(t, 0.0, GetPercentageOrDefault100(percentagePtr(0)))

	assert.Equal(t, 0.0, GetPercentageOrDefault0(nil))
	assert.Equal(t, 100.0, GetPercentageOrDefault100(nil))
}

// The per-category defaults are what an omitted percentage means, and the two are opposites: a
// noisy rule without a percentage drops the traffic it matches, while a highly relevant rule
// without a percentage keeps all of it. Swapping them would either drop every trace the user
// flagged as important or keep every trace they flagged as noise.
func TestPrecomputeWorkloadConfigPercentageDefaults(t *testing.T) {
	computed := precomputeWorkloadConfig(&commonapisampling.TailSamplingSourceConfig{
		NoisyOperations: []commonapisampling.NoisyOperation{
			{Id: "noise-default"},
			{Id: "noise-explicit", PercentageAtMost: percentagePtr(30)},
		},
		HighlyRelevantOperations: []commonapisampling.HighlyRelevantOperation{
			{Id: "hr-default"},
			{Id: "hr-explicit", PercentageAtLeast: percentagePtr(30)},
		},
		CostReductionRules: []commonapisampling.CostReductionRule{
			{Id: "cost-zero"},
			{Id: "cost-explicit", PercentageAtMost: 30},
		},
	}, false)

	require.Len(t, computed.NoisyOperations, 2)
	assert.Equal(t, 0.0, computed.NoisyOperations[0].Percentage)
	assert.Equal(t, 30.0, computed.NoisyOperations[1].Percentage)

	require.Len(t, computed.HighlyRelevantOperations, 2)
	assert.Equal(t, 100.0, computed.HighlyRelevantOperations[0].Percentage)
	assert.Equal(t, 30.0, computed.HighlyRelevantOperations[1].Percentage)

	require.Len(t, computed.CostReductionRules, 2)
	assert.Equal(t, 0.0, computed.CostReductionRules[0].Percentage)
	assert.Equal(t, 30.0, computed.CostReductionRules[1].Percentage)
}

func TestPrecomputeWorkloadConfigCopiesRuleIdentity(t *testing.T) {
	computed := precomputeWorkloadConfig(&commonapisampling.TailSamplingSourceConfig{
		NoisyOperations:          []commonapisampling.NoisyOperation{{Id: "noise-1", Name: "health checks", Disabled: true}},
		HighlyRelevantOperations: []commonapisampling.HighlyRelevantOperation{{Id: "hr-1", Name: "checkout errors"}},
		CostReductionRules:       []commonapisampling.CostReductionRule{{Id: "cost-1", Name: "static assets", Disabled: true}},
	}, false)

	assert.Equal(t, "noise-1", computed.NoisyOperations[0].RuleId)
	assert.Equal(t, "health checks", computed.NoisyOperations[0].Name)
	assert.True(t, computed.NoisyOperations[0].Disabled)

	assert.Equal(t, "hr-1", computed.HighlyRelevantOperations[0].RuleId)
	assert.Equal(t, "checkout errors", computed.HighlyRelevantOperations[0].Name)
	assert.False(t, computed.HighlyRelevantOperations[0].Disabled)

	assert.Equal(t, "cost-1", computed.CostReductionRules[0].RuleId)
	assert.Equal(t, "static assets", computed.CostReductionRules[0].Name)
	assert.True(t, computed.CostReductionRules[0].Disabled)
}

func TestPrecomputeWorkloadConfigWithoutRules(t *testing.T) {
	computed := precomputeWorkloadConfig(&commonapisampling.TailSamplingSourceConfig{}, false)
	assert.Empty(t, computed.NoisyOperations)
	assert.Empty(t, computed.HighlyRelevantOperations)
	assert.Empty(t, computed.CostReductionRules)
}

// A rule with no operation matcher matches every span in the trace. That is the difference between
// "sample this endpoint at 10%" and "sample the whole service at 10%".
func TestPrecomputeWorkloadConfigRuleWithoutAnOperationMatchesAnySpan(t *testing.T) {
	computed := precomputeWorkloadConfig(&commonapisampling.TailSamplingSourceConfig{
		NoisyOperations:          []commonapisampling.NoisyOperation{{Id: "noise-1"}},
		HighlyRelevantOperations: []commonapisampling.HighlyRelevantOperation{{Id: "hr-1"}},
		CostReductionRules:       []commonapisampling.CostReductionRule{{Id: "cost-1"}},
	}, false)

	bareSpan := ptrace.NewSpan()
	assert.True(t, computed.NoisyOperations[0].Matcher.Match(bareSpan))
	assert.True(t, computed.HighlyRelevantOperations[0].Matcher.Match(bareSpan))
	assert.True(t, computed.CostReductionRules[0].Matcher.Match(bareSpan))
}

// The highly relevant category is the only one that can combine an operation with an error and a
// duration condition, and all three have to hold for the rule to match.
func TestPrecomputeHighlyRelevantOperationsBuildsAConjunction(t *testing.T) {
	computed := precomputeWorkloadConfig(&commonapisampling.TailSamplingSourceConfig{
		HighlyRelevantOperations: []commonapisampling.HighlyRelevantOperation{{
			Id:                "hr-1",
			Error:             true,
			DurationAtLeastMs: intPointer(100),
			Operation: &commonapisampling.TailSamplingOperationMatcher{
				HttpServer: &commonapisampling.TailSamplingHttpServerOperationMatcher{Route: "/checkout"},
			},
		}},
	}, false)
	require.Len(t, computed.HighlyRelevantOperations, 1)
	matcher := computed.HighlyRelevantOperations[0].Matcher

	slowErrorOnRoute := computedRuleTestSpan()
	slowErrorOnRoute.Status().SetCode(ptrace.StatusCodeError)
	slowErrorOnRoute.SetEndTimestamp(pcommon.Timestamp(200 * 1e6))
	assert.True(t, matcher.Match(slowErrorOnRoute))

	slowOnRouteWithoutError := computedRuleTestSpan()
	slowOnRouteWithoutError.SetEndTimestamp(pcommon.Timestamp(200 * 1e6))
	assert.False(t, matcher.Match(slowOnRouteWithoutError))

	fastErrorOnRoute := computedRuleTestSpan()
	fastErrorOnRoute.Status().SetCode(ptrace.StatusCodeError)
	fastErrorOnRoute.SetEndTimestamp(pcommon.Timestamp(50 * 1e6))
	assert.False(t, matcher.Match(fastErrorOnRoute))

	slowErrorOnAnotherRoute := computedRuleTestSpan()
	slowErrorOnAnotherRoute.Attributes().PutStr("http.route", "/cart")
	slowErrorOnAnotherRoute.Status().SetCode(ptrace.StatusCodeError)
	slowErrorOnAnotherRoute.SetEndTimestamp(pcommon.Timestamp(200 * 1e6))
	assert.False(t, matcher.Match(slowErrorOnAnotherRoute))
}

// These attribute sets are the only dimensions the sampling metrics are broken down by, so a
// missing one makes a rule indistinguishable from another in the dashboards.
func TestComputeRuleMetricsAttributes(t *testing.T) {
	tests := []struct {
		name     string
		disabled bool
		dryRun   bool
		want     map[string]any
	}{
		{
			name: "enabled rule outside dry run",
			want: map[string]any{
				odigosattributes.SamplingCategory: "noise",
				odigosattributes.SamplingRuleId:   "noise-1",
				odigosattributes.SamplingRuleName: "health checks",
			},
		},
		{
			name:     "disabled rule is marked",
			disabled: true,
			want: map[string]any{
				odigosattributes.SamplingCategory:     "noise",
				odigosattributes.SamplingRuleId:       "noise-1",
				odigosattributes.SamplingRuleName:     "health checks",
				odigosattributes.SamplingRuleDisabled: true,
			},
		},
		{
			name:   "dry run is marked",
			dryRun: true,
			want: map[string]any{
				odigosattributes.SamplingCategory: "noise",
				odigosattributes.SamplingRuleId:   "noise-1",
				odigosattributes.SamplingRuleName: "health checks",
				odigosattributes.SamplingDryRun:   true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set := compteRuleMetricsAttributes("noise", "noise-1", "health checks", tt.disabled, tt.dryRun)

			got := map[string]any{}
			for _, kv := range set.ToSlice() {
				switch kv.Value.Type() {
				case attribute.BOOL:
					got[string(kv.Key)] = kv.Value.AsBool()
				default:
					got[string(kv.Key)] = kv.Value.AsString()
				}
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

// The dry run flag reaches the rules through the cache, so a rule computed in dry run mode must
// carry it; otherwise dry run and committed decisions land on the same metric series.
func TestPrecomputeWorkloadConfigPropagatesDryRunToRuleMetrics(t *testing.T) {
	computed := precomputeWorkloadConfig(&commonapisampling.TailSamplingSourceConfig{
		NoisyOperations:          []commonapisampling.NoisyOperation{{Id: "noise-1"}},
		HighlyRelevantOperations: []commonapisampling.HighlyRelevantOperation{{Id: "hr-1"}},
		CostReductionRules:       []commonapisampling.CostReductionRule{{Id: "cost-1"}},
	}, true)

	for _, rule := range []ComputedRule{
		computed.NoisyOperations[0],
		computed.HighlyRelevantOperations[0],
		computed.CostReductionRules[0],
	} {
		value, found := rule.MetricsAttributes.Value(odigosattributes.SamplingDryRun)
		require.True(t, found, "rule %q is missing the dry run attribute", rule.RuleId)
		assert.True(t, value.AsBool())
	}
}

// Each category must tag its rules with its own category name, or the per-category breakdown of the
// sampling metrics collapses.
func TestPrecomputeWorkloadConfigTagsRulesWithTheirCategory(t *testing.T) {
	computed := precomputeWorkloadConfig(&commonapisampling.TailSamplingSourceConfig{
		NoisyOperations:          []commonapisampling.NoisyOperation{{Id: "noise-1"}},
		HighlyRelevantOperations: []commonapisampling.HighlyRelevantOperation{{Id: "hr-1"}},
		CostReductionRules:       []commonapisampling.CostReductionRule{{Id: "cost-1"}},
	}, false)

	categoryOf := func(rule ComputedRule) string {
		value, found := rule.MetricsAttributes.Value(odigosattributes.SamplingCategory)
		require.True(t, found)
		return value.AsString()
	}

	assert.Equal(t, "noise", categoryOf(computed.NoisyOperations[0]))
	assert.Equal(t, "highly relevant", categoryOf(computed.HighlyRelevantOperations[0]))
	assert.Equal(t, "cost reduction", categoryOf(computed.CostReductionRules[0]))
}
