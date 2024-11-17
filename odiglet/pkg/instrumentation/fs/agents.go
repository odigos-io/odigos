package fs

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/odigos-io/odigos/odiglet/pkg/log"
)

const (
	containerDir   = "/instrumentations"
	hostDir        = "/var/odigos"
	chrootDir      = "/host"
	semanagePath   = "/sbin/semanage"
	restoreconPath = "/sbin/restorecon"
)

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

	err := removeChangedFilesFromKeepMap(filesToKeep, containerDir, hostDir)
	if err != nil {
		log.Logger.Error(err, "Error getting changed files")
	}

	err = removeFilesInDir(hostDir, filesToKeep)
	if err != nil {
		log.Logger.Error(err, "Error removing instrumentation directory from host")
		return err
	}

	err = copyDirectories(containerDir, hostDir, filesToKeep)
	if err != nil {
		log.Logger.Error(err, "Error copying instrumentation directory to host")
		return err
	}

	// Check if the semanage command exists when running on RHEL/CoreOS
	_, err = exec.LookPath(filepath.Join(chrootDir, semanagePath))
	if err == nil {
		// Run the semanage command to add the new directory to the container_ro_file_t context
		cmd := exec.Command(semanagePath, "fcontext", "-a", "-t", "container_ro_file_t", "/var/odigos(/.*)?")
		syscall.Chroot(chrootDir)
		err = cmd.Run()
		if err != nil {
			log.Logger.Error(err, "Error running semanage command")
		}
	}

	// Check if the restorecon command exists when running on RHEL/CoreOS
	_, err = exec.LookPath(filepath.Join(chrootDir, restoreconPath))
	if err == nil {
		// Run the restorecon command to apply the new context
		cmd := exec.Command(restoreconPath, "-r", "/var/odigos")
		syscall.Chroot(chrootDir)
		err = cmd.Run()
		if err != nil {
			log.Logger.Error(err, "Error running restorecon command")
		}
	}

	return nil
}

func removeChangedFilesFromKeepMap(filesToKeepMap map[string]struct{}, containerDir string, hostDir string) error {
	for hostPath := range filesToKeepMap {
		// Convert host path to container path
		containerPath := strings.Replace(hostPath, hostDir, containerDir, 1)

		// Check if both files exist
		hostInfo, hostErr := os.Stat(hostPath)
		containerInfo, containerErr := os.Stat(containerPath)

		// If either file doesn't exist, mark as changed and remove from filesToKeepMap
		if hostErr != nil || containerErr != nil {
			delete(filesToKeepMap, hostPath)
			log.Logger.V(0).Info("File marked for deletion (missing)", "file", hostPath)
			continue
		}

		// If sizes are different, mark as changed
		if hostInfo.Size() != containerInfo.Size() {
			delete(filesToKeepMap, hostPath)
			log.Logger.V(0).Info("File marked for deletion (size mismatch)", "file", hostPath)
			continue
		}

		// Compare file hashes
		hostHash, err := calculateFileHash(hostPath)
		if err != nil {
			return fmt.Errorf("error calculating hash for host file %s: %v", hostPath, err)
		}

		containerHash, err := calculateFileHash(containerPath)
		if err != nil {
			return fmt.Errorf("error calculating hash for container file %s: %v", containerPath, err)
		}

		fmt.Println("hostHash: ", hostHash, " containerHash: ", containerHash)

		// If hashes are different, mark as changed
		if hostHash != containerHash {
			delete(filesToKeepMap, hostPath)
			log.Logger.V(0).Info("File marked for deletion (content mismatch)", "file", hostPath)
		}
	}

	return nil
}

// calculateFileHash computes the SHA-256 hash of a file
func calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}
