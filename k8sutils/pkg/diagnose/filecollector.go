package diagnose

import (
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"sync"

	"k8s.io/klog/v2"
)

// FileCollector writes data to the local filesystem.
// This is used by the CLI for collecting data to a temporary directory
// before creating the final tar.gz archive.
type FileCollector struct {
	mu    sync.Mutex
	stats CollectorStats
}

// NewFileCollector creates a new FileCollector
func NewFileCollector() *FileCollector {
	return &FileCollector{}
}

// AddFile writes a file to the filesystem
func (c *FileCollector) AddFile(dir, filename string, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.stats.TotalSize += int64(len(data))
	c.stats.FileCount++

	// Ensure directory exists
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}

	filePath := filepath.Join(dir, filename)
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o666)
	if err != nil {
		return err
	}
	//nolint:errcheck // this close is deferred to the end of the function
	defer file.Close()

	if _, err := file.Write(data); err != nil {
		return err
	}

	return file.Sync()
}

// AddFileGzipped writes a gzip-compressed file to the filesystem
func (c *FileCollector) AddFileGzipped(dir, filename string, reader io.Reader) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Ensure directory exists
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}

	filePath := filepath.Join(dir, filename)
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o666)
	if err != nil {
		return err
	}
	//nolint:errcheck // this close is deferred to the end of the function
	defer file.Close()

	// Create a gzip writer
	gzWriter := gzip.NewWriter(file)
	//nolint:errcheck // this close is deferred to the end of the function
	defer gzWriter.Close()

	// Read and compress in chunks
	buffer := make([]byte, logBufferSize)
	var totalWritten int64
	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			written, writeErr := gzWriter.Write(buffer[:n])
			if writeErr != nil {
				klog.V(1).ErrorS(writeErr, "Failed to write to gzip", "filePath", filePath)
				return writeErr
			}
			totalWritten += int64(written)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	c.stats.TotalSize += totalWritten
	c.stats.FileCount++

	return nil
}

// GetStats returns collection statistics
func (c *FileCollector) GetStats() CollectorStats {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.stats
}
