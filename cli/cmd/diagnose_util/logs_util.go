package diagnose_util

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/odigos-io/odigos/cli/cmd/resources"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

const logBufferSize = 1024 * 1024 // 1MB buffer size for reading logs in chunks

func FetchOdigosComponentsLogs(ctx context.Context, client *kube.Client, logDir string) error {
	fmt.Printf("Fetching Odigos Components Logs...\n")

	odigosNamespace, err := resources.GetOdigosNamespace(client, ctx)
	if err != nil {
		return err
	}

	pods, err := client.CoreV1().Pods(odigosNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	var wg sync.WaitGroup

	for _, pod := range pods.Items {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fetchPodLogs(ctx, client, odigosNamespace, pod, logDir)
		}()
	}

	wg.Wait()

	return nil
}

func fetchPodLogs(ctx context.Context, client *kube.Client, odigosNamespace string, pod v1.Pod, logDir string) {
	for _, container := range pod.Spec.Containers {
		fetchingContainerLogs(ctx, client, odigosNamespace, pod, container, logDir, false)

		// Check if the pod has been restarted
		if pod.Status.ContainerStatuses != nil {
			for _, status := range pod.Status.ContainerStatuses {
				if status.RestartCount > 0 {
					// Fetch logs from the previous instance of the container
					fetchingContainerLogs(ctx, client, odigosNamespace, pod, container, logDir, true)
				}
			}
		}

	}
}

func fetchingContainerLogs(ctx context.Context, client *kube.Client, odigosNamespace string, pod v1.Pod, container v1.Container, logDir string, previous bool) {
	klog.V(2).InfoS("Fetching logs for Pod", "podName", pod.Name, "containerName", container.Name, "node", pod.Spec.NodeName)

	// Define the log file path for saving compressed logs
	logFileName := getLogFileName(pod, container, previous)
	logFilePath := filepath.Join(logDir, logFileName)
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		klog.V(1).ErrorS(err, "Failed to create log file", "logFilePath", logFilePath)
		return
	}
	defer logFile.Close()

	req := client.CoreV1().Pods(odigosNamespace).GetLogs(pod.Name, &v1.PodLogOptions{Previous: previous})
	logStream, err := req.Stream(ctx)
	if err != nil {
		klog.V(1).ErrorS(err, "Failed to create log stream", "logFilePath", logFilePath)
		return
	}
	defer logStream.Close()

	if err = saveLogsToGzipFileInBatches(logFile, logStream, logBufferSize); err != nil {
		klog.V(1).ErrorS(err, "Failed to save logs to file", "logFilePath", logFilePath)
		return
	}
}

func getLogFileName(pod v1.Pod, container v1.Container, previous bool) string {
	if previous {
		return pod.Name + "_" + container.Name + "_" + pod.Spec.NodeName + "_previous.log.gz"
	}
	return pod.Name + "_" + container.Name + "_" + pod.Spec.NodeName + ".log.gz"
}

func saveLogsToGzipFileInBatches(logFile *os.File, logStream io.ReadCloser, bufferSize int) error {
	// Create a gzip writer to compress the logs
	gzipWriter := gzip.NewWriter(logFile)
	defer gzipWriter.Close()

	// Read logs in chunks and write them to the file
	buffer := make([]byte, bufferSize)
	for {
		n, err := logStream.Read(buffer)
		if n > 0 {
			// Write the chunk to the gzip file
			if _, err := gzipWriter.Write(buffer[:n]); err != nil {
				return err
			}
		}

		if err == io.EOF {
			// End of the log stream; break out of the loop
			break
		}

		if err != nil {
			return err
		}
	}

	return nil
}
