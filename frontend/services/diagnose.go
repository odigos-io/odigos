package services

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/diagnose"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
)

// Single diagnose output - replaced on each new request
var (
	diagnoseTempDir  string
	diagnoseLock     sync.RWMutex
)

// DiagnoseGraphQL is the GraphQL resolver for diagnose
func DiagnoseGraphQL(
	ctx context.Context,
	input *model.DiagnoseInput,
	dryRun *bool,
) (*model.DiagnoseResponse, error) {
	ns := env.GetCurrentNamespace()

	// Parse options from input
	includeProfiles := true
	includeMetrics := true
	includeSourceWorkloads := false
	var sourceWorkloadNamespaces []string

	if input != nil {
		if input.IncludeProfiles != nil {
			includeProfiles = *input.IncludeProfiles
		}
		if input.IncludeMetrics != nil {
			includeMetrics = *input.IncludeMetrics
		}
		if input.IncludeSourceWorkloads != nil {
			includeSourceWorkloads = *input.IncludeSourceWorkloads
		}
		sourceWorkloadNamespaces = input.SourceWorkloadNamespaces
	}

	// Configure options matching the CLI diagnose behavior
	opts := diagnose.DefaultOptions()
	opts.OdigosNamespace = ns
	opts.IncludeProfiles = includeProfiles
	opts.IncludeMetrics = includeMetrics
	opts.IncludeSourceWorkloads = includeSourceWorkloads
	opts.SourceWorkloadNamespaces = sourceWorkloadNamespaces

	// Generate root directory name
	rootDir := diagnose.GetRootDir()

	isDryRun := dryRun != nil && *dryRun

	if isDryRun {
		// Create dry-run collector to estimate size
		collector := diagnose.NewDryRunCollector()

		if err := diagnose.RunDiagnose(
			ctx,
			kube.DefaultClient,
			kube.DefaultClient.DynamicClient,
			kube.DefaultClient.Discovery(),
			kube.DefaultClient.OdigosClient,
			collector,
			rootDir,
			opts,
		); err != nil {
			return nil, fmt.Errorf("failed to run diagnose: %w", err)
		}

		stats := collector.GetStats()
		return &model.DiagnoseResponse{
			Stats: &model.DiagnoseStats{
				FileCount:      int(stats.FileCount),
				TotalSizeBytes: int(stats.TotalSize),
				TotalSizeHuman: diagnose.FormatBytes(stats.TotalSize),
			},
			IncludeProfiles:          includeProfiles,
			IncludeMetrics:           includeMetrics,
			IncludeSourceWorkloads:   includeSourceWorkloads,
			SourceWorkloadNamespaces: sourceWorkloadNamespaces,
		}, nil
	}

	// Clean up any existing diagnose temp directory
	diagnoseLock.Lock()
	if diagnoseTempDir != "" {
		os.RemoveAll(diagnoseTempDir)
	}

	// Create temporary directory for collecting files
	mainTempDir, err := os.MkdirTemp("", "odigos-diagnose")
	if err != nil {
		diagnoseLock.Unlock()
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	diagnoseTempDir = mainTempDir
	diagnoseLock.Unlock()

	// The collector will write to mainTempDir/rootDir
	collectorRootDir := filepath.Join(mainTempDir, rootDir)
	if err := os.MkdirAll(collectorRootDir, os.ModePerm); err != nil {
		os.RemoveAll(mainTempDir)
		return nil, fmt.Errorf("failed to create collector root directory: %w", err)
	}

	// Create file collector (writes to temp directory)
	collector := diagnose.NewFileCollector()

	// Run the diagnose collection
	if err := diagnose.RunDiagnose(
		ctx,
		kube.DefaultClient,
		kube.DefaultClient.DynamicClient,
		kube.DefaultClient.Discovery(),
		kube.DefaultClient.OdigosClient,
		collector,
		collectorRootDir,
		opts,
	); err != nil {
		os.RemoveAll(mainTempDir)
		return nil, fmt.Errorf("failed to run diagnose: %w", err)
	}

	// Get file stats
	fileCount, totalSize := countFilesAndSize(mainTempDir)

	downloadURL := "/diagnose/download"

	return &model.DiagnoseResponse{
		Stats: &model.DiagnoseStats{
			FileCount:      fileCount,
			TotalSizeBytes: int(totalSize),
			TotalSizeHuman: diagnose.FormatBytes(totalSize),
		},
		IncludeProfiles:          includeProfiles,
		IncludeMetrics:           includeMetrics,
		IncludeSourceWorkloads:   includeSourceWorkloads,
		SourceWorkloadNamespaces: sourceWorkloadNamespaces,
		DownloadURL:              &downloadURL,
	}, nil
}

// DiagnoseDownload handles the download of the current diagnose output
func DiagnoseDownload(c *gin.Context) {
	diagnoseLock.RLock()
	tempDir := diagnoseTempDir
	diagnoseLock.RUnlock()

	if tempDir == "" {
		c.JSON(404, gin.H{"error": "No diagnose output available. Run diagnose query first."})
		return
	}

	// Get the root dir name from temp directory
	entries, err := os.ReadDir(tempDir)
	if err != nil || len(entries) == 0 {
		c.JSON(500, gin.H{"error": "Failed to read diagnose output"})
		return
	}
	rootDirName := entries[0].Name()

	// Set headers for file download
	filename := fmt.Sprintf("%s.tar.gz", rootDirName)
	c.Header("Content-Type", "application/gzip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	// Stream tar.gz directly to response
	if err := writeTarGzToWriter(tempDir, c.Writer); err != nil {
		return
	}

	c.Status(200)
}

// countFilesAndSize counts files and total size in a directory
func countFilesAndSize(dir string) (int, int64) {
	var count int
	var size int64
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.Mode().IsRegular() {
			count++
			size += info.Size()
		}
		return nil
	})
	return count, size
}

// writeTarGzToWriter creates a tar.gz archive from sourceDir and writes it to w
func writeTarGzToWriter(sourceDir string, w io.Writer) error {
	gzipWriter := gzip.NewWriter(w)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	return filepath.Walk(sourceDir, func(file string, fi os.FileInfo, err error) error {
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
}
