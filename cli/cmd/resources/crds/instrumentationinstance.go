package crds

import (
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewInstrumentationInstance() *v1.CustomResourceDefinition {
    return &v1.CustomResourceDefinition{
        TypeMeta: metav1.TypeMeta{
            Kind:       "CustomResourceDefinition",
            APIVersion: "apiextensions.k8s.io/v1",
        },
        ObjectMeta: metav1.ObjectMeta{
            Name: "instrumentationinstances.odigos.io",
        },
        Spec: v1.CustomResourceDefinitionSpec{
            Group: "odigos.io",
            Names: v1.CustomResourceDefinitionNames{
                Plural:     "instrumentationinstances",
                Singular:   "instrumentationinstance",
                Kind:       "InstrumentationInstance",
                ListKind:   "InstrumentationInstanceList",
            },
            Scope: v1.NamespaceScoped,
            Versions: []v1.CustomResourceDefinitionVersion{
                {
                    Name:    "v1alpha1",
                    Served:  true,
                    Storage: true,
                    Schema: &v1.CustomResourceValidation{
                        OpenAPIV3Schema: &v1.JSONSchemaProps{
                            Description: "InstrumentationInstance is the Schema for the InstrumentationInstances API",
                            Type:        "object",
                            Properties: map[string]v1.JSONSchemaProps{
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
                                    Type: "object",
                                },
                                "status": {
                                    Description: "InstrumentationInstanceStatus defines the observed state of InstrumentationInstance. If the instrumentation is not active, this CR should be deleted.",
                                    Type:        "object",
                                    Required: []string{
                                        "lastStatusTime",
                                        "startTime",
                                    },
                                    Properties: map[string]v1.JSONSchemaProps{
                                        "components": {
                                            Type: "array",
                                            Items: &v1.JSONSchemaPropsOrArray{
                                                Schema: &v1.JSONSchemaProps{
                                                    Type: "object",
                                                    Required: []string{
                                                        "lastStatusTime",
                                                        "name",
                                                        "type",
                                                    },
                                                    Properties: map[string]v1.JSONSchemaProps{
                                                        "healthy": {
                                                            Type: "boolean",
                                                        },
                                                        "identifyingAttributes": {
                                                            Type: "array",
                                                            Items: &v1.JSONSchemaPropsOrArray{
                                                                Schema: &v1.JSONSchemaProps{
                                                                    Type: "object",
                                                                    Required: []string{
                                                                        "key",
                                                                        "value",
                                                                    },
                                                                    Properties: map[string]v1.JSONSchemaProps{
                                                                        "key": {
                                                                            Type: "string",
                                                                            MinLength: 1,
                                                                        },
                                                                        "value": {
                                                                            Type: "string",
                                                                        },
                                                                    },
                                                                },
                                                            },
                                                        },
                                                        "lastStatusTime": {
                                                            Type:   "string",
                                                            Format: "date-time",
                                                        },
                                                        "message": {
                                                            Type:        "string",
                                                            MaxLength:   32768,
                                                            Description: "Message is a human readable message indicating details about the component health. Can be omitted if healthy is true.",
                                                        },
                                                        "name": {
                                                            Type:      "string",
                                                            MinLength: 1,
                                                            Description: "For example (\"net/http\", \"@opentelemetry/instrumentation-redis\")",
                                                        },
                                                        "nonIdentifyingAttributes": {
                                                            Type: "array",
                                                            Items: &v1.JSONSchemaPropsOrArray{
                                                                Schema: &v1.JSONSchemaProps{
                                                                    Type: "object",
                                                                    Required: []string{
                                                                        "key",
                                                                        "value",
                                                                    },
                                                                    Properties: map[string]v1.JSONSchemaProps{
                                                                        "key": {
                                                                            Type: "string",
                                                                            MinLength: 1,
                                                                        },
                                                                        "value": {
                                                                            Type: "string",
                                                                        },
                                                                    },
                                                                },
                                                            },
                                                        },
                                                        "reason": {
                                                            Type:        "string",
                                                            Description: "Reason contains a programmatic identifier indicating the reason for the SDK status. Producers of specific condition types may define expected values and meanings for this field, and whether the values are considered a guaranteed API.",
                                                        },
                                                        "type": {
                                                            Type: "string",
                                                            Enum: []v1.JSON{
                                                                {Raw: []byte(`"instrumentation"`)},
                                                                {Raw: []byte(`"sampler"`)},
                                                                {Raw: []byte(`"exporter"`)},
                                                            },
                                                        },
                                                    },
                                                },
                                            },
                                        },
                                        "healthy": {
                                            Type: "boolean",
                                        },
                                        "lastStatusTime": {
                                            Type:   "string",
                                            Format: "date-time",
                                        },
                                        "message": {
                                            Type:        "string",
                                            MaxLength:   32768,
                                            Description: "Message is a human readable message indicating details about the SDK general health. Can be omitted if healthy is true.",
                                        },
                                        "nonIdentifyingAttributes": {
                                            Type: "array",
                                            Items: &v1.JSONSchemaPropsOrArray{
                                                Schema: &v1.JSONSchemaProps{
                                                    Type: "object",
                                                    Required: []string{
                                                        "key",
                                                        "value",
                                                    },
                                                    Properties: map[string]v1.JSONSchemaProps{
                                                        "key": {
                                                            Type: "string",
                                                            MinLength: 1,
                                                        },
                                                        "value": {
                                                            Type: "string",
                                                        },
                                                    },
                                                },
                                            },
                                        },
                                        "reason": {
                                            Type:        "string",
                                            Description: "Reason contains a programmatic identifier indicating the reason for the component status. Producers of specific condition types may define expected values and meanings for this field, and whether the values are considered a guaranteed API.",
                                        },
                                        "startTime": {
                                            Type:   "string",
                                            Format: "date-time",
                                        },
                                    },
                                },
                            },
                        },
                    },
                    Subresources: &v1.CustomResourceSubresources{
                        Status: &v1.CustomResourceSubresourceStatus{},
                    },
                },
            },
        },
    }
}
