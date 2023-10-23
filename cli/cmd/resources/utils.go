package resources

import (
	"encoding/json"

	"github.com/keyval-dev/odigos/cli/pkg/containers"
)

type jsonPatchOperation struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

type jsonPatchDocument []jsonPatchOperation

// changes the image tag in the template spec for the first container in a deployment or daemonset
func patchTemplateSpecImageTag(name string, newVersion string, containerName string) []byte {
	newImage := containers.GetImageName(name, newVersion)
	patchDocument := jsonPatchDocument{
		{
			Op:    "test",
			Path:  "/spec/template/spec/containers/0/name",
			Value: containerName,
		},
		{
			Op:    "replace",
			Path:  "/spec/template/spec/containers/0/image",
			Value: newImage,
		},
	}

	jsonBytes, _ := json.Marshal(patchDocument)
	return jsonBytes
}
