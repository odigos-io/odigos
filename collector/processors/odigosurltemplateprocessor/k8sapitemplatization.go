package odigosurltemplateprocessor

import (
	"regexp"
	"strings"
)

// Kubernetes API groups follow DNS-like rules:
// - lowercase letters: a–z
// - digits: 0–9
// - hyphens: -
// - dots separating segments
var k8sApiGroupNameRegex = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`)

// Kubernetes API group versions follow this format: v[0-9]+((alpha|beta)[0-9]+)?
// - Start with v followed by a number: v1, v2, etc.
// - Optional alpha or beta with a number: v1alpha1, v1beta2, etc.
// Examples: v1, v2, v1alpha1, v1beta2, etc.
var k8sApiGroupVersionRegex = regexp.MustCompile(`^v[0-9]+((alpha|beta)[0-9]+)?$`)

// limit the possible values for cluster-scoped core resource type names
// this is to avoid capturing non-k8s api endpoints that are similar to k8s api endpoints
var clusterScopedCoreResources = map[string]struct{}{
	"nodes": {}, "persistentvolumes": {}, "componentstatuses": {}, "bindings": {},
}

// limit the possible values for namespace-scoped core resource type names
// this is to avoid capturing non-k8s api endpoints that are similar to k8s api endpoints
var namespaceScopedCoreResources = map[string]struct{}{
	"pods": {}, "services": {}, "replicationcontrollers": {}, "configmaps": {}, "secrets": {}, "endpoints": {}, "serviceaccounts": {}, "persistentvolumeclaims": {}, "events": {},
}

// attempt to templatize the url path segments as k8s api endpoint
// return templated path if successful, otherwise return empty string and false
func TemplatizeK8sApiEndpoint(pathSegments []string) (string, bool) {
	// k8s api has minimum of 3 segments: /api/v1/namespaces
	// and maximum of 8 segments: /apis/{group}/{version}/namespaces/{namespace}/{resource}/{name}/{subresource}
	if len(pathSegments) < 3 || len(pathSegments) > 8 {
		return "", false
	}

	switch pathSegments[0] {
	case "api":
		return templatizedSegmentAndSuccessIntoUrl(templatizeCoreApi(pathSegments))
	case "apis":
		return templatizedSegmentAndSuccessIntoUrl(templatizeNamedApiGroups(pathSegments))
	default:
		// k8s api endpoint starts with "api" or "apis"
		return "", false
	}
}

func templatizedSegmentAndSuccessIntoUrl(templatedPathSegments []string, success bool) (string, bool) {
	if !success {
		return "", false
	}
	return strings.Join(templatedPathSegments, "/"), true
}

// Prefix: /api/{version}/...
// Contains the built-in, core Kubernetes resources.
// Only core API group resources live here — these are the original Kubernetes objects.
// Examples: /api/v1/namespaces, /api/v1/pods/foo/status, /api/v1/events, etc.
func templatizeCoreApi(pathSegments []string) ([]string, bool) {
	if pathSegments[1] != "v1" {
		return nil, false
	}

	var prefixSegments []string
	var success bool
	var templatedPathSegments []string
	if pathSegments[2] == "namespaces" {
		switch len(pathSegments) {
		case 3:
			// /api/v1/namespaces. return original path segments
			return pathSegments, true
		case 4:
			// /api/v1/namespaces/{namespace-name}.
			// treat it as cluster-scoped resource
			templatedPathSegments, success = templatizeNamespacesK8sApiResource(pathSegments[3:])
			prefixSegments = pathSegments[:3]
		default:
			// not query on namespace itself, but on namespaced resources
			// e.g. /api/v1/namespaces/default/pods
			// verify the resource type is an actual namespace-scoped resource type
			if _, ok := namespaceScopedCoreResources[pathSegments[4]]; !ok {
				return nil, false
			}
			// templatize the namespace name and resource type
			templatedPathSegments, success = templatizeNamespacesK8sApiResource(pathSegments[3:])
			prefixSegments = pathSegments[:3]
		}
	} else {
		// verify resource type for cluster-scoped core resources is in the list
		if _, ok := clusterScopedCoreResources[pathSegments[2]]; !ok {
			return nil, false
		}
		prefixSegments = pathSegments[:2]
		// call the function without 1. "/api" 2. "v1" segments
		templatedPathSegments, success = templatizeClusterScopedK8sApiResources(pathSegments[2:])
	}
	if !success {
		return nil, false
	}
	return append(prefixSegments, templatedPathSegments...), true
}

// Prefix: /apis/{group}/{version}/...
// Contains all other API groups, including:
// - Extensions (apps, batch)
// - RBAC (rbac.authorization.k8s.io)
// - Admission controllers (admissionregistration.k8s.io)
// - Events (events.k8s.io)
// - Custom Resource Definitions (CRDs)
func templatizeNamedApiGroups(pathSegments []string) ([]string, bool) {

	groupName := pathSegments[1]

	// make sure it looks like a k8s api group name (e.g. "apps", "batch", "networking.k8s.io", "reports.kyverno.io", "external-secrets.io")
	if !k8sApiGroupNameRegex.MatchString(groupName) {
		return nil, false
	}

	groupVersion := pathSegments[2]
	if !k8sApiGroupVersionRegex.MatchString(groupVersion) {
		return nil, false
	}

	var success bool
	var prefixSegments []string
	var templatedPathSegments []string
	if pathSegments[3] == "namespaces" {
		templatedPathSegments, success = templatizeNamespacesK8sApiResource(pathSegments[4:])
		prefixSegments = pathSegments[:4]
	} else {
		templatedPathSegments, success = templatizeClusterScopedK8sApiResources(pathSegments[3:])
		prefixSegments = pathSegments[:3]
	}
	if !success {
		return nil, false
	}
	return append(prefixSegments, templatedPathSegments...), true
}

// Prefix: /apis/{group}/{version}/namespaces/... or /api/v1/namespaces/...
// parameter for the function is everything after the "namespaces" segment, to work for both cases.
// Contains the namespace-scoped resources (either core or named api group)
// Examples: /apis/apps/v1/namespaces/default/deployments, /api/v1/namespaces/default/pods/status, etc.
// return: the templated path segements if successful, otherwise return empty slice and false
func templatizeNamespacesK8sApiResource(nsPathSegments []string) ([]string, bool) {
	switch len(nsPathSegments) {
	case 1:
		// no resource type, it's a namespace itself (e.g. /api/v1/namespaces/default)
		// /apis/{group}/{version}/namespaces/{namespace}
		// /api/v1/namespaces/{namespace}
		// Example: /api/v1/namespaces/default
		// templatize the namespace name
		nsPathSegments[0] = "{namespace-name}"
		return nsPathSegments, true
	case 2:
		// resource list
		// /apis/{group}/{version}/namespaces/{namespace}/{resource}
		// /api/v1/namespaces/{namespace}/{resource}
		// Example: /apis/apps/v1/namespaces/default/deployments
		// templatize just the namespace name (which can have high cardinality)
		nsPathSegments[0] = "{namespace-name}"
		return nsPathSegments, true
	case 3:
		// resource by name
		// /apis/{group}/{version}/namespaces/{namespace}/{resource}/{name}
		// /api/v1/namespaces/{namespace}/{resource}/{name}
		// Example: /apis/apps/v1/namespaces/default/deployments/foo
		// templatize namespace name and resource name
		nsPathSegments[0] = "{namespace-name}"
		nsPathSegments[2] = "{resource-name}"
		return nsPathSegments, true
	case 4:
		// subresource
		// /apis/{group}/{version}/namespaces/{namespace}/{resource}/{name}/{subresource}
		// /api/v1/namespaces/{namespace}/{resource}/{name}/{subresource}
		// Example: /apis/apps/v1/namespaces/default/deployments/foo/status
		// templatize namespace name, and resource name. keep subresource untemplated as it has low cardinality
		nsPathSegments[0] = "{namespace-name}"
		nsPathSegments[2] = "{resource-name}"
		return nsPathSegments, true
	default:
		// number of url path segment for namespace-scoped resource does not match k8s api endpoint pattern
		return nil, false
	}
}

// Prefix: /apis/{group}/{version}/... or /api/v1/...
// Examples: "/apis/external-secrets.io/v1/clustersecretstores/admin", "/api/v1/nodes"
// parameter for the function is everything after the version (starting with resource type)
// return: the templated path segements if successful, otherwise return empty slice and false
func templatizeClusterScopedK8sApiResources(clusterResourceSegments []string) ([]string, bool) {
	switch len(clusterResourceSegments) {
	case 1:
		// resource list
		// /apis/{group}/{version}/{resource}
		// /api/v1/{resource}
		// Example: /apis/rbac.authorization.k8s.io/v1/clusterroles
		// resource type is low cardinality, so no need for templating
		// return the url as is
		return clusterResourceSegments, true
	case 2:
		// resource by name
		// /apis/{group}/{version}/{resource}/{name}
		// /api/v1/{resource}/{name}
		// Example: /apis/rbac.authorization.k8s.io/v1/clusterroles/admin
		// templatize just the resource name (high cardinality)
		clusterResourceSegments[1] = "{resource-name}"
		return clusterResourceSegments, true
	case 3:
		// subresource
		// /apis/{group}/{version}/{resource}/{name}/{subresource}
		// /api/v1/{resource}/{name}/{subresource}
		// Example: /apis/rbac.authorization.k8s.io/v1/clusterroles/admin/status
		// templatize resource name only, subresource is low cardinality, so no need for templating
		clusterResourceSegments[1] = "{resource-name}"
		return clusterResourceSegments, true
	default:
		// number of url path segment for cluster-scoped resource does not match k8s api endpoint pattern
		return nil, false
	}
}
