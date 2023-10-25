package migrations

import (
	"github.com/keyval-dev/odigos/cli/pkg/kube"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

func getResourceInterfaceDeployment(client *kube.Client, ns string) dynamic.ResourceInterface {
	return client.Dynamic.Resource(schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "deployments",
	}).Namespace(
		ns,
	)
}

func getResourceInterfaceDaemonSet(client *kube.Client, ns string) dynamic.ResourceInterface {
	return client.Dynamic.Resource(schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "daemonsets",
	}).Namespace(
		ns,
	)
}
