package sampling

import (
	"errors"

	"go.opentelemetry.io/collector/pdata/ptrace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

type ServiceName struct {
	ServiceName           string  `mapstructure:"service_name"`
	FallbackSamplingRatio float64 `mapstructure:"fallback_sampling_ratio"`
}

func (s *ServiceName) Validate() error {
	if s.ServiceName == "" {
		return errors.New("service name cannot be empty")
	}
	return nil
}

func (s *ServiceName) Evaluate(td ptrace.Traces) (filterMatch bool, conditionMatch bool, fallbackRatio float64) {
	rs := td.ResourceSpans()
	for i := 0; i < rs.Len(); i++ {
		attrs := rs.At(i).Resource().Attributes()
		if val, ok := attrs.Get(string(semconv.ServiceNameKey)); ok && val.Str() == s.ServiceName {
			return true, true, s.FallbackSamplingRatio
		}
	}
	return false, false, s.FallbackSamplingRatio
}
