package cmd

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"github.com/odigos-io/odigos/cli/cmd/resources"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/client"
	"github.com/spf13/cobra"
	"io"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"sync"
	"time"
)

const (
	logBufferSize   = 1024 * 1024 // 1MB buffer size for reading logs in chunks
	LogsDir         = "Logs"
	CRDsDir         = "CRDs"
	CRDName         = "crdName"
	CRDGroup        = "crdGroup"
	actionGroupName = "actions.odigos.io"
	odigosGroupName = "odigos.io"
)

var (
	diagnoseDirs = []string{LogsDir, CRDsDir}
	CRDsList     = []map[string]string{
		{
			CRDName:  "addclusterinfos",
			CRDGroup: actionGroupName,
		},
		{
			CRDName:  "deleteattributes",
			CRDGroup: actionGroupName,
		},
		{
			CRDName:  "renameattributes",
			CRDGroup: actionGroupName,
		},
		{
			CRDName:  "probabilisticsamplers",
			CRDGroup: actionGroupName,
		},
		{
			CRDName:  "piimaskings",
			CRDGroup: actionGroupName,
		},
		{
			CRDName:  "latencysamplers",
			CRDGroup: actionGroupName,
		},
		{
			CRDName:  "errorsamplers",
			CRDGroup: actionGroupName,
		},
		{
			CRDName:  "instrumentedapplications",
			CRDGroup: odigosGroupName,
		},
		{
			CRDName:  "instrumentationconfigs",
			CRDGroup: odigosGroupName,
		},
	}
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
	mainTempDir, err := createAllDirs()
	if err != nil {
		return err
	}
	defer os.RemoveAll(mainTempDir)

	var wg sync.WaitGroup

	// Fetch Odigos components logs
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := fetchOdigosComponentsLogs(ctx, client, filepath.Join(mainTempDir, LogsDir)); err != nil {
			fmt.Printf("Error fetching Odigos components logs: %v\n", err)
		}
	}()

	// Fetch Odigos CRDs
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err = fetchOdigosCRDs(ctx, client, filepath.Join(mainTempDir, CRDsDir)); err != nil {
			fmt.Printf("Error fetching Odigos CRDs: %v\n", err)
		}
	}()

	wg.Wait()

	// Package the results into a tar.gz file
	if err = createTarGz(mainTempDir); err != nil {
		return err
	}

	return nil
}

func createAllDirs() (string, error) {
	mainTempDir, err := os.MkdirTemp("", "odigos-diagnose")
	if err != nil {
		return "", err
	}

	for _, dir := range diagnoseDirs {
		tempDir := filepath.Join(mainTempDir, dir)
		err = os.Mkdir(tempDir, os.ModePerm) // os.ModePerm gives full permissions (0777)
		if err != nil {
			return "", err
		}
	}

	return mainTempDir, nil
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

func fetchOdigosCRDs(ctx context.Context, kubeClient *kube.Client, crdDir string) error {
	var wg sync.WaitGroup

	for _, resourceData := range CRDsList {
		crdDataDirPath := filepath.Join(crdDir, resourceData[CRDName])
		err := os.Mkdir(crdDataDirPath, os.ModePerm) // os.ModePerm gives full permissions (0777)
		if err != nil {
			fmt.Printf("Error creating directory for CRD: %v, because: %v", resourceData, err)
			continue
		}

		wg.Add(1)

		go func() {
			defer wg.Done()
			err = fetchSingleResource(ctx, kubeClient, crdDataDirPath, resourceData)
			if err != nil {
				fmt.Printf("Error Getting CRDs of: %v, because: %v", resourceData[CRDName], err)
			}
		}()
	}

	wg.Wait()

	return nil
}

func fetchSingleResource(ctx context.Context, kubeClient *kube.Client, crdDataDirPath string, resourceData map[string]string) error {
	fmt.Printf("Fetching Resource: %s", resourceData[CRDName]+"\n")

	gvr := schema.GroupVersionResource{
		Group:    resourceData[CRDGroup], // The API group
		Version:  "v1alpha1",             // The version of the resourceData
		Resource: resourceData[CRDName],  // The resourceData type
	}

	err := client.ListWithPages(client.DefaultPageSize, kubeClient.Dynamic.Resource(gvr).List, ctx, metav1.ListOptions{}, func(crds *unstructured.UnstructuredList) error {
		for _, crd := range crds.Items {
			crdDirPath := filepath.Join(crdDataDirPath, crd.GetName()+".yaml.gz")
			crdFile, err := os.OpenFile(crdDirPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
			if err != nil {
				continue
			}

			gzipWriter := gzip.NewWriter(crdFile)

			crdYAML, err := yaml.Marshal(crd)
			if err != nil {
				continue
			}

			_, err = gzipWriter.Write(crdYAML)
			if err != nil {
				continue
			}
			if err = gzipWriter.Flush(); err != nil {
				continue
			}

			gzipWriter.Close()
			crdFile.Close()
		}
		return nil
	},
	)

	if err != nil {
		return err
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
