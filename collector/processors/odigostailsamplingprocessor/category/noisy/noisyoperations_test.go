package noisy

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/ptrace"

	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
)

func TestEvaluateIgnoresDisabledRulesForDecision(t *testing.T) {
	span := ptrace.NewTraces().ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()

	result := Evaluate(span, []commonapisampling.NoisyOperation{
		{
			Id:       "disabled",
			Name:     "disabled noisy rule",
			Disabled: true,
		},
	})

	require.Nil(t, result.DecidingRule)
	require.True(t, result.RulesEvalResults["disabled"].Matched)
	require.Equal(t, 1, result.RulesEvalResults["disabled"].SpanMatchedCount)
}

func TestEvaluateSelectsLeastPercentageEnabledRule(t *testing.T) {
	span := ptrace.NewTraces().ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	disabledZero := 0.0
	enabledHigh := 50.0
	enabledLow := 10.0

	result := Evaluate(span, []commonapisampling.NoisyOperation{
		{
			Id:               "disabled-zero",
			Disabled:         true,
			PercentageAtMost: &disabledZero,
		},
		{
			Id:               "enabled-high",
			PercentageAtMost: &enabledHigh,
		},
		{
			Id:               "enabled-low",
			PercentageAtMost: &enabledLow,
		},
	})

	require.NotNil(t, result.DecidingRule)
	require.Equal(t, "enabled-low", result.DecidingRule.Id)
}
