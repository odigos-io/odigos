package sampling

type SamplingConfig struct {
	GlobalRules   []Rule `json:"global_rules,omitempty"`
	ServiceRules  []Rule `json:"service_rules,omitempty"`
	EndpointRules []Rule `json:"endpoint_rules,omitempty"`
}

// Rule representes a rule in odigossampling processor rule
type Rule struct {
	Name     string      `json:"name"`
	RuleType RuleType    `json:"type"`
	Details  RuleDetails `json:"rule_details"`
}

type RuleType string

const (
	LatencyRule       RuleType = "http_latency"
	ErrorRule         RuleType = "error"
	SpanAttributeRule RuleType = "span_attribute"
	ServiceNameRule   RuleType = "service_name"
)
