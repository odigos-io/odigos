package instrumentationrules

import "github.com/odigos-io/odigos/common"

// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
// Deprecated: Use OtelDistro instead
type OtelSdks struct {
	OtelSdkByLanguage map[common.ProgrammingLanguage]common.OtelSdk `json:"otelSdkByLanguage"`
}

// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type OtelDistros struct {

	// Set a list of distribution names that take priority over the default distributions.
	// if a language is not in this list, the default distribution will be used.
	// it multiple distributions are specified for the same language, in one or many rules,
	// the behavior is undefined.
	OtelDistroNames []string `json:"otelDistroNames"`
}
