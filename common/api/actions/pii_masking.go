package actions

// +kubebuilder:validation:Enum=CREDIT_CARD
type PiiCategory string

const (
	CreditCardMasking PiiCategory = "CREDIT_CARD"
)

// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type PiiMaskingConfig struct {
	PiiCategories []PiiCategory `json:"piiCategories"`
}
