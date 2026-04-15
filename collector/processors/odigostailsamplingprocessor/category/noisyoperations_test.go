package category

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/ptrace"

	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
)

func ptrFloat64(v float64) *float64 {
	return &v
}

func newEmptySpan() ptrace.Span {
	td := ptrace.NewTraces()
	return td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
}

func TestEvaluateNoisyOperationsIgnoresDisabledRules(t *testing.T) {
	span := newEmptySpan()

	rules := []commonapisampling.NoisyOperation{
		{
			Id:               "disabled-rule",
			Disabled:         true,
			PercentageAtMost: nil, // 0% if applied (this should be ignored)
		},
		{
			Id:               "enabled-rule",
			PercentageAtMost: ptrFloat64(100),
		},
	}

	matched, decidingRule := EvaluateNoisyOperations(span, rules)
	require.True(t, matched)
	require.NotNil(t, decidingRule)
	assert.Equal(t, "enabled-rule", decidingRule.Id)
}

func TestEvaluateNoisyOperationsAllRulesDisabled(t *testing.T) {
	span := newEmptySpan()

	rules := []commonapisampling.NoisyOperation{
		{
			Id:       "disabled-rule",
			Disabled: true,
		},
	}

	matched, decidingRule := EvaluateNoisyOperations(span, rules)
	assert.False(t, matched)
	assert.Nil(t, decidingRule)
}

func TestEvaluateNoisyOperationsChoosesLeastPercentageAmongEnabled(t *testing.T) {
	span := newEmptySpan()

	rules := []commonapisampling.NoisyOperation{
		{
			Id:               "high-percentage",
			PercentageAtMost: ptrFloat64(80),
		},
		{
			Id:               "low-percentage",
			PercentageAtMost: ptrFloat64(20),
		},
	}

	matched, decidingRule := EvaluateNoisyOperations(span, rules)
	require.True(t, matched)
	require.NotNil(t, decidingRule)
	assert.Equal(t, "low-percentage", decidingRule.Id)
}
