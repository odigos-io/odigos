package cmd

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/odigos-io/odigos/cli/cmd/resources"
	cmdcontext "github.com/odigos-io/odigos/cli/pkg/cmd_context"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/diagnose"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

// Diagnose command flags
var (
	diagnoseIncludeLogs              bool
	diagnoseIncludeCRDs              bool
	diagnoseIncludeProfiles          bool
	diagnoseIncludeMetrics           bool
	diagnoseIncludeConfigMaps        bool
	diagnoseAllWorkloadNamespaces    bool
	diagnoseSourceWorkloadNamespaces []string
)

var diagnoseCmd = &cobra.Command{
	Use:   "diagnose",
	Short: "Diagnose Client Cluster",
	Long: `Retrieves Logs of all Odigos components in the odigos-system namespace and CRDs of Actions, instrumentation resources and more.
The results will be saved in a compressed file for further troubleshooting.
The file will be saved in this format: odigos_debug_ddmmyyyyhhmmss.tar.gz`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		client := cmdcontext.KubeClientFromContextOrExit(ctx)

		err := startDiagnose(ctx, client)
		if err != nil {
			klog.V(1).ErrorS(err, "Failed to start diagnose")
		}
	},
}

func startDiagnose(ctx context.Context, client *kube.Client) error {
	// Get the Odigos namespace
	odigosNamespace, err := resources.GetOdigosNamespace(client, ctx)
	if err != nil {
		return fmt.Errorf("failed to get odigos namespace: %w", err)
	}

	fmt.Printf("Starting diagnose for Odigos in namespace: %s\n", odigosNamespace)

	// Create the root directory name
	rootDir := diagnose.GetRootDir()

	// Create temporary directory for collecting files
	mainTempDir, err := os.MkdirTemp("", "odigos-diagnose")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(mainTempDir)

	// The builder will write to mainTempDir/rootDir
	builderRootDir := filepath.Join(mainTempDir, rootDir)
	if err := os.MkdirAll(builderRootDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create builder root directory: %w", err)
	}

	// Create the diagnose builder
	builder := diagnose.NewBuilder()

	// Determine if source workloads should be collected:
	// - If --source-workload-namespaces is provided, collect from those namespaces
	// - If --all-workload-namespaces is provided, collect from all namespaces
	includeSourceWorkloads := diagnoseAllWorkloadNamespaces || len(diagnoseSourceWorkloadNamespaces) > 0

	// Configure options from flags
	opts := diagnose.Options{
		OdigosNamespace:          odigosNamespace,
		IncludeLogs:              diagnoseIncludeLogs,
		IncludeCRDs:              diagnoseIncludeCRDs,
		IncludeProfiles:          diagnoseIncludeProfiles,
		IncludeMetrics:           diagnoseIncludeMetrics,
		IncludeConfigMaps:        diagnoseIncludeConfigMaps,
		IncludeSourceWorkloads:   includeSourceWorkloads,
		SourceWorkloadNamespaces: diagnoseSourceWorkloadNamespaces,
	}

	// Run the diagnose collection
	if err := diagnose.RunDiagnose(ctx, client.Clientset, client.Dynamic, client.Clientset.Discovery(), client.OdigosClient, builder, builderRootDir, opts); err != nil {
		klog.V(1).ErrorS(err, "Some diagnose operations had errors")
	}

	// Package the results into a tar.gz file
	tarGzFileName, err := createTarGz(mainTempDir)
	if err != nil {
		return fmt.Errorf("failed to create tar.gz file: %w", err)
	}

	fmt.Printf("Diagnose completed successfully, the results are saved in the file: %s\n", tarGzFileName)

	return nil
}

func createTarGz(sourceDir string) (string, error) {
	timestamp := time.Now().Format("02012006150405")
	tarGzFileName := fmt.Sprintf("odigos_debug_%s.tar.gz", timestamp)

	tarGzFile, err := os.Create(tarGzFileName)
	if err != nil {
		return "", err
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

	return tarGzFileName, err
}

func init() {
	rootCmd.AddCommand(diagnoseCmd)

	// Add flags with defaults matching DefaultOptions()
	diagnoseCmd.Flags().BoolVar(&diagnoseIncludeLogs, "odigos-logs", true, "Include Odigos component pod logs in the diagnose output")
	diagnoseCmd.Flags().BoolVar(&diagnoseIncludeCRDs, "odigos-crds", true, "Include Odigos CRDs in the diagnose output")
	diagnoseCmd.Flags().BoolVar(&diagnoseIncludeProfiles, "odigos-profiles", true, "Include Odigos pprof profiles in the diagnose output")
	diagnoseCmd.Flags().BoolVar(&diagnoseIncludeMetrics, "odigos-metrics", true, "Include Odigos Prometheus metrics in the diagnose output")
	diagnoseCmd.Flags().BoolVar(&diagnoseIncludeConfigMaps, "odigos-configmaps", true, "Include Odigos ConfigMaps in the diagnose output")
	diagnoseCmd.Flags().StringSliceVar(&diagnoseSourceWorkloadNamespaces, "source-workload-namespaces", []string{}, "Collect instrumented source workloads from specific namespaces (comma-separated)")
	diagnoseCmd.Flags().BoolVar(&diagnoseAllWorkloadNamespaces, "all-workload-namespaces", false, "Collect instrumented source workloads from all namespaces")
}
