package sampling

import "go.opentelemetry.io/collector/pdata/ptrace"

// SamplingDecision defines the interface for implementing tail-based sampling rules
// that operate on entire traces. Each implementation can decide whether to keep
// or drop a trace based on its spans, attributes, timing, or other conditions.
type SamplingDecision interface {
	// Evaluate inspects the given trace and determines whether the rule matches it.
	//
	// It returns three values:
	//
	// - matched (bool):
	//     Indicates whether the trace matches the *scope or filter* of the rule.
	//     For example, a rule might only apply to traces from a specific service or with a specific span attribute.
	//     If false, the rule is skipped and does not affect the sampling decision.
	//
	// - satisfied (bool):
	//     Indicates whether the rule's *condition* was met.
	//     If true, the trace should be sampled (kept).
	//     If false, the rule suggests the trace may be dropped — but it might still be kept based on fallback sampling.
	//
	// - fallbackRatio (float64):
	//     Specifies the probability (in percent, 0.0 to 100.0) to sample the trace
	//     *if the rule matched but the condition was not satisfied*.
	//
	// The fallback ratio is ignored if the condition was satisfied or if the trace did not match the rule at all.
	Evaluate(td ptrace.Traces) (filterMatch bool, conditionMatch bool, fallbackRatio float64)

	// Validate ensures the rule is correctly configured (e.g. required fields are set).
	// It returns an error if the rule is invalid and should not be used.
	Validate() error
}
