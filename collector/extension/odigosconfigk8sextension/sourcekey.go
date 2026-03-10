package odigosconfigk8sextension

import "fmt"

func k8sSourceKey(namespace, kind, name, containerName string) string {
	return fmt.Sprintf("%s/%s/%s/%s", namespace, kind, name, containerName)
}

// KeyPrefixForWorkload returns the cache key prefix for a workload (namespace/kind/name/).
// Used when collecting or clearing keys for a workload.
func KeyPrefixForWorkload(namespace, kind, name string) string {
	return k8sSourceKey(namespace, kind, name, "")
}

// WorkloadKeyString returns a short string for logging (namespace/kind/name, no trailing slash).
func WorkloadKeyString(namespace, kind, name string) string {
	return fmt.Sprintf("%s/%s/%s", namespace, kind, name)
}
