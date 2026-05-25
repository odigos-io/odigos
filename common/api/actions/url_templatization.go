package actions

// +kubebuilder:object:generate=true
type UrlTemplatizationConfig struct {
	// Template rules to apply to URLs
	TemplatizationRules []string `json:"templatizationRules,omitempty"`

	// indicate that the workload can accept garbage requests that can contaminate the url-templatization process
	// leading to high-cardinality of templated routes.
	// when set to true, requests that did not match any custom templatization rule and ended with 404 status code
	// will not be templatized (no route and span name will be GET or the http method)
	AvoidDefaultTemplatizationOnError bool `json:"avoidDefaultTemplatizationOnError,omitempty"`
}
