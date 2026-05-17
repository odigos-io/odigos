package actions

// +kubebuilder:object:generate=true
type UrlTemplatizationConfig struct {
	// Template rules to apply to URLs
	TemplatizationRules []string `json:"templatizationRules,omitempty"`
}
