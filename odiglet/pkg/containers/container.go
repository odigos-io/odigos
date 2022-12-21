package containers

import (
	"context"
	"errors"
	"github.com/keyval-dev/odigos/odiglet/pkg/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"strings"
	"time"
)

type ContainerID struct {
	Runtime string
	ID      string
}

func newContainerID(str string) (*ContainerID, error) {
	parts := strings.Split(str, "://")
	if len(parts) != 2 {
		return nil, errors.New("invalid container id")
	}

	return &ContainerID{
		Runtime: parts[0],
		ID:      parts[1],
	}, nil
}

func FindIDs(ctx context.Context, podName string, podNamespace string, kubeClient kubernetes.Interface) ([]*ContainerID, error) {
	// Wait for all the containers to be running
	err := wait.PollImmediate(time.Second, 30*time.Second, isPodRunning(ctx, kubeClient, podName, podNamespace))
	if err != nil {
		return nil, err
	}

	// Get the pod
	pod, err := kubeClient.CoreV1().Pods(podNamespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	// Get container ids of found containers
	var containerIds []*ContainerID
	for _, container := range pod.Status.ContainerStatuses {
		if !isInstrumentationContainerStatus(&container) {
			log.Logger.V(0).Info("container status", "container", container)
			id, err := newContainerID(container.ContainerID)
			if err != nil {
				return nil, err
			}

			containerIds = append(containerIds, id)
		}
	}

	log.Logger.V(0).Info("Found container ids", "containerIds", containerIds)
	return containerIds, nil
}

// return a condition function that indicates whether the given pod is
// currently running
func isPodRunning(ctx context.Context, kubeClient kubernetes.Interface, podName string, namespace string) wait.ConditionFunc {
	return func() (bool, error) {
		pod, err := kubeClient.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		for _, containerStatus := range pod.Status.ContainerStatuses {
			if !isInstrumentationContainerStatus(&containerStatus) && !containerStatus.Ready {
				log.Logger.V(0).Info("Container not ready", "container", containerStatus)
				return false, nil
			}
		}

		return true, nil
	}
}

func isInstrumentationContainerStatus(containerStatus *corev1.ContainerStatus) bool {
	return strings.Contains(containerStatus.Image, "otel-go-agent")
}
