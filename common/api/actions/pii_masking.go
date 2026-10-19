package actions

// +kubebuilder:validation:Enum=CREDIT_CARD;EMAIL;JWT;UUID
type PiiCategory string

const (
	CreditCardMasking PiiCategory = "CREDIT_CARD"
	EmailMasking      PiiCategory = "EMAIL"
	JwtMasking        PiiCategory = "JWT"
	UuidMasking       PiiCategory = "UUID"
)

// CustomFormatMasking masks values found via a LookupKey inside a DataFormat
// (e.g. JSON key, SQL column, resource path segment).
//
// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type CustomFormatMasking struct {
	// LookupKey is the field or path segment whose value should be masked
	// (e.g. a JSON key, SQL column name, or URL path segment).
	// +kubebuilder:validation:Required
	LookupKey string `json:"lookupKey" yaml:"lookupKey" mapstructure:"lookup_key"`

	// DataFormat is the format of the data to search in (json, sql, or resource_path).
	// +kubebuilder:validation:Required
	DataFormat DataFormat `json:"dataFormat" yaml:"dataFormat" mapstructure:"data_format"`
}

// CustomRegexMasking masks values matched by a user-supplied regular expression.
//
// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type CustomRegexMasking struct {
	// Regex is a custom regular expression with a single capture group whose value is masked.
	// +kubebuilder:validation:Required
	Regex string `json:"regex" yaml:"regex" mapstructure:"regex"`
}

// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type PiiMaskingConfig struct {
	// PiiCategories are predefined PII patterns to mask (e.g. CREDIT_CARD, EMAIL).
	// +kubebuilder:validation:Optional
	PiiCategories []PiiCategory `json:"piiCategories,omitempty" yaml:"piiCategories,omitempty" mapstructure:"pii_categories"`

	// CustomFormatMaskings is the list of format-based masking rules to apply, in order.
	// +kubebuilder:validation:Optional
	//nolint:lll // CustomFormatMaskings line is long due to struct tag requirements
	CustomFormatMaskings []CustomFormatMasking `json:"customFormatMaskings,omitempty" yaml:"customFormatMaskings,omitempty" mapstructure:"custom_format_maskings"`

	// CustomRegexMaskings is the list of regex-based masking rules to apply, in order.
	// +kubebuilder:validation:Optional
	//nolint:lll // CustomRegexMaskings line is long due to struct tag requirements
	CustomRegexMaskings []CustomRegexMasking `json:"customRegexMaskings,omitempty" yaml:"customRegexMaskings,omitempty" mapstructure:"custom_regex_maskings"`
}
