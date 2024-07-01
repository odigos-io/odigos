package sampling

type SamplingConfig struct {
	Rules []Rule `json:"rules"`
}

// Rule representes a rule in odigossampling processor rule
type Rule struct {
	Name     string      `json:"name"`
	RuleType string      `json:"type"`
	Details  RuleDetails `json:"rule_details"`
}
