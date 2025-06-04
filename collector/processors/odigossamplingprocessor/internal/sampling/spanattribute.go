package sampling

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
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
	SamplingRatio         float64                `mapstructure:"sampling_ratio"`
	FallbackSamplingRatio float64                `mapstructure:"fallback_sampling_ratio"`
}

var _ SamplingDecision = (*SpanAttributeRule)(nil)

func (s *SpanAttributeRule) Validate() error {
	if s.SamplingRatio < 0 || s.SamplingRatio > 100 {
		return errors.New("sampling ratio must be between 0 and 100")
	}
	if s.FallbackSamplingRatio < 0 || s.FallbackSamplingRatio > 100 {
		return errors.New("fallback sampling ratio must be between 0 and 100")
	}

	if s.ServiceName == "" {
		return errors.New("service_name cannot be empty")
	}
	if s.AttributeKey == "" {
		return errors.New("attribute_key cannot be empty")
	}
	switch s.ConditionType {
	case TypeString:
		validOps := map[string]bool{
			"exists":       true,
			"equals":       true,
			"not_equals":   true,
			"contains":     true,
			"not_contains": true,
			"regex":        true,
		}
		if !validOps[s.Operation] {
			return errors.New("invalid string operation")
		}
		if s.Operation != "exists" && s.ExpectedValue == "" {
			return errors.New("expected_value required for string operations")
		}
	case TypeNumber:
		validOps := map[string]bool{
			"exists":                true,
			"equals":                true,
			"not_equals":            true,
			"greater_than":          true,
			"less_than":             true,
			"greater_than_or_equal": true,
			"less_than_or_equal":    true,
		}
		if !validOps[s.Operation] {
			return errors.New("invalid number operation")
		}
		if s.Operation != "exists" && s.ExpectedValue == "" {
			return errors.New("expected_value required for number operations")
		}
	case TypeBoolean:
		validOps := map[string]bool{
			"exists": true,
			"equals": true,
		}
		if !validOps[s.Operation] {
			return errors.New("invalid boolean operation")
		}
		if s.Operation == "equals" && s.ExpectedValue == "" {
			return errors.New("expected_value required for boolean equals operation")
		}
	case TypeJSON:
		validOps := map[string]bool{
			"exists":           true,
			"is_valid_json":    true,
			"is_invalid_json":  true,
			"jsonpath_exists":  true,
			"contains_key":     true,
			"not_contains_key": true,
			"key_equals":       true,
			"key_not_equals":   true,
		}
		if !validOps[s.Operation] {
			return errors.New("invalid json operation")
		}
		// For some JSON operations, a jsonPath is required.
		if (s.Operation != "exists" && s.Operation != "is_valid_json" && s.Operation != "is_invalid_json") && s.JsonPath == "" {
			return errors.New("json_path required for json operations")
		}
		if (s.Operation == "key_equals" || s.Operation == "key_not_equals") && s.ExpectedValue == "" {
			return errors.New("expected_value required for key comparison")
		}
	default:
		return fmt.Errorf("unsupported condition type: %q", s.ConditionType)
	}
	return nil
}

// Evaluate checks if the trace contains a specific attribute and if it matching the attribute condition
// - matched: Attribute exists in the trace.
// - satisfied: The attribute rule is satisfied with it's value.
// - samplingRatio: sample ration on satisfy and fallback ration otherwise.
func (s *SpanAttributeRule) Evaluate(td ptrace.Traces) (bool, bool, float64) {
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
				// At this point, the attribute exists.
				switch s.ConditionType {
				case TypeString:
					if s.Operation == "exists" {
						if attr.Type() == pcommon.ValueTypeStr && attr.AsString() != "" {
							return true, true, s.SamplingRatio
						}
					}
					if attr.Type() != pcommon.ValueTypeStr {
						continue
					}
					val := attr.AsString()
					switch s.Operation {
					case "equals":
						if val == s.ExpectedValue {
							return true, true, s.SamplingRatio
						}
					case "not_equals":
						if val != s.ExpectedValue {
							return true, true, s.SamplingRatio
						}
					case "contains":
						if strings.Contains(val, s.ExpectedValue) {
							return true, true, s.SamplingRatio
						}
					case "not_contains":
						if !strings.Contains(val, s.ExpectedValue) {
							return true, true, s.SamplingRatio
						}
					case "regex":
						re, err := regexp.Compile(s.ExpectedValue)
						if err != nil {
							continue
						}
						if re.MatchString(val) {
							return true, true, s.SamplingRatio
						}
					}
				case TypeNumber:
					if s.Operation == "exists" {
						if attr.Type() == pcommon.ValueTypeInt || attr.Type() == pcommon.ValueTypeDouble {
							return true, true, s.SamplingRatio
						}
					}
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
					case "equals":
						if attrNum == numVal {
							return true, true, s.SamplingRatio
						}
					case "not_equals":
						if attrNum != numVal {
							return true, true, s.SamplingRatio
						}
					case "greater_than":
						if attrNum > numVal {
							return true, true, s.SamplingRatio
						}
					case "less_than":
						if attrNum < numVal {
							return true, true, s.SamplingRatio
						}
					case "greater_than_or_equal":
						if attrNum >= numVal {
							return true, true, s.SamplingRatio
						}
					case "less_than_or_equal":
						if attrNum <= numVal {
							return true, true, s.SamplingRatio
						}
					}
				case TypeBoolean:
					if s.Operation == "exists" {
						if attr.Type() == pcommon.ValueTypeBool {
							return true, true, s.SamplingRatio
						}
					}
					expectedBool, err := strconv.ParseBool(s.ExpectedValue)
					if err != nil || attr.Type() != pcommon.ValueTypeBool {
						continue
					}
					if s.Operation == "equals" && attr.Bool() == expectedBool {
						return true, true, s.SamplingRatio
					}
				case TypeJSON:
					if attr.Type() != pcommon.ValueTypeStr {
						continue
					}
					jsonStr := attr.AsString()
					var jsonVal interface{}
					err := json.Unmarshal([]byte(jsonStr), &jsonVal)
					switch s.Operation {
					case "is_valid_json":
						if err == nil {
							return true, true, s.SamplingRatio
						}
					case "is_invalid_json":
						if err != nil {
							return true, true, s.SamplingRatio
						}
					case "contains_key":
						if err == nil {
							if res, err2 := jsonpath.Get(s.JsonPath, jsonVal); err2 == nil && res != nil {
								return true, true, s.SamplingRatio
							}
						}
					case "not_contains_key":
						if err == nil {
							// If the key does not exist, jsonpath.Get should return an error.
							if _, err2 := jsonpath.Get(s.JsonPath, jsonVal); err2 != nil {
								return true, true, s.SamplingRatio
							}
						}
					case "key_equals":
						if err == nil {
							res, err2 := jsonpath.Get(s.JsonPath, jsonVal)
							if err2 != nil {
								continue
							}
							valStr := ""
							switch v := res.(type) {
							case string:
								valStr = v
							case float64:
								valStr = strconv.FormatFloat(v, 'f', -1, 64)
							case bool:
								valStr = strconv.FormatBool(v)
							case nil:
								valStr = "null"
							default:
								b, _ := json.Marshal(v)
								valStr = string(b)
							}
							if valStr == s.ExpectedValue {
								return true, true, s.SamplingRatio
							}
						}
					case "key_not_equals":
						if err == nil {
							res, err2 := jsonpath.Get(s.JsonPath, jsonVal)
							if err2 != nil {
								continue
							}
							valStr := ""
							switch v := res.(type) {
							case string:
								valStr = v
							case float64:
								valStr = strconv.FormatFloat(v, 'f', -1, 64)
							case bool:
								valStr = strconv.FormatBool(v)
							case nil:
								valStr = "null"
							default:
								b, _ := json.Marshal(v)
								valStr = string(b)
							}
							if valStr != s.ExpectedValue {
								return true, true, s.SamplingRatio
							}
						}
					}
				}
			}
		}
	}
	return false, false, s.FallbackSamplingRatio
}
