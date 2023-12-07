// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

package v1alpha1

import (
	"github.com/keyval-dev/odigos/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SupportedOtelSdk struct {
	SdkType common.OtelSdkType `json:"sdkType"`
	SdkTier common.OtelSdkTier `json:"sdkTier"`
}

// type SupportedOtelSdksPerLanguage map[common.ProgrammingLanguage][]SupportedOtelSdk

// type DefaultOtelSdkPerLanguage map[common.ProgrammingLanguage]SupportedOtelSdk

// OdigosConfigurationSpec defines the desired state of OdigosConfiguration
type OdigosConfigurationSpec struct {
	OdigosVersion     string                                            `json:"odigosVersion"`
	ConfigVersion     int                                               `json:"configVersion"`
	TelemetryEnabled  bool                                              `json:"telemetryEnabled,omitempty"`
	IgnoredNamespaces []string                                          `json:"ignoredNamespaces,omitempty"`
	Psp               bool                                              `json:"psp,omitempty"`
	ImagePrefix       string                                            `json:"imagePrefix,omitempty"`
	OdigletImage      string                                            `json:"odigletImage,omitempty"`
	InstrumentorImage string                                            `json:"instrumentorImage,omitempty"`
	AutoscalerImage   string                                            `json:"autoscalerImage,omitempty"`
	SupportedSDKs     map[common.ProgrammingLanguage][]SupportedOtelSdk `json:"supportedSDKs,omitempty"`
	DefaultSDKs       map[common.ProgrammingLanguage]SupportedOtelSdk   `json:"defaultSDKs,omitempty"`
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
