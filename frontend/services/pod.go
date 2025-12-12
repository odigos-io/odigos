package services

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetPodDetails(ctx context.Context, namespace, name string) (*model.PodDetails, error) {
	pod, err := kube.DefaultClient.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	var nodePtr *string
	if pod.Spec.NodeName != "" {
		nodePtr = StringPtr(pod.Spec.NodeName)
	}

	var statusPtr *model.PodPhase
	statusPtr = mapPodPhase(pod.Status.Phase)

	containers := buildContainersOverview(pod)

	manifestYAML, err := K8sManifest(ctx, namespace, model.K8sResourceKindPod, name)
	if err != nil {
		return nil, err
	}

	return &model.PodDetails{
		Name:         pod.Name,
		Namespace:    pod.Namespace,
		Node:         nodePtr,
		Status:       statusPtr,
		Containers:   containers,
		ManifestYaml: manifestYAML,
	}, nil
}

// TODO: Create a dedicated services conversion file and move all conversion helpers there.
func buildContainersOverview(pod *corev1.Pod) []*model.ContainerOverview {

	statusByName := make(map[string]corev1.ContainerStatus, len(pod.Status.ContainerStatuses))
	for _, cs := range pod.Status.ContainerStatuses {
		statusByName[cs.Name] = cs
	}

	containers := make([]*model.ContainerOverview, 0, len(pod.Spec.Containers))
	for _, c := range pod.Spec.Containers {
		cs, ok := statusByName[c.Name]

		ready := false
		restarts := 0
		status := model.ContainerLifecycleStatusWaiting
		var stateReasonPtr *string
		var startedAtPtr *string

		if ok {
			ready = cs.Ready
			restarts = int(cs.RestartCount)
			if cs.State.Running != nil {
				status = model.ContainerLifecycleStatusRunning
				if !cs.State.Running.StartedAt.IsZero() {
					startedAtPtr = StringPtr(cs.State.Running.StartedAt.Time.Format(time.RFC3339))
				}
			} else if cs.State.Waiting != nil {
				status = model.ContainerLifecycleStatusWaiting
				if cs.State.Waiting.Reason != "" {
					stateReasonPtr = StringPtr(cs.State.Waiting.Reason)
				}
			} else if cs.State.Terminated != nil {
				status = model.ContainerLifecycleStatusTerminated
				if cs.State.Terminated.Reason != "" {
					stateReasonPtr = StringPtr(cs.State.Terminated.Reason)
				}
			}
		}

		containers = append(containers, &model.ContainerOverview{
			Name:        c.Name,
			Image:       StringPtr(c.Image),
			Status:      status,
			StateReason: stateReasonPtr,
			Ready:       ready,
			Restarts:    restarts,
			StartedAt:   startedAtPtr,
			Resources:   buildContainerResources(c.Resources),
		})
	}
	return containers
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

// RestartPod deletes a pod to trigger a restart. If the pod is managed by a Deployment,
// StatefulSet, DaemonSet, etc., the controller will automatically recreate it.
func RestartPod(ctx context.Context, namespace, name string) error {
	policy := metav1.DeletePropagationBackground
	err := kube.DefaultClient.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{
		PropagationPolicy: &policy,
	})

	// If the pod doesn't exist, consider it already deleted/restarted
	if apierrors.IsNotFound(err) {
		return nil
	}

	return err
}

// StreamPodLogs streams pod logs via SSE (Server-Sent Events).
// It supports optional query parameters:
//   - container: specific container name (defaults to first container)
//   - follow: "true" to stream logs in real-time (defaults to "true")
//   - tailLines: number of lines to show from the end (defaults to 100)
//   - previous: "true" to get logs from previous terminated container
func StreamPodLogs(c *gin.Context) {
	namespace := c.Param("namespace")
	podName := c.Param("name")
	containerName := c.Query("container")
	follow := c.DefaultQuery("follow", "true") == "true"
	previous := c.Query("previous") == "true"

	// Parse tailLines, default to 100
	var tailLines int64 = 100
	if tl := c.Query("tailLines"); tl != "" {
		fmt.Sscanf(tl, "%d", &tailLines)
	}

	// If no container specified, get the first container from the pod
	if containerName == "" {
		pod, err := kube.DefaultClient.CoreV1().Pods(namespace).Get(c.Request.Context(), podName, metav1.GetOptions{})
		if err != nil {
			c.JSON(500, gin.H{"error": fmt.Sprintf("failed to get pod: %v", err)})
			return
		}
		if len(pod.Spec.Containers) == 0 {
			c.JSON(400, gin.H{"error": "pod has no containers"})
			return
		}
		containerName = pod.Spec.Containers[0].Name
	}

	// Build log options
	logOptions := &corev1.PodLogOptions{
		Container: containerName,
		Follow:    follow,
		TailLines: &tailLines,
		Previous:  previous,
	}

	// Get the log stream
	req := kube.DefaultClient.CoreV1().Pods(namespace).GetLogs(podName, logOptions)
	stream, err := req.Stream(c.Request.Context())
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("failed to stream logs: %v", err)})
		return
	}
	defer stream.Close()

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// Stream logs line by line
	reader := bufio.NewReader(stream)
	for {
		select {
		case <-c.Request.Context().Done():
			return
		default:
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					if !follow {
						// For non-follow mode, we're done when we reach EOF
						return
					}
					// For follow mode, EOF means the container stopped
					fmt.Fprintf(c.Writer, "data: [stream ended]\n\n")
					c.Writer.Flush()
					return
				}
				// Other error
				fmt.Fprintf(c.Writer, "data: [error: %v]\n\n", err)
				c.Writer.Flush()
				return
			}

			// Send the log line as an SSE event
			fmt.Fprintf(c.Writer, "data: %s\n", line)
			c.Writer.Flush()
		}
	}
}
