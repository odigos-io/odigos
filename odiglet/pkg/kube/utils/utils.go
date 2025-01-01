package utils

import (
	"context"
	"fmt"

	k8scontainer "github.com/odigos-io/odigos/k8sutils/pkg/container"
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
	if err != nil {
		return nil, err
	}

	var filteredPods []corev1.Pod
	for _, pod := range podList.Items {
		if IsPodInCurrentNode(&pod) && pod.DeletionTimestamp == nil {
			if k8scontainer.AllContainersReady(&pod) {
				filteredPods = append(filteredPods, pod)
			}
		}
	}

	return filteredPods, nil
}

func GetPodExternalURL(ip string, ports []corev1.ContainerPort) string {
	if ports != nil && len(ports) > 0 {
		return fmt.Sprintf("http://%s:%d", ip, ports[0].ContainerPort)
	}

	return ""
}
