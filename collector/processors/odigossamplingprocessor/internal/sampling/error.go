package sampling

import (
	"errors"

	"go.opentelemetry.io/collector/pdata/ptrace"
)

type ErrorRule struct {
	// FallbackSamplingRatio determines the percentage of traces to sample
	// if no error spans are present. Valid range: 0–100.
	FallbackSamplingRatio float64 `mapstructure:"fallback_sampling_ratio"`
}

var _ SamplingDecision = (*ErrorRule)(nil)

// Validate ensures the fallback ratio is within acceptable bounds.
func (r *ErrorRule) Validate() error {
	if r.FallbackSamplingRatio < 0 || r.FallbackSamplingRatio > 100 {
		return errors.New("fallback_sampling_ratio must be between 0 and 100")
	}
	return nil
}

// Evaluate scans all spans in the trace and returns:
// - filterMatch: always true (this rule applies globally)
// - conditionMatch: true if any span has error status
// - ratio: 100 if error is found (sample always), fallback ratio otherwise
func (r *ErrorRule) Evaluate(td ptrace.Traces) (bool, bool, float64) {
	rs := td.ResourceSpans()
	for i := 0; i < rs.Len(); i++ {
		scopeSpans := rs.At(i).ScopeSpans()
		for j := 0; j < scopeSpans.Len(); j++ {
			spans := scopeSpans.At(j).Spans()
			for k := 0; k < spans.Len(); k++ {
				if spans.At(k).Status().Code() == ptrace.StatusCodeError {
					return true, true, 100.0 // satisfied; RuleEngine will always sample this
				}
			}
		}
	}
	// No error spans; matched but not satisfied — fallback may apply
	return true, false, r.FallbackSamplingRatio
}
