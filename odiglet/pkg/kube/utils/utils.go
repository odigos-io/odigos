package utils

import (
	"context"

	"github.com/odigos-io/odigos/common"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"

	"github.com/odigos-io/odigos/odiglet/pkg/env"
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

func GetResourceAttributes(workload *common.PodWorkload, podName string) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		semconv.K8SNamespaceName(workload.Namespace),
		semconv.K8SPodName(podName),
	}

	switch workload.Kind {
	case "Deployment":
		attrs = append(attrs, semconv.K8SDeploymentName(workload.Name))
	case "StatefulSet":
		attrs = append(attrs, semconv.K8SStatefulSetName(workload.Name))
	case "DaemonSet":
		attrs = append(attrs, semconv.K8SDaemonSetName(workload.Name))
	}

	return attrs
}
