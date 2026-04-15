package odigosattributes

const (
	// The sampling category that matched the trace (e.g. "noise", "highly relevant", "cost reduction"). Type: string.
	SamplingCategory = "odigos.sampling.category"
	// The unique identifier of the deciding sampling rule for the trace. Type: string.
	SamplingTraceDecidingRuleId = "odigos.sampling.trace.deciding_rule.id"
	// The human-readable name of the deciding sampling rule. Type: string.
	// It the rule has no name, this attribute will not be set.
	SamplingTraceDecidingRuleName = "odigos.sampling.trace.deciding_rule.name"
	// The keep percentage configured on the deciding sampling rule. Type: double.
	SamplingTraceDecidingRuleKeepPercentage = "odigos.sampling.trace.deciding_rule.keep_percentage"
	// Whether the sampling decision was made in dry-run mode. Type: bool.
	// When dry run is disabled, this attribute will not be set.
	SamplingDryRun = "odigos.sampling.dry_run"
	// Whether the trace would be kept (true) or dropped (false) by the sampling decision. Only set in dry-run mode. Type: bool.
	// Set only when dry run is enabled, to indicate the trace would have been kept or dropped if dry run was disabled.
	SamplingTraceKept = "odigos.sampling.trace.kept"
)
