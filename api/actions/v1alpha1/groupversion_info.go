package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var SchemeGroupVersion = schema.GroupVersion{Group: "actions.odigos.io", Version: "v1alpha1"}

func addKnownTypes(s *runtime.Scheme) error {
	s.AddKnownTypes(SchemeGroupVersion,
		&AddClusterInfo{}, &AddClusterInfoList{},
		&SpanAttributeSampler{}, &SpanAttributeSamplerList{},
		&ServiceNameSampler{}, &ServiceNameSamplerList{},
		&RenameAttribute{}, &RenameAttributeList{},
		&ProbabilisticSampler{}, &ProbabilisticSamplerList{},
		&PiiMasking{}, &PiiMaskingList{},
		&LatencySampler{}, &LatencySamplerList{},
		&K8sAttributesResolver{}, &K8sAttributesResolverList{},
		&ErrorSampler{}, &ErrorSamplerList{},
		&DeleteAttribute{}, &DeleteAttributeList{},
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
