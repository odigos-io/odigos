package category

// CommonRuleMetrics holds the shared per-rule metrics used by both highly-relevant and cost-reduction categories.
type CommonRuleMetrics struct {
	RuleId         string
	RuleName       string
	RulePercentage float64
	Matched        bool
	// number of spans on which we invoked this rule.
	SpanCheckedCount int
	// number of spans on which we matched this rule.
	SpanMatchingCount int
	// number of traces on which we matched this rule at least once.
	TraceMatchingCount int
}
