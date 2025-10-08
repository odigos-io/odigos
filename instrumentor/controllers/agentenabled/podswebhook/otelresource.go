package podswebhook

import (
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const otelServiceNameEnvVarName = "OTEL_SERVICE_NAME"
const otelResourceAttributesEnvVarName = "OTEL_RESOURCE_ATTRIBUTES"

type resourceAttribute struct {
	Key   attribute.Key
	Value string
}

func getResourceAttributes(podWorkload k8sconsts.PodWorkload, containerName string, ownerReferences []metav1.OwnerReference) []resourceAttribute {
	workloadKindKey := getWorkloadKindAttributeKey(podWorkload.Kind)
	resourceAttributes := []resourceAttribute{
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
	resourceAttributes = append(resourceAttributes, getOwnerReferenceAttributes(ownerReferences)...)
	return resourceAttributes
}

func getOwnerReferenceAttributes(ownerReferences []metav1.OwnerReference) []resourceAttribute {
	resourceAttributes := []resourceAttribute{}
	for _, ownerReference := range ownerReferences {
		switch ownerReference.Kind {
		case "Deployment":
			resourceAttributes = append(resourceAttributes,
				resourceAttribute{
					Key:   semconv.K8SDeploymentNameKey,
					Value: ownerReference.Name,
				},
				resourceAttribute{
					Key:   semconv.K8SDeploymentUIDKey,
					Value: string(ownerReference.UID),
				},
			)
		case "StatefulSet":
			resourceAttributes = append(resourceAttributes,
				resourceAttribute{
					Key:   semconv.K8SStatefulSetNameKey,
					Value: ownerReference.Name,
				},
				resourceAttribute{
					Key:   semconv.K8SStatefulSetUIDKey,
					Value: string(ownerReference.UID),
				},
			)
		case "DaemonSet":
			resourceAttributes = append(resourceAttributes,
				resourceAttribute{
					Key:   semconv.K8SDaemonSetNameKey,
					Value: ownerReference.Name,
				},
				resourceAttribute{
					Key:   semconv.K8SDaemonSetUIDKey,
					Value: string(ownerReference.UID),
				},
			)
		case "Job":
			resourceAttributes = append(resourceAttributes,
				resourceAttribute{
					Key:   semconv.K8SJobNameKey,
					Value: ownerReference.Name,
				},
				resourceAttribute{
					Key:   semconv.K8SJobUIDKey,
					Value: string(ownerReference.UID),
				},
			)
		case "ReplicaSet":
			resourceAttributes = append(resourceAttributes,
				resourceAttribute{
					Key:   semconv.K8SReplicaSetNameKey,
					Value: ownerReference.Name,
				},
			)
			resourceAttributes = append(resourceAttributes,
				resourceAttribute{
					Key:   semconv.K8SReplicaSetUIDKey,
					Value: string(ownerReference.UID),
				},
			)
		}
	}
	return resourceAttributes
}

func getWorkloadKindAttributeKey(workloadKind k8sconsts.WorkloadKind) attribute.Key {
	switch workloadKind {
	case k8sconsts.WorkloadKindDeployment:
		return semconv.K8SDeploymentNameKey
	case k8sconsts.WorkloadKindStatefulSet:
		return semconv.K8SStatefulSetNameKey
	case k8sconsts.WorkloadKindDaemonSet:
		return semconv.K8SDaemonSetNameKey
	case k8sconsts.WorkloadKindCronJob:
		return semconv.K8SCronJobNameKey
	case k8sconsts.WorkloadKindJob:
		return semconv.K8SJobNameKey
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

func InjectOtelResourceAndServiceNameEnvVars(existingEnvNames EnvVarNamesMap, container *corev1.Container, distroName string, pw k8sconsts.PodWorkload, serviceName string, ownerReferences []metav1.OwnerReference) EnvVarNamesMap {

	// OTEL_SERVICE_NAME
	existingEnvNames = InjectConstEnvVarToPodContainer(existingEnvNames, container, otelServiceNameEnvVarName, serviceName)

	// OTEL_RESOURCE_ATTRIBUTES
	resourceAttributes := getResourceAttributes(pw, container.Name, ownerReferences)
	resourceAttributesEnvValue := getResourceAttributesEnvVarValue(resourceAttributes)
	existingEnvNames = InjectConstEnvVarToPodContainer(existingEnvNames, container, otelResourceAttributesEnvVarName, resourceAttributesEnvValue)
	return existingEnvNames
}
