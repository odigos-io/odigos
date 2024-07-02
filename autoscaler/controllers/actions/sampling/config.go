package sampling

type SamplingConfig struct {
	GlobalRules   []Rule `json:"global_rules,omitempty"`
	ServiceRules  []Rule `json:"service_rules,omitempty"`
	EndpointRules []Rule `json:"endpoint_rules,omitempty"`
}

// Rule representes a rule in odigossampling processor rule
type Rule struct {
	Name     string      `json:"name"`
	RuleType string      `json:"type"`
	Details  RuleDetails `json:"rule_details"`
}
