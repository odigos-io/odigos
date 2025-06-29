package fs

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
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

func removeChangedFilesFromKeepMap(filesToKeepMap map[string]struct{}, containerDir string, hostDir string) (map[string]struct{}, error) {

	updatedFilesToKeepMap := make(map[string]struct{})

	for hostPath := range filesToKeepMap {
		// Convert host path to container path
		containerPath := strings.Replace(hostPath, hostDir, containerDir, 1)

		// If either file doesn't exist, mark as changed and remove from filesToKeepMap
		_, hostErr := os.Stat(hostPath)
		_, containerErr := os.Stat(containerPath)

		if hostErr != nil || containerErr != nil {
			log.Logger.V(0).Info("File marked for recreate (missing)", "file", hostPath)
			continue
		}

		// Compare file hashes
		hostHash, err := calculateFileHash(hostPath)
		if err != nil {
			return nil, fmt.Errorf("error calculating hash for host file %s: %v", hostPath, err)
		}

		containerHash, err := calculateFileHash(containerPath)
		if err != nil {
			return nil, fmt.Errorf("error calculating hash for container file %s: %v", containerPath, err)
		}

		// If the hashes are different, keep the old version of the file in the host with the new name <ORIGINAL_FILE_NAME_{12_CHARS_OF_HASH}>
		// and ensure the renamed file is added to filesToKeepMap to protect it from deletion.
		if hostHash != containerHash {
			newHostPath, err := renameFileWithHashSuffix(hostPath, hostHash)
			if err != nil {
				return nil, fmt.Errorf("error renaming file: %v", err)
			}

			updatedFilesToKeepMap[newHostPath] = struct{}{}

			continue // original file is renamed, recreate hostPath and keep NewHostPath
		}

		updatedFilesToKeepMap[hostPath] = struct{}{}
	}

	return updatedFilesToKeepMap, nil
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
