package instrumentationrules

import "github.com/odigos-io/odigos/common"

// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type OtelSdks struct {
	OtelSdkByLanguage map[common.ProgrammingLanguage]common.OtelSdk `json:"otelSdkByLanguage"`
}
