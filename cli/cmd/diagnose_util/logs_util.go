package diagnose_util

import (
	"compress/gzip"
	"context"
	"fmt"
	"github.com/odigos-io/odigos/cli/cmd/resources"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"io"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"path/filepath"
	"sync"
)

const logBufferSize = 1024 * 1024 // 1MB buffer size for reading logs in chunks

func FetchOdigosComponentsLogs(ctx context.Context, client *kube.Client, logDir string) error {
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
		fetchingContainerLogs(ctx, client, odigosNamespace, pod, container, logDir)

	}
}

func fetchingContainerLogs(ctx context.Context, client *kube.Client, odigosNamespace string, pod v1.Pod, container v1.Container, logDir string) {
	logPrefix := fmt.Sprintf("Fetching logs for Pod: %s, Container: %s, Node: %s", pod.Name, container.Name, pod.Spec.NodeName)
	fmt.Printf(logPrefix + "\n")

	// Define the log file path for saving compressed logs
	logFilePath := filepath.Join(logDir, pod.Name+"_"+container.Name+"_"+pod.Spec.NodeName+".log.gz")
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		fmt.Printf(logPrefix+" - Failed - Error creating log file: %v\n", err)
		return
	}
	defer logFile.Close()

	req := client.CoreV1().Pods(odigosNamespace).GetLogs(pod.Name, &v1.PodLogOptions{})
	logStream, err := req.Stream(ctx)
	if err != nil {
		fmt.Printf(logPrefix+" - Failed - Error creating log stream: %v\n", err)
		return
	}
	defer logStream.Close()

	if err = saveLogsToGzipFileInBatches(logFile, logStream, logBufferSize); err != nil {
		fmt.Printf(logPrefix+" - Failed - Error saving logs to file: %v\n", err)
		return
	}
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
