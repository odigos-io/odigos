package fs

import (
	"bufio"
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
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/odiglet/pkg/log"
)

const (
	containerDir   = "/instrumentations"
	chrootDir      = "/host"
	semanagePath   = "/sbin/semanage"
	restoreconPath = "/sbin/restorecon"
	keeplistPath   = "/tmp/keeplist"
)

func CopyAgentsDirectoryToHost() error {

	startTime := time.Now()
	// We kept the following list of files to avoid removing instrumentations that are already loaded in the process memory
	filesToKeep := map[string]struct{}{
		"/var/odigos/nodejs-ebpf/build/Release/dtrace-injector-native.node":                            {},
		"/var/odigos/nodejs-ebpf/build/Release/obj.target/dtrace-injector-native.node":                 {},
		"/var/odigos/nodejs-ebpf/build/Release/.deps/Release/dtrace-injector-native.node.d":            {},
		"/var/odigos/nodejs-ebpf/build/Release/.deps/Release/obj.target/dtrace-injector-native.node.d": {},
		"/var/odigos/java-ebpf/tracing_probes.so":                                                      {},
		"/var/odigos/java-ext-ebpf/end_span_usdt.so":                                                   {},
		"/var/odigos/java-ext-ebpf/javaagent.jar":                                                      {},
		"/var/odigos/java-ext-ebpf/otel_agent_extension.jar":                                           {},
		"/var/odigos/python-ebpf/pythonUSDT.abi3.so":                                                   {},
		"/var/odigos/loader/loader.so":                                                                 {},
		// Google protobuf library shared object loaded by Python processes.
		// This file gets mapped into process memory and cannot be replaced while loaded.
		// Therefore, we need to keep this file in the host filesystem to avoid removing it.
		// This file is versioned and renamed if changed (python protobuf library version changes).
		"/var/odigos/python/google/_upb/_message.abi3.so": {},
	}
	empty, err := isDirEmptyOrNotExist(k8sconsts.OdigosAgentsDirectory)
	if err != nil {
		return fmt.Errorf("failed to inspect destination: %w", err)
	}

	if empty {
		// if empty, we can just copy the directory to the host
		log.Logger.Info("Odigos agents directory is empty, copying agents directory to host")
		err = copyDirectories(containerDir, k8sconsts.OdigosAgentsDirectory)
		if err != nil {
			log.Logger.Error(err, "Error copying instrumentation directory to host")
			return err
		}
	} else {
		log.Logger.Info("Odigos agents directory is not empty, syncing files with rsync")
		updatedFilesToKeepMap, err := removeChangedFilesFromKeepMap(filesToKeep, containerDir, k8sconsts.OdigosAgentsDirectory)

		if err != nil {
			log.Logger.Error(err, "Error getting changed files")
		}

		if err := writeKeeplist(keeplistPath, updatedFilesToKeepMap); err != nil {
			log.Logger.Error(err, "failed to write keeplist")
			return err
		}

		if err := runSingleRsyncSync(containerDir, k8sconsts.OdigosAgentsDirectory, keeplistPath); err != nil {
			log.Logger.Error(err, "rsync failed")
			return err
		}
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

	log.Logger.Info("Odigos agents directory copied to host", "elapsed", time.Since(startTime))

	return nil
}

func removeChangedFilesFromKeepMap(filesToKeepMap map[string]struct{}, containerDir string, hostDir string) (map[string]struct{}, error) {

	updatedFilesToKeepMap := make(map[string]struct{})

	for hostPath := range filesToKeepMap {
		// Convert host path to container path
		containerPath := strings.Replace(hostPath, hostDir, containerDir, 1)

		// Find and preserve existing hash version files for this base file
		existingHashVersionFiles, err := findExistingHashVersionFiles(hostPath)
		if err != nil {
			log.Logger.Error(err, "Error finding existing hash version files", "basePath", hostPath)
		} else {
			// Add all existing hash version files to the keep map
			for _, hashVersionFile := range existingHashVersionFiles {
				updatedFilesToKeepMap[hashVersionFile] = struct{}{}
				log.Logger.V(0).Info("Preserving existing hash version file", "file", hashVersionFile)
			}
		}

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

// findExistingHashVersionFiles searches for existing files with _hash_version pattern
// for the given base file path. For example, if basePath is "/var/odigos/python-ebpf/pythonUSDT.abi3.so",
// it will search for files like "/var/odigos/python-ebpf/pythonUSDT.abi3_hash_version-*.so"
func findExistingHashVersionFiles(basePath string) ([]string, error) {
	// Extract directory and base filename
	dir := filepath.Dir(basePath)
	ext := filepath.Ext(basePath)
	baseNameWithoutExt := strings.TrimSuffix(filepath.Base(basePath), ext)

	// Create the pattern to search for: basefilename_hash_version-*
	pattern := baseNameWithoutExt + "_hash_version-*" + ext

	// Read directory contents
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			// Directory doesn't exist, return empty slice
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	var matchingFiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Check if filename matches our pattern
		matched, err := filepath.Match(pattern, entry.Name())
		if err != nil {
			log.Logger.Error(err, "Error matching pattern", "pattern", pattern, "filename", entry.Name())
			continue
		}

		if matched {
			fullPath := filepath.Join(dir, entry.Name())
			matchingFiles = append(matchingFiles, fullPath)
		}
	}

	return matchingFiles, nil
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

func isDirEmptyOrNotExist(dir string) (bool, error) {
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	if !info.IsDir() {
		return false, fmt.Errorf("not a directory: %s", dir)
	}
	f, err := os.Open(dir)
	if err != nil {
		return false, err
	}
	defer f.Close()
	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}

// writeKeeplist creates an exclude file for rsync with relative paths.
// rsync --exclude-from expects patterns relative to the source directory, not absolute paths.
// Since we're syncing to /var/odigos, we need to convert absolute paths like:
//
//	/var/odigos/python-ebpf/pythonUSDT.abi3_hash_version-e3b0c44298fc.so
//
// to relative patterns like:
//
//	python-ebpf/pythonUSDT.abi3_hash_version-e3b0c44298fc.so
//
// This ensures the --delete flag won't remove files we want to keep.
func writeKeeplist(file string, keeps map[string]struct{}) error {
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for hostPath := range keeps {
		// Convert absolute path to relative path for rsync exclude pattern
		relativePath := strings.TrimPrefix(hostPath, k8sconsts.OdigosAgentsDirectory+"/")
		fmt.Fprintln(w, relativePath)
	}
	return w.Flush()
}

// runSingleRsyncSync performs a single-threaded rsync from srcDir to dstDir using the given exclude file.
// This is used when the destination already contains files and we want to sync changes while keeping versioned files.
func runSingleRsyncSync(srcDir, dstDir, excludeFile string) error {
	// rsync flags:
	// -a: archive mode (preserves permissions, symlinks, modification times, etc.)
	// -v: verbose output (shows which files were copied)
	// --delete: removes files in dstDir that are not in srcDir (clean sync)
	// --whole-file: disables delta-transfer algorithm (lower CPU, better for local copying)
	// --inplace: update files in-place without temp files (avoids disk pressure)
	// --exclude-from: skip deleting or overwriting files listed in keeplist.txt
	args := []string{
		"-av", "--delete", "--whole-file", "--inplace",
		fmt.Sprintf("--exclude-from=%s", excludeFile),
		srcDir + "/", dstDir + "/",
	}

	cmd := exec.Command("rsync", args...)
	var _, stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Logger.Error(err, "rsync failed", "stderr", stderr.String())
		return err
	}

	log.Logger.V(0).Info("rsync completed")
	return nil
}
