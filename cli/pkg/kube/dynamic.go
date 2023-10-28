package kube

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func parseAPIVersion(apiVersion string) (group, version string) {
	parts := strings.Split(apiVersion, "/")
	if len(parts) == 1 {
		return "", parts[0]
	}
	return parts[0], parts[1]
}

func TypeMetaToDynamicResource(typemeta metav1.TypeMeta) schema.GroupVersionResource {
	group, version := parseAPIVersion(typemeta.APIVersion)
	resource := objectKindToResourceName(typemeta.Kind)
	return schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: resource,
	}
}
