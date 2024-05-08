package actions

import (
	"github.com/odigos-io/odigos/cli/cmd/resources/crds/common"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewRenameAttributeCRD() *apiextensionsv1.CustomResourceDefinition {
	return &apiextensionsv1.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CustomResourceDefinition",
			APIVersion: "apiextensions.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "renameattributes.actions.odigos.io",
		},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: "actions.odigos.io",
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Kind:       "RenameAttribute",
				ListKind:   "RenameAttributeList",
				Plural:     "renameattributes",
				Singular:   "renameattribute",
			},
			Scope: apiextensionsv1.NamespaceScoped,
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
				{
					Name:    "v1alpha1",
					Served:  true,
					Storage: true,
					Schema: &apiextensionsv1.CustomResourceValidation{
						OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
							Description: "RenameAttribute is the Schema for the renameattribute odigos action API",
							Type:        "object",
							Properties: map[string]apiextensionsv1.JSONSchemaProps{
								"apiVersion": {
									Description: "APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources",
									Type:        "string",
								},
								"kind": {
									Description: "Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds",
									Type:        "string",
								},
								"metadata": {
									Type: "object",
								},
								"spec": {
									Description: "RenameAttributeSpec defines the desired state of RenameAttribute action",
									Type:        "object",
									Required:    []string{"renames", "signals"},
									Properties: map[string]apiextensionsv1.JSONSchemaProps{
										"actionName": {
											Type: "string",
										},
										"renames": {
											Description: "Renames define attributes to rename and what to rename them to",
											Type:        "object",
											AdditionalProperties: &apiextensionsv1.JSONSchemaPropsOrBool{
												Allows: true,
												Schema: &apiextensionsv1.JSONSchemaProps{
													Type: "string",
												},
											},
										},
										"disabled": {
											Type: "boolean",
										},
										"notes": {
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
									},
								},
								"status": {
									Description: "RenameAttributeStatus defines the observed state of RenameAttribute action",
									Type:        "object",
									Properties: map[string]apiextensionsv1.JSONSchemaProps{
										"conditions": common.Conditions,
									},
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
