package collectors

import (
	"context"
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
	started := formatStarted(pod.Status.ContainerStatuses)
	status := deriveStatus(pod)
	restarts := sumRestarts(pod.Status.ContainerStatuses)
	node := pod.Spec.NodeName
	creationTimestamp := strings.ToLower(pod.CreationTimestamp.Time.Format(time.RFC3339))

	containerName := containers.GetCollectorContainerName(pod)
	imageVersion := extractImageVersionForContainer(pod.Spec.Containers, containerName)

	return &model.PodInfo{
		Namespace:         pod.Namespace,
		Name:              pod.Name,
		Ready:             ready,
		Started:           started,
		Status:            status,
		RestartsCount:     restarts,
		NodeName:          node,
		CreationTimestamp: creationTimestamp,
		Image:             imageVersion,
	}
}

func formatReady(statuses []corev1.ContainerStatus) bool {
	if len(statuses) == 0 {
		return false
	}
	total := len(statuses)
	ready := 0
	for _, cs := range statuses {
		if cs.Ready {
			ready++
		}
	}
	return ready == total
}

func formatStarted(statuses []corev1.ContainerStatus) bool {
	if len(statuses) == 0 {
		return false
	}
	total := len(statuses)
	started := 0
	for _, cs := range statuses {
		if cs.Started != nil && *cs.Started {
			started++
		}
	}
	return started == total
}

func deriveStatus(pod *corev1.Pod) string {
	// Container status reason when pod is in waiting or terminated state (e.g., "ImagePullBackOff", "CrashLoopBackOff", "Completed").
	// "Running" if all containers are running normally.
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Waiting != nil && cs.State.Waiting.Reason != "" {
			return cs.State.Waiting.Reason
		}
		if cs.State.Terminated != nil && cs.State.Terminated.Reason != "" {
			return cs.State.Terminated.Reason
		}
	}

	return string(corev1.PodRunning)
}

func sumRestarts(statuses []corev1.ContainerStatus) int {
	sum := 0
	for _, cs := range statuses {
		sum += int(cs.RestartCount)
	}
	return sum
}
