package configresolvers

import (
	"github.com/odigos-io/odigos/common/resourceattributes"

	"github.com/odigos-io/odigos/opampserver/pkg/deviceid"
)

type ResourceAttribute struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// the server resolved 6 resource attribute for the agent which it cannot discover itself:
// - node name
// - namespace
// - service name
// - pod name
// - container name
// - object name and kind (deployment, statefulset, daemonset, pod)
func CalculateServerAttributes(k8sAttributes *deviceid.K8sResourceAttributes, serviceName string) ([]ResourceAttribute, error) {
	resAttributes := resourceattributes.AfterPodStart(&resourceattributes.ContainerIdentifier{
		PodName:       k8sAttributes.PodName,
		Namespace:     k8sAttributes.Namespace,
		ContainerName: k8sAttributes.ContainerName,
	}).IncludeServiceName(serviceName)

	var result []ResourceAttribute
	for _, attr := range resAttributes {
		result = append(result, ResourceAttribute{
			Key:   string(attr.Key),
			Value: attr.Value.AsString(),
		})
	}

	return result, nil
}
