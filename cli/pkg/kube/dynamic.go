package kube

import (
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// the k8s manifest yamls use `Kind` which can be multiple words,
// e.g. `Deployment`, `Deployments`, etc.
// the k8s api, however, requires something calls `Resource` which is
// always plural and lowercase, e.g. `deployments`
// Currently, I will translate the `Kind` to `Resource` by appending an `s`
// but this is not guaranteed to work for all resources.
func objectKindToResourceName(kind string) string {
	return strings.ToLower(kind) + "s"
}

func TypeMetaToDynamicResource(gvk schema.GroupVersionKind) schema.GroupVersionResource {
	resource := objectKindToResourceName(gvk.Kind)
	return schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: resource,
	}
}
