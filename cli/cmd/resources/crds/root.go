package crds

import v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

func NewCRDs() []v1.CustomResourceDefinition {
	return []v1.CustomResourceDefinition{
		NewCollectorsGroup(),
		NewInstrumentedApp(),
		NewConfiguration(),
		NewDestination(),
	}
}
