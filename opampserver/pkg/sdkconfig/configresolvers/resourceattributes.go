package configresolvers

import (
	"errors"

	"github.com/odigos-io/odigos/api/k8sconsts"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

type K8sResourceAttributes struct {
	Namespace     string
	WorkloadKind  string
	WorkloadName  string
	PodName       string
	ContainerName string
}

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
func CalculateServerAttributes(k8sAttributes *K8sResourceAttributes, nodeName string, serviceName string) ([]ResourceAttribute, error) {

	serverOfferResourceAttributes := []ResourceAttribute{
		{
			Key:   string(semconv.K8SNodeNameKey),
			Value: nodeName,
		},
		{
			Key:   string(semconv.K8SNamespaceNameKey),
			Value: k8sAttributes.Namespace,
		},
		{
			Key:   string(semconv.ServiceNameKey),
			Value: serviceName,
		},
		{
			Key:   string(semconv.K8SPodNameKey),
			Value: k8sAttributes.PodName,
		},
		{
			Key:   string(semconv.K8SContainerNameKey),
			Value: k8sAttributes.ContainerName,
		},
	}

	var objectNameKey string
	switch k8sAttributes.WorkloadKind {
	case "Deployment":
		objectNameKey = string(semconv.K8SDeploymentNameKey)
	case "StatefulSet":
		objectNameKey = string(semconv.K8SStatefulSetNameKey)
	case "DaemonSet":
		objectNameKey = string(semconv.K8SDaemonSetNameKey)
	case "Pod":
		objectNameKey = string(semconv.K8SPodNameKey)
	case "Job":
		objectNameKey = string(semconv.K8SJobNameKey)
	case "CronJob":
		objectNameKey = string(semconv.K8SCronJobNameKey)
	case "Rollout":
		// Argo Rollout - use custom key with argoproj prefix since it's an Argo-specific resource
		objectNameKey = k8sconsts.K8SArgoRolloutNameAttribute
	default:
		return serverOfferResourceAttributes, errors.New("unsupported workload kind")
	}

	serverOfferResourceAttributes = append(serverOfferResourceAttributes, ResourceAttribute{
		Key:   objectNameKey,
		Value: k8sAttributes.WorkloadName,
	})
	return serverOfferResourceAttributes, nil
}
