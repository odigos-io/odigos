package category

type RuleEvaluationResult struct {
	RuleId         string
	RuleName       string
	RulePercentage float64
	Matched        bool

	// number of spans on which we evaluated this rule
	SpanCheckedCount int

	// number of spans on which we matched this rule
	SpanMatchedCount int

	// number of traces which matched this rule.
	TraceMatchedCount int
}

type CategoryEvaluationResult map[string]*RuleEvaluationResult
