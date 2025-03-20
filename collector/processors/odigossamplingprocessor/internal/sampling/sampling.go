package sampling

import "go.opentelemetry.io/collector/pdata/ptrace"

type SamplingDecision interface {
	Evaluate(ptrace.Traces) (matched bool, satisfied bool, fallbackRatio float64)
	Validate() error
}
