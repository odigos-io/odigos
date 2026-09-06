package costreduction

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/category/config"
	"github.com/odigos-io/odigos/common/odigosattributes"
)

const costTestWorkloadAttr = "odigos.test.workload.key"

// nameMatcher matches a span by name, which keeps these tests about rule selection rather than
// about the attribute matchers (those are covered in the matchers package).
type nameMatcher struct{ name string }

func (m nameMatcher) Match(span ptrace.Span) bool { return span.Name() == m.name }

type anySpanMatcher struct{}

func (anySpanMatcher) Match(ptrace.Span) bool { return true }

// costTestConfigProvider resolves the workload config from a resource attribute so one trace can
// carry resources belonging to workloads with different rules.
type costTestConfigProvider map[string]*config.ComputedWorkloadConfig

func (p costTestConfigProvider) GetTailSamplingConfig(resource pcommon.Resource) (*config.ComputedWorkloadConfig, bool) {
	key, found := resource.Attributes().Get(costTestWorkloadAttr)
	if !found {
		return nil, false
	}
	cfg, found := p[key.Str()]
	return cfg, found
}

func costTestRule(id string, percentage float64, matchName string) config.ComputedRule {
	return config.ComputedRule{
		RuleId:     id,
		Name:       id,
		Percentage: percentage,
		Matcher:    nameMatcher{name: matchName},
	}
}

func costTestTrace(workloadSpans map[string][]string) ptrace.Traces {
	td := ptrace.NewTraces()
	for workload, spanNames := range workloadSpans {
		rs := td.ResourceSpans().AppendEmpty()
		rs.Resource().Attributes().PutStr(costTestWorkloadAttr, workload)
		scope := rs.ScopeSpans().AppendEmpty()
		for _, name := range spanNames {
			scope.Spans().AppendEmpty().SetName(name)
		}
	}
	return td
}

func spanByName(t *testing.T, td ptrace.Traces, name string) ptrace.Span {
	t.Helper()
	rss := td.ResourceSpans()
	for i := 0; i < rss.Len(); i++ {
		scopes := rss.At(i).ScopeSpans()
		for j := 0; j < scopes.Len(); j++ {
			spans := scopes.At(j).Spans()
			for k := 0; k < spans.Len(); k++ {
				if spans.At(k).Name() == name {
					return spans.At(k)
				}
			}
		}
	}
	t.Fatalf("no span named %q in the trace", name)
	return ptrace.Span{}
}

func TestEvaluateWithoutConfigForTheWorkload(t *testing.T) {
	td := costTestTrace(map[string][]string{"checkout": {"GET /assets/app.js"}})

	res := Evaluate(td, costTestConfigProvider{})
	assert.Nil(t, res.DecidingRule)
	assert.Empty(t, res.RulesEvalResults)
}

func TestEvaluateWithoutCostReductionRules(t *testing.T) {
	td := costTestTrace(map[string][]string{"checkout": {"GET /assets/app.js"}})
	provider := costTestConfigProvider{"checkout": {
		HighlyRelevantOperations: []config.ComputedRule{{RuleId: "hr-1", Matcher: anySpanMatcher{}}},
	}}

	res := Evaluate(td, provider)
	assert.Nil(t, res.DecidingRule)
	assert.Empty(t, res.RulesEvalResults, "highly relevant rules must not be evaluated by this category")
}

func TestEvaluateWithoutAMatch(t *testing.T) {
	td := costTestTrace(map[string][]string{"checkout": {"GET /checkout"}})
	provider := costTestConfigProvider{"checkout": {
		CostReductionRules: []config.ComputedRule{costTestRule("cost-1", 10, "GET /assets/app.js")},
	}}

	res := Evaluate(td, provider)
	assert.Nil(t, res.DecidingRule)
	require.Contains(t, res.RulesEvalResults, "cost-1")
	assert.Equal(t, 1, res.RulesEvalResults["cost-1"].SpanCheckedCount)
	assert.Zero(t, res.RulesEvalResults["cost-1"].SpanMatchedCount)
}

// Cost reduction rules are an instruction to sample down, so the most aggressive match has to win.
// Picking the highest would keep traffic the user is paying to drop, and the outcome must not
// depend on the order the rules happen to arrive in.
func TestEvaluatePicksTheLowestPercentageRegardlessOfOrder(t *testing.T) {
	aggressive := costTestRule("cost-aggressive", 5, "GET /assets/app.js")
	permissive := costTestRule("cost-permissive", 60, "GET /assets/app.js")

	tests := []struct {
		name  string
		rules []config.ComputedRule
	}{
		{name: "aggressive rule first", rules: []config.ComputedRule{aggressive, permissive}},
		{name: "permissive rule first", rules: []config.ComputedRule{permissive, aggressive}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			td := costTestTrace(map[string][]string{"checkout": {"GET /assets/app.js"}})
			res := Evaluate(td, costTestConfigProvider{"checkout": {CostReductionRules: tt.rules}})

			require.NotNil(t, res.DecidingRule)
			assert.Equal(t, "cost-aggressive", res.DecidingRule.RuleId)
		})
	}
}

// A zero percent rule is the most aggressive possible, so it must win over anything else.
func TestEvaluateZeroPercentRuleWins(t *testing.T) {
	td := costTestTrace(map[string][]string{"checkout": {"GET /assets/app.js"}})
	res := Evaluate(td, costTestConfigProvider{"checkout": {CostReductionRules: []config.ComputedRule{
		costTestRule("cost-zero", 0, "GET /assets/app.js"),
		costTestRule("cost-ten", 10, "GET /assets/app.js"),
	}}})

	require.NotNil(t, res.DecidingRule)
	assert.Equal(t, "cost-zero", res.DecidingRule.RuleId)
}

// A disabled rule must not be able to drop a trace, but it is still evaluated so users can see in
// the metrics what enabling it would do.
func TestEvaluateDisabledRuleIsMeasuredButDoesNotDecide(t *testing.T) {
	disabled := costTestRule("cost-disabled", 0, "GET /assets/app.js")
	disabled.Disabled = true

	t.Run("a disabled rule alone does not decide", func(t *testing.T) {
		td := costTestTrace(map[string][]string{"checkout": {"GET /assets/app.js"}})
		res := Evaluate(td, costTestConfigProvider{"checkout": {CostReductionRules: []config.ComputedRule{disabled}}})

		assert.Nil(t, res.DecidingRule)
		require.Contains(t, res.RulesEvalResults, "cost-disabled")
		assert.Equal(t, 1, res.RulesEvalResults["cost-disabled"].SpanMatchedCount)
	})

	// the shortcut for a 0% rule must not return a disabled rule.
	t.Run("a disabled zero percent rule does not short circuit the selection", func(t *testing.T) {
		td := costTestTrace(map[string][]string{"checkout": {"GET /assets/app.js"}})
		res := Evaluate(td, costTestConfigProvider{"checkout": {CostReductionRules: []config.ComputedRule{
			disabled,
			costTestRule("cost-enabled", 60, "GET /assets/app.js"),
			costTestRule("cost-enabled-higher", 80, "GET /assets/app.js"),
		}}})

		require.NotNil(t, res.DecidingRule)
		assert.Equal(t, "cost-enabled", res.DecidingRule.RuleId)
	})
}

func TestEvaluateAggregatesAcrossSpans(t *testing.T) {
	td := costTestTrace(map[string][]string{
		"checkout": {"GET /assets/app.js", "GET /checkout", "GET /assets/app.js"},
	})
	res := Evaluate(td, costTestConfigProvider{"checkout": {CostReductionRules: []config.ComputedRule{
		costTestRule("cost-assets", 30, "GET /assets/app.js"),
	}}})

	require.NotNil(t, res.DecidingRule)
	require.Contains(t, res.RulesEvalResults, "cost-assets")
	assert.Equal(t, 3, res.RulesEvalResults["cost-assets"].SpanCheckedCount)
	assert.Equal(t, 2, res.RulesEvalResults["cost-assets"].SpanMatchedCount)
}

// Rules are scoped to the workload that owns the span, so a rule from one workload must not drop a
// trace because of an identically named span belonging to another.
func TestEvaluateAppliesEachResourceItsOwnRules(t *testing.T) {
	td := ptrace.NewTraces()

	unconfigured := td.ResourceSpans().AppendEmpty()
	unconfigured.Resource().Attributes().PutStr(costTestWorkloadAttr, "frontend")
	unconfigured.ScopeSpans().AppendEmpty().Spans().AppendEmpty().SetName("GET /assets/app.js")

	configured := td.ResourceSpans().AppendEmpty()
	configured.Resource().Attributes().PutStr(costTestWorkloadAttr, "checkout")
	configured.ScopeSpans().AppendEmpty().Spans().AppendEmpty().SetName("GET /checkout")

	res := Evaluate(td, costTestConfigProvider{"checkout": {CostReductionRules: []config.ComputedRule{
		costTestRule("cost-assets", 5, "GET /assets/app.js"),
	}}})

	assert.Nil(t, res.DecidingRule, "the asset span belongs to a workload without rules")
	assert.Equal(t, 1, res.RulesEvalResults["cost-assets"].SpanCheckedCount)
}

// The span level attributes are what lets a user find which span drove the decision. Each span
// records its own most aggressive match, which is not necessarily the trace level deciding rule.
func TestEvaluateRecordsThePerSpanMatchingRule(t *testing.T) {
	td := costTestTrace(map[string][]string{"checkout": {"GET /assets/app.js", "GET /healthz"}})
	res := Evaluate(td, costTestConfigProvider{"checkout": {CostReductionRules: []config.ComputedRule{
		costTestRule("cost-assets", 5, "GET /assets/app.js"),
		costTestRule("cost-healthz", 40, "GET /healthz"),
	}}})

	require.NotNil(t, res.DecidingRule)
	assert.Equal(t, "cost-assets", res.DecidingRule.RuleId)

	assetSpan := spanByName(t, td, "GET /assets/app.js")
	ruleID, found := assetSpan.Attributes().Get(odigosattributes.SamplingSpanMatchingRuleId)
	require.True(t, found)
	assert.Equal(t, "cost-assets", ruleID.Str())

	healthzSpan := spanByName(t, td, "GET /healthz")
	ruleID, found = healthzSpan.Attributes().Get(odigosattributes.SamplingSpanMatchingRuleId)
	require.True(t, found)
	assert.Equal(t, "cost-healthz", ruleID.Str(), "each span records the rule that matched it, not the trace decision")
	percentage, found := healthzSpan.Attributes().Get(odigosattributes.SamplingSpanMatchingRuleKeepPercentage)
	require.True(t, found)
	assert.Equal(t, 40.0, percentage.Double())
}

func TestEvaluateLeavesUnmatchedSpansUnannotated(t *testing.T) {
	td := costTestTrace(map[string][]string{"checkout": {"GET /assets/app.js", "GET /checkout"}})
	Evaluate(td, costTestConfigProvider{"checkout": {CostReductionRules: []config.ComputedRule{
		costTestRule("cost-assets", 5, "GET /assets/app.js"),
	}}})

	_, found := spanByName(t, td, "GET /checkout").Attributes().Get(odigosattributes.SamplingSpanMatchingRuleId)
	assert.False(t, found)
}
