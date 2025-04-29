package podswebhook

import (
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	corev1 "k8s.io/api/core/v1"
)

const otelServiceNameEnvVarName = "OTEL_SERVICE_NAME"
const otelResourceAttributesEnvVarName = "OTEL_RESOURCE_ATTRIBUTES"

type resourceAttribute struct {
	Key   attribute.Key
	Value string
}

func getResourceAttributes(podWorkload k8sconsts.PodWorkload, containerName string) []resourceAttribute {
	workloadKindKey := getWorkloadKindAttributeKey(podWorkload.Kind)
	return []resourceAttribute{
		{
			Key:   semconv.K8SPodNameKey,
			Value: "$(ODIGOS_POD_NAME)",
		},
		{
			Key:   semconv.K8SContainerNameKey,
			Value: containerName,
		},
		{
			Key:   semconv.K8SNamespaceNameKey,
			Value: podWorkload.Namespace,
		},
		{
			Key:   workloadKindKey,
			Value: podWorkload.Name,
		},
	}
}

func getWorkloadKindAttributeKey(workloadKind k8sconsts.WorkloadKind) attribute.Key {
	switch workloadKind {
	case k8sconsts.WorkloadKindDeployment:
		return semconv.K8SDeploymentNameKey
	case k8sconsts.WorkloadKindStatefulSet:
		return semconv.K8SStatefulSetNameKey
	case k8sconsts.WorkloadKindDaemonSet:
		return semconv.K8SDaemonSetNameKey
	}
	return attribute.Key("")
}

func getResourceAttributesEnvVarValue(ra []resourceAttribute) string {
	var attrs []string
	for _, a := range ra {
		attrs = append(attrs, fmt.Sprintf("%s=%s", a.Key, a.Value))
	}
	return strings.Join(attrs, ",")
}

func InjectOtelResourceAndServiceNameEnvVars(existingEnvNames EnvVarNamesMap, container *corev1.Container, distroName string, pw k8sconsts.PodWorkload, serviceName string) EnvVarNamesMap {

	// OTEL_SERVICE_NAME
	existingEnvNames = InjectEnvVarToPodContainer(existingEnvNames, container, otelServiceNameEnvVarName, serviceName, nil)

	// OTEL_RESOURCE_ATTRIBUTES
	resourceAttributes := getResourceAttributes(pw, container.Name)
	resourceAttributesEnvValue := getResourceAttributesEnvVarValue(resourceAttributes)
	existingEnvNames = InjectEnvVarToPodContainer(existingEnvNames, container, otelResourceAttributesEnvVarName, resourceAttributesEnvValue, nil)
	return existingEnvNames
}
