package utils

import (
	"context"

	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"github.com/odigos-io/odigos/odiglet/pkg/env"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func IsPodInCurrentNode(pod *corev1.Pod) bool {
	return pod.Spec.NodeName == env.Current.NodeName
}

func GetRunningPods(ctx context.Context, labels map[string]string, ns string, kubeClient client.Client) ([]corev1.Pod, error) {
	var podList corev1.PodList
	err := kubeClient.List(ctx, &podList, client.MatchingLabels(labels), client.InNamespace(ns))

	var filteredPods []corev1.Pod
	for _, pod := range podList.Items {
		if IsPodInCurrentNode(&pod) && pod.Status.Phase == corev1.PodRunning && pod.DeletionTimestamp == nil {
			filteredPods = append(filteredPods, pod)
		}
	}

	if err != nil {
		return nil, err
	}

	return filteredPods, nil
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
		return ip + ":" + string(ports[0].ContainerPort)
	}

	return ""
}
