package collectors

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetGatewayPods(ctx context.Context) ([]*model.PodInfo, error) {
	ns := env.GetCurrentNamespace()
	selector := fmt.Sprintf("%s=%s", k8sconsts.OdigosCollectorRoleLabel, string(k8sconsts.CollectorsRoleClusterGateway))
	gatewayPods, err := kube.DefaultClient.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return nil, err
	}

	gatewayPodsInfo := make([]*model.PodInfo, 0, len(gatewayPods.Items))
	for _, p := range gatewayPods.Items {
		gatewayPodsInfo = append(gatewayPodsInfo, podToInfo(&p))
	}

	// Sort: more restarts first, then error status, then not ready
	sort.Slice(gatewayPodsInfo, func(i, j int) bool {

		if gatewayPodsInfo[i].Restarts != gatewayPodsInfo[j].Restarts {
			return gatewayPodsInfo[i].Restarts > gatewayPodsInfo[j].Restarts
		}

		ie, je := isErrorStatus(gatewayPodsInfo[i].Status), isErrorStatus(gatewayPodsInfo[j].Status)
		if ie != je {
			return ie
		}

		inr, jnr := isNotReady(gatewayPodsInfo[i].Ready), isNotReady(gatewayPodsInfo[j].Ready)
		if inr != jnr {
			return inr
		}

		return gatewayPodsInfo[i].Name < gatewayPodsInfo[j].Name
	})
	return gatewayPodsInfo, nil
}

func GetOdigletPods(ctx context.Context) ([]*model.PodInfo, error) {
	ns := env.GetCurrentNamespace()
	selector := fmt.Sprintf("%s=%s", k8sconsts.OdigosCollectorRoleLabel, string(k8sconsts.CollectorsRoleNodeCollector))
	pods, err := kube.DefaultClient.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return nil, err
	}

	out := make([]*model.PodInfo, 0, len(pods.Items))
	for _, p := range pods.Items {
		out = append(out, podToInfo(&p))
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Restarts != out[j].Restarts {
			return out[i].Restarts > out[j].Restarts
		}
		ie, je := isErrorStatus(out[i].Status), isErrorStatus(out[j].Status)
		if ie != je {
			return ie
		}
		inr, jnr := isNotReady(out[i].Ready), isNotReady(out[j].Ready)
		if inr != jnr {
			return inr
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

func podToInfo(pod *corev1.Pod) *model.PodInfo {
	ready := formatReady(pod.Status.ContainerStatuses)
	status := deriveStatus(pod)
	restarts := sumRestarts(pod.Status.ContainerStatuses)
	node := pod.Spec.NodeName
	age := humanizeAge(pod.CreationTimestamp.Time)
	image := gatewayImageTag(pod)
	return &model.PodInfo{
		Name:     pod.Name,
		Ready:    ready,
		Status:   status,
		Restarts: restarts,
		NodeName: node,
		Age:      age,
		Image:    image,
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

func humanizeAge(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)

	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

func gatewayImageTag(pod *corev1.Pod) string {
	for _, c := range pod.Spec.Containers {
		if c.Name == k8sconsts.OdigosClusterCollectorContainerName {
			return extractImageTag(c.Image)
		}
	}
	if len(pod.Spec.Containers) > 0 {
		return extractImageTag(pod.Spec.Containers[0].Image)
	}
	return ""
}

func extractImageTag(image string) string {
	if idx := strings.Index(image, "@"); idx >= 0 {
		image = image[:idx]
	}
	parts := strings.Split(image, "/")
	last := parts[len(parts)-1]
	if colon := strings.LastIndex(last, ":"); colon >= 0 {
		return last[colon+1:]
	}
	return last
}

func isErrorStatus(status string) bool {
	if status == "" {
		return false
	}
	s := strings.ToLower(status)

	if strings.Contains(s, "backoff") {
		return true
	}
	if strings.Contains(s, "errimagepull") || strings.Contains(s, "imagepullbackoff") {
		return true
	}
	if strings.Contains(s, "crashloopbackoff") {
		return true
	}
	if strings.Contains(s, "oomkilled") {
		return true
	}
	if strings.Contains(s, "error") {
		return true
	}
	if strings.Contains(s, "failed") {
		return true
	}
	return false
}

func isNotReady(readyStr string) bool {
	parts := strings.Split(readyStr, "/")
	if len(parts) != 2 {
		return false
	}
	x, y := parts[0], parts[1]

	var xi, yi int
	fmt.Sscanf(x, "%d", &xi)
	fmt.Sscanf(y, "%d", &yi)

	return yi > 0 && xi < yi
}
