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
							Description: "InstrumentationConfig is the Schema for the instrumentationconfig API",
							Properties: map[string]apiextensionsv1.JSONSchemaProps{
								"apiVersion": {Type: "string"},
								"kind":       {Type: "string"},
								"metadata":   {Type: "object"},
								"spec": {
									Type:        "object",
									Description: "Config for the OpenTelemetry SDKs that should be applied to a workload. The workload is identified by the owner reference",
									Properties: map[string]apiextensionsv1.JSONSchemaProps{
										"config": {
											Type: "array",
											Items: &apiextensionsv1.JSONSchemaPropsOrArray{
												Schema: &apiextensionsv1.JSONSchemaProps{
													Type: "object",
													Properties: map[string]apiextensionsv1.JSONSchemaProps{
														"instrumentationLibraries": {
															Type: "array",
															Items: &apiextensionsv1.JSONSchemaPropsOrArray{
																Schema: &apiextensionsv1.JSONSchemaProps{
																	Type: "object",
																	Properties: map[string]apiextensionsv1.JSONSchemaProps{
																		"instrumentationLibraryName": {Type: "string"},
																		"language":                   {Type: "string"},
																	},
																},
															},
														},
														"optionKey":          {Type: "string"},
														"optionValueBoolean": {Type: "boolean"},
													},
												},
											},
										},
									},
									Required: []string{"config"},
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
