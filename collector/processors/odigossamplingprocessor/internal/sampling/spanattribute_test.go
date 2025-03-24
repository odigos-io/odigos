package sampling

import (
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/odigos/processor/odigossamplingprocessor/internal/sampling/testutil"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// TestSpanAttribute_ExistsCondition verifies that the sampler matches
// when the specified attribute key exists on any span in the trace.
func TestSpanAttribute_ExistsCondition(t *testing.T) {
	s := SpanAttributeRule{
		AttributeKey:          "env",
		AttributeCondition:    AttributeConditionExists,
		FallbackSamplingRatio: 10.0,
	}

	trace := testutil.NewTrace().
		AddResource("test").
		AddSpan("span1", testutil.WithAttribute("env", "prod")).
		Done().
		Build()

	filterMatch, conditionMatch, fallback := s.Evaluate(trace)

	assert.True(t, filterMatch)
	assert.True(t, conditionMatch)
	assert.Equal(t, 10.0, fallback)
}

// TestSpanAttribute_Equals_Match ensures the sampler matches when the attribute value
// exactly equals the expected value (case-insensitive).
func TestSpanAttribute_Equals_Match(t *testing.T) {
	s := SpanAttributeRule{
		AttributeKey:          "env",
		AttributeCondition:    AttributeConditionEquals,
		ExpectedValue:         "prod",
		FallbackSamplingRatio: 5.0,
	}

	trace := testutil.NewTrace().
		AddResource("web").
		AddSpan("homepage", testutil.WithAttribute("env", "prod")).
		Done().
		Build()

	filterMatch, conditionMatch, fallback := s.Evaluate(trace)

	assert.True(t, filterMatch)
	assert.True(t, conditionMatch)
	assert.Equal(t, 5.0, fallback)
}

// TestSpanAttribute_Equals_NoMatch ensures the sampler does not match
// if the attribute value does not match the expected value.
func TestSpanAttribute_Equals_NoMatch(t *testing.T) {
	s := SpanAttributeRule{
		AttributeKey:          "env",
		AttributeCondition:    AttributeConditionEquals,
		ExpectedValue:         "staging",
		FallbackSamplingRatio: 5.0,
	}

	trace := testutil.NewTrace().
		AddResource("web").
		AddSpan("homepage", testutil.WithAttribute("env", "prod")).
		Done().
		Build()

	filterMatch, conditionMatch, _ := s.Evaluate(trace)

	assert.False(t, filterMatch)
	assert.False(t, conditionMatch)
}

// TestSpanAttribute_NotEquals_Match checks that the sampler matches
// when the attribute value is different from the expected value.
func TestSpanAttribute_NotEquals_Match(t *testing.T) {
	s := SpanAttributeRule{
		AttributeKey:          "env",
		AttributeCondition:    AttributeConditionNotEquals,
		ExpectedValue:         "prod",
		FallbackSamplingRatio: 12.0,
	}

	trace := testutil.NewTrace().
		AddResource("service").
		AddSpan("process", testutil.WithAttribute("env", "dev")).
		Done().
		Build()

	filterMatch, conditionMatch, fallback := s.Evaluate(trace)

	assert.True(t, filterMatch)
	assert.True(t, conditionMatch)
	assert.Equal(t, 12.0, fallback)
}

// TestSpanAttribute_NotEquals_NoMatch ensures the sampler does not match
// when the attribute value exactly equals the expected value.
func TestSpanAttribute_NotEquals_NoMatch(t *testing.T) {
	s := SpanAttributeRule{
		AttributeKey:          "env",
		AttributeCondition:    AttributeConditionNotEquals,
		ExpectedValue:         "prod",
		FallbackSamplingRatio: 12.0,
	}

	trace := testutil.NewTrace().
		AddResource("service").
		AddSpan("process", testutil.WithAttribute("env", "prod")).
		Done().
		Build()

	filterMatch, conditionMatch, _ := s.Evaluate(trace)

	assert.False(t, filterMatch)
	assert.False(t, conditionMatch)
}

// TestSpanAttribute_Validate tests validation logic for missing keys,
// missing values, and unsupported conditions.
func TestSpanAttribute_Validate(t *testing.T) {
	valid := SpanAttributeRule{
		AttributeKey:       "feature.flag",
		AttributeCondition: AttributeConditionEquals,
		ExpectedValue:      "enabled",
	}

	err := valid.Validate()
	assert.NoError(t, err)

	missingKey := SpanAttributeRule{
		AttributeKey:       "",
		AttributeCondition: AttributeConditionExists,
	}
	assert.ErrorContains(t, missingKey.Validate(), "attribute key cannot be empty")

	missingValue := SpanAttributeRule{
		AttributeKey:       "env",
		AttributeCondition: AttributeConditionEquals,
	}
	assert.ErrorContains(t, missingValue.Validate(), "expected_value must be set")

	invalidCondition := SpanAttributeRule{
		AttributeKey:       "env",
		AttributeCondition: "invalid",
	}
	assert.ErrorContains(t, invalidCondition.Validate(), "invalid attribute condition")
}

// TestSpanAttribute_MultipleSpans_OneMatch verifies that if any one span
// in the trace satisfies the rule, the entire trace is marked for sampling.
func TestSpanAttribute_MultipleSpans_OneMatch(t *testing.T) {
	s := SpanAttributeRule{
		AttributeKey:          "env",
		AttributeCondition:    AttributeConditionEquals,
		ExpectedValue:         "prod",
		FallbackSamplingRatio: 50.0,
	}

	trace := testutil.NewTrace().
		AddResource("multi").
		AddSpan("init", testutil.WithAttribute("env", "dev")).
		AddSpan("main", testutil.WithAttribute("env", "prod")).
		Done().Build()

	filterMatch, conditionMatch, fallback := s.Evaluate(trace)

	assert.True(t, filterMatch)
	assert.True(t, conditionMatch)
	assert.Equal(t, 50.0, fallback)
}

// TestSpanAttribute_NonStringAttribute ensures that attributes with non-string values
// are ignored and do not cause a crash or false match.
func TestSpanAttribute_NonStringAttribute(t *testing.T) {
	s := SpanAttributeRule{
		AttributeKey:          "http.status_code",
		AttributeCondition:    AttributeConditionEquals,
		ExpectedValue:         "200",
		FallbackSamplingRatio: 25.0,
	}

	trace := testutil.NewTrace().
		AddResource("service").
		AddSpan("span", func(span ptrace.Span) {
			span.Attributes().PutInt("http.status_code", 200)
		}).
		Done().Build()

	filterMatch, conditionMatch, fallback := s.Evaluate(trace)

	assert.False(t, filterMatch)
	assert.False(t, conditionMatch)
	assert.Equal(t, 25.0, fallback)
}

// TestSpanAttribute_EmptyTrace verifies that an empty trace does not crash
// and returns a fallback decision with no match.
func TestSpanAttribute_EmptyTrace(t *testing.T) {
	s := SpanAttributeRule{
		AttributeKey:          "env",
		AttributeCondition:    AttributeConditionExists,
		FallbackSamplingRatio: 42.0,
	}

	trace := testutil.NewTrace().Build()

	filterMatch, conditionMatch, fallback := s.Evaluate(trace)

	assert.False(t, filterMatch)
	assert.False(t, conditionMatch)
	assert.Equal(t, 42.0, fallback)
}
