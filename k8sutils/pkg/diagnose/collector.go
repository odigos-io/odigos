package diagnose

import (
	"fmt"
	"io"
	"path"
	"sync"
	"time"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/odigos-io/odigos/api/k8sconsts"
)

const (
	logBufferSize    = 1024 * 1024 // 1MB buffer size for reading logs in chunks
	odigosGroupName  = "odigos.io"
	actionsGroupName = "actions.odigos.io"
)

// DiagnoseClient is an interface that abstracts the Kubernetes client operations
// needed for the diagnose functionality. This allows the package to work with
// both CLI and frontend clients.
type DiagnoseClient interface {
	kubernetes.Interface
	GetDynamicClient() dynamic.Interface
	GetDiscoveryClient() discovery.DiscoveryInterface
}

// Collector is an interface for collecting diagnose data.
// It can be implemented for different output targets (file system, HTTP stream, etc.)
type Collector interface {
	// AddFile adds a file to the collection
	AddFile(dir, filename string, data []byte) error
	// AddFileGzipped adds a gzip-compressed file to the collection
	AddFileGzipped(dir, filename string, reader io.Reader) error
	// GetStats returns collection statistics
	GetStats() CollectorStats
}

// CollectorStats holds statistics about the collection
type CollectorStats struct {
	TotalSize int64
	FileCount int
}

// Options configures what data to collect during diagnose
type Options struct {
	// OdigosNamespace is the namespace where Odigos is installed
	OdigosNamespace string
	// IncludeSourceWorkloads includes workload YAMLs for instrumented sources (not odigos components)
	IncludeSourceWorkloads bool
	// SourceWorkloadNamespaces filters which namespaces to collect source workloads from (empty means all)
	SourceWorkloadNamespaces []string
	// IncludeProfiles collects pprof profiles
	IncludeProfiles bool
	// IncludeMetrics collects Prometheus metrics
	IncludeMetrics bool
	// IncludeLogs collects pod logs (under component folders)
	IncludeLogs bool
	// IncludeCRDs collects Odigos CRDs
	IncludeCRDs bool
	// IncludeConfigMaps collects ConfigMaps
	IncludeConfigMaps bool
}

// DefaultOptions returns the default diagnose options matching the CLI behavior
func DefaultOptions() Options {
	return Options{
		IncludeProfiles:   true,
		IncludeMetrics:    true,
		IncludeLogs:       true,
		IncludeCRDs:       true,
		IncludeConfigMaps: true,
	}
}

// DryRunCollector only tracks stats without writing any data.
// This is used for estimating the size before actual collection.
type DryRunCollector struct {
	mu    sync.Mutex
	stats CollectorStats
}

// NewDryRunCollector creates a new DryRunCollector
func NewDryRunCollector() *DryRunCollector {
	return &DryRunCollector{}
}

// AddFile tracks the file size without writing
func (c *DryRunCollector) AddFile(dir, filename string, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stats.TotalSize += int64(len(data))
	c.stats.FileCount++
	return nil
}

// AddFileGzipped tracks the estimated compressed size without writing
func (c *DryRunCollector) AddFileGzipped(dir, filename string, reader io.Reader) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Read to count bytes (we can't know compressed size without actually compressing)
	buffer := make([]byte, logBufferSize)
	var totalRead int64
	for {
		n, err := reader.Read(buffer)
		totalRead += int64(n)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	// Estimate compressed size as ~30% of original (rough gzip estimate for text)
	c.stats.TotalSize += totalRead * 30 / 100
	c.stats.FileCount++
	return nil
}

// GetStats returns collection statistics
func (c *DryRunCollector) GetStats() CollectorStats {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.stats
}

// GetRootDir returns the root directory name for the diagnose output
func GetRootDir() string {
	timestamp := time.Now().Format("02012006150405")
	return fmt.Sprintf("odigos_debug_%s", timestamp)
}

// GetCRDsDir returns the CRDs directory path for a specific namespace
func GetCRDsDir(rootDir, namespace string) string {
	return path.Join(rootDir, namespace)
}

// GetProfileDir returns the profile directory path under the odigos namespace
func GetProfileDir(rootDir, odigosNamespace string) string {
	return path.Join(rootDir, odigosNamespace, k8sconsts.ProfileDir)
}

// GetMetricsDir returns the metrics directory path under the odigos namespace
func GetMetricsDir(rootDir, odigosNamespace string) string {
	return path.Join(rootDir, odigosNamespace, k8sconsts.MetricsDir)
}

// GetConfigMapsDir returns the configmaps directory path under the odigos namespace
func GetConfigMapsDir(rootDir, odigosNamespace string) string {
	return path.Join(rootDir, odigosNamespace, "ConfigMaps")
}

// GetWorkloadDir returns the workload directory path under a specific namespace
func GetWorkloadDir(rootDir, namespace, workloadDirName string) string {
	return path.Join(rootDir, namespace, workloadDirName)
}

// FormatBytes converts bytes to a human-readable string
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
