package sampling

import (
	"errors"

	"go.opentelemetry.io/collector/pdata/ptrace"
)

type SpanAttributeExistsSampler struct {
	AttributeKey          string  `mapstructure:"attribute_key"`
	FallbackSamplingRatio float64 `mapstructure:"fallback_sampling_ratio"`
}

func (s *SpanAttributeExistsSampler) Validate() error {
	if s.AttributeKey == "" {
		return errors.New("attribute key cannot be empty")
	}
	return nil
}

func (s *SpanAttributeExistsSampler) Evaluate(td ptrace.Traces) (filterMatch bool, conditionMatch bool, fallbackRatio float64) {
	rs := td.ResourceSpans()
	for i := 0; i < rs.Len(); i++ {
		scopeSpans := rs.At(i).ScopeSpans()
		for j := 0; j < scopeSpans.Len(); j++ {
			spans := scopeSpans.At(j).Spans()
			for k := 0; k < spans.Len(); k++ {
				if _, ok := spans.At(k).Attributes().Get(s.AttributeKey); ok {
					return true, true, s.FallbackSamplingRatio
				}
			}
		}
	}
	return false, false, s.FallbackSamplingRatio
}
