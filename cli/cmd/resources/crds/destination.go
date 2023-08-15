package crds

import (
	"github.com/keyval-dev/odigos/cli/pkg/labels"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewDestination() apiextensionsv1.CustomResourceDefinition {
	return apiextensionsv1.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CustomResourceDefinition",
			APIVersion: "apiextensions.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   "destinations.odigos.io",
			Labels: labels.OdigosSystem,
		},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: "odigos.io",
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Plural:   "destinations",
				Singular: "destination",
				Kind:     "Destination",
				ListKind: "DestinationList",
			},
			Scope: apiextensionsv1.NamespaceScoped,
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
				{
					Name:    "v1alpha1",
					Served:  true,
					Storage: true,
					Schema: &apiextensionsv1.CustomResourceValidation{
						OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
							Description: "Destination is the Schema for the destinations API",
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
									Description: "DestinationSpec defines the desired state of Destination",
									Type:        "object",
									Required: []string{
										"data",
										"destinationName",
										"signals",
										"type",
									},
									Properties: map[string]apiextensionsv1.JSONSchemaProps{
										"data": {
											Type: "object",
											AdditionalProperties: &apiextensionsv1.JSONSchemaPropsOrBool{
												Allows: true,
												Schema: &apiextensionsv1.JSONSchemaProps{
													Type: "string",
												},
											},
										},
										"destinationName": {
											Type: "string",
										},
										"secretRef": {
											Description: "LocalObjectReference contains enough information to let you locate the referenced object inside the same namespace.",
											Type:        "object",
											Properties: map[string]apiextensionsv1.JSONSchemaProps{
												"name": {
													Type: "string",
												},
											},
										},
										"signals": {
											Type: "array",
											Items: &apiextensionsv1.JSONSchemaPropsOrArray{
												Schema: &apiextensionsv1.JSONSchemaProps{
													Type: "string",
													Enum: []apiextensionsv1.JSON{
														{Raw: []byte(`"LOGS"`)},
														{Raw: []byte(`"TRACES"`)},
														{Raw: []byte(`"METRICS"`)},
													},
												},
											},
										},
										"type": {
											Type: "string",
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
