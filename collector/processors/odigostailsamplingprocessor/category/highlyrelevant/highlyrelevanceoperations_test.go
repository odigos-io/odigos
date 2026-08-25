package highlyrelevant

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/category/config"
	"github.com/odigos-io/odigos/common/odigosattributes"
)

const hrTestWorkloadAttr = "odigos.test.workload.key"

// nameMatcher matches a span by name, which keeps these tests about rule selection rather than
// about the attribute matchers (those are covered in the matchers package).
type nameMatcher struct{ name string }

func (m nameMatcher) Match(span ptrace.Span) bool { return span.Name() == m.name }

type anySpanMatcher struct{}

func (anySpanMatcher) Match(ptrace.Span) bool { return true }

// hrTestConfigProvider resolves the workload config from a resource attribute so one trace can
// carry resources belonging to workloads with different rules.
type hrTestConfigProvider map[string]*config.ComputedWorkloadConfig

func (p hrTestConfigProvider) GetTailSamplingConfig(resource pcommon.Resource) (*config.ComputedWorkloadConfig, bool) {
	key, found := resource.Attributes().Get(hrTestWorkloadAttr)
	if !found {
		return nil, false
	}
	cfg, found := p[key.Str()]
	return cfg, found
}

func hrTestRule(id string, percentage float64, matchName string) config.ComputedRule {
	return config.ComputedRule{
		RuleId:     id,
		Name:       id,
		Percentage: percentage,
		Matcher:    nameMatcher{name: matchName},
	}
}

// hrTestTrace builds a trace with one resource per entry in workloadSpans, keyed by workload.
func hrTestTrace(workloadSpans map[string][]string) ptrace.Traces {
	td := ptrace.NewTraces()
	for workload, spanNames := range workloadSpans {
		rs := td.ResourceSpans().AppendEmpty()
		rs.Resource().Attributes().PutStr(hrTestWorkloadAttr, workload)
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
	td := hrTestTrace(map[string][]string{"checkout": {"GET /checkout"}})

	res := Evaluate(td, hrTestConfigProvider{})
	assert.Nil(t, res.DecidingRule)
	assert.Empty(t, res.RulesEvalResults)
}

func TestEvaluateWithoutHighlyRelevantRules(t *testing.T) {
	td := hrTestTrace(map[string][]string{"checkout": {"GET /checkout"}})
	provider := hrTestConfigProvider{"checkout": {
		CostReductionRules: []config.ComputedRule{{RuleId: "cost-1", Matcher: anySpanMatcher{}}},
	}}

	res := Evaluate(td, provider)
	assert.Nil(t, res.DecidingRule)
	assert.Empty(t, res.RulesEvalResults, "cost reduction rules must not be evaluated by this category")
}

func TestEvaluateWithoutAMatch(t *testing.T) {
	td := hrTestTrace(map[string][]string{"checkout": {"GET /checkout"}})
	provider := hrTestConfigProvider{"checkout": {
		HighlyRelevantOperations: []config.ComputedRule{hrTestRule("hr-1", 100, "GET /cart")},
	}}

	res := Evaluate(td, provider)
	assert.Nil(t, res.DecidingRule)
	require.Contains(t, res.RulesEvalResults, "hr-1")
	assert.Equal(t, 1, res.RulesEvalResults["hr-1"].SpanCheckedCount)
	assert.Zero(t, res.RulesEvalResults["hr-1"].SpanMatchedCount)
}

// Highly relevant rules are an instruction to keep, so the most permissive match has to win.
// Picking the lowest would drop traces the user explicitly flagged as important, and the outcome
// must not depend on the order the rules happen to arrive in.
func TestEvaluatePicksTheHighestPercentageRegardlessOfOrder(t *testing.T) {
	permissive := hrTestRule("hr-permissive", 90, "GET /checkout")
	restrictive := hrTestRule("hr-restrictive", 10, "GET /checkout")

	tests := []struct {
		name  string
		rules []config.ComputedRule
	}{
		{name: "permissive rule first", rules: []config.ComputedRule{permissive, restrictive}},
		{name: "restrictive rule first", rules: []config.ComputedRule{restrictive, permissive}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			td := hrTestTrace(map[string][]string{"checkout": {"GET /checkout"}})
			res := Evaluate(td, hrTestConfigProvider{"checkout": {HighlyRelevantOperations: tt.rules}})

			require.NotNil(t, res.DecidingRule)
			assert.Equal(t, "hr-permissive", res.DecidingRule.RuleId)
		})
	}
}

// A hundred percent rule is the most permissive possible, so it must win over anything else.
func TestEvaluateFullPercentRuleWins(t *testing.T) {
	td := hrTestTrace(map[string][]string{"checkout": {"GET /checkout"}})
	res := Evaluate(td, hrTestConfigProvider{"checkout": {HighlyRelevantOperations: []config.ComputedRule{
		hrTestRule("hr-full", 100, "GET /checkout"),
		hrTestRule("hr-half", 50, "GET /checkout"),
	}}})

	require.NotNil(t, res.DecidingRule)
	assert.Equal(t, "hr-full", res.DecidingRule.RuleId)
}

// A disabled rule must not be able to keep a trace, but it is still evaluated so users can see in
// the metrics what enabling it would do.
func TestEvaluateDisabledRuleIsMeasuredButDoesNotDecide(t *testing.T) {
	disabled := hrTestRule("hr-disabled", 100, "GET /checkout")
	disabled.Disabled = true

	t.Run("a disabled rule alone does not decide", func(t *testing.T) {
		td := hrTestTrace(map[string][]string{"checkout": {"GET /checkout"}})
		res := Evaluate(td, hrTestConfigProvider{"checkout": {HighlyRelevantOperations: []config.ComputedRule{disabled}}})

		assert.Nil(t, res.DecidingRule)
		require.Contains(t, res.RulesEvalResults, "hr-disabled")
		assert.Equal(t, 1, res.RulesEvalResults["hr-disabled"].SpanMatchedCount)
	})

	t.Run("an enabled rule still decides alongside a disabled one", func(t *testing.T) {
		td := hrTestTrace(map[string][]string{"checkout": {"GET /checkout"}})
		res := Evaluate(td, hrTestConfigProvider{"checkout": {HighlyRelevantOperations: []config.ComputedRule{
			disabled,
			hrTestRule("hr-enabled", 20, "GET /checkout"),
		}}})

		require.NotNil(t, res.DecidingRule)
		assert.Equal(t, "hr-enabled", res.DecidingRule.RuleId)
		assert.Equal(t, 20.0, res.DecidingRule.Percentage)
	})

	// the shortcut for a 100% rule must not return a disabled rule.
	t.Run("a disabled full percent rule does not short circuit the selection", func(t *testing.T) {
		td := hrTestTrace(map[string][]string{"checkout": {"GET /checkout"}})
		res := Evaluate(td, hrTestConfigProvider{"checkout": {HighlyRelevantOperations: []config.ComputedRule{
			disabled,
			hrTestRule("hr-enabled", 20, "GET /checkout"),
			hrTestRule("hr-enabled-lower", 5, "GET /checkout"),
		}}})

		require.NotNil(t, res.DecidingRule)
		assert.Equal(t, "hr-enabled", res.DecidingRule.RuleId)
	})
}

// Any span of the trace can make it highly relevant, not just the root, and the rule counters have
// to add up across all of them.
func TestEvaluateAggregatesAcrossSpansAndResources(t *testing.T) {
	td := hrTestTrace(map[string][]string{
		"checkout": {"GET /checkout", "SELECT orders", "GET /checkout"},
	})
	res := Evaluate(td, hrTestConfigProvider{"checkout": {HighlyRelevantOperations: []config.ComputedRule{
		hrTestRule("hr-checkout", 30, "GET /checkout"),
	}}})

	require.NotNil(t, res.DecidingRule)
	assert.Equal(t, "hr-checkout", res.DecidingRule.RuleId)

	require.Contains(t, res.RulesEvalResults, "hr-checkout")
	assert.Equal(t, 3, res.RulesEvalResults["hr-checkout"].SpanCheckedCount)
	assert.Equal(t, 2, res.RulesEvalResults["hr-checkout"].SpanMatchedCount)
}

// Rules are scoped to the workload that owns the span. A rule from one workload must not be applied
// to the spans of another, and a workload with no rules must not stop the others from being
// evaluated.
func TestEvaluateAppliesEachResourceItsOwnRules(t *testing.T) {
	td := ptrace.NewTraces()

	unconfigured := td.ResourceSpans().AppendEmpty()
	unconfigured.Resource().Attributes().PutStr(hrTestWorkloadAttr, "frontend")
	unconfigured.ScopeSpans().AppendEmpty().Spans().AppendEmpty().SetName("GET /checkout")

	configured := td.ResourceSpans().AppendEmpty()
	configured.Resource().Attributes().PutStr(hrTestWorkloadAttr, "checkout")
	configured.ScopeSpans().AppendEmpty().Spans().AppendEmpty().SetName("GET /checkout")

	res := Evaluate(td, hrTestConfigProvider{"checkout": {HighlyRelevantOperations: []config.ComputedRule{
		hrTestRule("hr-checkout", 30, "GET /checkout"),
	}}})

	require.NotNil(t, res.DecidingRule)
	assert.Equal(t, "hr-checkout", res.DecidingRule.RuleId)
	// the identically named span on the unconfigured workload must not have been evaluated.
	assert.Equal(t, 1, res.RulesEvalResults["hr-checkout"].SpanCheckedCount)
}

// The span level attributes are what lets a user find which span kept a trace. Each span records
// its own most permissive match, which is not necessarily the trace level deciding rule.
func TestEvaluateRecordsThePerSpanMatchingRule(t *testing.T) {
	td := hrTestTrace(map[string][]string{"checkout": {"GET /checkout", "SELECT orders"}})
	res := Evaluate(td, hrTestConfigProvider{"checkout": {HighlyRelevantOperations: []config.ComputedRule{
		hrTestRule("hr-checkout", 90, "GET /checkout"),
		hrTestRule("hr-db", 20, "SELECT orders"),
	}}})

	require.NotNil(t, res.DecidingRule)
	assert.Equal(t, "hr-checkout", res.DecidingRule.RuleId)

	checkoutSpan := spanByName(t, td, "GET /checkout")
	ruleID, found := checkoutSpan.Attributes().Get(odigosattributes.SamplingSpanMatchingRuleId)
	require.True(t, found)
	assert.Equal(t, "hr-checkout", ruleID.Str())
	percentage, found := checkoutSpan.Attributes().Get(odigosattributes.SamplingSpanMatchingRuleKeepPercentage)
	require.True(t, found)
	assert.Equal(t, 90.0, percentage.Double())

	dbSpan := spanByName(t, td, "SELECT orders")
	ruleID, found = dbSpan.Attributes().Get(odigosattributes.SamplingSpanMatchingRuleId)
	require.True(t, found)
	assert.Equal(t, "hr-db", ruleID.Str(), "each span records the rule that matched it, not the trace decision")
}

func TestEvaluateLeavesUnmatchedSpansUnannotated(t *testing.T) {
	td := hrTestTrace(map[string][]string{"checkout": {"GET /checkout", "SELECT orders"}})
	Evaluate(td, hrTestConfigProvider{"checkout": {HighlyRelevantOperations: []config.ComputedRule{
		hrTestRule("hr-checkout", 90, "GET /checkout"),
	}}})

	dbSpan := spanByName(t, td, "SELECT orders")
	_, found := dbSpan.Attributes().Get(odigosattributes.SamplingSpanMatchingRuleId)
	assert.False(t, found)
}

// A span that matches only a disabled rule must not be annotated as if a rule kept it.
func TestEvaluateDoesNotAnnotateSpansMatchedOnlyByADisabledRule(t *testing.T) {
	disabled := hrTestRule("hr-disabled", 100, "GET /checkout")
	disabled.Disabled = true

	td := hrTestTrace(map[string][]string{"checkout": {"GET /checkout"}})
	Evaluate(td, hrTestConfigProvider{"checkout": {HighlyRelevantOperations: []config.ComputedRule{disabled}}})

	_, found := spanByName(t, td, "GET /checkout").Attributes().Get(odigosattributes.SamplingSpanMatchingRuleId)
	assert.False(t, found)
}
