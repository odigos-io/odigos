package utils

import (
	"fmt"

	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"github.com/odigos-io/odigos/odiglet/pkg/env"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	corev1 "k8s.io/api/core/v1"
)

func IsPodInCurrentNode(pod *corev1.Pod) bool {
	return pod.Spec.NodeName == env.Current.NodeName
}

func GetResourceAttributes(podWorkload *workload.PodWorkload, podName string) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		semconv.K8SNamespaceName(podWorkload.Namespace),
		semconv.K8SPodName(podName),
	}

	switch podWorkload.Kind {
	case workload.WorkloadKindDeployment:
		attrs = append(attrs, semconv.K8SDeploymentName(podWorkload.Name))
	case workload.WorkloadKindStatefulSet:
		attrs = append(attrs, semconv.K8SStatefulSetName(podWorkload.Name))
	case workload.WorkloadKindDaemonSet:
		attrs = append(attrs, semconv.K8SDaemonSetName(podWorkload.Name))
	}

	return attrs
}

func GetPodExternalURL(ip string, ports []corev1.ContainerPort) string {
	if ports != nil && len(ports) > 0 {
		return fmt.Sprintf("http://%s:%d", ip, ports[0].ContainerPort)
	}

	return ""
}
