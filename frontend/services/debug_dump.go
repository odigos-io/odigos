package services

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/diagnose"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
)

// DebugDump generates a tar.gz file containing logs and YAML manifests
// for all Odigos components running in the odigos system namespace.
// Query params:
//   - includeSourceWorkloads: if "true", also include workload and pod YAMLs for each instrumented Source
//   - sourceWorkloadNamespaces: comma-separated list of namespaces to collect source workloads from (only used when includeSourceWorkloads=true, defaults to all namespaces)
//   - dryRun: if "true", returns JSON with estimated size instead of generating the file
func DebugDump(c *gin.Context) {
	ctx := c.Request.Context()
	ns := env.GetCurrentNamespace()
	includeSourceWorkloads := c.Query("includeSourceWorkloads") == "true"
	dryRun := c.Query("dryRun") == "true"

	// Parse sourceWorkloadNamespaces - comma-separated list of namespaces to filter by
	var sourceWorkloadNamespaces []string
	if nsParam := c.Query("sourceWorkloadNamespaces"); nsParam != "" {
		for _, n := range strings.Split(nsParam, ",") {
			if trimmed := strings.TrimSpace(n); trimmed != "" {
				sourceWorkloadNamespaces = append(sourceWorkloadNamespaces, trimmed)
			}
		}
	}

	// Configure options matching the CLI diagnose behavior
	opts := diagnose.DefaultOptions()
	opts.OdigosNamespace = ns
	opts.IncludeSourceWorkloads = includeSourceWorkloads
	opts.SourceWorkloadNamespaces = sourceWorkloadNamespaces

	// Generate root directory name
	rootDir := diagnose.GetRootDir()

	if dryRun {
		// Create dry-run collector to estimate size
		collector := diagnose.NewDryRunCollector()

		if err := diagnose.RunDiagnose(ctx, kube.DefaultClient, kube.DefaultClient.DynamicClient, kube.DefaultClient.Discovery(), collector, rootDir, opts); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to run diagnose: %v", err)})
			return
		}

		stats := collector.GetStats()
		c.JSON(http.StatusOK, gin.H{
			"dryRun":                   true,
			"includeSourceWorkloads":   includeSourceWorkloads,
			"sourceWorkloadNamespaces": sourceWorkloadNamespaces,
			"fileCount":                stats.FileCount,
			"totalSizeBytes":           stats.TotalSize,
			"totalSizeHuman":           diagnose.FormatBytes(stats.TotalSize),
		})
		return
	}

	// Create temporary directory for collecting files (same approach as CLI)
	mainTempDir, err := os.MkdirTemp("", "odigos-diagnose")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to create temp directory: %v", err)})
		return
	}
	defer os.RemoveAll(mainTempDir)

	// The collector will write to mainTempDir/rootDir
	collectorRootDir := filepath.Join(mainTempDir, rootDir)
	if err := os.MkdirAll(collectorRootDir, os.ModePerm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to create collector root directory: %v", err)})
		return
	}

	// Create file collector (writes to temp directory)
	collector := diagnose.NewFileCollector()

	// Run the diagnose collection
	if err := diagnose.RunDiagnose(ctx, kube.DefaultClient, kube.DefaultClient.DynamicClient, kube.DefaultClient.Discovery(), collector, collectorRootDir, opts); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to run diagnose: %v", err)})
		return
	}

	// Set headers for file download
	filename := fmt.Sprintf("%s.tar.gz", rootDir)
	c.Header("Content-Type", "application/gzip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	// Stream tar.gz directly to response
	if err := writeTarGzToWriter(mainTempDir, c.Writer); err != nil {
		// If we haven't started writing yet, we can send an error
		// Otherwise the client will receive a truncated archive
		return
	}

	c.Status(http.StatusOK)
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
