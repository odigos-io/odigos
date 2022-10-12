package crds

import (
	"github.com/keyval-dev/odigos/cli/pkg/labels"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewConfiguration() apiextensionsv1.CustomResourceDefinition {
	return apiextensionsv1.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CustomResourceDefinition",
			APIVersion: "apiextensions.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   "odigosconfigurations.odigos.io",
			Labels: labels.OdigosSystem,
		},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: "odigos.io",
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Plural:   "odigosconfigurations",
				Singular: "odigosconfiguration",
				Kind:     "OdigosConfiguration",
				ListKind: "OdigosConfigurationList",
			},
			Scope: apiextensionsv1.NamespaceScoped,
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
				{
					Name:    "v1alpha1",
					Served:  true,
					Storage: true,
					Schema: &apiextensionsv1.CustomResourceValidation{
						OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
							Description: "OdigosConfiguration is the Schema for the odigos configuration",
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
									Description: "OdigosConfigurationSpec defines the desired state of OdigosConfiguration",
									Type:        "object",
									Required: []string{
										"instrumentationMode",
									},
									Properties: map[string]apiextensionsv1.JSONSchemaProps{
										"instrumentationMode": {
											Type: "string",
											Enum: []apiextensionsv1.JSON{
												{Raw: []byte(`"OPT_IN"`)},
												{Raw: []byte(`"OPT_OUT"`)},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
