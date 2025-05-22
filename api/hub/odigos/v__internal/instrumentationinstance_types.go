package v__internal

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type InstrumentationInstanceSpec struct {
	ContainerName string `json:"containerName"`
}

type InstrumentationLibraryType string

type Attribute struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type InstrumentationLibraryStatus struct {
	Name                     string                     `json:"name"`
	Type                     InstrumentationLibraryType `json:"type"`
	IdentifyingAttributes    []Attribute                `json:"identifyingAttributes,omitempty"`
	NonIdentifyingAttributes []Attribute                `json:"nonIdentifyingAttributes,omitempty"`
	Healthy                  *bool                      `json:"healthy,omitempty"`
	Message                  string                     `json:"message,omitempty"`
	Reason                   string                     `json:"reason,omitempty"`
	LastStatusTime           metav1.Time                `json:"lastStatusTime"`
}

type InstrumentationInstanceStatus struct {
	IdentifyingAttributes    []Attribute                    `json:"identifyingAttributes,omitempty"`
	NonIdentifyingAttributes []Attribute                    `json:"nonIdentifyingAttributes,omitempty"`
	Healthy                  *bool                          `json:"healthy,omitempty"`
	Message                  string                         `json:"message,omitempty"`
	Reason                   string                         `json:"reason,omitempty"`
	LastStatusTime           metav1.Time                    `json:"lastStatusTime"`
	Components               []InstrumentationLibraryStatus `json:"components,omitempty"`
}

// +kubebuilder:skipversion
// +kubebuilder:storageversion
// +kubebuilder:object:root=true
type InstrumentationInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              InstrumentationInstanceSpec   `json:"spec,omitempty"`
	Status            InstrumentationInstanceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type InstrumentationInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InstrumentationInstance `json:"items"`
}

func (*InstrumentationInstance) Hub() {}
