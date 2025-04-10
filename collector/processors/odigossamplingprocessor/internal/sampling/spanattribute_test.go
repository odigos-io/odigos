package sampling

import (
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/odigos/processor/odigossamplingprocessor/internal/sampling/testutil"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// buildTrace creates a trace with the given service name and span attributes.
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

// ----------- String Conditions -----------

// Exists
func TestSpanAttribute_StringCondition_Exists(t *testing.T) {
	rule := SpanAttributeRule{
		ServiceName:           "test-service",
		AttributeKey:          "env",
		ConditionType:         TypeString,
		Operation:             "exists",
		FallbackSamplingRatio: 10.0,
	}
	trace := buildTrace("test-service", map[string]interface{}{"env": "prod"})
	matched, satisfied, ratio := rule.Evaluate(trace)
	assert.True(t, matched)
	assert.True(t, satisfied)
	assert.Equal(t, 10.0, ratio)
}

// Equals
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
	matched, satisfied, ratio := rule.Evaluate(trace)
	assert.True(t, matched)
	assert.True(t, satisfied)
	assert.Equal(t, 5.0, ratio)
}

// Not Equals
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
	matched, satisfied, ratio := rule.Evaluate(trace)
	assert.True(t, matched)
	assert.True(t, satisfied)
	assert.Equal(t, 5.0, ratio)
}

// Contains
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
	matched, satisfied, ratio := rule.Evaluate(trace)
	assert.True(t, matched)
	assert.True(t, satisfied)
	assert.Equal(t, 5.0, ratio)
}

// Not Contains
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
	matched, satisfied, ratio := rule.Evaluate(trace)
	assert.True(t, matched)
	assert.True(t, satisfied)
	assert.Equal(t, 5.0, ratio)
}

// Regex Match
func TestSpanAttribute_StringCondition_Regex_Match(t *testing.T) {
	rule := SpanAttributeRule{
		ServiceName:           "test-service",
		AttributeKey:          "version",
		ConditionType:         TypeString,
		Operation:             "regex",
		ExpectedValue:         "^v[0-9]+\\.[0-9]+$",
		FallbackSamplingRatio: 12.0,
	}
	trace := buildTrace("test-service", map[string]interface{}{"version": "v1.23"})
	matched, satisfied, ratio := rule.Evaluate(trace)
	assert.True(t, matched)
	assert.True(t, satisfied)
	assert.Equal(t, 12.0, ratio)
}

// Regex No Match
func TestSpanAttribute_StringCondition_Regex_NoMatch(t *testing.T) {
	rule := SpanAttributeRule{
		ServiceName:           "test-service",
		AttributeKey:          "version",
		ConditionType:         TypeString,
		Operation:             "regex",
		ExpectedValue:         "^v[0-9]+\\.[0-9]+$",
		FallbackSamplingRatio: 12.0,
	}
	trace := buildTrace("test-service", map[string]interface{}{"version": "version1.23"})
	matched, satisfied, ratio := rule.Evaluate(trace)
	assert.False(t, matched)
	assert.False(t, satisfied)
	assert.Equal(t, 12.0, ratio)
}

// ----------- Number Conditions -----------

// Greater Than
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
	matched, satisfied, ratio := rule.Evaluate(trace)
	assert.True(t, matched)
	assert.True(t, satisfied)
	assert.Equal(t, 20.0, ratio)
}

// Equals
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
	matched, satisfied, ratio := rule.Evaluate(trace)
	assert.True(t, matched)
	assert.True(t, satisfied)
	assert.Equal(t, 30.0, ratio)
}

// Exists (Number)
func TestSpanAttribute_NumberCondition_Exists(t *testing.T) {
	rule := SpanAttributeRule{
		ServiceName:           "test-service",
		AttributeKey:          "latency",
		ConditionType:         TypeNumber,
		Operation:             "exists",
		FallbackSamplingRatio: 10.0,
	}
	trace := buildTrace("test-service", map[string]interface{}{"latency": 77.0})
	matched, satisfied, ratio := rule.Evaluate(trace)
	assert.True(t, matched)
	assert.True(t, satisfied)
	assert.Equal(t, 10.0, ratio)
}

// Greater Than or Equal & Less Than or Equal
func TestSpanAttribute_NumberCondition_GTE_LTE(t *testing.T) {
	// Greater Than or Equal
	ruleGTE := SpanAttributeRule{
		ServiceName:           "test-service",
		AttributeKey:          "duration",
		ConditionType:         TypeNumber,
		Operation:             "greater_than_or_equal",
		ExpectedValue:         "50",
		FallbackSamplingRatio: 18.0,
	}
	trace1 := buildTrace("test-service", map[string]interface{}{"duration": 50})
	fm1, cm1, fb1 := ruleGTE.Evaluate(trace1)
	assert.True(t, fm1)
	assert.True(t, cm1)
	assert.Equal(t, 18.0, fb1)

	// Less Than or Equal
	ruleLTE := SpanAttributeRule{
		ServiceName:           "test-service",
		AttributeKey:          "duration",
		ConditionType:         TypeNumber,
		Operation:             "less_than_or_equal",
		ExpectedValue:         "100",
		FallbackSamplingRatio: 22.0,
	}
	trace2 := buildTrace("test-service", map[string]interface{}{"duration": 100})
	fm2, cm2, fb2 := ruleLTE.Evaluate(trace2)
	assert.True(t, fm2)
	assert.True(t, cm2)
	assert.Equal(t, 22.0, fb2)
}

// ----------- Boolean Conditions -----------

// Equals (Boolean)
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
	matched, satisfied, ratio := rule.Evaluate(trace)
	assert.True(t, matched)
	assert.True(t, satisfied)
	assert.Equal(t, 15.0, ratio)
}

// Exists (Boolean)
func TestSpanAttribute_BooleanCondition_Exists(t *testing.T) {
	rule := SpanAttributeRule{
		ServiceName:           "test-service",
		AttributeKey:          "cache_hit",
		ConditionType:         TypeBoolean,
		Operation:             "exists",
		FallbackSamplingRatio: 20.0,
	}
	trace := buildTrace("test-service", map[string]interface{}{"cache_hit": false})
	matched, satisfied, ratio := rule.Evaluate(trace)
	assert.True(t, matched)
	assert.True(t, satisfied)
	assert.Equal(t, 20.0, ratio)
}

// ----------- JSON Conditions -----------

// Exists (JSON)
func TestSpanAttribute_JSONCondition_Exists(t *testing.T) {
	rule := SpanAttributeRule{
		ServiceName:           "test-service",
		AttributeKey:          "payload",
		ConditionType:         TypeJSON,
		Operation:             "exists",
		JsonPath:              "$", // check the root
		FallbackSamplingRatio: 50.0,
	}
	trace := buildTrace("test-service", map[string]interface{}{"payload": `{"foo": "bar"}`})
	matched, satisfied, ratio := rule.Evaluate(trace)
	assert.True(t, matched)
	assert.True(t, satisfied)
	assert.Equal(t, 50.0, ratio)
}

// is_valid_json
func TestSpanAttribute_JSONCondition_IsValidJSON_Match(t *testing.T) {
	rule := SpanAttributeRule{
		ServiceName:           "test-service",
		AttributeKey:          "payload",
		ConditionType:         TypeJSON,
		Operation:             "is_valid_json",
		JsonPath:              "$", // check the root
		FallbackSamplingRatio: 25.0,
	}
	trace := buildTrace("test-service", map[string]interface{}{"payload": `{"valid":true}`})
	matched, satisfied, ratio := rule.Evaluate(trace)
	assert.True(t, matched)
	assert.True(t, satisfied)
	assert.Equal(t, 25.0, ratio)
}

// is_invalid_json
func TestSpanAttribute_JSONCondition_IsInvalidJSON(t *testing.T) {
	rule := SpanAttributeRule{
		ServiceName:           "test-service",
		AttributeKey:          "payload",
		ConditionType:         TypeJSON,
		Operation:             "is_invalid_json",
		JsonPath:              "$",
		FallbackSamplingRatio: 33.0,
	}
	trace := buildTrace("test-service", map[string]interface{}{"payload": `{"invalid":`})
	matched, satisfied, ratio := rule.Evaluate(trace)
	assert.True(t, matched)
	assert.True(t, satisfied)
	assert.Equal(t, 33.0, ratio)
}

// jsonpath_exists
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
	matched, satisfied, ratio := rule.Evaluate(trace)
	assert.True(t, matched)
	assert.True(t, satisfied)
	assert.Equal(t, 70.0, ratio)
}

// contains_key: sample if the JSON key exists (using JsonPath)
func TestSpanAttribute_JSONCondition_ContainsKey(t *testing.T) {
	rule := SpanAttributeRule{
		ServiceName:           "test-service",
		AttributeKey:          "payload",
		ConditionType:         TypeJSON,
		Operation:             "contains_key",
		JsonPath:              "$.foo.bar",
		FallbackSamplingRatio: 40.0,
	}
	trace := buildTrace("test-service", map[string]interface{}{"payload": `{"foo": {"bar": "baz"}}`})
	matched, satisfied, ratio := rule.Evaluate(trace)
	assert.True(t, matched)
	assert.True(t, satisfied)
	assert.Equal(t, 40.0, ratio)
}

// not_contains_key: sample if the JSON key does not exist.
func TestSpanAttribute_JSONCondition_NotContainsKey(t *testing.T) {
	rule := SpanAttributeRule{
		ServiceName:           "test-service",
		AttributeKey:          "payload",
		ConditionType:         TypeJSON,
		Operation:             "not_contains_key",
		JsonPath:              "$.missing.key",
		FallbackSamplingRatio: 60.0,
	}
	trace := buildTrace("test-service", map[string]interface{}{"payload": `{"foo": "bar"}`})
	matched, satisfied, ratio := rule.Evaluate(trace)
	assert.True(t, matched)
	assert.True(t, satisfied)
	assert.Equal(t, 60.0, ratio)
}

// key_equals: sample if the JSON value at the given JsonPath equals ExpectedValue.
func TestSpanAttribute_JSONCondition_KeyEquals(t *testing.T) {
	rule := SpanAttributeRule{
		ServiceName:           "test-service",
		AttributeKey:          "payload",
		ConditionType:         TypeJSON,
		Operation:             "key_equals",
		JsonPath:              "$.foo.bar",
		ExpectedValue:         "123",
		FallbackSamplingRatio: 80.0,
	}
	trace := buildTrace("test-service", map[string]interface{}{"payload": `{"foo": {"bar": 123}}`})
	matched, satisfied, ratio := rule.Evaluate(trace)
	assert.True(t, matched)
	assert.True(t, satisfied)
	assert.Equal(t, 80.0, ratio)
}

// key_not_equals: sample if the JSON value at the given JsonPath does not equal ExpectedValue.
func TestSpanAttribute_JSONCondition_KeyNotEquals(t *testing.T) {
	rule := SpanAttributeRule{
		ServiceName:           "test-service",
		AttributeKey:          "payload",
		ConditionType:         TypeJSON,
		Operation:             "key_not_equals",
		JsonPath:              "$.foo.bar",
		ExpectedValue:         "wrong",
		FallbackSamplingRatio: 90.0,
	}
	trace := buildTrace("test-service", map[string]interface{}{"payload": `{"foo": {"bar": "right"}}`})
	matched, satisfied, ratio := rule.Evaluate(trace)
	assert.True(t, matched)
	assert.True(t, satisfied)
	assert.Equal(t, 90.0, ratio)
}
