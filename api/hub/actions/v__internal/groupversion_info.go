package v__internal

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var SchemeGroupVersion = schema.GroupVersion{
	Group:   "actions.odigos.io",
	Version: "__internal",
}

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
	// register the metadata (API version) for your types
	metav1.AddToGroupVersion(s, SchemeGroupVersion)
	return nil
}

var SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)

// AddToScheme applies all of the above into a Scheme
var AddToScheme = SchemeBuilder.AddToScheme
