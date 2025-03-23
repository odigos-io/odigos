package sampling

import (
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/odigos/processor/odigossamplingprocessor/internal/sampling/testutil"
	"github.com/stretchr/testify/assert"
)

// TestServiceName_MatchingServiceExists ensures that a trace with a resource span
// matching the specified service name is correctly identified and marked for sampling.
func TestServiceName_MatchingServiceExists(t *testing.T) {
	s := ServiceNameRule{
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

// TestServiceName_NoMatchingService verifies that when no resource span matches
// the specified service name, the rule does not match and fallback is returned.
func TestServiceName_NoMatchingService(t *testing.T) {
	s := ServiceNameRule{
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

// TestServiceName_MultipleResources_OneMatches confirms that if at least one resource
// in the trace matches the service name, the rule is satisfied even if other resources don't match.
func TestServiceName_MultipleResources_OneMatches(t *testing.T) {
	s := ServiceNameRule{
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

// TestServiceName_EmptyTrace ensures that an empty trace does not cause panic
// and results in a non-match with fallback ratio applied.
func TestServiceName_EmptyTrace(t *testing.T) {
	s := ServiceNameRule{
		ServiceName:           "any-service",
		FallbackSamplingRatio: 20.0,
	}

	trace := testutil.NewTrace().Build()

	filterMatch, conditionMatch, fallback := s.Evaluate(trace)

	assert.False(t, filterMatch)
	assert.False(t, conditionMatch)
	assert.Equal(t, 20.0, fallback)
}

// TestServiceName_InvalidServiceKey ensures that a trace with no "service.name" attribute
// does not match the sampler rule and returns fallback.
func TestServiceName_InvalidServiceKey(t *testing.T) {
	s := ServiceNameRule{
		ServiceName:           "expected-service",
		FallbackSamplingRatio: 30.0,
	}

	// Create a resource with a different key
	trace := testutil.NewTrace().
		AddEmptyResource(). // this doesn't set any attributes
		AddSpan("generic").
		Done().
		Build()

	filterMatch, conditionMatch, fallback := s.Evaluate(trace)

	assert.False(t, filterMatch)
	assert.False(t, conditionMatch)
	assert.Equal(t, 30.0, fallback)
}
