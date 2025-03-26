package sampling

import (
	"errors"
	"math/rand"

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

func (s *ServiceNameRule) Evaluate(td ptrace.Traces) (filterMatch bool, conditionMatch bool, fallbackRatio float64) {
	rs := td.ResourceSpans()
	for i := 0; i < rs.Len(); i++ {
		attrs := rs.At(i).Resource().Attributes()
		if val, ok := attrs.Get(string(semconv.ServiceNameKey)); ok && val.Str() == s.ServiceName {
			filterMatch = true
			if rand.Float64()*100 < s.SamplingRatio {
				// Trace matched and passes the sampling ratio
				return true, true, s.FallbackSamplingRatio
			} else {
				// Trace matched service, but didn't pass the sampling ratio
				// Set fallback sampling ratio to 0 to ensure the trace won't be sampled at the last exhausted filter
				return true, false, 0
			}
		}
	}
	return false, false, s.FallbackSamplingRatio
}
