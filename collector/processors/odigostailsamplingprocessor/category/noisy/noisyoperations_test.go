package noisy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/category/config"
)

// nameMatcher matches a span by name, which keeps these tests about rule selection rather than
// about the attribute matchers (those are covered in the matchers package).
type nameMatcher struct{ name string }

func (m nameMatcher) Match(span ptrace.Span) bool { return span.Name() == m.name }

func noisyTestRule(id string, percentage float64, matchName string) config.ComputedRule {
	return config.ComputedRule{
		RuleId:     id,
		Name:       id,
		Percentage: percentage,
		Matcher:    nameMatcher{name: matchName},
	}
}

func noisyTestSpan(name string) ptrace.Span {
	span := ptrace.NewSpan()
	span.SetName(name)
	return span
}

func TestEvaluateWithoutRules(t *testing.T) {
	res := Evaluate(noisyTestSpan("GET /healthz"), nil)
	assert.Nil(t, res.DecidingRule)
	assert.Empty(t, res.RulesEvalResults)
}

func TestEvaluateWithoutAMatch(t *testing.T) {
	res := Evaluate(noisyTestSpan("GET /checkout"), []config.ComputedRule{
		noisyTestRule("noise-1", 10, "GET /healthz"),
	})

	assert.Nil(t, res.DecidingRule)
	require.Contains(t, res.RulesEvalResults, "noise-1")
	assert.Equal(t, 1, res.RulesEvalResults["noise-1"].SpanCheckedCount)
	assert.Zero(t, res.RulesEvalResults["noise-1"].SpanMatchedCount)
}

func TestEvaluateSingleMatch(t *testing.T) {
	res := Evaluate(noisyTestSpan("GET /healthz"), []config.ComputedRule{
		noisyTestRule("noise-1", 10, "GET /healthz"),
	})

	require.NotNil(t, res.DecidingRule)
	assert.Equal(t, "noise-1", res.DecidingRule.RuleId)
	assert.Equal(t, 10.0, res.DecidingRule.Percentage)

	require.Contains(t, res.RulesEvalResults, "noise-1")
	assert.Equal(t, 1, res.RulesEvalResults["noise-1"].SpanCheckedCount)
	assert.Equal(t, 1, res.RulesEvalResults["noise-1"].SpanMatchedCount)
}

// Noise rules are an instruction to sample down, so when several of them match the most aggressive
// one has to win. Picking the highest instead would keep traffic the user explicitly called noisy,
// and the outcome must not depend on the order the rules happen to arrive in.
func TestEvaluatePicksTheLowestPercentageRegardlessOfOrder(t *testing.T) {
	aggressive := noisyTestRule("noise-aggressive", 5, "GET /healthz")
	permissive := noisyTestRule("noise-permissive", 60, "GET /healthz")

	tests := []struct {
		name  string
		rules []config.ComputedRule
	}{
		{name: "aggressive rule first", rules: []config.ComputedRule{aggressive, permissive}},
		{name: "permissive rule first", rules: []config.ComputedRule{permissive, aggressive}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := Evaluate(noisyTestSpan("GET /healthz"), tt.rules)
			require.NotNil(t, res.DecidingRule)
			assert.Equal(t, "noise-aggressive", res.DecidingRule.RuleId)
		})
	}
}

func TestEvaluateIgnoresRulesThatDoNotMatch(t *testing.T) {
	res := Evaluate(noisyTestSpan("GET /healthz"), []config.ComputedRule{
		noisyTestRule("noise-other-route", 1, "GET /metrics"),
		noisyTestRule("noise-healthz", 60, "GET /healthz"),
	})

	require.NotNil(t, res.DecidingRule)
	assert.Equal(t, "noise-healthz", res.DecidingRule.RuleId)
}

// A disabled rule is kept around so the user can re-enable it later. Until then it must not be able
// to drop anything, even when it is the most aggressive rule configured.
func TestEvaluateSkipsDisabledRules(t *testing.T) {
	disabled := noisyTestRule("noise-disabled", 0, "GET /healthz")
	disabled.Disabled = true

	t.Run("a disabled rule alone does not decide", func(t *testing.T) {
		res := Evaluate(noisyTestSpan("GET /healthz"), []config.ComputedRule{disabled})
		assert.Nil(t, res.DecidingRule)
	})

	t.Run("a disabled rule does not override an enabled one", func(t *testing.T) {
		res := Evaluate(noisyTestSpan("GET /healthz"), []config.ComputedRule{
			disabled,
			noisyTestRule("noise-enabled", 60, "GET /healthz"),
		})
		require.NotNil(t, res.DecidingRule)
		assert.Equal(t, "noise-enabled", res.DecidingRule.RuleId)
		assert.Equal(t, 60.0, res.DecidingRule.Percentage)
	})
}

// A zero percent rule is the most aggressive possible, so it must win over anything else.
func TestEvaluateZeroPercentRuleWins(t *testing.T) {
	res := Evaluate(noisyTestSpan("GET /healthz"), []config.ComputedRule{
		noisyTestRule("noise-zero", 0, "GET /healthz"),
		noisyTestRule("noise-ten", 10, "GET /healthz"),
	})

	require.NotNil(t, res.DecidingRule)
	assert.Equal(t, "noise-zero", res.DecidingRule.RuleId)
}

// The evaluation result carries the rule itself, which is what the processor uses to attribute the
// metrics. A result pointing at the wrong rule would misreport every noise decision.
func TestEvaluateResultCarriesTheEvaluatedRule(t *testing.T) {
	rule := noisyTestRule("noise-1", 42, "GET /healthz")
	res := Evaluate(noisyTestSpan("GET /healthz"), []config.ComputedRule{rule})

	require.Contains(t, res.RulesEvalResults, "noise-1")
	assert.Equal(t, "noise-1", res.RulesEvalResults["noise-1"].ComputedRule.RuleId)
	assert.Equal(t, 42.0, res.RulesEvalResults["noise-1"].ComputedRule.Percentage)
}
