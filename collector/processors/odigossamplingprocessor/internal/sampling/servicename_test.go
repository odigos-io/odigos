package sampling

import (
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/odigos/processor/odigossamplingprocessor/internal/sampling/testutil"
	"github.com/stretchr/testify/assert"
)

func TestServiceNameSampler_MatchingServiceExists(t *testing.T) {
	s := ServiceNameSampler{
		ServiceName:           "checkout-service",
		FallbackSamplingRatio: 10.0,
	}

	trace := testutil.NewTrace().
		AddResource("checkout-service").
		AddSpan("checkout").
		Done().
		Build()

	filterMatch, conditionMatch, fallback := s.Evaluate(trace)

	assert.True(t, filterMatch)
	assert.True(t, conditionMatch)
	assert.Equal(t, 10.0, fallback)
}

func TestServiceNameSampler_NoMatchingService(t *testing.T) {
	s := ServiceNameSampler{
		ServiceName:           "checkout-service",
		FallbackSamplingRatio: 5.0,
	}

	trace := testutil.NewTrace().
		AddResource("inventory-service").
		AddSpan("inventory").
		Done().
		Build()

	filterMatch, conditionMatch, fallback := s.Evaluate(trace)

	assert.False(t, filterMatch)
	assert.False(t, conditionMatch)
	assert.Equal(t, 5.0, fallback)
}

func TestServiceNameSampler_MultipleResources_OneMatches(t *testing.T) {
	s := ServiceNameSampler{
		ServiceName:           "auth-service",
		FallbackSamplingRatio: 15.0,
	}

	trace := testutil.NewTrace().
		AddResource("frontend").
		AddSpan("render").
		Done().
		AddResource("auth-service").
		AddSpan("login").
		Done().
		Build()

	filterMatch, conditionMatch, fallback := s.Evaluate(trace)

	assert.True(t, filterMatch)
	assert.True(t, conditionMatch)
	assert.Equal(t, 15.0, fallback)
}
