package odigosattributes

const (
	// The sampling category that matched the trace (e.g. "noise", "highly relevant", "cost reduction"). Type: string.
	SamplingCategory = "odigos.sampling.category"

	// The unique identifier of the deciding sampling rule for the trace. Type: string.
	// Multiple spans can match different rules, and this one is the most restrictive one.
	SamplingTraceDecidingRuleId = "odigos.sampling.trace.deciding_rule.id"
	// The human-readable name of the deciding sampling rule. Type: string.
	// It the rule has no name, this attribute will not be set.
	SamplingTraceDecidingRuleName = "odigos.sampling.trace.deciding_rule.name"
	// The keep percentage configured on the deciding sampling rule. Type: double.
	SamplingTraceDecidingRuleKeepPercentage = "odigos.sampling.trace.deciding_rule.keep_percentage"

	// The unique identifier of the matching sampling rule for the span. Type: string.
	// In the same trace, different spans can match different rules,
	// and this attribute indicate the most restrictive rule that matched a specific span.
	SamplingSpanMatchingRuleId = "odigos.sampling.span.matching_rule.id"
	// The human-readable name of the matching sampling rule. Type: string.
	// It the rule has no name, this attribute will not be set.
	SamplingSpanMatchingRuleName = "odigos.sampling.span.matching_rule.name"
	// The keep percentage configured on the matching sampling rule. Type: double.
	SamplingSpanMatchingRuleKeepPercentage = "odigos.sampling.span.matching_rule.keep_percentage"

	// Whether the sampling decision was made in dry-run mode. Type: bool.
	// When dry run is disabled, this attribute will not be set.
	SamplingDryRun = "odigos.sampling.dry_run"
	// Whether the trace would be kept (true) or dropped (false) by the sampling decision. Only set in dry-run mode. Type: bool.
	// Set only when dry run is enabled, to indicate the trace would have been kept or dropped if dry run was disabled.
	SamplingTraceKept = "odigos.sampling.trace.kept"
)
