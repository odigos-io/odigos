package sampling

import (
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/odigos/processor/odigossamplingprocessor/internal/sampling/testutil"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// TestErrorRule_Evaluate ensures the trace is sampled when an error span exists.
func TestErrorRule_Evaluate(t *testing.T) {
	rule := &ErrorRule{FallbackSamplingRatio: 30}

	trace := testutil.NewTrace().
		AddResource("billing-service").
		AddSpan("Process Payment", testutil.WithStatus(ptrace.StatusCodeError)).
		Done().Build()

	matched, satisfied, fallback := rule.Evaluate(trace)
	assert.True(t, matched)
	assert.True(t, satisfied)
	assert.Equal(t, 0.0, fallback)
}

// TestErrorRule_Evaluate_MultipleSpans_OneError checks that if any span has an error status,
// the trace is sampled.
func TestErrorRule_Evaluate_MultipleSpans_OneError(t *testing.T) {
	rule := &ErrorRule{FallbackSamplingRatio: 30}

	trace := testutil.NewTrace().
		AddResource("order-service").
		AddSpan("Create Order").
		AddSpan("Charge Card", testutil.WithStatus(ptrace.StatusCodeError)).
		AddSpan("Send Email").
		Done().Build()

	matched, satisfied, fallback := rule.Evaluate(trace)

	assert.True(t, matched)
	assert.True(t, satisfied)
	assert.Equal(t, 0.0, fallback)
}

// TestErrorRule_Evaluate_NoErrorSpans confirms that if no spans have errors,
// fallback sampling ratio is applied.
func TestErrorRule_Evaluate_NoErrorSpans(t *testing.T) {
	rule := &ErrorRule{FallbackSamplingRatio: 30}

	trace := testutil.NewTrace().
		AddResource("user-service").
		AddSpan("Create User").
		AddSpan("Validate Input").
		AddSpan("Send Welcome Email").
		Done().Build()

	matched, satisfied, fallback := rule.Evaluate(trace)

	assert.True(t, matched)         // ✅ rule always matches
	assert.False(t, satisfied)      // ✅ no error found
	assert.Equal(t, 30.0, fallback) // ✅ fallback applied
}
