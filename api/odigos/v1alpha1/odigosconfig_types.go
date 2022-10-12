// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

//+kubebuilder:validation:Enum=OPT_IN;OPT_OUT
type InstrumentationMode string

const (
	OptInInstrumentationMode  InstrumentationMode = "OPT_IN"
	OptOutInstrumentationMode InstrumentationMode = "OPT_OUT"
)

// OdigosConfigurationSpec defines the desired state of OdigosConfiguration
type OdigosConfigurationSpec struct {
	InstrumentationMode InstrumentationMode `json:"instrumentationMode"`
}

//+genclient
//+kubebuilder:object:root=true

// OdigosConfiguration is the Schema for the odigos configuration
type OdigosConfiguration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec OdigosConfigurationSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// OdigosConfigurationList contains a list of OdigosConfiguration
type OdigosConfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OdigosConfiguration `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OdigosConfiguration{}, &OdigosConfigurationList{})
}
