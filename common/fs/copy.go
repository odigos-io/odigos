package fs

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	commonlogger "github.com/odigos-io/odigos/common/logger"
)

// criticalFiles lists paths relative to the agents directory root that must be
// preserved during upgrades because they may be memory-mapped by running processes.
var criticalFiles = map[string]struct{}{
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
	// Python dependency shared objects - special handling:
	// These shared objects (.so files) are loaded by Python processes and mapped into process memory.
	// They cannot be replaced while loaded, so we must keep them in the host filesystem to avoid removal.
	// These files are versioned and renamed when their respective library versions change.
	"/var/odigos/python/google/_upb/_message.abi3.so":                        {}, // Google protobuf library
	"/var/odigos/python/wrapt/_wrappers.cpython-311-aarch64-linux-gnu.so":    {}, // Wrapt library on arm64
	"/var/odigos/python/wrapt/_wrappers.cpython-311-x86_64-linux-gnu.so":     {}, // Wrapt library on x86_64
	"/var/odigos/python3.8/google/_upb/_message.abi3.so":                     {}, // Google protobuf library [python 3.8 distro]
	"/var/odigos/python3.8/wrapt/_wrappers.cpython-311-aarch64-linux-gnu.so": {}, // Wrapt library on arm64 [python 3.8 distro]
	"/var/odigos/python3.8/wrapt/_wrappers.cpython-311-x86_64-linux-gnu.so":  {}, // Wrapt library on x86_64 [python 3.8 distro]
	// PHP native extension loaded by the PHP runtime via dlopen().
	// Must be preserved during upgrades to avoid crashing running PHP-FPM processes.
	"/var/odigos/php/8.1/opentelemetry.so": {},
	"/var/odigos/php/8.2/opentelemetry.so": {},
	"/var/odigos/php/8.3/opentelemetry.so": {},
	"/var/odigos/php/8.4/opentelemetry.so": {},
}

func CopyDirectories(srcDir, dstDir string, excludes map[string]bool) error {
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return fmt.Errorf("create destination dir: %w", err)
	}

	// Track every relative path we see in srcDir so we can delete extras.
	srcPaths := make(map[string]bool)
	err := filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		dst := filepath.Join(dstDir, rel)

		if d.IsDir() {
			srcPaths[rel] = true
			return os.MkdirAll(dst, 0o755)
		}

		srcPaths[rel] = true

		if excludes[rel] {
			return nil
		}

		srcInfo, err := d.Info()
		if err != nil {
			return err
		}
		if fileUnchanged(srcInfo, dst) {
			return nil
		}

		return copyFile(path, dst, srcInfo)
	})
	if err != nil {
		return fmt.Errorf("walking source: %w", err)
	}

	// Delete files/dirs in dstDir that aren't in srcDir and aren't excluded.
	// Errors are intentionally swallowed — this is a best-effort cleanup pass.
	// Individual entry failures should not abort the entire cleanup walk.
	var toRemove []string
	_ = filepath.WalkDir(dstDir, func(path string, d fs.DirEntry, walkErr error) error { //nolint:errcheck // best-effort cleanup
		if walkErr != nil {
			return nil //nolint:nilerr // skip inaccessible entries, continue walking
		}
		rel, relErr := filepath.Rel(dstDir, path)
		if relErr != nil || rel == "." {
			return nil //nolint:nilerr // skip entries with path errors
		}
		if excludes[rel] {
			return nil
		}
		if !srcPaths[rel] {
			toRemove = append(toRemove, path)
			if d.IsDir() {
				return filepath.SkipDir
			}
		}
		return nil
	})

	for i := len(toRemove) - 1; i >= 0; i-- {
		if err := os.RemoveAll(toRemove[i]); err != nil {
			commonlogger.LoggerCompat().Error("Error removing file", "err", err, "file", toRemove[i])
		}
	}

	return nil
}

// fileUnchanged reports whether dst exists and matches srcInfo in size and
// modification time, meaning the file can be skipped during sync.
func fileUnchanged(srcInfo os.FileInfo, dst string) bool {
	dstInfo, err := os.Lstat(dst)
	if err != nil {
		return false
	}
	return dstInfo.Size() == srcInfo.Size() && dstInfo.ModTime().Equal(srcInfo.ModTime())
}

func copyFile(src, dst string, srcInfo os.FileInfo) error {
	if srcInfo == nil {
		var err error
		srcInfo, err = os.Lstat(src)
		if err != nil {
			return err
		}
	}

	if srcInfo.Mode()&fs.ModeSymlink != 0 {
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		_ = os.Remove(dst)
		return os.Symlink(target, dst)
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := in.Close(); err != nil {
			commonlogger.LoggerCompat().Error("Error closing file", "err", err, "file", src)
		}
	}()

	dir := filepath.Dir(dst)
	tmp, err := os.CreateTemp(dir, ".odigos-sync-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	if _, err := io.Copy(tmp, in); err != nil {
		if err := tmp.Close(); err != nil {
			commonlogger.LoggerCompat().Error("Error closing file", "err", err, "file", tmpName)
		}
		if err := os.Remove(tmpName); err != nil {
			commonlogger.LoggerCompat().Error("Error removing file", "err", err, "file", tmpName)
		}
		return err
	}
	if err := tmp.Close(); err != nil {
		if err := os.Remove(tmpName); err != nil {
			commonlogger.LoggerCompat().Error("Error removing file", "err", err, "file", tmpName)
		}
		return err
	}
	if err := os.Chmod(tmpName, srcInfo.Mode().Perm()); err != nil {
		if err := os.Remove(tmpName); err != nil {
			commonlogger.LoggerCompat().Error("Error removing file", "err", err, "file", tmpName)
		}
		return err
	}
	if err := os.Rename(tmpName, dst); err != nil {
		if err := os.Remove(tmpName); err != nil {
			commonlogger.LoggerCompat().Error("Error removing file", "err", err, "file", tmpName)
		}
		return err
	}
	// Preserve source mtime so subsequent syncs can skip unchanged files via
	// size+mtime comparison, avoiding unnecessary I/O.
	mtime := srcInfo.ModTime()
	return os.Chtimes(dst, mtime, mtime)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func ProcessCriticalFiles(files map[string]struct{}, stagingDir, targetDir string) (map[string]bool, error) {
	excludes := make(map[string]bool)

	for relPath := range files {
		targetPath := filepath.Join(targetDir, relPath)
		stagingPath := filepath.Join(stagingDir, relPath)

		// Preserve all existing hash-versioned files for this base file.
		hvFiles, _ := findHashVersionFiles(targetPath)
		for _, hvf := range hvFiles {
			rel, err := filepath.Rel(targetDir, hvf)
			if err == nil {
				excludes[rel] = true
			}
		}

		targetExists := fileExists(targetPath)
		stagingExists := fileExists(stagingPath)

		if !targetExists || !stagingExists {
			continue
		}

		targetHash, err := fileHash(targetPath)
		if err != nil {
			return nil, fmt.Errorf("hash target %s: %w", relPath, err)
		}
		stagingHash, err := fileHash(stagingPath)
		if err != nil {
			return nil, fmt.Errorf("hash staging %s: %w", relPath, err)
		}

		if targetHash == stagingHash {
			// Unchanged -- exclude from sync to avoid unnecessary I/O.
			excludes[relPath] = true
			continue
		}

		// Changed -- rename old version so running processes keep their
		// memory-mapped file, then let sync copy the new version.
		renamed, err := renameWithHashSuffix(targetPath, targetHash)
		if err != nil {
			return nil, fmt.Errorf("rename critical file %s: %w", relPath, err)
		}
		if renamed != "" {
			rel, err := filepath.Rel(targetDir, renamed)
			if err == nil {
				excludes[rel] = true
			}
		}
	}

	return excludes, nil
}

func renameWithHashSuffix(filePath, hash string) (string, error) {
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("stat %s: %w", filePath, err)
	}

	suffix := hash
	if len(suffix) > 12 {
		suffix = suffix[:12]
	}

	dir := filepath.Dir(filePath)
	filename := filepath.Base(filePath)
	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, ext)

	newName := fmt.Sprintf("%s_hash_version-%s%s", base, suffix, ext)
	newPath := filepath.Join(dir, newName)

	if err := os.Rename(filePath, newPath); err != nil {
		return "", fmt.Errorf("rename %s -> %s: %w", filePath, newPath, err)
	}
	return newPath, nil
}

func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := f.Close(); err != nil {
			commonlogger.LoggerCompat().Error("Error closing file", "err", err, "file", path)
		}
	}()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func findHashVersionFiles(filePath string) ([]string, error) {
	dir := filepath.Dir(filePath)
	filename := filepath.Base(filePath)
	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, ext)

	pattern := filepath.Join(dir, base+"_hash_version-*"+ext)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	return matches, nil
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

func getArch() string {
	if runtime.GOARCH == "arm64" {
		return "arm64"
	}

	return "x64"
}
