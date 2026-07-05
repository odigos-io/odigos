package sampling

// QueryParamMatcher is a matcher for a query parameter in a url.
// +kubebuilder:object:generate=true
type QueryParamMatcher struct {

	// name of the query parameter
	Name string `json:"name"`

	// value of the query parameter, that should be matched exactly
	ValueExact string `json:"valueExact"`
}
