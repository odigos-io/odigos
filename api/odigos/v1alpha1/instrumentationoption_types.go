package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// InstrumentationOption is the Schema for the instrumentationoptions API
type InstrumentationOption struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InstrumentationOptionSpec   `json:"spec,omitempty"`
	Status InstrumentationOptionStatus `json:"status,omitempty"`
}

// InstrumentationOptionSpec defines the desired state of InstrumentationOption
// Each field in the struct will be converted to an OpenAPI v3 schema
// with the comments used as the description.
type InstrumentationOptionSpec struct {
	// OptionName is the name of the option
	OptionName string `json:"optionName"`

	// OptionValueBoolean is the boolean value of the option
	OptionValueBoolean bool `json:"optionValueBoolean,omitempty"`

	// Workloads is an optional list of k8s ns+kind+name to which this option applies.
	// If not specified, the option applies to all workloads.
	Workloads []Workload `json:"workloads,omitempty"`

	// InstrumentationLibraries is a list of instrumentation libraries
	// to which this option applies.
	InstrumentationLibraries []InstrumentationLibrary `json:"instrumentationLibraries"`

	// Filters define how to apply the instrumentation options
	Filters []InstrumentationOptionFilter `json:"filters,omitempty"`
}

type Workload struct {
	// Namespace is the k8s namespace of the workload
	Namespace string `json:"namespace"`

	// Kind is the k8s kind of the workload, e.g., 'Deployment'
	// +kubebuilder:validation:Enum=Deployment;DaemonSet;StatefulSet
	Kind string `json:"kind"`

	// Name is the name of the k8s object of the workload
	Name string `json:"name"`
}

// InstrumentationLibrary represents a library for instrumentation
type InstrumentationLibrary struct {
	// Language is the programming language of the library
	Language string `json:"language"`

	// InstrumentationLibraryName is the name of the instrumentation library
	InstrumentationLibraryName string `json:"instrumentationLibraryName"`
}

// InstrumentationOptionFilter defines a filter for applying instrumentation options
type InstrumentationOptionFilter struct {
	// Key is the attribute key to filter (e.g., 'http.route', 'url.path')
	Key string `json:"key"`

	// MatchType is the type of match (e.g., 'equals', 'startsWith')
	MatchType string `json:"matchType"`

	// MatchValue is the value to match against
	MatchValue string `json:"matchValue"`
}

// InstrumentationOptionStatus defines the observed state of InstrumentationOption
type InstrumentationOptionStatus struct {
	// Status fields should be added here
}

// +kubebuilder:object:root=true

// InstrumentationOptionList contains a list of InstrumentationOption
type InstrumentationOptionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InstrumentationOption `json:"items"`
}

func init() {
	SchemeBuilder.Register(&InstrumentationOption{}, &InstrumentationOptionList{})
}
