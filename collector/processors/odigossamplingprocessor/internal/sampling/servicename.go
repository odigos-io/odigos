package sampling

import (
	"errors"

	"go.opentelemetry.io/collector/pdata/ptrace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.uber.org/zap"
)

type ServiceNameRule struct {
	ServiceName           string  `mapstructure:"service_name"`
	FallbackSamplingRatio float64 `mapstructure:"fallback_sampling_ratio"`
}

var _ SamplingDecision = (*ServiceNameRule)(nil)

func (s *ServiceNameRule) Validate() error {
	if s.ServiceName == "" {
		return errors.New("service name cannot be empty")
	}
	return nil
}

func (s *ServiceNameRule) Evaluate(td ptrace.Traces) (filterMatch bool, conditionMatch bool, fallbackRatio float64) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	rs := td.ResourceSpans()
	for i := 0; i < rs.Len(); i++ {
		attrs := rs.At(i).Resource().Attributes()
		if val, ok := attrs.Get(string(semconv.ServiceNameKey)); ok && val.Str() == s.ServiceName {
			logger.Info("ServiceNameRule matched: trace contains target service",
				zap.String("matched_service", s.ServiceName),
				zap.Float64("fallback_sampling_ratio", s.FallbackSamplingRatio),
			)
			return true, true, s.FallbackSamplingRatio
		}
	}
	logger.Info("ServiceNameRule not matched: no target service found in trace",
		zap.String("expected_service", s.ServiceName),
		zap.Float64("fallback_sampling_ratio", s.FallbackSamplingRatio),
	)
	return false, false, s.FallbackSamplingRatio
}
