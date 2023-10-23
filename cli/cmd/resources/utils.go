package resources

import (
	"encoding/json"

	"github.com/keyval-dev/odigos/cli/pkg/containers"
)

type jsonPatchOperation struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value,omitempty"`
	From  string `json:"from,omitempty"`
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

// the app label makes sense on pods to group them into a replicaset,
// but not on deployments or daemonsets where it doesn't mean anything.
// this patch removes the "app" label from those resources
func patchRemoveAppLabel(expectedValue string) ([]byte, []byte) {
	patchDocument := jsonPatchDocument{
		{
			Op:    "test",
			Path:  "/metadata/labels/app",
			Value: expectedValue,
		},
		{
			Op:   "remove",
			Path: "/metadata/labels/app",
		},
	}

	unpatchDocument := jsonPatchDocument{
		{
			Op:    "add",
			Path:  "/metadata/labels/app",
			Value: expectedValue,
		},
	}

	patchBytes, _ := json.Marshal(patchDocument)
	unpatchBytes, _ := json.Marshal(unpatchDocument)
	return patchBytes, unpatchBytes
}
