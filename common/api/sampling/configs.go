package sampling

// SpanSamplingAttributesConfiguration controls whether spans are enhanced with sampling attributes
// (e.g. category and decisions). These attributes add context when viewing traces and inspecting costs,
// so you can understand how sampling decisions were made for an individual span and apply changes to fine-tune rules.
// When dry run is enabled, each span includes the sampling decision (kept or dropped) as it would apply once dry run is disabled.
// +kubebuilder:object:generate=true
type SpanSamplingAttributesConfiguration struct {
	// Set to true to disable span sampling attributes.
	// When disabled, Odigos will not set span attributes for sampling decisions
	// (unless explicitly enabled for specific attributes set).
	Disabled *bool `json:"disabled,omitempty"`

	// Set to true to disable recording the sampling category as a span attribute.
	// Attributes: odigos.sampling.category (noise, highly relevant, cost reduction, or empty)
	SamplingCategoryDisabled *bool `json:"samplingCategoryDisabled,omitempty"`

	// Set to true to disable recording the trace-level deciding rule as span attributes.
	// When a trace matches an enabled sampling rule for a category, the most "severe" rule
	// is chosen as the "deciding rule" and recorded as span attributes on all spans in the trace.
	// If multiple rules with the same percentage match, one is chosen arbitrarily.
	// Use these attributes to find rules that are too permissive (high cost) or too strict (dropping important traces).
	// For example, if a rare and important trace is dropped by a rule that is too strict, increase its keep percentage
	// or add a separate rule for that use case. If cost analysis shows expensive traces, find rules that are too permissive
	// and decrease their percentage or remove them.
	// Attributes:
	//   - odigos.sampling.trace.deciding_rule.id
	//   - odigos.sampling.trace.deciding_rule.name
	//   - odigos.sampling.trace.deciding_rule.keep_percentage
	TraceDecidingRuleDisabled *bool `json:"traceDecidingRuleDisabled,omitempty"`

	// Set to true to disable recording the span-level deciding rule as span attributes.
	// The trace-level decision is an aggregation of all spans in the trace; these attributes record
	// which spans contributed to it and how, so you can link decisions back to the specific spans and operations.
	// For example, if a trace is kept due to high duration, the attributes pinpoint the span(s) that drove that decision.
	// Attributes:
	//   - odigos.sampling.span.deciding_rule.id
	//   - odigos.sampling.span.deciding_rule.name
	//   - odigos.sampling.span.deciding_rule.keep_percentage
	SpanDecisionAttributesDisabled *bool `json:"spanDecisionAttributesDisabled,omitempty"`
}

// TailSamplingConfiguration configures tail sampling behavior.
// +kubebuilder:object:generate=true
type TailSamplingConfiguration struct {

	// If set to true, tail sampling will be disabled globally
	// regardless of any other configurations or rules set.
	// Can be used to reduce collectors resource usage, troubleshooting, etc,
	// or when tail-sampling is not needed or desired and should be shut off.
	Disabled *bool `json:"disabled,omitempty"`

	// Time to wait from the first span of a trace until a trace is considered completed.
	// At this time, all spans received for this trace are aggregated and a tail-sampling decision is applied.
	// Introduces this amount of latency in the pipeline and for trace to hit the destination.
	// Also increases memory usage for keeping spans in memory until the wait duration time is reached.
	// Setting it too low might introduce fragmentation of traces - sampling decisions based on incomplete traces,
	// and broken traces due to sampling each trace in few pieces.
	TraceAggregationWaitDuration *string `json:"traceAggregationWaitDuration,omitempty"`
}
