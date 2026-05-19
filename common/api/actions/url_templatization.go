package actions

// +kubebuilder:object:generate=true
type UrlTemplatizationConfig struct {
	// Template rules to apply to URLs
	TemplatizationRules []string `json:"templatizationRules,omitempty"`

	// indicate that the workload accepts incoming traffic from the internet.
	// this is used to avoid running url-templatization
	// when an endpoint from this service returns with 404 status code.
	// internet exposed services are commonly being "tested" by malicious actors
	// with irrelevant or garbage requests that can contaminate the url-templatization process
	// leading to high-cardinality of templated routes.
	// +kubebuilder:validation:Optional
	// +optional
	PubliclyAccessible bool `json:"publiclyAccessible,omitempty"`
}
