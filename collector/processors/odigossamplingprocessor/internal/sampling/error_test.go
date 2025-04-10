package sampling

import (
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/odigos/processor/odigossamplingprocessor/internal/sampling/testutil"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// TestErrorRule_Evaluate triggers when a single error span is present in the trace.
func TestErrorRule_Evaluate(t *testing.T) {
	rule := &ErrorRule{FallbackSamplingRatio: 30}

	trace := testutil.NewTrace().
		AddResource("billing-service").
		AddSpan("Process Payment", testutil.WithStatus(ptrace.StatusCodeError)).
		Done().Build()

	matched, satisfied, ratio := rule.Evaluate(trace)
	assert.True(t, matched)
	assert.True(t, satisfied)
	assert.Equal(t, 100.0, ratio) // Updated fallback
}

// TestErrorRule_Evaluate_MultipleSpans_OneError ensures that even one error span causes sampling.
func TestErrorRule_Evaluate_MultipleSpans_OneError(t *testing.T) {
	rule := &ErrorRule{FallbackSamplingRatio: 30}

	trace := testutil.NewTrace().
		AddResource("order-service").
		AddSpan("Create Order").
		AddSpan("Charge Card", testutil.WithStatus(ptrace.StatusCodeError)).
		AddSpan("Send Email").
		Done().Build()

	matched, satisfied, ratio := rule.Evaluate(trace)
	assert.True(t, matched)
	assert.True(t, satisfied)
	assert.Equal(t, 100.0, ratio) // Updated fallback
}

// TestErrorRule_Evaluate_NoErrorSpans returns fallback when no error spans are found.
func TestErrorRule_Evaluate_NoErrorSpans(t *testing.T) {
	rule := &ErrorRule{FallbackSamplingRatio: 30}

	trace := testutil.NewTrace().
		AddResource("user-service").
		AddSpan("Create User").
		AddSpan("Validate Input").
		AddSpan("Send Welcome Email").
		Done().Build()

	matched, satisfied, ratio := rule.Evaluate(trace)
	assert.True(t, matched)
	assert.False(t, satisfied)
	assert.Equal(t, 30.0, ratio)
}

// TestErrorRule_Evaluate_EmptyTrace ensures an empty trace still matches and uses fallback.
func TestErrorRule_Evaluate_EmptyTrace(t *testing.T) {
	rule := &ErrorRule{FallbackSamplingRatio: 20}

	trace := testutil.NewTrace().Build()

	matched, satisfied, ratio := rule.Evaluate(trace)
	assert.True(t, matched)
	assert.False(t, satisfied)
	assert.Equal(t, 20.0, ratio)
}

// TestErrorRule_Evaluate_ResourceWithNoSpans ensures no error is found if spans are missing.
func TestErrorRule_Evaluate_ResourceWithNoSpans(t *testing.T) {
	rule := &ErrorRule{FallbackSamplingRatio: 15}

	trace := testutil.NewTrace().
		AddEmptyResource().
		Done().Build()

	matched, satisfied, ratio := rule.Evaluate(trace)
	assert.True(t, matched)
	assert.False(t, satisfied)
	assert.Equal(t, 15.0, ratio)
}

// TestErrorRule_Validate_InvalidRatio confirms validation catches fallback > 100.
func TestErrorRule_Validate_InvalidRatio(t *testing.T) {
	rule := &ErrorRule{FallbackSamplingRatio: 150}

	err := rule.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be between 0 and 100")
}
