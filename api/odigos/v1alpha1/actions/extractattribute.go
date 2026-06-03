package actions

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
)

const ActionNameExtractAttribute = "ExtractAttribute"

// +kubebuilder:validation:Enum=resource_path;json;sql
type DataFormat string

const (
	FormatResourcePath  DataFormat = "resource_path"
	FormatJSON DataFormat = "json"
	FormatSQL  DataFormat = "sql"
)

// Extraction describes a single extraction rule applied by the
// odigosextractattribute processor. Either (LookupKey + DataFormat) or
// Regex must be provided; the captured value is written to a new span
// attribute named TargetAttributeName.
//
// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type Extraction struct {
	// TargetAttributeName is the name of the new span attribute the extracted value will be written to.
	// +kubebuilder:validation:Required
	TargetAttributeName string `json:"targetAttributeName"`

	// LookupKey is the literal key to search for inside the scanned span attributes.
	// It is plugged into the pre-set regex pattern selected by DataFormat (e.g. for JSON,
	// the processor will look for `"<lookupKey>": "<value>"` and capture the value).
	// Required when using DataFormat.
	// You can input either LookupKey+DataFormat, or supply your own Regex.
	// +kubebuilder:validation:Optional
	LookupKey string `json:"lookupKey,omitempty"`

	// A pre-set definition which resolves into a regex (e.g. JSON DataFormat would give a regex that searches
	// inside a JSON string), which also applies LookupKey for searching inside it.
	// Required when using LookupKey.
	// You can input either LookupKey+DataFormat, or supply your own Regex.
	// +kubebuilder:validation:Optional
	DataFormat DataFormat `json:"dataFormat,omitempty"`

	// Regex is a custom regular expression with a single capture group whose value is
	// written to the new span attribute named TargetAttributeName.
	// You can input either Regex, or LookupKey+DataFormat.
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
