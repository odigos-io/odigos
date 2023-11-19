package utils

import (
	"context"

	"github.com/keyval-dev/odigos/odiglet/pkg/env"
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
		if IsPodInCurrentNode(&pod) && pod.Status.Phase == corev1.PodRunning {
			filteredPods = append(filteredPods, pod)
		}
	}

	if err != nil {
		return nil, err
	}

	return filteredPods, nil
}
