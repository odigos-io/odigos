package cmd

import (
	"compress/gzip"
	"context"
	"fmt"
	"github.com/odigos-io/odigos/cli/cmd/resources"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/spf13/cobra"
	"io"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"path/filepath"
)

const (
	mainDir       = "odigos-diagnose"
	logDir        = "logs"
	logBufferSize = 1024 * 1024 // 1MB buffer size for reading logs in chunks
)

var diagnozeCmd = &cobra.Command{
	Use:   "diagnose",
	Short: "Diagnose Client Cluster",
	Long:  `Diagnose Client Cluster to identify issues and resolve them. This command is useful for troubleshooting and debugging.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		client, err := kube.CreateClient(cmd)
		if err != nil {
			kube.PrintClientErrorAndExit(err)
		}

		err = startDiagnose(ctx, client)
		if err != nil {
			fmt.Printf("The diagnose script crashed on: %v\n", err)
		}
	},
}

func startDiagnose(ctx context.Context, client *kube.Client) error {
	if err := createAllDirs(); err != nil {
		return err
	}

	if err := fetchOdigosComponentsLogs(ctx, client); err != nil {
		return err
	}

	return nil
}

func createAllDirs() error {
	if err := os.RemoveAll(mainDir); err != nil {
		return err
	}
	if err := os.MkdirAll(mainDir, 0755); err != nil {
		return err
	}

	logsPath := filepath.Join(mainDir, logDir)
	if err := os.MkdirAll(logsPath, 0755); err != nil {
		return err
	}

	return nil
}

func fetchOdigosComponentsLogs(ctx context.Context, client *kube.Client) error {
	odigosNamespace, err := resources.GetOdigosNamespace(client, ctx)
	if err != nil {
		return err
	}

	pods, err := client.CoreV1().Pods(odigosNamespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, pod := range pods.Items {
		if err = fetchPodLogs(ctx, client, odigosNamespace, pod); err != nil {
			return err
		}
	}

	return nil
}

func fetchPodLogs(ctx context.Context, client *kube.Client, odigosNamespace string, pod v1.Pod) error {
	for _, container := range pod.Spec.Containers {
		fmt.Printf("Fetching logs for Pod: %s, Container: %s\n", pod.Name, container.Name)

		// Define the log file path for saving compressed logs
		logFilePath := filepath.Join(mainDir, logDir, pod.Name+"_"+container.Name+".log.gz")
		logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			return err
		}

		req := client.CoreV1().Pods(odigosNamespace).GetLogs(pod.Name, &v1.PodLogOptions{})
		logStream, err := req.Stream(ctx)
		if err != nil {
			return err
		}

		if err = saveLogsToGzipFileInBatches(logFile, logStream, logBufferSize); err != nil {
			return err
		}

		logStream.Close()
		logFile.Close()
	}

	return nil
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

func init() {
	rootCmd.AddCommand(diagnozeCmd)
}
