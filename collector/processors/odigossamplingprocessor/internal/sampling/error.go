package sampling

import (
	"errors"

	"go.opentelemetry.io/collector/pdata/ptrace"
)

type ErrorRule struct {
	FallbackSamplingRatio float64 `mapstructure:"fallback_sampling_ratio"`
}

var _ SamplingDecision = (*ErrorRule)(nil)

// Validate ensures the rule's configuration is correct.
func (r *ErrorRule) Validate() error {
	if r.FallbackSamplingRatio < 0 || r.FallbackSamplingRatio > 100 {
		return errors.New("fallback_sampling_ratio must be between 0 and 100")
	}
	return nil
}

// Evaluate checks if the trace contains any spans with errors.
// - matched is always true because the error rule is global (or service-level, if configured).
// - satisfied is true if an error span is found (always sample).
// - fallbackRatio used if no error span is found (probabilistic sampling).
func (r *ErrorRule) Evaluate(td ptrace.Traces) (bool, bool, float64) {
	rs := td.ResourceSpans()
	for i := 0; i < rs.Len(); i++ {
		scopeSpans := rs.At(i).ScopeSpans()
		for j := 0; j < scopeSpans.Len(); j++ {
			spans := scopeSpans.At(j).Spans()
			for k := 0; k < spans.Len(); k++ {
				if spans.At(k).Status().Code() == ptrace.StatusCodeError {
					return true, true, 0.0 // Immediate sample if an error is found
				}
			}
		}
	}
	return true, false, r.FallbackSamplingRatio // Probabilistic fallback if no errors found
}
