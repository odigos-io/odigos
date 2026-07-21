package actions

// +kubebuilder:validation:Enum=CREDIT_CARD;EMAIL;JWT;UUID
type PiiCategory string

const (
	CreditCardMasking PiiCategory = "CREDIT_CARD"
	EmailMasking      PiiCategory = "EMAIL"
	JwtMasking        PiiCategory = "JWT"
	UuidMasking       PiiCategory = "UUID"
)

// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type PiiMaskingConfig struct {
	PiiCategories []PiiCategory `json:"piiCategories" mapstructure:"pii_categories"`
}
