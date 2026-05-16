package actions

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
)

const ActionNameExtractAttribute = "ExtractAttribute"

// +kubebuilder:validation:Enum=url;json
type DataFormat string

const (
	FormatURL  DataFormat = "url"
	FormatJSON DataFormat = "json"
)

// Extraction describes a single extraction rule applied by the
// odigosextractattribute processor. Either (Source + DataFormat) or
// Regex must be provided; the captured value is written to Target.
//
// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type Extraction struct {
	// Target is the attribute key the extracted value will be written to on the span.
	// +kubebuilder:validation:Required
	Target string `json:"target"`

	// The string key that is searched within the pre-set regex patterns resolved by DataFormat.
	// Required when using DataFormat.
	// You can input either Source+DataFormat, or supply your own Regex.
	// +kubebuilder:validation:Optional
	Source string `json:"source,omitempty"`

	// A pre-set definition which resolves into a regex (e.g. JSON DataFormat would give a regex that searches
	// inside a JSON string), which also applies Source for searching inside it.
	// Required when using Source.
	// You can input either Source+DataFormat, or supply your own Regex.
	// +kubebuilder:validation:Optional
	DataFormat DataFormat `json:"dataFormat,omitempty"`

	// Regex is a custom regular expression with a single capture group whose value is
	// written to Target.
	// You can input either Regex, or Source+DataFormat.
	// +kubebuilder:validation:Optional
	Regex string `json:"regex,omitempty"`
}

// ExtractAttributeConfig is the action config for the odigosextractattribute processor.
//
// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type ExtractAttributeConfig struct {
	// Extractions is the list of extraction rules to apply, in order.
	// +kubebuilder:validation:MinItems=1
	Extractions []Extraction `json:"extractions"`
}

func (ExtractAttributeConfig) ProcessorType() string {
	return "odigosextractattribute"
}

// OrderHint is 2 so extraction runs after K8sAttributes (0) and before downstream
// transforms that may consume the extracted attributes.
func (ExtractAttributeConfig) OrderHint() int {
	return 2
}

func (ExtractAttributeConfig) CollectorRoles() []k8sconsts.CollectorRole {
	return []k8sconsts.CollectorRole{
		k8sconsts.CollectorsRoleClusterGateway,
	}
}
