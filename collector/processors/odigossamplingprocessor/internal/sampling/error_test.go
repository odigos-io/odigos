package sampling

import (
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/odigos/processor/odigossamplingprocessor/internal/sampling/testutil"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

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
