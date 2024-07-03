package sampling

import (
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type ErrorRule struct {
	FallbackSamplingRatio float64 `mapstructure:"fallback_sampling_ratio"`
}

func (tlr *ErrorRule) Validate() error {
	return nil
}

func (tlr *ErrorRule) KeepTraceDecision(td ptrace.Traces) (conditionMatch bool) {

	resources := td.ResourceSpans()

	// Iterate over resources
	for r := 0; r < resources.Len(); r++ {
		scoreSpan := resources.At(r).ScopeSpans()

		// Iterate over scopes
		for j := 0; j < scoreSpan.Len(); j++ {
			ils := scoreSpan.At(j)

			// iterate over spans
			for k := 0; k < ils.Spans().Len(); k++ {
				span := ils.Spans().At(k)

				statusCode := span.Status().Code().String()
				if statusCode == ptrace.StatusCodeError.String() {
					return true
				}
			}
		}
	}
	return false
}
