package sampling

import (
	"errors"

	"go.opentelemetry.io/collector/pdata/ptrace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

type ServiceNameRule struct {
	ServiceName           string  `mapstructure:"service_name"`
	SamplingRatio         float64 `mapstructure:"sampling_ratio"`
	FallbackSamplingRatio float64 `mapstructure:"fallback_sampling_ratio"`
}

var _ SamplingDecision = (*ServiceNameRule)(nil)

func (s *ServiceNameRule) Validate() error {
	if s.ServiceName == "" {
		return errors.New("service name cannot be empty")
	}
	if s.SamplingRatio < 0 || s.SamplingRatio > 100 {
		return errors.New("sampling ratio must be between 0 and 100")
	}
	if s.FallbackSamplingRatio < 0 || s.FallbackSamplingRatio > 100 {
		return errors.New("fallback sampling ratio must be between 0 and 100")
	}
	return nil
}

func (s *ServiceNameRule) Evaluate(td ptrace.Traces) (filterMatch, conditionMatch bool, fallbackRatio float64) {
	rs := td.ResourceSpans()

	for i := range rs.Len() {
		resourceAttrs := rs.At(i).Resource().Attributes()
		serviceAttr, ok := resourceAttrs.Get(string(semconv.ServiceNameKey))
		if !ok || serviceAttr.Str() != s.ServiceName {
			continue
		}

		// Matched a span from the target service
		filterMatch = true
		conditionMatch = true // Report that this rule is satisfied
		return filterMatch, conditionMatch, s.SamplingRatio
	}

	// No match → report fallback sampling ratio
	return false, false, s.FallbackSamplingRatio
}
