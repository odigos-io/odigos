package sampling

import (
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/odigos/processor/odigossamplingprocessor/internal/sampling/testutil"
	"github.com/stretchr/testify/assert"
)

func TestSpanAttributeExistsSampler_AttributeExists(t *testing.T) {
	s := SpanAttributeExistsSampler{
		AttributeKey:          "user.id",
		FallbackSamplingRatio: 20.0,
	}

	trace := testutil.NewTrace().
		AddResource("user-service").
		AddSpan("create-user", testutil.WithAttribute("user.id", "abc123")).
		Done().
		Build()

	filterMatch, conditionMatch, fallback := s.Evaluate(trace)

	assert.True(t, filterMatch)
	assert.True(t, conditionMatch)
	assert.Equal(t, 20.0, fallback)
}

func TestSpanAttributeExistsSampler_AttributeMissing(t *testing.T) {
	s := SpanAttributeExistsSampler{
		AttributeKey:          "session.token",
		FallbackSamplingRatio: 3.0,
	}

	trace := testutil.NewTrace().
		AddResource("auth-service").
		AddSpan("auth-request").
		Done().
		Build()

	filterMatch, conditionMatch, fallback := s.Evaluate(trace)

	assert.False(t, filterMatch)
	assert.False(t, conditionMatch)
	assert.Equal(t, 3.0, fallback)
}

func TestSpanAttributeExistsSampler_ExistsInOneOfManySpans(t *testing.T) {
	s := SpanAttributeExistsSampler{
		AttributeKey:          "feature.flag",
		FallbackSamplingRatio: 12.0,
	}

	trace := testutil.NewTrace().
		AddResource("feature-service").
		AddSpan("evaluate").
		Done().
		AddResource("feature-service").
		AddSpan("check-flag", testutil.WithAttribute("feature.flag", "beta-mode")).
		Done().
		Build()

	filterMatch, conditionMatch, fallback := s.Evaluate(trace)

	assert.True(t, filterMatch)
	assert.True(t, conditionMatch)
	assert.Equal(t, 12.0, fallback)
}
