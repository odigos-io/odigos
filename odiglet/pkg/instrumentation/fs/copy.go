package fs

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/odigos-io/odigos/odiglet/pkg/log"
)

const (
	// Optimize worker count for I/O bound operations
	maxWorkers = 8

	// Increased buffer size for better I/O performance
	bufferSize = 64 * 1024
)

// CopyJob represents a file copy operation
type CopyJob struct {
	src string
	dst string
}

// CopyWorkerPool manages parallel file copying operations
type CopyWorkerPool struct {
	workers int
	jobs    chan CopyJob
	wg      sync.WaitGroup
	errors  chan error
	bufPool sync.Pool
}

// getNumberOfWorkers returns the number of workers to use for copying files.
func getNumberOfWorkers() int {
	numCPU := runtime.NumCPU()
	// For I/O bound operations, we can use more workers than CPU cores
	workers := numCPU * 2
	if workers > maxWorkers {
		workers = maxWorkers
	}
	if workers < 2 {
		workers = 2
	}
	return workers
}

func copyDirectories(srcDir string, destDir string, filesToKeep map[string]struct{}) error {
	start := time.Now()

	hostContainEbpfDir := HostContainsEbpfDir(destDir)

	// If the host directory NOT contains ebpf directories we copy all files
	CopyCFiles := !hostContainEbpfDir
	log.Logger.V(0).Info("Copying instrumentation files to host", "srcDir", srcDir, "destDir", destDir, "CopyCFiles", CopyCFiles)

	// Create the destination directory if it doesn't exist
	err := os.MkdirAll(destDir, os.ModePerm)
	if err != nil {
		return err
	}

	// Collect copy jobs efficiently in a single directory traversal
	copyJobs, err := collectCopyJobs(srcDir, destDir, CopyCFiles, filesToKeep)
	if err != nil {
		return err
	}

	if len(copyJobs) == 0 {
		log.Logger.V(0).Info("No files to copy")
		return nil
	}

	// Process copy jobs in parallel
	err = processCopyJobsInParallel(copyJobs)
	if err != nil {
		return err
	}

	log.Logger.V(0).Info("Finished copying instrumentation files to host", "duration", time.Since(start), "fileCount", len(copyJobs))
	return nil
}

// collectCopyJobs efficiently collects all copy operations in a single directory traversal
func collectCopyJobs(srcDir, destDir string, copyCFiles bool, filesToKeep map[string]struct{}) ([]CopyJob, error) {
	var jobs []CopyJob
	srcDirLen := len(srcDir) + 1 // +1 for the path separator

	err := filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Logger.Error(err, "Error accessing path during copy scan", "path", path)
			return nil // Continue processing other files
		}

		if d.IsDir() {
			return nil
		}

		// Apply filtering logic efficiently
		if !copyCFiles {
			// Convert container path to host path for comparison
			hostPath := "/var/odigos/" + path[srcDirLen:]
			if _, found := filesToKeep[hostPath]; found {
				log.Logger.V(1).Info("Skipping copying file", "file", path)
				return nil
			}
		}

		// Create destination path
		destFile := filepath.Join(destDir, path[srcDirLen:])
		jobs = append(jobs, CopyJob{src: path, dst: destFile})

		return nil
	})

	return jobs, err
}

// processCopyJobsInParallel processes copy jobs using optimized worker pool
func processCopyJobsInParallel(jobs []CopyJob) error {
	if len(jobs) == 0 {
		return nil
	}

	numWorkers := getNumberOfWorkers()
	if numWorkers > len(jobs) {
		numWorkers = len(jobs)
	}

	pool := &CopyWorkerPool{
		workers: numWorkers,
		jobs:    make(chan CopyJob, len(jobs)),
		errors:  make(chan error, len(jobs)),
		bufPool: sync.Pool{
			New: func() interface{} {
				return make([]byte, bufferSize)
			},
		},
	}

	// Start workers
	for i := 0; i < pool.workers; i++ {
		pool.wg.Add(1)
		go pool.worker()
	}

	// Send jobs
	go func() {
		for _, job := range jobs {
			pool.jobs <- job
		}
		close(pool.jobs)
	}()

	// Wait for completion
	go func() {
		pool.wg.Wait()
		close(pool.errors)
	}()

	// Collect any errors
	var firstError error
	errorCount := 0
	for err := range pool.errors {
		if err != nil {
			errorCount++
			if firstError == nil {
				firstError = err
			}
			log.Logger.Error(err, "Error during file copy")
		}
	}

	if errorCount > 0 && firstError != nil {
		log.Logger.Error(firstError, "Copy operation completed with errors", "errorCount", errorCount, "totalJobs", len(jobs))
		return firstError
	}

	return nil
}

// worker processes copy jobs with optimized buffer management
func (pool *CopyWorkerPool) worker() {
	defer pool.wg.Done()

	for job := range pool.jobs {
		err := pool.copyFileOptimized(job.src, job.dst)
		pool.errors <- err
	}
}

// copyFileOptimized uses buffer pool and optimized I/O operations
func (pool *CopyWorkerPool) copyFileOptimized(src, dst string) error {
	// Get buffer from pool
	buf := pool.bufPool.Get().([]byte)
	defer pool.bufPool.Put(buf)

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Create destination directories if needed
	err = os.MkdirAll(filepath.Dir(dst), os.ModePerm)
	if err != nil {
		return err
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// Optimized copy loop with larger buffer
	for {
		n, readErr := srcFile.Read(buf)
		if n > 0 {
			if _, writeErr := dstFile.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}

	return nil
}

func createDotnetDeprecatedDirectories(destDir string) error {

	var err error

	arch := getArch()
	dotnetSoFile := "OpenTelemetry.AutoInstrumentation.Native.so"
	glibcDir := filepath.Join(destDir, "linux-glibc")
	muslDir := filepath.Join(destDir, "linux-musl")
	glibcDirWithArch := filepath.Join(destDir, "linux-glibc-"+arch)
	muslDirWithArch := filepath.Join(destDir, "linux-musl-"+arch)

	err = os.MkdirAll(glibcDirWithArch, os.ModePerm)
	if err != nil {
		return err
	}
	err = os.MkdirAll(muslDirWithArch, os.ModePerm)
	if err != nil {
		return err
	}

	err = os.Symlink(filepath.Join(glibcDir, dotnetSoFile), filepath.Join(glibcDirWithArch, dotnetSoFile))
	if err != nil {
		return err
	}
	err = os.Symlink(filepath.Join(muslDir, dotnetSoFile), filepath.Join(muslDirWithArch, dotnetSoFile))
	if err != nil {
		return err
	}

	return nil
}

// Deprecated: Use processCopyJobsInParallel instead
func worker(fileChan <-chan string, sourceDir, destDir string, wg *sync.WaitGroup) {
	defer wg.Done()

	// Allocate a buffer once per goroutine.
	buf := make([]byte, bufferSize)
	for file := range fileChan {
		destFile := filepath.Join(destDir, file[len(sourceDir)+1:])
		err := copyFile(file, destFile, buf)
		if err != nil {
			log.Logger.Error(err, "error copying file", "file", file)
		}
	}
}

// Deprecated: Use collectCopyJobs instead
func getFiles(dir string, CopyCFiles bool, filesToKeep map[string]struct{}) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			if !CopyCFiles {
				if _, found := filesToKeep[strings.Replace(path, "/instrumentations/", "/var/odigos/", 1)]; found {
					log.Logger.V(0).Info("Skipping copying file", "file", path)
					return nil
				}
			}

			files = append(files, path)
		}
		return nil
	})

	return files, err
}

// Deprecated: Use copyFileOptimized instead
func copyFile(src, dst string, buf []byte) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Create destination file and directories if needed
	err = os.MkdirAll(filepath.Dir(dst), os.ModePerm)
	if err != nil {
		return err
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// Copy using the provided buffer.
	for {
		n, err := srcFile.Read(buf)
		if n > 0 {
			if _, err := dstFile.Write(buf[:n]); err != nil {
				return err
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	return nil
}

// HostContainsEbpfDir efficiently checks if directory contains ebpf subdirectories
func HostContainsEbpfDir(dir string) bool {
	// Use a more efficient approach with early termination
	found := false
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || found {
			return err
		}
		if d.IsDir() && strings.Contains(d.Name(), "ebpf") {
			found = true
			return filepath.SkipAll // Early termination
		}
		return nil
	})
	return found
}

func getArch() string {
	if runtime.GOARCH == "arm64" {
		return "arm64"
	}

	return "x64"
}
