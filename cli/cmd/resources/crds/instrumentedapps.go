package crds

import (
	"github.com/keyval-dev/odigos/cli/pkg/labels"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewInstrumentedApp() apiextensionsv1.CustomResourceDefinition {
	return apiextensionsv1.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CustomResourceDefinition",
			APIVersion: "apiextensions.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   "instrumentedapplications.odigos.io",
			Labels: labels.OdigosSystem,
		},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: "odigos.io",
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Plural:   "instrumentedapplications",
				Singular: "instrumentedapplication",
				Kind:     "InstrumentedApplication",
				ListKind: "InstrumentedApplicationList",
			},
			Scope: apiextensionsv1.NamespaceScoped,
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
				{
					Name:    "v1alpha1",
					Served:  true,
					Storage: true,
					Schema: &apiextensionsv1.CustomResourceValidation{
						OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
							Description: "InstrumentedApplication is the Schema for the instrumentedapplications API",
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
									Description: "InstrumentedApplicationSpec defines the desired state of InstrumentedApplication",
									Type:        "object",
									Properties: map[string]apiextensionsv1.JSONSchemaProps{
										"languages": {
											Type: "array",
											Items: &apiextensionsv1.JSONSchemaPropsOrArray{
												Schema: &apiextensionsv1.JSONSchemaProps{
													Type: "object",
													Required: []string{
														"containerName",
														"language",
													},
													Properties: map[string]apiextensionsv1.JSONSchemaProps{
														"containerName": {
															Type: "string",
														},
														"language": {
															Type: "string",
															Enum: []apiextensionsv1.JSON{
																{
																	Raw: []byte(`"java"`),
																},
																{
																	Raw: []byte(`"python"`),
																},
																{
																	Raw: []byte(`"go"`),
																},
																{
																	Raw: []byte(`"dotnet"`),
																},
																{
																	Raw: []byte(`"javascript"`),
																},
															},
														},
														"processName": {
															Type: "string",
														},
													},
												},
											},
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
