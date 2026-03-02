package odigosconfigk8sextension

import "fmt"

func k8sSourceKey(namespace, kind, name, containerName string) string {
	return fmt.Sprintf("%s/%s/%s/%s", namespace, kind, name, containerName)
}
