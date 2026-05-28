package actions

// Used to mark sources to avoid default templatization on error.
// Publicly accessible services are commonly being "tested" by malicious actors
// with irrelevant or garbage requests that can contaminate the url-templatization process
// leading to high-cardinality of templated routes.
// when no custom templatization rule matched, the request status code is checked, and skipped based on the config.
// either skipForNonSuccessCodes or statusCodes must be set for it to take effect.
// if both are set, skipForNonSuccessCodes takes precedence.
//
// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type DefaultTemplatizationSkipPolicyConfig struct {

	// If set to true, default templatization will be skipped for any non-success HTTP status code (i.e., any code not in the 2xx range).
	// When this is true, the StatusCodes list is ignored.
	SkipForNonSuccessCodes bool `json:"skipForNonSuccessCodes,omitempty"`

	// the http status codes for which the default templatization will be skipped.
	// for example: [404, 401].
	// advanced users can use to cherry pick specific codes for tailoring to specific use cases.
	SkipHttpStatusCodes []int `json:"skipHttpStatusCodes,omitempty"`
}

// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type DefaultTemplatizationConfig struct {

	// if set to true, the default templatization will be disabled for the services in the scope.
	// in case of conflict (if other action set this to false), the default templatization will be disabled.
	Disabled bool `json:"enabled,omitempty"`

	// config if default templatization should be skipped on error.
	// use it when sources scope describes a service that is publicly accessible to the internet
	// to filter garbage requests that can contaminate the url-templatization process.
	SkipPolicy *DefaultTemplatizationSkipPolicyConfig `json:"skipPolicy,omitempty"`
}

// +kubebuilder:object:generate=true
type UrlTemplatizationConfig struct {
	// Template rules to apply to URLs
	TemplatizationRules []string `json:"templatizationRules,omitempty"`

	// configurations for default templatization.
	// default templatization is applied on a single http span if none of the custom templatization rules matched.
	DefaultTemplatization *DefaultTemplatizationConfig `json:"defaultTemplatization,omitempty"`
}
