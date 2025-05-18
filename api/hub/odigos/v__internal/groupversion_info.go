package v__internal

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var SchemeGroupVersion = schema.GroupVersion{Group: "odigos.io", Version: "__internal"}

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

var SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)

// AddToScheme applies all of the above into a Scheme
var AddToScheme = SchemeBuilder.AddToScheme
