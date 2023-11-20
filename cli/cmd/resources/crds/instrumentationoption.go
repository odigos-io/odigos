package crds

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewInstrumentationOption() *apiextensionsv1.CustomResourceDefinition {
	return &apiextensionsv1.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CustomResourceDefinition",
			APIVersion: "apiextensions.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "instrumentationoptions.odigos.io",
		},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: "odigos.io",
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Plural:   "instrumentationoptions",
				Singular: "instrumentationoption",
				Kind:     "InstrumentationOption",
				ListKind: "InstrumentationOptionList",
			},
			Scope: apiextensionsv1.NamespaceScoped,
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
				{
					Name:    "v1alpha1",
					Served:  true,
					Storage: true,
					Schema: &apiextensionsv1.CustomResourceValidation{
						OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
							Description: "InstrumentationOption is the Schema for the instrumentation options API",
							Type:        "object",
							Properties: map[string]apiextensionsv1.JSONSchemaProps{
								"apiVersion": {
									Type: "string",
								},
								"kind": {
									Type: "string",
								},
								"metadata": {
									Type: "object",
								},
								"spec": {
									Description: "InstrumentationOptionSpec defines the desired state of InstrumentationOption",
									Type:        "object",
									Required: []string{
										"optionName",
										"instrumentationLibraries",
									},
									Properties: map[string]apiextensionsv1.JSONSchemaProps{
										"optionName": {
											Type: "string",
										},
										"optionValueBoolean": {
											Type:        "boolean",
											Description: "The value of the option if it is a boolean",
										},
										"serviceNames": {
											Type: "array",
											Items: &apiextensionsv1.JSONSchemaPropsOrArray{
												Schema: &apiextensionsv1.JSONSchemaProps{
													Type: "string",
												},
											},
											Description: "An optional list of service names to which this option applies. If not specified, the option applies to all services.",
										},
										"instrumentationLibraries": {
											Type: "array",
											Items: &apiextensionsv1.JSONSchemaPropsOrArray{
												Schema: &apiextensionsv1.JSONSchemaProps{
													Type: "object",
													Properties: map[string]apiextensionsv1.JSONSchemaProps{
														"language": {
															Type: "string",
														},
														"instrumentationLibraryName": {
															Type: "string",
														},
													},
												},
											},
											Description: "An optional list of instrumentation libraries to which this option applies.",
										},
										"filters": {
											Type: "array",
											Items: &apiextensionsv1.JSONSchemaPropsOrArray{
												Schema: &apiextensionsv1.JSONSchemaProps{
													Type: "object",
													Properties: map[string]apiextensionsv1.JSONSchemaProps{
														"key": {
															Type:        "string",
															Description: "The attribute key to filter",
														},
														"matchType": {
															Type:        "string",
															Description: "Type of match",
															Enum: []apiextensionsv1.JSON{
																{Raw: []byte(`"equals"`)},
																{Raw: []byte(`"startsWith"`)},
																{Raw: []byte(`"greaterThan"`)},
																{Raw: []byte(`"lessThan"`)},
																{Raw: []byte(`"regex"`)},
															},
														},
														"matchValue": {
															Type:        "string",
															Description: "The value to match against",
														},
													},
												},
											},
											Description: "Defines filters for applying instrumentation options",
										},
									},
								},
								"status": {
									Type: "object",
								},
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
