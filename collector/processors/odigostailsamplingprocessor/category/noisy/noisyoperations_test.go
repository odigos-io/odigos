package noisy

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/ptrace"

	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
)

func TestEvaluateIgnoresDisabledRulesForDecision(t *testing.T) {
	disabledPercentage := 0.0
	enabledPercentage := 75.0
	span := ptrace.NewTraces().ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()

	result := Evaluate(span, []commonapisampling.NoisyOperation{
		{
			Id:               "disabled-rule",
			Disabled:         true,
			PercentageAtMost: &disabledPercentage,
		},
		{
			Id:               "enabled-rule",
			PercentageAtMost: &enabledPercentage,
		},
	})

	require.NotNil(t, result.DecidingRule)
	require.Equal(t, "enabled-rule", result.DecidingRule.Id)
	require.Equal(t, 1, result.RulesEvalResults["disabled-rule"].SpanMatchedCount)
	require.Equal(t, 1, result.RulesEvalResults["enabled-rule"].SpanMatchedCount)
}

func TestEvaluateOnlyDisabledMatchesNoDecision(t *testing.T) {
	percentage := 0.0
	span := ptrace.NewTraces().ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()

	result := Evaluate(span, []commonapisampling.NoisyOperation{
		{
			Id:               "disabled-rule",
			Disabled:         true,
			PercentageAtMost: &percentage,
		},
	})

	require.Nil(t, result.DecidingRule)
	require.Equal(t, 1, result.RulesEvalResults["disabled-rule"].SpanMatchedCount)
}
