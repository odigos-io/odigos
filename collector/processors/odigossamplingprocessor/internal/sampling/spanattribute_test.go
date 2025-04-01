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

// TestSpanAttribute_StringCondition_NotEquals ensures sampling when a string attribute does not match the expected value.
func TestSpanAttribute_StringCondition_NotEquals(t *testing.T) {
	rule := SpanAttributeRule{
		ServiceName:           "test-service",
		AttributeKey:          "env",
		ConditionType:         TypeString,
		Operation:             "not_equals",
		ExpectedValue:         "staging",
		FallbackSamplingRatio: 5.0,
	}

	trace := buildTrace("test-service", map[string]interface{}{"env": "prod"})
	filterMatch, conditionMatch, fallback := rule.Evaluate(trace)
	assert.True(t, filterMatch)
	assert.True(t, conditionMatch)
	assert.Equal(t, 5.0, fallback)
}

// TestSpanAttribute_StringCondition_Contains verifies sampling when a string attribute contains the expected substring.
func TestSpanAttribute_StringCondition_Contains(t *testing.T) {
	rule := SpanAttributeRule{
		ServiceName:           "test-service",
		AttributeKey:          "env",
		ConditionType:         TypeString,
		Operation:             "contains",
		ExpectedValue:         "prod",
		FallbackSamplingRatio: 5.0,
	}

	trace := buildTrace("test-service", map[string]interface{}{"env": "production"})
	filterMatch, conditionMatch, fallback := rule.Evaluate(trace)
	assert.True(t, filterMatch)
	assert.True(t, conditionMatch)
	assert.Equal(t, 5.0, fallback)
}

// TestSpanAttribute_StringCondition_NotContains ensures sampling when a string attribute does not contain a given substring.
func TestSpanAttribute_StringCondition_NotContains(t *testing.T) {
	rule := SpanAttributeRule{
		ServiceName:           "test-service",
		AttributeKey:          "env",
		ConditionType:         TypeString,
		Operation:             "not_contains",
		ExpectedValue:         "dev",
		FallbackSamplingRatio: 5.0,
	}

	trace := buildTrace("test-service", map[string]interface{}{"env": "prod"})
	filterMatch, conditionMatch, fallback := rule.Evaluate(trace)
	assert.True(t, filterMatch)
	assert.True(t, conditionMatch)
	assert.Equal(t, 5.0, fallback)
}

// TestSpanAttribute_NumberCondition_Equals checks sampling when a numeric attribute exactly matches the expected value.
func TestSpanAttribute_NumberCondition_Equals(t *testing.T) {
	rule := SpanAttributeRule{
		ServiceName:           "test-service",
		AttributeKey:          "latency",
		ConditionType:         TypeNumber,
		Operation:             "equals",
		ExpectedValue:         "123.45",
		FallbackSamplingRatio: 30.0,
	}

	trace := buildTrace("test-service", map[string]interface{}{"latency": 123.45})
	filterMatch, conditionMatch, fallback := rule.Evaluate(trace)
	assert.True(t, filterMatch)
	assert.True(t, conditionMatch)
	assert.Equal(t, 30.0, fallback)
}

// TestSpanAttribute_NumberCondition_Exists verifies sampling when a numeric attribute key exists.
func TestSpanAttribute_NumberCondition_Exists(t *testing.T) {
	rule := SpanAttributeRule{
		ServiceName:           "test-service",
		AttributeKey:          "latency",
		ConditionType:         TypeNumber,
		Operation:             "exists",
		FallbackSamplingRatio: 10.0,
	}

	trace := buildTrace("test-service", map[string]interface{}{"latency": 77.0})
	filterMatch, conditionMatch, fallback := rule.Evaluate(trace)
	assert.True(t, filterMatch)
	assert.True(t, conditionMatch)
	assert.Equal(t, 10.0, fallback)
}

// TestSpanAttribute_BooleanCondition_Exists ensures sampling when a boolean attribute key exists.
func TestSpanAttribute_BooleanCondition_Exists(t *testing.T) {
	rule := SpanAttributeRule{
		ServiceName:           "test-service",
		AttributeKey:          "cache_hit",
		ConditionType:         TypeBoolean,
		Operation:             "exists",
		FallbackSamplingRatio: 20.0,
	}

	trace := buildTrace("test-service", map[string]interface{}{"cache_hit": false})
	filterMatch, conditionMatch, fallback := rule.Evaluate(trace)
	assert.True(t, filterMatch)
	assert.True(t, conditionMatch)
	assert.Equal(t, 20.0, fallback)
}

// TestSpanAttribute_JSONCondition_Exists confirms sampling when a JSON attribute key exists.
func TestSpanAttribute_JSONCondition_Exists(t *testing.T) {
	rule := SpanAttributeRule{
		ServiceName:           "test-service",
		AttributeKey:          "payload",
		ConditionType:         TypeJSON,
		Operation:             "exists",
		FallbackSamplingRatio: 50.0,
	}

	trace := buildTrace("test-service", map[string]interface{}{"payload": `{"foo": "bar"}`})
	filterMatch, conditionMatch, fallback := rule.Evaluate(trace)
	assert.True(t, filterMatch)
	assert.True(t, conditionMatch)
	assert.Equal(t, 50.0, fallback)
}

// TestSpanAttribute_JSONCondition_IsInvalidJSON verifies sampling when the attribute is invalid JSON.
func TestSpanAttribute_JSONCondition_IsInvalidJSON(t *testing.T) {
	rule := SpanAttributeRule{
		ServiceName:           "test-service",
		AttributeKey:          "payload",
		ConditionType:         TypeJSON,
		Operation:             "is_invalid_json",
		FallbackSamplingRatio: 33.0,
	}

	trace := buildTrace("test-service", map[string]interface{}{"payload": `{"invalid":`})
	filterMatch, conditionMatch, fallback := rule.Evaluate(trace)
	assert.True(t, filterMatch)
	assert.True(t, conditionMatch)
	assert.Equal(t, 33.0, fallback)
}

// TestSpanAttribute_JSONCondition_ContainsKey ensures sampling when a nested JSON key exists.
func TestSpanAttribute_JSONCondition_ContainsKey(t *testing.T) {
	rule := SpanAttributeRule{
		ServiceName:           "test-service",
		AttributeKey:          "payload",
		ConditionType:         TypeJSON,
		Operation:             "contains_key",
		ExpectedKey:           "foo.bar",
		FallbackSamplingRatio: 40.0,
	}

	trace := buildTrace("test-service", map[string]interface{}{"payload": `{"foo": {"bar": "baz"}}`})
	filterMatch, conditionMatch, fallback := rule.Evaluate(trace)
	assert.True(t, filterMatch)
	assert.True(t, conditionMatch)
	assert.Equal(t, 40.0, fallback)
}

// TestSpanAttribute_JSONCondition_NotContainsKey ensures sampling when a nested JSON key does not exist.
func TestSpanAttribute_JSONCondition_NotContainsKey(t *testing.T) {
	rule := SpanAttributeRule{
		ServiceName:           "test-service",
		AttributeKey:          "payload",
		ConditionType:         TypeJSON,
		Operation:             "not_contains_key",
		ExpectedKey:           "missing.key",
		FallbackSamplingRatio: 60.0,
	}

	trace := buildTrace("test-service", map[string]interface{}{"payload": `{"foo": "bar"}`})
	filterMatch, conditionMatch, fallback := rule.Evaluate(trace)
	assert.True(t, filterMatch)
	assert.True(t, conditionMatch)
	assert.Equal(t, 60.0, fallback)
}

// TestSpanAttribute_JSONCondition_JsonPathExists confirms sampling when a JSONPath expression resolves to a value.
func TestSpanAttribute_JSONCondition_JsonPathExists(t *testing.T) {
	rule := SpanAttributeRule{
		ServiceName:           "test-service",
		AttributeKey:          "payload",
		ConditionType:         TypeJSON,
		Operation:             "jsonpath_exists",
		JsonPath:              "$.foo.bar",
		FallbackSamplingRatio: 70.0,
	}

	trace := buildTrace("test-service", map[string]interface{}{"payload": `{"foo": {"bar": "value"}}`})
	filterMatch, conditionMatch, fallback := rule.Evaluate(trace)
	assert.True(t, filterMatch)
	assert.True(t, conditionMatch)
	assert.Equal(t, 70.0, fallback)
}

// TestSpanAttribute_JSONCondition_KeyEquals verifies sampling when a nested JSON key equals the expected value.
func TestSpanAttribute_JSONCondition_KeyEquals(t *testing.T) {
	rule := SpanAttributeRule{
		ServiceName:           "test-service",
		AttributeKey:          "payload",
		ConditionType:         TypeJSON,
		Operation:             "key_equals",
		ExpectedKey:           "foo.bar",
		ExpectedValue:         "123",
		FallbackSamplingRatio: 80.0,
	}

	trace := buildTrace("test-service", map[string]interface{}{"payload": `{"foo": {"bar": 123}}`})
	filterMatch, conditionMatch, fallback := rule.Evaluate(trace)
	assert.True(t, filterMatch)
	assert.True(t, conditionMatch)
	assert.Equal(t, 80.0, fallback)
}

// TestSpanAttribute_JSONCondition_KeyNotEquals confirms sampling when a nested JSON key does not match the expected value.
func TestSpanAttribute_JSONCondition_KeyNotEquals(t *testing.T) {
	rule := SpanAttributeRule{
		ServiceName:           "test-service",
		AttributeKey:          "payload",
		ConditionType:         TypeJSON,
		Operation:             "key_not_equals",
		ExpectedKey:           "foo.bar",
		ExpectedValue:         "wrong",
		FallbackSamplingRatio: 90.0,
	}

	trace := buildTrace("test-service", map[string]interface{}{"payload": `{"foo": {"bar": "right"}}`})
	filterMatch, conditionMatch, fallback := rule.Evaluate(trace)
	assert.True(t, filterMatch)
	assert.True(t, conditionMatch)
	assert.Equal(t, 90.0, fallback)
}
