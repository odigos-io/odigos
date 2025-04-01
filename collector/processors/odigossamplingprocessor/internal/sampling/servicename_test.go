package sampling

import (
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/odigos/processor/odigossamplingprocessor/internal/sampling/testutil"
	"github.com/stretchr/testify/assert"
)

// TestServiceName_MatchingService_Sampled ensures that a matching service is marked as satisfied.
func TestServiceName_MatchingService_Sampled(t *testing.T) {
	s := ServiceNameRule{
		ServiceName:           "checkout-service",
		SamplingRatio:         100.0,
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
	assert.Equal(t, 100.0, fallback)
}

// TestServiceName_MatchingService_NotSampled ensures that a matching service is reported as satisfied even with 0% ratio.
func TestServiceName_MatchingService_NotSampled(t *testing.T) {
	s := ServiceNameRule{
		ServiceName:           "checkout-service",
		SamplingRatio:         0.0,
		FallbackSamplingRatio: 10.0,
	}

	trace := testutil.NewTrace().
		AddResource("checkout-service").
		AddSpan("checkout").
		Done().
		Build()

	filterMatch, conditionMatch, fallback := s.Evaluate(trace)

	assert.True(t, filterMatch)
	assert.True(t, conditionMatch) // satisfied == true since rule matched
	assert.Equal(t, 0.0, fallback)
}

// TestServiceName_NoMatchingService verifies behavior when no service matches; fallback ratio applies.
func TestServiceName_NoMatchingService(t *testing.T) {
	s := ServiceNameRule{
		ServiceName:           "checkout-service",
		SamplingRatio:         50.0,
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

// TestServiceName_MultipleResources_OneMatches confirms that when one of multiple resource spans
// matches the target service name, the rule is marked as matched and satisfied, and returns its SamplingRatio.
func TestServiceName_MultipleResources_OneMatches(t *testing.T) {
	s := ServiceNameRule{
		ServiceName:           "auth-service",
		SamplingRatio:         100.0,
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
	assert.Equal(t, 100.0, fallback)
}

// TestServiceName_EmptyTrace ensures an empty trace is handled gracefully with fallback ratio.
func TestServiceName_EmptyTrace(t *testing.T) {
	s := ServiceNameRule{
		ServiceName:           "any-service",
		SamplingRatio:         50.0,
		FallbackSamplingRatio: 20.0,
	}

	trace := testutil.NewTrace().Build()

	filterMatch, conditionMatch, fallback := s.Evaluate(trace)

	assert.False(t, filterMatch)
	assert.False(t, conditionMatch)
	assert.Equal(t, 20.0, fallback)
}

// TestServiceName_InvalidServiceKey verifies handling when service.name attribute is missing.
func TestServiceName_InvalidServiceKey(t *testing.T) {
	s := ServiceNameRule{
		ServiceName:           "expected-service",
		SamplingRatio:         100.0,
		FallbackSamplingRatio: 30.0,
	}

	trace := testutil.NewTrace().
		AddEmptyResource().
		AddSpan("generic").
		Done().
		Build()

	filterMatch, conditionMatch, fallback := s.Evaluate(trace)

	assert.False(t, filterMatch)
	assert.False(t, conditionMatch)
	assert.Equal(t, 30.0, fallback)
}

// TestServiceName_InvalidSamplingRatio validates rule with invalid SamplingRatio.
func TestServiceName_InvalidSamplingRatio(t *testing.T) {
	s := ServiceNameRule{
		ServiceName:           "checkout-service",
		SamplingRatio:         -10.0, // Invalid
		FallbackSamplingRatio: 20.0,
	}

	err := s.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sampling ratio must be between 0 and 100")
}

// TestServiceName_InvalidFallbackSamplingRatio validates rule with invalid FallbackSamplingRatio.
func TestServiceName_InvalidFallbackSamplingRatio(t *testing.T) {
	s := ServiceNameRule{
		ServiceName:           "checkout-service",
		SamplingRatio:         50.0,
		FallbackSamplingRatio: 150.0, // Invalid
	}

	err := s.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fallback sampling ratio must be between 0 and 100")
}
