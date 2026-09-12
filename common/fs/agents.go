package fs

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/unix"

	commonlogger "github.com/odigos-io/odigos/common/logger"
)

const (
	selinuxXattr = "security.selinux"
	// index of "type" in a "user:role:type:level" context
	typeField = 2
	// what RHEL policy gives files under /var, and containers may not read it
	varFileType = "var_t"
	// what container-selinux gives files that any container may read. We set this on
	// the files only, so anything that relabels the node afterwards undoes it, and
	// the agents stay unreadable until the init container runs again.
	agentsFileType   = "container_ro_file_t"
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

// ApplyOpenShiftSELinuxSettings makes the agent files readable by instrumented
// containers, and reports how many it relabeled. A file's SELinux context is
// "user:role:type:level", and only the type decides who may read it.
// It also reports the type the directory had on entry, which explains the count.
func ApplyOpenShiftSELinuxSettings(dstDir string) (int, string, error) {
	root, err := fileContext(dstDir)
	if err != nil {
		return 0, "", err
	}
	rootType := contextType(root)
	// nothing to do on a node without SELinux, which gives no context at all, or
	// under a policy that gives the agents some type we don't expect:
	// - varFileType is what the root gets by default on OpenShift
	// - agentsFileType is what we put and might be here from a previous run
	if rootType != varFileType && rootType != agentsFileType {
		return 0, rootType, nil
	}

	relabeled := 0
	err = filepath.WalkDir(dstDir, func(path string, _ fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		current, err := fileContext(path)
		if err != nil {
			return err
		}
		// not everything under a var_t directory is var_t: a named type transition
		// gives a directory called "debug" or "man" a type its contents inherit
		if t := contextType(current); t == "" || t == agentsFileType {
			return nil
		}
		fields := strings.Split(current, ":")
		fields[typeField] = agentsFileType

		if err := unix.Lsetxattr(path, selinuxXattr, []byte(strings.Join(fields, ":")), 0); err != nil {
			if errors.Is(err, unix.EINVAL) {
				// the node's policy has no such type; nothing was written
				return fmt.Errorf("SELinux policy does not define %s, needed to make %s readable by containers", agentsFileType, path)
			}
			return fmt.Errorf("labeling %s: %w", path, err)
		}
		relabeled++
		return nil
	})
	return relabeled, rootType, err
}

// fileContext returns the file's SELinux context, or "" when it has none.
// Buffer handling follows go-selinux: ask the kernel for the size only if the
// fixed buffer turns out to be too small.
// https://github.com/opencontainers/selinux/blob/v1.12.0/go-selinux/xattrs_linux.go
func fileContext(path string) (string, error) {
	buf := make([]byte, 128)
	n, err := unix.Lgetxattr(path, selinuxXattr, buf)
	for errors.Is(err, unix.ERANGE) {
		n, err = unix.Lgetxattr(path, selinuxXattr, nil)
		if err != nil {
			break
		}
		buf = make([]byte, n)
		n, err = unix.Lgetxattr(path, selinuxXattr, buf)
	}
	if err != nil {
		// no SELinux on this node
		if errors.Is(err, unix.ENODATA) || errors.Is(err, unix.ENOTSUP) {
			return "", nil
		}
		return "", fmt.Errorf("reading SELinux context of %s: %w", path, err)
	}
	return string(bytes.TrimRight(buf[:n], "\x00")), nil
}

func contextType(c string) string {
	fields := strings.Split(c, ":")
	if len(fields) <= typeField {
		return ""
	}
	return fields[typeField]
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
	// Community Java agent jar, attached via -javaagent at JVM startup. The agent reads its
	// own classfiles out of the jar lazily, at request time, so rewriting the jar in place
	// under a live JVM breaks instrumentation ("Failed to read classfile URL") until the pod
	// restarts. The enterprise agent jar below is preserved for the same reason.
	cf[filepath.Join(bp, "java", "javaagent.jar")] = struct{}{}
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
