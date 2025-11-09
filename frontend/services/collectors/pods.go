package collectors

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/containers"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetPodsBySelector(ctx context.Context, selector string) ([]*model.PodInfo, error) {
	ns := env.GetCurrentNamespace()
	pods, err := kube.DefaultClient.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return nil, err
	}

	podsInfo := make([]*model.PodInfo, 0, len(pods.Items))
	for _, p := range pods.Items {
		podsInfo = append(podsInfo, podToInfo(&p))
	}

	return podsInfo, nil
}

func podToInfo(pod *corev1.Pod) *model.PodInfo {
	ready := formatReady(pod.Status.ContainerStatuses)
	status := deriveStatus(pod)
	restarts := sumRestarts(pod.Status.ContainerStatuses)
	node := pod.Spec.NodeName
	age := strings.ToLower(pod.CreationTimestamp.Time.Format(time.RFC3339))

	containerName := containers.GetCollectorContainerName(pod)
	imageVersion := extractImageVersionForContainer(pod.Spec.Containers, containerName)
	return &model.PodInfo{
		Name:     pod.Name,
		Ready:    ready,
		Status:   status,
		Restarts: restarts,
		NodeName: node,
		Age:      age,
		Image:    imageVersion,
	}
}

func formatReady(statuses []corev1.ContainerStatus) string {
	if len(statuses) == 0 {
		return "0/0"
	}
	total := len(statuses)
	ready := 0
	for _, cs := range statuses {
		if cs.Ready {
			ready++
		}
	}
	return fmt.Sprintf("%d/%d", ready, total)
}

func deriveStatus(pod *corev1.Pod) string {

	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Waiting != nil && cs.State.Waiting.Reason != "" {
			return cs.State.Waiting.Reason
		}
		if cs.State.Terminated != nil && cs.State.Terminated.Reason != "" {
			return cs.State.Terminated.Reason
		}
	}
	if string(pod.Status.Phase) != "" {
		return string(pod.Status.Phase)
	}
	return "Unknown"
}

func sumRestarts(statuses []corev1.ContainerStatus) int {
	sum := 0
	for _, cs := range statuses {
		sum += int(cs.RestartCount)
	}
	return sum
}
