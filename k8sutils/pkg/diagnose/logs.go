package diagnose

import (
	"context"
	"fmt"
	"sync"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

// FetchWorkloadLogs fetches logs for the given pods and stores them in the workloadDir
func FetchWorkloadLogs(
	ctx context.Context,
	client kubernetes.Interface,
	builder Builder,
	namespace, workloadDir string,
	pods []corev1.Pod,
) error {
	var wg sync.WaitGroup

	for i := 0; i < len(pods); i++ {
		pod := &pods[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < len(pod.Spec.Containers); i++ {
				container := &pod.Spec.Containers[i]
				addContainerLogs(ctx, client, builder, namespace, workloadDir, pod.Name, container.Name, false)

				// Check if container has been restarted and get previous logs
				for i := 0; i < len(pod.Status.ContainerStatuses); i++ {
					status := &pod.Status.ContainerStatuses[i]
					if status.Name == container.Name && status.RestartCount > 0 {
						addContainerLogs(ctx, client, builder, namespace, workloadDir, pod.Name, container.Name, true)
					}
				}
			}

			// Also collect logs from init containers
			for i := 0; i < len(pod.Spec.InitContainers); i++ {
				container := &pod.Spec.InitContainers[i]
				addContainerLogs(ctx, client, builder, namespace, workloadDir, pod.Name, container.Name, false)
			}
		}()
	}

	wg.Wait()
	return nil
}

func addContainerLogs(
	ctx context.Context,
	client kubernetes.Interface,
	builder Builder,
	namespace, componentDir, podName, containerName string,
	previous bool,
) {
	var filename string
	if previous {
		filename = fmt.Sprintf("pod-%s.%s.previous.log.gz", podName, containerName)
	} else {
		filename = fmt.Sprintf("pod-%s.%s.log.gz", podName, containerName)
	}

	req := client.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
		Container: containerName,
		Previous:  previous,
	})

	logStream, err := req.Stream(ctx)
	if err != nil {
		// Write error message to log file so user knows what happened
		errorMsg := fmt.Sprintf("Error fetching logs: %v\n", err)
		if writeErr := builder.AddFile(componentDir, filename, []byte(errorMsg)); writeErr != nil {
			klog.V(1).ErrorS(writeErr, "Failed to write error message", "podName", podName)
		}
		return
	}
	//nolint:errcheck // this close is deferred to the end of the function
	defer logStream.Close()

	if err := builder.AddFileGzipped(componentDir, filename, logStream); err != nil {
		klog.V(1).ErrorS(err, "Failed to add container logs", "podName", podName, "containerName", containerName)
	}
}
