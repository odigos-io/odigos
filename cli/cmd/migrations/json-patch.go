package migrations

import "encoding/json"

// when you want to update a k8s object, you have few options:
// 1. use the k8s client to get the object, update it and then save it back
// 2. use the k8s client patch method to apply just the changes you want
// 3. delete the object and create it from scratch
//
// The preferred way is to use the patch method, because it's clear what you're doing
// and it's more efficient than the other options.
//
// K8s also support multiple patch types. You can read more here:
// https://erosb.github.io/post/json-patch-vs-merge-patch/
// This file includes types and helpers for the JSON-patch payload:
// https://datatracker.ietf.org/doc/html/rfc6902

type jsonPatchOperation struct {
	Op    string `json:"op"`              // can be "add", "remove", "replace", "move", "copy", "test"
	Path  string `json:"path"`            // required for all operations
	Value string `json:"value,omitempty"` // required for "add", "replace" and "test"
	From  string `json:"from,omitempty"`  // required for "move" and "copy
}

type jsonPatchDocument []jsonPatchOperation

func encodeJsonPatchDocument(patchDocument jsonPatchDocument) []byte {
	data, _ := json.Marshal(patchDocument)
	return data
}
