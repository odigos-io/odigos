package fs

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"syscall"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/odiglet/pkg/log"
)

const (
	containerDir   = "/instrumentations"
	chrootDir      = "/host"
	semanagePath   = "/sbin/semanage"
	restoreconPath = "/sbin/restorecon"
)

// FileInfo holds essential file information for quick comparison
type FileInfo struct {
	Path    string
	Size    int64
	ModTime int64
	Exists  bool
}

// getFileInfo efficiently gets file information without full stat
func getFileInfo(path string) FileInfo {
	info, err := os.Stat(path)
	if err != nil {
		return FileInfo{Path: path, Exists: false}
	}
	return FileInfo{
		Path:    path,
		Size:    info.Size(),
		ModTime: info.ModTime().Unix(),
		Exists:  true,
	}
}

// fastFileCompare does a quick comparison using size and mtime before expensive hash
func fastFileCompare(hostPath, containerPath string) (bool, error) {
	hostInfo := getFileInfo(hostPath)
	containerInfo := getFileInfo(containerPath)

	// If either doesn't exist, they're different
	if !hostInfo.Exists || !containerInfo.Exists {
		return false, nil
	}

	// Quick comparison - if size or modtime differ, files are different
	if hostInfo.Size != containerInfo.Size {
		return false, nil
	}

	// If sizes match, use fast CRC32 instead of SHA-256 for final comparison
	return fastHashCompare(hostPath, containerPath)
}

// fastHashCompare uses CRC32 which is much faster than SHA-256
func fastHashCompare(hostPath, containerPath string) (bool, error) {
	hostHash, err := calculateFastHash(hostPath)
	if err != nil {
		return false, err
	}

	containerHash, err := calculateFastHash(containerPath)
	if err != nil {
		return false, err
	}

	return hostHash == containerHash, nil
}

// calculateFastHash uses CRC32 which is much faster than SHA-256
func calculateFastHash(filePath string) (uint32, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	hasher := crc32.NewIEEE()
	// Use a larger buffer for better I/O performance
	buf := make([]byte, 64*1024)
	for {
		n, err := file.Read(buf)
		if n > 0 {
			hasher.Write(buf[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
	}

	return hasher.Sum32(), nil
}

func CopyAgentsDirectoryToHost() error {
	// remove the current content of /var/odigos
	// as we want a fresh copy of instrumentation agents with no files leftover from previous odigos versions.
	// we cannot remove /var/odigos itself: "unlinkat /var/odigos: device or resource busy"
	// so we will just remove it's content

	// We kept the following list of files to avoid removing instrumentations that are already loaded in the process memory
	filesToKeep := map[string]struct{}{
		"/var/odigos/nodejs-ebpf/build/Release/dtrace-injector-native.node":                            {},
		"/var/odigos/nodejs-ebpf/build/Release/obj.target/dtrace-injector-native.node":                 {},
		"/var/odigos/nodejs-ebpf/build/Release/.deps/Release/dtrace-injector-native.node.d":            {},
		"/var/odigos/nodejs-ebpf/build/Release/.deps/Release/obj.target/dtrace-injector-native.node.d": {},
		"/var/odigos/java-ebpf/tracing_probes.so":                                                      {},
		"/var/odigos/java-ext-ebpf/end_span_usdt.so":                                                   {},
		"/var/odigos/python-ebpf/pythonUSDT.abi3.so":                                                   {},
	}

	updatedFilesToKeepMap, err := removeChangedFilesFromKeepMap(filesToKeep, containerDir, k8sconsts.OdigosAgentsDirectory)
	if err != nil {
		log.Logger.Error(err, "Error getting changed files")
	}

	err = removeFilesInDir(k8sconsts.OdigosAgentsDirectory, updatedFilesToKeepMap)
	if err != nil {
		log.Logger.Error(err, "Error removing instrumentation directory from host")
		return err
	}

	err = copyDirectories(containerDir, k8sconsts.OdigosAgentsDirectory, updatedFilesToKeepMap)
	if err != nil {
		log.Logger.Error(err, "Error copying instrumentation directory to host")
		return err
	}

	// temporary workaround for dotnet.
	// dotnet used to have directories containing the arch suffix (linux-glibc-arm64).
	// this works will with virtual device that knows the arch it is running on.
	// however, the webhook cannot know in advance which arch the pod is going to run on.
	// thus, the directory names are renamed so they do not contain the arch suffix (linux-glibc)
	// which can be used by the webhook.
	// The following link is a temporary support for the deprecated dotnet virtual devices.
	// TODO: remove this once we delete the virtual devices.
	err = createDotnetDeprecatedDirectories(path.Join(k8sconsts.OdigosAgentsDirectory, "dotnet"))
	if err != nil {
		log.Logger.Error(err, "Error creating dotnet deprecated directories")
		return err
	}

	return nil
}

// WorkerPool manages parallel file processing
type WorkerPool struct {
	workers    int
	jobs       chan string
	results    chan fileComparisonResult
	wg         sync.WaitGroup
	mu         sync.Mutex
	keepMap    map[string]struct{}
	containerDir string
	hostDir    string
}

type fileComparisonResult struct {
	hostPath    string
	containerPath string
	keep        bool
	newPath     string
	err         error
}

func removeChangedFilesFromKeepMap(filesToKeepMap map[string]struct{}, containerDir string, hostDir string) (map[string]struct{}, error) {
	// Use worker pool for parallel processing
	numWorkers := runtime.NumCPU()
	if numWorkers > len(filesToKeepMap) {
		numWorkers = len(filesToKeepMap)
	}

	wp := &WorkerPool{
		workers:      numWorkers,
		jobs:         make(chan string, len(filesToKeepMap)),
		results:      make(chan fileComparisonResult, len(filesToKeepMap)),
		keepMap:      make(map[string]struct{}),
		containerDir: containerDir,
		hostDir:      hostDir,
	}

	// Start workers
	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.worker()
	}

	// Send jobs
	go func() {
		for hostPath := range filesToKeepMap {
			wp.jobs <- hostPath
		}
		close(wp.jobs)
	}()

	// Wait for workers to finish
	go func() {
		wp.wg.Wait()
		close(wp.results)
	}()

	// Collect results
	updatedFilesToKeepMap := make(map[string]struct{})
	for result := range wp.results {
		if result.err != nil {
			return nil, result.err
		}
		if result.keep {
			updatedFilesToKeepMap[result.hostPath] = struct{}{}
		}
		if result.newPath != "" {
			updatedFilesToKeepMap[result.newPath] = struct{}{}
		}
	}

	return updatedFilesToKeepMap, nil
}

func (wp *WorkerPool) worker() {
	defer wp.wg.Done()
	
	for hostPath := range wp.jobs {
		result := fileComparisonResult{hostPath: hostPath}
		
		// Convert host path to container path
		containerPath := strings.Replace(hostPath, wp.hostDir, wp.containerDir, 1)
		result.containerPath = containerPath

		// Use fast comparison instead of expensive SHA-256
		filesMatch, err := fastFileCompare(hostPath, containerPath)
		if err != nil {
			result.err = fmt.Errorf("error comparing files %s and %s: %v", hostPath, containerPath, err)
			wp.results <- result
			continue
		}

		if !filesMatch {
			// Only calculate SHA-256 for renaming when files actually differ
			hostHash, err := calculateFileHash(hostPath)
			if err != nil {
				result.err = fmt.Errorf("error calculating hash for host file %s: %v", hostPath, err)
				wp.results <- result
				continue
			}

			newHostPath, err := renameFileWithHashSuffix(hostPath, hostHash)
			if err != nil {
				result.err = fmt.Errorf("error renaming file: %v", err)
				wp.results <- result
				continue
			}

			result.newPath = newHostPath
			log.Logger.V(0).Info("File marked for recreate (changed)", "file", hostPath)
		} else {
			result.keep = true
		}

		wp.results <- result
	}
}

// Helper function to rename a file using the first 12 characters of its hash
func renameFileWithHashSuffix(originalPath, fileHash string) (string, error) {
	// Extract the first 12 characters of the hash
	hashSuffix := fileHash[:12]

	newPath := generateRenamedFilePath(originalPath, hashSuffix)

	if err := os.Rename(originalPath, newPath); err != nil {
		return "", fmt.Errorf("failed to rename file %s to %s: %w", originalPath, newPath, err)
	}

	log.Logger.V(0).Info("File successfully renamed", "oldPath", originalPath, "newPath", newPath)
	return newPath, nil
}

// Construct a renamed file path
func generateRenamedFilePath(originalPath, hashSuffix string) string {
	ext := filepath.Ext(originalPath)                                 // Get the file extension (e.g., ".so")
	base := strings.TrimSuffix(originalPath, ext)                     // Remove the extension from the original path
	return fmt.Sprintf("%s_hash_version-%s%s", base, hashSuffix, ext) // Append the hash and add back the extension
}

// calculateFileHash computes the SHA-256 hash of a file (kept for compatibility where SHA-256 is required)
func calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	// Use larger buffer for better I/O performance
	buf := make([]byte, 64*1024)
	for {
		n, err := file.Read(buf)
		if n > 0 {
			hasher.Write(buf[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// ApplyOpenShiftSELinuxSettings makes auto-instrumentation agents readable by containers on RHEL hosts.
// Note: This function calls chroot to use the host's PATH to execute selinux commands. Calling it will
// affect the odiglet running process's apparent filesystem.
func ApplyOpenShiftSELinuxSettings() error {
	// Check if the semanage command exists when running on RHEL/CoreOS
	log.Logger.Info("Applying selinux settings to host")
	_, err := exec.LookPath(filepath.Join(chrootDir, semanagePath))
	if err == nil {
		syscall.Chroot(chrootDir)

		// list existing semanage rules to check if Odigos has already been set
		cmd := exec.Command(semanagePath, "fcontext", "-l")
		var out bytes.Buffer
		cmd.Stdout = &out

		err := cmd.Run()
		if err != nil {
			log.Logger.Error(err, "Error executing semanage")
			return err
		}

		pattern := regexp.MustCompile(`/var/odigos(\(/.\*\)\?)?\s+.*container_ro_file_t`)
		if pattern.Match(out.Bytes()) {
			log.Logger.Info("Rule for /var/odigos already exists with container_ro_file_t.")
			return nil
		}

		// Run the semanage command to add the new directory to the container_ro_file_t context
		// semanage writes SELinux config to host
		cmd = exec.Command(semanagePath, "fcontext", "-a", "-t", "container_ro_file_t", "/var/odigos(/.*)?")
		stdoutBytes, err := cmd.CombinedOutput()
		if err != nil {
			log.Logger.Error(err, "Error running semanage command", "stdout", string(stdoutBytes))
			if strings.Contains(string(stdoutBytes), "already defined") {
				// some versions of selinux return an error when trying to set fcontext where it already exists
				// if that's the case, we don't need to return an error here
				return nil
			}
			return err
		}

		// Check if the restorecon command exists when running on RHEL/CoreOS
		// restorecon applies the SELinux settings we just created to the host
		// And we are already chrooted to the host path, so we can just look for restoreconPath now
		_, err = exec.LookPath(restoreconPath)
		if err == nil {
			// Run the restorecon command to apply the new context
			cmd := exec.Command(restoreconPath, "-r", k8sconsts.OdigosAgentsDirectory)
			err = cmd.Run()
			if err != nil {
				log.Logger.Error(err, "Error running restorecon command")
				return err
			}
		} else {
			log.Logger.Error(err, "Unable to find restorecon path")
			return err
		}
	} else {
		log.Logger.Info("Unable to find semanage path, possibly not on RHEL host")
	}
	return nil
}
