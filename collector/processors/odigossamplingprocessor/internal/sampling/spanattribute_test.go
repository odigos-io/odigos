package sampling

import (
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/odigos/processor/odigossamplingprocessor/internal/sampling/testutil"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func buildTrace(serviceName string, attrs map[string]interface{}) ptrace.Traces {
	builder := testutil.NewTrace().AddResource(serviceName)
	builder.AddSpan("span", func(span ptrace.Span) {
		for k, v := range attrs {
			switch val := v.(type) {
			case string:
				span.Attributes().PutStr(k, val)
			case int:
				span.Attributes().PutInt(k, int64(val))
			case float64:
				span.Attributes().PutDouble(k, val)
			case bool:
				span.Attributes().PutBool(k, val)
			}
		}
	})
	return builder.Done().Build()
}

// TestSpanAttribute_StringCondition_Exists verifies that a trace is correctly sampled
// when a string attribute key exists.
func TestSpanAttribute_StringCondition_Exists(t *testing.T) {
	rule := SpanAttributeRule{
		ServiceName:           "test-service",
		AttributeKey:          "env",
		ConditionType:         TypeString,
		Operation:             "exists",
		FallbackSamplingRatio: 10.0,
	}

	trace := buildTrace("test-service", map[string]interface{}{"env": "prod"})
	filterMatch, conditionMatch, fallback := rule.Evaluate(trace)

	assert.True(t, filterMatch)
	assert.True(t, conditionMatch)
	assert.Equal(t, 10.0, fallback)
}

// TestSpanAttribute_StringCondition_Equals_Match ensures that a trace is correctly sampled
// when a string attribute matches the expected value exactly.
func TestSpanAttribute_StringCondition_Equals_Match(t *testing.T) {
	rule := SpanAttributeRule{
		ServiceName:           "test-service",
		AttributeKey:          "env",
		ConditionType:         TypeString,
		Operation:             "equals",
		ExpectedValue:         "prod",
		FallbackSamplingRatio: 5.0,
	}

	trace := buildTrace("test-service", map[string]interface{}{"env": "prod"})
	filterMatch, conditionMatch, fallback := rule.Evaluate(trace)

	assert.True(t, filterMatch)
	assert.True(t, conditionMatch)
	assert.Equal(t, 5.0, fallback)
}

// TestSpanAttribute_NumberCondition_GreaterThan_Match ensures that a trace is correctly sampled
// when a numeric attribute value is greater than the specified threshold.
func TestSpanAttribute_NumberCondition_GreaterThan_Match(t *testing.T) {
	rule := SpanAttributeRule{
		ServiceName:           "test-service",
		AttributeKey:          "latency",
		ConditionType:         TypeNumber,
		Operation:             "greater_than",
		ExpectedValue:         "100",
		FallbackSamplingRatio: 20.0,
	}

	trace := buildTrace("test-service", map[string]interface{}{"latency": 150})
	filterMatch, conditionMatch, fallback := rule.Evaluate(trace)

	assert.True(t, filterMatch)
	assert.True(t, conditionMatch)
	assert.Equal(t, 20.0, fallback)
}

// TestSpanAttribute_BooleanCondition_Equals_Match verifies that a trace is correctly sampled
// when a boolean attribute matches the expected value.
func TestSpanAttribute_BooleanCondition_Equals_Match(t *testing.T) {
	rule := SpanAttributeRule{
		ServiceName:           "test-service",
		AttributeKey:          "cache_hit",
		ConditionType:         TypeBoolean,
		Operation:             "equals",
		ExpectedValue:         "true",
		FallbackSamplingRatio: 15.0,
	}

	trace := buildTrace("test-service", map[string]interface{}{"cache_hit": true})
	filterMatch, conditionMatch, fallback := rule.Evaluate(trace)

	assert.True(t, filterMatch)
	assert.True(t, conditionMatch)
	assert.Equal(t, 15.0, fallback)
}

// TestSpanAttribute_JSONCondition_IsValidJSON_Match confirms that a trace is correctly sampled
// when a JSON attribute contains valid JSON.
func TestSpanAttribute_JSONCondition_IsValidJSON_Match(t *testing.T) {
	rule := SpanAttributeRule{
		ServiceName:           "test-service",
		AttributeKey:          "payload",
		ConditionType:         TypeJSON,
		Operation:             "is_valid_json",
		FallbackSamplingRatio: 25.0,
	}

	trace := buildTrace("test-service", map[string]interface{}{"payload": `{"valid":true}`})
	filterMatch, conditionMatch, fallback := rule.Evaluate(trace)

	assert.True(t, filterMatch)
	assert.True(t, conditionMatch)
	assert.Equal(t, 25.0, fallback)
}
