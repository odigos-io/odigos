package crds

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewInstrumentationConfig() *apiextensionsv1.CustomResourceDefinition {
	return &apiextensionsv1.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CustomResourceDefinition",
			APIVersion: "apiextensions.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "instrumentationconfigs.odigos.io",
		},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: "odigos.io",
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Plural:   "instrumentationconfigs",
				Singular: "instrumentationconfig",
				Kind:     "InstrumentationConfig",
				ListKind: "InstrumentationConfigList",
			},
			Scope: apiextensionsv1.NamespaceScoped,
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
				{
					Name:    "v1alpha1",
					Served:  true,
					Storage: true,
					Schema: &apiextensionsv1.CustomResourceValidation{
						OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
							Type:        "object",
							Description: "InstrumentationConfig is the Schema for the instrumentation config API",
							Properties: map[string]apiextensionsv1.JSONSchemaProps{
								"apiVersion": {Type: "string"},
								"kind":       {Type: "string"},
								"metadata":   {Type: "object"},
								"spec": {
									Type:        "object",
									Description: "InstrumentationConfigSpec defines the desired state of InstrumentationConfig",
									Properties: map[string]apiextensionsv1.JSONSchemaProps{
										"name": {
											Type: "string",
										},
										"optionKey": {
											Type: "string",
										},
										"optionValueBoolean": {
											Type: "boolean",
										},
										"workloads": {
											Type: "array",
											Items: &apiextensionsv1.JSONSchemaPropsOrArray{
												Schema: &apiextensionsv1.JSONSchemaProps{
													Type: "object",
													Properties: map[string]apiextensionsv1.JSONSchemaProps{
														"namespace": {Type: "string"},
														"kind":      {Type: "string"},
														"name":      {Type: "string"},
													},
												},
											},
										},
										"instrumentationLibraries": {
											Type: "array",
											Items: &apiextensionsv1.JSONSchemaPropsOrArray{
												Schema: &apiextensionsv1.JSONSchemaProps{
													Type: "object",
													Properties: map[string]apiextensionsv1.JSONSchemaProps{
														"language":                   {Type: "string"},
														"instrumentationLibraryName": {Type: "string"},
													},
												},
											},
										},
										"filters": {
											Type: "array",
											Items: &apiextensionsv1.JSONSchemaPropsOrArray{
												Schema: &apiextensionsv1.JSONSchemaProps{
													Type: "object",
													Properties: map[string]apiextensionsv1.JSONSchemaProps{
														"key":        {Type: "string"},
														"matchType":  {Type: "string"},
														"matchValue": {Type: "string"},
													},
												},
											},
										},
									},
								},
								"status": {Type: "object"},
							},
						},
					},
					Subresources: &apiextensionsv1.CustomResourceSubresources{
						Status: &apiextensionsv1.CustomResourceSubresourceStatus{},
					},
				},
			},
		},
	}
}
