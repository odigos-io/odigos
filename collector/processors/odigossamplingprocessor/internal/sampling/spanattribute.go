package sampling

import (
	"errors"
	"strings"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type SpanAttributeCondition string

const (
	AttributeConditionExists    SpanAttributeCondition = "exists"
	AttributeConditionEquals    SpanAttributeCondition = "equals"
	AttributeConditionNotEquals SpanAttributeCondition = "not_equals"
)

type SpanAttributeRule struct {
	AttributeKey          string                 `mapstructure:"attribute_key"`
	AttributeCondition    SpanAttributeCondition `mapstructure:"condition"`
	ExpectedValue         string                 `mapstructure:"expected_value,omitempty"`
	FallbackSamplingRatio float64                `mapstructure:"fallback_sampling_ratio"`
}

var _ SamplingDecision = (*SpanAttributeRule)(nil)

func (s *SpanAttributeRule) Validate() error {
	if s.AttributeKey == "" {
		return errors.New("attribute key cannot be empty")
	}
	switch s.AttributeCondition {
	case AttributeConditionExists:
		// no value needed
	case AttributeConditionEquals, AttributeConditionNotEquals:
		if s.ExpectedValue == "" {
			return errors.New("expected_value must be set for 'equals' and 'not_equals'")
		}
	default:
		return errors.New("invalid attribute condition")
	}
	return nil
}

func (s *SpanAttributeRule) Evaluate(td ptrace.Traces) (filterMatch bool, conditionMatch bool, fallbackRatio float64) {
	rs := td.ResourceSpans()
	for i := 0; i < rs.Len(); i++ {
		scopeSpans := rs.At(i).ScopeSpans()
		for j := 0; j < scopeSpans.Len(); j++ {
			spans := scopeSpans.At(j).Spans()
			for k := 0; k < spans.Len(); k++ {
				attrs := spans.At(k).Attributes()
				val, found := attrs.Get(s.AttributeKey)
				if !found || val.Type() != pcommon.ValueTypeStr {
					continue // skip if not found or not a string
				}

				switch s.AttributeCondition {
				case AttributeConditionExists:
					return true, true, s.FallbackSamplingRatio
				case AttributeConditionEquals:
					if strings.EqualFold(val.AsString(), s.ExpectedValue) {
						return true, true, s.FallbackSamplingRatio
					}
				case AttributeConditionNotEquals:
					if !strings.EqualFold(val.AsString(), s.ExpectedValue) {
						return true, true, s.FallbackSamplingRatio
					}
				}
			}
		}
	}
	return false, false, s.FallbackSamplingRatio
}
