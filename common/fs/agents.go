package fs

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	commonlogger "github.com/odigos-io/odigos/common/logger"
)

const (
	chrootDir        = "/host"
	semanagePath     = "/sbin/semanage"
	restoreconPath   = "/sbin/restorecon"
	keeplistPath     = "/tmp/keeplist"
	rsyncDefaultPath = "rsync"
)

func CopyAgentsDirectoryToHost(srcDir, dstDir string, optionalRsyncPath *string) error {
	logger := commonlogger.LoggerCompat().With("subsystem", "agents")
	startTime := time.Now()
	empty, err := isDirEmptyOrNotExist(dstDir)
	if err != nil {
		return fmt.Errorf("failed to inspect destination: %w", err)
	}

	if empty {
		// if empty, we can just copy the directory to the host
		logger.Info("Odigos agents directory is empty, copying agents directory to host")
		err = CopyDirectories(srcDir, dstDir, nil)
		if err != nil {
			logger.Error("Error copying instrumentation directory to host", "err", err)
			return err
		}
	} else {
		logger.Info("Odigos agents directory is not empty, syncing files with rsync")
		criticalFiles := getCriticalFiles(dstDir)
		updatedFilesToKeepMap, err := removeChangedFilesFromKeepMap(criticalFiles, srcDir, dstDir)

		if err != nil {
			logger.Error("Error getting changed files", "err", err)
			return fmt.Errorf("failed to prepare critical files keeplist: %w", err)
		}

		if err := writeKeeplist(dstDir, keeplistPath, updatedFilesToKeepMap); err != nil {
			logger.Error("failed to write keeplist", "err", err)
			return err
		}

		if err := runSingleRsyncSync(srcDir, dstDir, keeplistPath, optionalRsyncPath); err != nil {
			logger.Error("rsync failed", "err", err)
			return err
		}
	}

	logger.Info("Odigos agents directory copied to host", "elapsed", time.Since(startTime))

	return nil
}

// ApplyOpenShiftSELinuxSettings makes auto-instrumentation agents readable by containers on RHEL hosts.
// Note: This function calls chroot to use the host's PATH to execute selinux commands. Calling it will
// affect the odiglet running process's apparent filesystem.
func ApplyOpenShiftSELinuxSettings(dstDir string) error {
	// Check if the semanage command exists when running on RHEL/CoreOS
	logger := commonlogger.LoggerCompat().With("subsystem", "agents")
	logger.Info("Applying selinux settings to host")
	_, err := exec.LookPath(filepath.Join(chrootDir, semanagePath))
	if err == nil {
		err = syscall.Chroot(chrootDir)
		if err != nil {
			logger.Error("Error chrooting to host", "err", err)
		}

		// list existing semanage rules to check if Odigos has already been set
		cmd := exec.CommandContext(context.Background(), semanagePath, "fcontext", "-l")
		var out bytes.Buffer
		cmd.Stdout = &out

		err := cmd.Run()
		if err != nil {
			logger.Error("Error executing semanage", "err", err)
			return err
		}

		pattern := regexp.MustCompile(`/var/odigos(\(/.\*\)\?)?\s+.*container_ro_file_t`)
		if pattern.Match(out.Bytes()) {
			logger.Info("Rule for /var/odigos already exists with container_ro_file_t.")
			return nil
		}

		// Run the semanage command to add the new directory to the container_ro_file_t context
		// semanage writes SELinux config to host
		cmd = exec.CommandContext(context.Background(), semanagePath, "fcontext", "-a", "-t", "container_ro_file_t", "/var/odigos(/.*)?")
		stdoutBytes, err := cmd.CombinedOutput()
		if err != nil {
			logger.Error("Error running semanage command", "err", err, "stdout", string(stdoutBytes))
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
			cmd := exec.CommandContext(context.Background(), restoreconPath, "-r", dstDir)
			err = cmd.Run()
			if err != nil {
				logger.Error("Error running restorecon command", "err", err)
				return err
			}
		} else {
			logger.Error("Unable to find restorecon path", "err", err)
			return err
		}
	} else {
		logger.Info("Unable to find semanage path, possibly not on RHEL host")
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
	defer func() {
		if err := f.Close(); err != nil {
			commonlogger.LoggerCompat().Error("Error closing file", "err", err)
		}
	}()
	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}

func removeChangedFilesFromKeepMap(filesToKeepMap map[string]struct{}, srcDir string, dstDir string) (map[string]struct{}, error) {
	logger := commonlogger.LoggerCompat().With("subsystem", "agents")
	updatedFilesToKeepMap := make(map[string]struct{})

	for dstPath := range filesToKeepMap {
		// Convert destination path to source path
		srcPath := strings.Replace(dstPath, dstDir, srcDir, 1)

		// Find and preserve existing hash version files for this base file
		existingHashVersionFiles, err := findHashVersionFiles(dstPath)
		if err != nil {
			logger.Error("Error finding existing hash version files", "err", err, "basePath", dstPath)
			return nil, fmt.Errorf("error finding existing hash version files for %s: %w", dstPath, err)
		} else {
			// Add all existing hash version files to the keep map
			for _, hashVersionFile := range existingHashVersionFiles {
				updatedFilesToKeepMap[hashVersionFile] = struct{}{}
				logger.Info("Preserving existing hash version file", "file", hashVersionFile)
			}
		}

		// If either file doesn't exist, mark as changed and remove from filesToKeepMap
		_, dstErr := os.Stat(dstPath)
		_, srcErr := os.Stat(srcPath)

		if dstErr != nil || srcErr != nil {
			logger.Info("File marked for recreate (missing)", "file", dstPath)
			continue
		}

		// Compare file hashes
		dstHash, err := fileHash(dstPath)
		if err != nil {
			return nil, fmt.Errorf("error calculating hash for destination file %s: %v", dstPath, err)
		}

		srcHash, err := fileHash(srcPath)
		if err != nil {
			return nil, fmt.Errorf("error calculating hash for source file %s: %v", srcPath, err)
		}

		// If the hashes are different, keep the old version of the file in the destination with the new name <ORIGINAL_FILE_NAME_{12_CHARS_OF_HASH}>
		// and ensure the renamed file is added to filesToKeepMap to protect it from deletion.
		if dstHash != srcHash {
			newDstPath, err := renameWithHashSuffix(dstPath, dstHash)
			if err != nil {
				return nil, fmt.Errorf("error renaming file: %v", err)
			}

			updatedFilesToKeepMap[newDstPath] = struct{}{}

			continue // original file is renamed, recreate dstPath and keep newDstPath
		}

		updatedFilesToKeepMap[dstPath] = struct{}{}
	}

	return updatedFilesToKeepMap, nil
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
func writeKeeplist(dstDir, file string, keeps map[string]struct{}) error {
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Close(); err != nil {
			commonlogger.LoggerCompat().Error("Error closing file", "err", err)
		}
	}()

	w := bufio.NewWriter(f)
	for hostPath := range keeps {
		// Convert absolute path to relative path for rsync exclude pattern
		relativePath := strings.TrimPrefix(hostPath, dstDir+"/")
		_, _ = fmt.Fprintln(w, relativePath) // ignore error
	}
	return w.Flush()
}

// runSingleRsyncSync performs a single-threaded rsync from srcDir to dstDir using the given exclude file.
// This is used when the destination already contains files and we want to sync changes while keeping versioned files.
func runSingleRsyncSync(srcDir, dstDir, excludeFile string, optionalRsyncPath *string) error {
	logger := commonlogger.LoggerCompat().With("subsystem", "agents")
	// rsync flags:
	// -a: archive mode (preserves permissions, symlinks, modification times, etc.)
	// -v: verbose output (shows which files were copied)
	// --numeric-ids: keep UID/GID numeric; avoids getpwuid/getgrgid NSS lookups
	//   which crash the statically-linked bundled rsync (NSS dlopen on static glibc)
	// --delete: removes files in dstDir that are not in srcDir (clean sync)
	// --whole-file: disables delta-transfer algorithm (lower CPU, better for local copying)
	// --inplace: update files in-place without temp files (avoids disk pressure)
	// --exclude-from: skip deleting or overwriting files listed in keeplist.txt
	rsyncPath := rsyncDefaultPath
	if optionalRsyncPath != nil {
		rsyncPath = *optionalRsyncPath
	}
	args := []string{
		"-av", "--numeric-ids", "--delete", "--whole-file", "--inplace",
		fmt.Sprintf("--exclude-from=%s", excludeFile),
		srcDir + "/", dstDir + "/",
	}

	cmd := exec.CommandContext(context.Background(), rsyncPath, args...)
	var _, stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		logger.Error("rsync failed", "err", err, "stderr", stderr.String())
		return err
	}

	logger.Info("rsync completed")
	return nil
}

// criticalFiles lists paths relative to the agents directory root that must be
// preserved during upgrades because they may be memory-mapped by running processes.
// The path is relative to the srcDir.
func getCriticalFiles(bp string) map[string]struct{} {
	cf := make(map[string]struct{})
	cf[filepath.Join(bp, "nodejs-ebpf", "build", "Release", "dtrace-injector-native.node")] = struct{}{}
	cf[filepath.Join(bp, "nodejs-ebpf", "build", "Release", "obj.target", "dtrace-injector-native.node")] = struct{}{}
	cf[filepath.Join(bp, "nodejs-ebpf", "build", "Release", ".deps", "Release", "dtrace-injector-native.node.d")] = struct{}{}
	cf[filepath.Join(bp, "nodejs-ebpf", "build", "Release", ".deps", "Release", "obj.target", "dtrace-injector-native.node.d")] = struct{}{}
	cf[filepath.Join(bp, "java-ebpf", "tracing_probes.so")] = struct{}{}
	cf[filepath.Join(bp, "java-ext-ebpf", "end_span_usdt.so")] = struct{}{}
	cf[filepath.Join(bp, "java-ext-ebpf", "javaagent.jar")] = struct{}{}
	cf[filepath.Join(bp, "java-ext-ebpf", "otel_agent_extension.jar")] = struct{}{}
	cf[filepath.Join(bp, "python-ebpf", "pythonUSDT.abi3.so")] = struct{}{}
	cf[filepath.Join(bp, "loader", "loader.so")] = struct{}{}
	// Python dependency shared objects - special handling:
	// These shared objects (.so files) are loaded by Python processes and mapped into process memory.
	// They cannot be replaced while loaded, so we must keep them in the host filesystem to avoid removal.
	// These files are versioned and renamed when their respective library versions change.
	cf[filepath.Join(bp, "python", "google", "_upb", "_message.abi3.so")] = struct{}{}                     // Google protobuf library
	cf[filepath.Join(bp, "python", "wrapt", "_wrappers.cpython-311-aarch64-linux-gnu.so")] = struct{}{}    // Wrapt library on arm64
	cf[filepath.Join(bp, "python", "wrapt", "_wrappers.cpython-311-x86_64-linux-gnu.so")] = struct{}{}     // Wrapt library on x86_64
	cf[filepath.Join(bp, "python3.8", "google", "_upb", "_message.abi3.so")] = struct{}{}                  // Google protobuf library [python 3.8 distro]
	cf[filepath.Join(bp, "python3.8", "wrapt", "_wrappers.cpython-311-aarch64-linux-gnu.so")] = struct{}{} // Wrapt library on arm64 [python 3.8 distro]
	cf[filepath.Join(bp, "python3.8", "wrapt", "_wrappers.cpython-311-x86_64-linux-gnu.so")] = struct{}{}  // Wrapt library on x86_64 [python 3.8 distro]
	// PHP native extension loaded by the PHP runtime via dlopen().
	// Must be preserved during upgrades to avoid crashing running PHP-FPM processes.
	cf[filepath.Join(bp, "php", "8.1", "opentelemetry.so")] = struct{}{}
	cf[filepath.Join(bp, "php", "8.2", "opentelemetry.so")] = struct{}{}
	cf[filepath.Join(bp, "php", "8.3", "opentelemetry.so")] = struct{}{}
	cf[filepath.Join(bp, "php", "8.4", "opentelemetry.so")] = struct{}{}
	return cf
}
