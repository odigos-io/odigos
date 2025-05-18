package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var SchemeGroupVersion = schema.GroupVersion{Group: "odigos.io", Version: "v1alpha1"}

func addKnownTypes(s *runtime.Scheme) error {
	s.AddKnownTypes(SchemeGroupVersion,
		&Source{}, &SourceList{},
		&Processor{}, &ProcessorList{},
		&InstrumentedApplication{}, &InstrumentedApplicationList{},
		&InstrumentationRule{}, &InstrumentationRuleList{},
		&InstrumentationInstance{}, &InstrumentationInstanceList{},
		&InstrumentationConfig{}, &InstrumentationConfigList{},
		&Destination{}, &DestinationList{},
		&CollectorsGroup{}, &CollectorsGroupList{},
		&Action{}, &ActionList{},
	)
	metav1.AddToGroupVersion(s, SchemeGroupVersion)
	return nil
}

var SchemeBuilder = runtime.NewSchemeBuilder(
	addKnownTypes,
	RegisterConversions,
)

// alias for conversion-gen’s init hook
var localSchemeBuilder = SchemeBuilder

// AddToScheme applies all of the above into a Scheme
var AddToScheme = SchemeBuilder.AddToScheme
