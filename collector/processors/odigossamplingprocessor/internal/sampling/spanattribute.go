package sampling

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/PaesslerAG/jsonpath"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type AttributeConditionType string

const (
	TypeString  AttributeConditionType = "string"
	TypeNumber  AttributeConditionType = "number"
	TypeBoolean AttributeConditionType = "boolean"
	TypeJSON    AttributeConditionType = "json"
)

type SpanAttributeRule struct {
	ServiceName           string                 `mapstructure:"service_name"`
	AttributeKey          string                 `mapstructure:"attribute_key"`
	ConditionType         AttributeConditionType `mapstructure:"condition_type"`
	Operation             string                 `mapstructure:"operation"`
	ExpectedValue         string                 `mapstructure:"expected_value,omitempty"`
	JsonPath              string                 `mapstructure:"json_path,omitempty"`
	FallbackSamplingRatio float64                `mapstructure:"fallback_sampling_ratio"`
}

var _ SamplingDecision = (*SpanAttributeRule)(nil)

func (s *SpanAttributeRule) Validate() error {
	if s.ServiceName == "" {
		return errors.New("service_name cannot be empty")
	}
	if s.AttributeKey == "" {
		return errors.New("attribute_key cannot be empty")
	}
	switch s.ConditionType {
	case TypeString:
		validOps := map[string]bool{"exists": true, "equals": true, "not_equals": true, "contains": true, "not_contains": true}
		if !validOps[s.Operation] {
			return errors.New("invalid string operation")
		}
		if s.Operation != "exists" && s.ExpectedValue == "" {
			return errors.New("expected_value required for string operations")
		}
	case TypeNumber:
		validOps := map[string]bool{"exists": true, "equals": true, "not_equals": true, "greater_than": true, "less_than": true}
		if !validOps[s.Operation] {
			return errors.New("invalid number operation")
		}
		if s.Operation != "exists" && s.ExpectedValue == "" {
			return errors.New("expected_value required for number operations")
		}
	case TypeBoolean:
		validOps := map[string]bool{"exists": true, "equals": true}
		if !validOps[s.Operation] {
			return errors.New("invalid boolean operation")
		}
		if s.Operation == "equals" && s.ExpectedValue == "" {
			return errors.New("expected_value required for boolean equals operation")
		}
	case TypeJSON:
		validOps := map[string]bool{"exists": true, "is_valid_json": true, "is_invalid_json": true, "contains_key": true, "jsonpath_exists": true}
		if !validOps[s.Operation] {
			return errors.New("invalid json operation")
		}
		if (s.Operation == "contains_key") && s.ExpectedValue == "" {
			return errors.New("expected_value required for json contains_key")
		}
		if s.Operation == "jsonpath_exists" && s.JsonPath == "" {
			return errors.New("json_path required for jsonpath_exists")
		}
	default:
		return errors.New("unsupported condition type")
	}
	return nil
}

func (s *SpanAttributeRule) Evaluate(td ptrace.Traces) (filterMatch, conditionMatch bool, fallbackRatio float64) {
	rs := td.ResourceSpans()
	for i := 0; i < rs.Len(); i++ {
		resourceAttrs := rs.At(i).Resource().Attributes()
		if svcAttr, ok := resourceAttrs.Get("service.name"); !ok || svcAttr.AsString() != s.ServiceName {
			continue
		}

		scopeSpans := rs.At(i).ScopeSpans()
		for j := 0; j < scopeSpans.Len(); j++ {
			spans := scopeSpans.At(j).Spans()
			for k := 0; k < spans.Len(); k++ {
				attr, found := spans.At(k).Attributes().Get(s.AttributeKey)
				if !found {
					continue
				}

				filterMatch = true
				switch s.ConditionType {
				case TypeString:
					if attr.Type() != pcommon.ValueTypeStr {
						continue
					}
					val := attr.AsString()
					switch s.Operation {
					case "exists":
						return true, true, s.FallbackSamplingRatio
					case "equals":
						if val == s.ExpectedValue {
							return true, true, s.FallbackSamplingRatio
						}
					case "not_equals":
						if val != s.ExpectedValue {
							return true, true, s.FallbackSamplingRatio
						}
					case "contains":
						if strings.Contains(val, s.ExpectedValue) {
							return true, true, s.FallbackSamplingRatio
						}
					case "not_contains":
						if !strings.Contains(val, s.ExpectedValue) {
							return true, true, s.FallbackSamplingRatio
						}
					}
				case TypeNumber:
					numVal, err := strconv.ParseFloat(s.ExpectedValue, 64)
					if err != nil {
						continue
					}

					var attrNum float64
					switch attr.Type() {
					case pcommon.ValueTypeInt:
						attrNum = float64(attr.Int())
					case pcommon.ValueTypeDouble:
						attrNum = attr.Double()
					default:
						continue
					}

					switch s.Operation {
					case "exists":
						return true, true, s.FallbackSamplingRatio
					case "equals":
						if attrNum == numVal {
							return true, true, s.FallbackSamplingRatio
						}
					case "greater_than":
						if attrNum > numVal {
							return true, true, s.FallbackSamplingRatio
						}
					case "less_than":
						if attrNum < numVal {
							return true, true, s.FallbackSamplingRatio
						}
					case "not_equals":
						if attrNum != numVal {
							return true, true, s.FallbackSamplingRatio
						}
					}
				case TypeBoolean:
					expectedBool, err := strconv.ParseBool(s.ExpectedValue)
					if err != nil || attr.Type() != pcommon.ValueTypeBool {
						continue
					}
					attrBool := attr.Bool()
					switch s.Operation {
					case "exists":
						return true, true, s.FallbackSamplingRatio
					case "equals":
						if attrBool == expectedBool {
							return true, true, s.FallbackSamplingRatio
						}
					}
				case TypeJSON:
					if attr.Type() != pcommon.ValueTypeStr {
						continue
					}
					var jsonVal interface{}
					err := json.Unmarshal([]byte(attr.AsString()), &jsonVal)

					switch s.Operation {
					case "exists":
						return true, true, s.FallbackSamplingRatio
					case "is_valid_json":
						if err == nil {
							return true, true, s.FallbackSamplingRatio
						}
					case "is_invalid_json":
						if err != nil {
							return true, true, s.FallbackSamplingRatio
						}
					case "contains_key":
						if err == nil {
							if m, ok := jsonVal.(map[string]interface{}); ok {
								if _, found := m[s.ExpectedValue]; found {
									return true, true, s.FallbackSamplingRatio
								}
							}
						}
					case "jsonpath_exists":
						if err == nil {
							if _, err := jsonpath.Get(s.JsonPath, jsonVal); err == nil {
								return true, true, s.FallbackSamplingRatio
							}
						}
					}
				}
			}
		}
	}
	return false, false, s.FallbackSamplingRatio
}
