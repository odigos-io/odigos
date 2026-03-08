package category

// CommonRuleMetrics holds the shared per-rule metrics used by both highly-relevant and cost-reduction categories.
type CommonRuleMetrics struct {
	// number of spans on which we invoked this rule.
	SpanCheckedCount int
	// number of spans on which we matched this rule.
	SpanMatchingCount int
	// number of traces on which we checked this rule (1 when the rule was checked for this trace).
	TraceCheckedCount int
	// number of traces on which we matched this rule at least once.
	TraceMatchingCount int
}
