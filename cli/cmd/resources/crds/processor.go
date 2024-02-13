package crds

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewProcessor() *apiextensionsv1.CustomResourceDefinition {
	xPreserveUnknownFields := true
	return &apiextensionsv1.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CustomResourceDefinition",
			APIVersion: "apiextensions.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "processors.odigos.io",
		},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: "odigos.io",
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Plural:   "processors",
				Singular: "processor",
				Kind:     "Processor",
				ListKind: "ProcessorList",
			},
			Scope: apiextensionsv1.NamespaceScoped,
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
				{
					Name:    "v1alpha1",
					Served:  true,
					Storage: true,
					Schema: &apiextensionsv1.CustomResourceValidation{
						OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
							Description: "Processor is the Schema for an Opentelemetry Collector Processor that is added to Odigos pipeline",
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
									Description: "ProcessorSpec defines the OpenTelemetry Collector processor in odigos telemetry pipeline",
									Type:        "object",
									Required: []string{
										"collectorRoles",
										"processorConfig",
										"signals",
										"type",
									},
									Properties: map[string]apiextensionsv1.JSONSchemaProps{
										"collectorRoles": {
											Type: "array",
											Items: &apiextensionsv1.JSONSchemaPropsOrArray{
												Schema: &apiextensionsv1.JSONSchemaProps{
													Type: "string",
													Enum: []apiextensionsv1.JSON{
														{Raw: []byte(`"CLUSTER_GATEWAY"`)},
														{Raw: []byte(`"NODE_COLLECTOR"`)},
													},
												},
											},
										},
										"processorConfig": {
											XPreserveUnknownFields: &xPreserveUnknownFields,
											Type:                   "object",
										},
										"disabled": {
											Type: "boolean",
										},
										"notes": {
											Type: "string",
										},
										"orderHint": {
											Type: "integer",
										},
										"processorName": {
											Type: "string",
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
