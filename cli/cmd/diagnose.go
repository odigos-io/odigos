package cmd

import (
	"archive/tar"
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
	"sync"
	"time"
)

const (
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
	mainTempDir, logTempDir, err := createAllDirs()
	if err != nil {
		return err
	}
	defer os.RemoveAll(mainTempDir)

	if err := fetchOdigosComponentsLogs(ctx, client, logTempDir); err != nil {
		return err
	}

	if err := createTarGz(mainTempDir); err != nil {
		return err
	}
	return nil
}

func createAllDirs() (string, string, error) {
	mainTempDir, err := os.MkdirTemp("", "parent-")
	if err != nil {
		return "", "", err
	}

	logTempDir := filepath.Join(mainTempDir, "Logs")
	err = os.Mkdir(logTempDir, os.ModePerm) // os.ModePerm gives full permissions (0777)
	if err != nil {
		return "", "", err
	}

	return mainTempDir, logTempDir, nil
}

func fetchOdigosComponentsLogs(ctx context.Context, client *kube.Client, logDir string) error {
	odigosNamespace, err := resources.GetOdigosNamespace(client, ctx)
	if err != nil {
		return err
	}

	pods, err := client.CoreV1().Pods(odigosNamespace).List(context.TODO(), metav1.ListOptions{})
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
		logPrefix := fmt.Sprintf("Fetching logs for Pod: %s, Container: %s, Node: %s", pod.Name, container.Name, pod.Spec.NodeName)
		fmt.Printf(logPrefix + "\n")

		// Define the log file path for saving compressed logs
		logFilePath := filepath.Join(logDir, pod.Name+"_"+container.Name+"_"+pod.Spec.NodeName+".log.gz")
		logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			fmt.Printf(logPrefix+" - Failed - Error creating log file: %v\n", err)
			return
		}

		req := client.CoreV1().Pods(odigosNamespace).GetLogs(pod.Name, &v1.PodLogOptions{})
		logStream, err := req.Stream(ctx)
		if err != nil {
			fmt.Printf(logPrefix+" - Failed - Error creating log stream: %v\n", err)
			return
		}

		if err = saveLogsToGzipFileInBatches(logFile, logStream, logBufferSize); err != nil {
			fmt.Printf(logPrefix+" - Failed - Error saving logs to file: %v\n", err)
			return
		}

		logStream.Close()
		logFile.Close()
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

func createTarGz(sourceDir string) error {
	timestamp := time.Now().Format("02012006150405")
	tarGzFileName := fmt.Sprintf("odigos_debug_%s.tar.gz", timestamp)

	tarGzFile, err := os.Create(tarGzFileName)
	if err != nil {
		return err
	}
	defer tarGzFile.Close()

	gzipWriter := gzip.NewWriter(tarGzFile)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	err = filepath.Walk(sourceDir, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(fi, fi.Name())
		if err != nil {
			return err
		}

		header.Name, err = filepath.Rel(sourceDir, file)
		if err != nil {
			return err
		}

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		if !fi.Mode().IsRegular() {
			return nil
		}

		fileContent, err := os.Open(file)
		if err != nil {
			return err
		}
		defer fileContent.Close()

		if _, err := io.Copy(tarWriter, fileContent); err != nil {
			return err
		}

		return nil
	})

	return err
}

func init() {
	rootCmd.AddCommand(diagnozeCmd)
}
