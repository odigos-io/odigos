package sampling

import "go.opentelemetry.io/collector/pdata/ptrace"

// SamplingDecision defines the interface for implementing tail-based sampling rules
// that operate on complete traces. Each implementation decides whether a trace should be
// sampled (kept) based on service identity, span attributes, error signals, latency, or other criteria.
type SamplingDecision interface {
	// Evaluate inspects the given trace and determines how the rule applies.
	//
	// Returns three values:
	//
	//   - matched (bool):
	//       True if the trace is in scope for this rule (e.g., from a specific service
	//       or containing spans with certain attributes). If false, the rule is ignored.
	//
	//   - satisfied (bool):
	//       True if the rule’s condition is fully met and the trace should be sampled
	//       according to the provided probability.
	//
	//   - samplingRatio (float64):
	//       The probability (0.0–100.0) of sampling the trace.
	//       - If satisfied is true, this is the intended sampling ratio.
	//       - If satisfied is false but matched is true, this is a fallback ratio.
	//       - If matched is false, the ratio is ignored.
	//
	// The RuleEngine uses these return values to determine the final sampling decision
	// based on priority, satisfaction, and fallback rules across all active samplers.
	Evaluate(td ptrace.Traces) (matched bool, satisfied bool, samplingRatio float64)

	// Validate checks whether the rule is properly configured.
	// Returns an error if the rule is invalid or incomplete.
	Validate() error
}
