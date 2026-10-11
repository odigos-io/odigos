package actions

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

	// LookupKey is the field or path segment whose value should be extracted
	// (e.g. a JSON key, SQL column name, or URL path segment).
	// Required when using DataFormat.
	// You can input either LookupKey+DataFormat, or supply your own Regex.
	// +kubebuilder:validation:Optional
	LookupKey string `json:"lookupKey,omitempty"`

	// DataFormat is the format of the data to search in (json, sql, or resource_path).
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
