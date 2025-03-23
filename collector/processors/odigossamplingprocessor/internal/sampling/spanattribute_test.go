package sampling

import (
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/odigos/processor/odigossamplingprocessor/internal/sampling/testutil"
	"github.com/stretchr/testify/assert"
)

func TestSpanAttributeSampler_ExistsCondition(t *testing.T) {
	s := SpanAttributeSampler{
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

func TestSpanAttributeSampler_Equals_Match(t *testing.T) {
	s := SpanAttributeSampler{
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

func TestSpanAttributeSampler_Equals_NoMatch(t *testing.T) {
	s := SpanAttributeSampler{
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

func TestSpanAttributeSampler_NotEquals_Match(t *testing.T) {
	s := SpanAttributeSampler{
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

func TestSpanAttributeSampler_NotEquals_NoMatch(t *testing.T) {
	s := SpanAttributeSampler{
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

func TestSpanAttributeSampler_Validate(t *testing.T) {
	valid := SpanAttributeSampler{
		AttributeKey:       "feature.flag",
		AttributeCondition: AttributeConditionEquals,
		ExpectedValue:      "enabled",
	}

	err := valid.Validate()
	assert.NoError(t, err)

	missingKey := SpanAttributeSampler{
		AttributeKey:       "",
		AttributeCondition: AttributeConditionExists,
	}
	assert.ErrorContains(t, missingKey.Validate(), "attribute key cannot be empty")

	missingValue := SpanAttributeSampler{
		AttributeKey:       "env",
		AttributeCondition: AttributeConditionEquals,
	}
	assert.ErrorContains(t, missingValue.Validate(), "expected_value must be set")

	invalidCondition := SpanAttributeSampler{
		AttributeKey:       "env",
		AttributeCondition: "invalid",
	}
	assert.ErrorContains(t, invalidCondition.Validate(), "invalid attribute condition")
}
