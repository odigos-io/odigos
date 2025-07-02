package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/odigos-io/odigos/odiglet/pkg/log"
)

// RemovalJob represents a file or directory to be removed
type RemovalJob struct {
	path  string
	isDir bool
}

// RemovalWorkerPool manages parallel file removal operations
type RemovalWorkerPool struct {
	workers int
	jobs    chan RemovalJob
	wg      sync.WaitGroup
	errors  chan error
}

func removeFilesInDir(hostDir string, filesToKeep map[string]struct{}) error {
	log.Logger.V(0).Info("Removing files in the host directory", "hostDir", hostDir)

	// Pre-compute protected directories for faster lookup
	protectedDirs := buildProtectedDirsSet(hostDir, filesToKeep)
	
	// Collect all removal jobs first to batch operations
	removalJobs, err := collectRemovalJobs(hostDir, filesToKeep, protectedDirs)
	if err != nil {
		return err
	}

	if len(removalJobs) == 0 {
		log.Logger.V(0).Info("No files to remove")
		return nil
	}

	// Process removals in parallel batches
	return processRemovalJobsInParallel(removalJobs)
}

// buildProtectedDirsSet pre-computes all protected directories for O(1) lookup
func buildProtectedDirsSet(hostDir string, filesToKeep map[string]struct{}) map[string]struct{} {
	protectedDirs := make(map[string]struct{})
	protectedDirs[hostDir] = struct{}{} // Always protect the main directory
	
	for file := range filesToKeep {
		dir := filepath.Dir(file)
		// Add all parent directories up to hostDir
		for dir != hostDir && dir != "." && dir != "/" {
			protectedDirs[dir] = struct{}{}
			dir = filepath.Dir(dir)
		}
	}
	
	return protectedDirs
}

// collectRemovalJobs efficiently traverses the directory tree once and collects all removal jobs
func collectRemovalJobs(hostDir string, filesToKeep map[string]struct{}, protectedDirs map[string]struct{}) ([]RemovalJob, error) {
	var jobs []RemovalJob
	var mu sync.Mutex
	
	// Use a more efficient directory traversal
	err := filepath.WalkDir(hostDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// Log error but continue processing other files
			log.Logger.Error(err, "Error accessing path during removal scan", "path", path)
			return nil
		}

		// Skip the root directory
		if path == hostDir {
			return nil
		}

		// Quick checks to avoid expensive operations
		isDir := d.IsDir()
		
		// For files: check if it should be kept
		if !isDir {
			// Fast path: check keep map first
			if _, shouldKeep := filesToKeep[path]; shouldKeep {
				log.Logger.V(1).Info("Skipping protected file", "file", path)
				return nil
			}

			// Fast path: check for hash version suffix without string operations in critical path
			if containsHashVersion(path) {
				log.Logger.V(1).Info("Skipping file with versioning suffix", "file", path)
				return nil
			}

			// This file should be removed
			mu.Lock()
			jobs = append(jobs, RemovalJob{path: path, isDir: false})
			mu.Unlock()
			return nil
		}

		// For directories: check if it's protected
		if _, isProtected := protectedDirs[path]; isProtected {
			log.Logger.V(1).Info("Skipping protected directory", "directory", path)
			return nil
		}

		// Check if directory is empty or only contains files that will be removed
		// This optimization helps us remove entire directory trees efficiently
		if canRemoveDirectory(path, filesToKeep, protectedDirs) {
			mu.Lock()
			jobs = append(jobs, RemovalJob{path: path, isDir: true})
			mu.Unlock()
			// Skip processing contents since we're removing the whole directory
			return filepath.SkipDir
		}

		return nil
	})

	return jobs, err
}

// containsHashVersion is optimized to avoid creating substrings
func containsHashVersion(path string) bool {
	// Look for the pattern "_hash_version-" more efficiently
	return strings.Contains(path, "_hash_version-")
}

// canRemoveDirectory checks if a directory can be safely removed entirely
func canRemoveDirectory(dirPath string, filesToKeep map[string]struct{}, protectedDirs map[string]struct{}) bool {
	// If it's in protected dirs, don't remove
	if _, isProtected := protectedDirs[dirPath]; isProtected {
		return false
	}

	// Check if any files in filesToKeep are under this directory
	dirPathWithSep := dirPath + string(os.PathSeparator)
	for keepFile := range filesToKeep {
		if strings.HasPrefix(keepFile, dirPathWithSep) {
			return false
		}
	}

	return true
}

// processRemovalJobsInParallel processes removal jobs using worker pool
func processRemovalJobsInParallel(jobs []RemovalJob) error {
	if len(jobs) == 0 {
		return nil
	}

	// Determine optimal number of workers
	numWorkers := runtime.NumCPU()
	if numWorkers > len(jobs) {
		numWorkers = len(jobs)
	}
	
	// Limit workers for I/O bound operations
	if numWorkers > 8 {
		numWorkers = 8
	}

	pool := &RemovalWorkerPool{
		workers: numWorkers,
		jobs:    make(chan RemovalJob, len(jobs)),
		errors:  make(chan error, len(jobs)),
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
			log.Logger.Error(err, "Error during file removal")
		}
	}

	if firstError != nil {
		return fmt.Errorf("encountered %d errors during removal, first error: %w", errorCount, firstError)
	}

	log.Logger.V(0).Info("Successfully removed files", "count", len(jobs))
	return nil
}

// worker processes removal jobs
func (pool *RemovalWorkerPool) worker() {
	defer pool.wg.Done()
	
	for job := range pool.jobs {
		var err error
		
		if job.isDir {
			err = os.RemoveAll(job.path)
			if err != nil {
				err = fmt.Errorf("error removing directory %s: %w", job.path, err)
			}
		} else {
			err = os.Remove(job.path)
			if err != nil {
				err = fmt.Errorf("error removing file %s: %w", job.path, err)
			}
		}
		
		pool.errors <- err
	}
}
