package collectors

import (
	"context"
	"strings"
	"time"

	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/frontend/services"
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
	containerName := containers.GetCollectorContainerName(pod)
	cs := getContainerStatusByName(pod.Status.ContainerStatuses, containerName)

	return &model.PodInfo{
		Namespace:         pod.Namespace,
		Name:              pod.Name,
		Status:            containerStatus(cs),
		RestartsCount:     containerRestarts(cs),
		NodeName:          pod.Spec.NodeName,
		CreationTimestamp: strings.ToLower(pod.CreationTimestamp.Time.Format(time.RFC3339)),
		Image:             extractImageVersionForContainer(pod.Spec.Containers, containerName),
	}
}

func getContainerStatusByName(statuses []corev1.ContainerStatus, name string) *corev1.ContainerStatus {
	for i := range statuses {
		if statuses[i].Name == name {
			return &statuses[i]
		}
	}
	return nil
}

func containerStatus(cs *corev1.ContainerStatus) string {
	if cs == nil {
		return "Unknown"
	}
	if cs.State.Waiting != nil && cs.State.Waiting.Reason != "" {
		return cs.State.Waiting.Reason
	}
	if cs.State.Terminated != nil && cs.State.Terminated.Reason != "" {
		return cs.State.Terminated.Reason
	}
	return string(corev1.PodRunning)
}

func containerRestarts(cs *corev1.ContainerStatus) int {
	if cs == nil {
		return 0
	}
	return int(cs.RestartCount)
}

// GetCollectorPodDetails returns pod details with only the collector container.
// This is used for the pod details drawer when clicking on a collector pod.
func GetCollectorPodDetails(ctx context.Context, namespace, name string) (*model.PodDetails, error) {
	pod, err := kube.DefaultClient.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	var nodePtr *string
	if pod.Spec.NodeName != "" {
		nodePtr = services.StringPtr(pod.Spec.NodeName)
	}

	statusPtr := mapPodPhase(pod.Status.Phase)

	// Get only the collector container
	containerName := containers.GetCollectorContainerName(pod)
	collectorContainers := buildCollectorContainerOverview(pod, containerName)

	manifestYAML, err := services.K8sManifest(ctx, namespace, model.K8sResourceKindPod, name)
	if err != nil {
		return nil, err
	}

	return &model.PodDetails{
		Name:         pod.Name,
		Namespace:    pod.Namespace,
		Node:         nodePtr,
		Status:       statusPtr,
		Containers:   collectorContainers,
		ManifestYaml: manifestYAML,
	}, nil
}

func mapPodPhase(p corev1.PodPhase) *model.PodPhase {
	switch p {
	case corev1.PodPending:
		v := model.PodPhasePending
		return &v
	case corev1.PodRunning:
		v := model.PodPhaseRunning
		return &v
	case corev1.PodSucceeded:
		v := model.PodPhaseSucceeded
		return &v
	case corev1.PodFailed:
		v := model.PodPhaseFailed
		return &v
	case corev1.PodUnknown:
		fallthrough
	default:
		v := model.PodPhaseUnknown
		return &v
	}
}

// buildCollectorContainerOverview builds the container overview for only the collector container.
func buildCollectorContainerOverview(pod *corev1.Pod, containerName string) []*model.ContainerOverview {
	// Find the container spec
	var containerSpec *corev1.Container
	for i := range pod.Spec.Containers {
		if pod.Spec.Containers[i].Name == containerName {
			containerSpec = &pod.Spec.Containers[i]
			break
		}
	}
	if containerSpec == nil {
		return []*model.ContainerOverview{}
	}

	// Find the container status
	cs := getContainerStatusByName(pod.Status.ContainerStatuses, containerName)

	ready := false
	restarts := 0
	status := model.ContainerLifecycleStatusWaiting
	var stateReasonPtr *string
	var startedAtPtr *string

	if cs != nil {
		ready = cs.Ready
		restarts = int(cs.RestartCount)
		if cs.State.Running != nil {
			status = model.ContainerLifecycleStatusRunning
			if !cs.State.Running.StartedAt.IsZero() {
				startedAtPtr = services.StringPtr(cs.State.Running.StartedAt.Time.Format(time.RFC3339))
			}
		} else if cs.State.Waiting != nil {
			status = model.ContainerLifecycleStatusWaiting
			if cs.State.Waiting.Reason != "" {
				stateReasonPtr = services.StringPtr(cs.State.Waiting.Reason)
			}
		} else if cs.State.Terminated != nil {
			status = model.ContainerLifecycleStatusTerminated
			if cs.State.Terminated.Reason != "" {
				stateReasonPtr = services.StringPtr(cs.State.Terminated.Reason)
			}
		}
	}

	return []*model.ContainerOverview{
		{
			Name:        containerSpec.Name,
			Image:       services.StringPtr(containerSpec.Image),
			Status:      status,
			StateReason: stateReasonPtr,
			Ready:       ready,
			Restarts:    restarts,
			StartedAt:   startedAtPtr,
			Resources:   buildContainerResources(containerSpec.Resources),
		},
	}
}

func buildContainerResources(reqs corev1.ResourceRequirements) *model.Resources {
	if reqs.Requests == nil && reqs.Limits == nil {
		return nil
	}

	var requests *model.ResourceAmounts
	if len(reqs.Requests) > 0 {
		var cpuPtr, memPtr *string
		if q, ok := reqs.Requests[corev1.ResourceCPU]; ok {
			s := q.String()
			cpuPtr = &s
		}
		if q, ok := reqs.Requests[corev1.ResourceMemory]; ok {
			s := q.String()
			memPtr = &s
		}
		if cpuPtr != nil || memPtr != nil {
			requests = &model.ResourceAmounts{
				CPU:    cpuPtr,
				Memory: memPtr,
			}
		}
	}

	var limits *model.ResourceAmounts
	if len(reqs.Limits) > 0 {
		var cpuPtr, memPtr *string
		if q, ok := reqs.Limits[corev1.ResourceCPU]; ok {
			s := q.String()
			cpuPtr = &s
		}
		if q, ok := reqs.Limits[corev1.ResourceMemory]; ok {
			s := q.String()
			memPtr = &s
		}
		if cpuPtr != nil || memPtr != nil {
			limits = &model.ResourceAmounts{
				CPU:    cpuPtr,
				Memory: memPtr,
			}
		}
	}

	if requests == nil && limits == nil {
		return nil
	}

	return &model.Resources{
		Requests: requests,
		Limits:   limits,
	}
}
