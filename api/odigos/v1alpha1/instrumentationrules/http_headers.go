package instrumentationrules

// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type HttpHeadersCollection struct {

	// Limit payload collection to specific header keys.
	HeaderKeys []string `json:"headerKeys,omitempty"`
}
