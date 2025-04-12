package actions

// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type CalculateUrlTemplate struct {

	// A list of url-templatization rules that when match a url.path,
	// will replace some of it's path-segments with templated name.
	// A rule looks like this: "/user/{userName}/profile/{profileId}"
	// The rule will match a path like "/user/john/profile/1234"
	// and will templatize it as "/user/{userName}/profile/{profileId}"
	TempltizationRules []string `json:"templtizationRules,omitempty"`
}
