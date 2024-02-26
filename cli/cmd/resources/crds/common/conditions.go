package common

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

var (
	messageMaxLength          int64   = 32768
	observedGenerationMinimum float64 = 0
	reasonMaxLength           int64   = 1024
	typeMaxLength             int64   = 316
	statusEnumValues                  = []apiextensionsv1.JSON{
		{Raw: []byte(`"True"`)},
		{Raw: []byte(`"False"`)},
		{Raw: []byte(`"Unknown"`)},
	}

	Conditions = apiextensionsv1.JSONSchemaProps{
		Description: "Represents the observations of a addclusterinfos's current state. Known .status.conditions.type are: \"Available\", \"Progressing\"",
		Type:        "array",
		Items: &apiextensionsv1.JSONSchemaPropsOrArray{
			Schema: &apiextensionsv1.JSONSchemaProps{
				Type: "object",
				Properties: map[string]apiextensionsv1.JSONSchemaProps{
					"lastTransitionTime": {
						Description: "lastTransitionTime is the last time the condition transitioned from one status to another. This should be when the underlying condition changed. If that is not known, then using the time when the API field changed is acceptable.",
						Type:        "string",
						Format:      "date-time",
					},
					"message": {
						Description: "message is a human readable message indicating details about the transition. This may be an empty string.",
						Type:        "string",
						MaxLength:   &messageMaxLength,
					},
					"observedGeneration": {
						Description: "observedGeneration represents the .metadata.generation that the condition was set based upon. For instance, if .metadata.generation is currently 12, but the .status.conditions[x].observedGeneration is 9, the condition is out of date with respect to the current state of the instance.",
						Type:        "integer",
						Format:      "int64",
						Minimum:     &observedGenerationMinimum,
					},
					"reason": {
						Description: "reason contains a programmatic identifier indicating the reason for the condition's last transition. Producers of specific condition types may define expected values and meanings for this field, and whether the values are considered a guaranteed API. The value should be a CamelCase string. This field may not be empty.",
						Type:        "string",
						MaxLength:   &reasonMaxLength,
						Pattern:     "^[A-Za-z]([A-Za-z0-9_,:]*[A-Za-z0-9_])?$",
					},
					"status": {
						Description: "status of the condition, one of True, False, Unknown.",
						Type:        "string",
						Enum:        statusEnumValues,
					},
					"type": {
						Description: "type of condition in CamelCase or in foo.example.com/CamelCase.",
						Type:        "string",
						MaxLength:   &typeMaxLength,
						Pattern:     `^([a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*/)?(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])$`,
					},
				},
				Required: []string{"lastTransitionTime", "message", "reason", "status", "type"},
			},
		},
	}
)
