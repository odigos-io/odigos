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

	commonlogger "github.com/odigos-io/odigos/common/logger"
)

const (
	// we don't want to overload the CPU so we limit to small number of goroutines
	workersPerCPU = 4

	// 32 KB buffer for I/O operations
	bufferSize = 32 * 1024

	directoryTraversalMode os.FileMode = 0755
)

// getNumberOfWorkers determines the number of concurrent workers to use for copying files.
// It is based on GOMAXPROCS, which reflects the effective CPU limit set for the init container.
// The returned value is calculated as GOMAXPROCS * workersPerCPU, where workersPerCPU is a heuristic multiplier representing the desired concurrency level per CPU unit.
func getNumberOfWorkers() int {
	return max(1, runtime.GOMAXPROCS(0)*workersPerCPU)
}

func copyDirectories(srcDir string, destDir string) error {
	logger := commonlogger.LoggerCompat().With("subsystem", "agentscopy")
	start := time.Now()

	files, err := getFiles(srcDir)
	if err != nil {
		return err
	}

	// Create the destination directory if it doesn't exist
	err = os.MkdirAll(destDir, directoryTraversalMode)
	if err != nil {
		return err
	}

	// Create a buffered channel to control concurrency
	numWorkers := getNumberOfWorkers()
	fileChan := make(chan string, numWorkers)
	var wg sync.WaitGroup

	// Start worker goroutines
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(fileChan, srcDir, destDir, &wg)
	}

	// Send files to the channel
	for _, file := range files {
		fileChan <- file
	}

	// Close the channel and wait for workers to finish
	close(fileChan)
	wg.Wait()
	logger.Info("Finished copying instrumentation files to host", "duration", time.Since(start))
	return ensureDirectoryTraversalPermissions(destDir)
}

func createDotnetDeprecatedDirectories(destDir string) error {

	var err error

	arch := getArch()
	dotnetSoFile := "OpenTelemetry.AutoInstrumentation.Native.so"
	glibcDir := filepath.Join(destDir, "linux-glibc")
	muslDir := filepath.Join(destDir, "linux-musl")
	glibcDirWithArch := filepath.Join(destDir, "linux-glibc-"+arch)
	muslDirWithArch := filepath.Join(destDir, "linux-musl-"+arch)

	err = os.MkdirAll(glibcDirWithArch, directoryTraversalMode)
	if err != nil {
		return err
	}
	err = os.MkdirAll(muslDirWithArch, directoryTraversalMode)
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

func worker(fileChan <-chan string, sourceDir, destDir string, wg *sync.WaitGroup) {
	defer wg.Done()
	logger := commonlogger.LoggerCompat().With("subsystem", "agentscopy")

	// Allocate a buffer once per goroutine.
	buf := make([]byte, bufferSize)
	for file := range fileChan {
		destFile := filepath.Join(destDir, file[len(sourceDir)+1:])
		err := copyFile(file, destFile, buf)
		if err != nil {
			logger.Error("Failed to copy file", "err", err, "file", file)
		}
	}
}

func getFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			files = append(files, path)
		}
		return nil
	})

	return files, err
}

func copyFile(src, dst string, buf []byte) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Create destination file and directories if needed
	err = os.MkdirAll(filepath.Dir(dst), directoryTraversalMode)
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

func ensureDirectoryTraversalPermissions(root string) error {
	if err := ensureDirectoryTraversalPermission(root); err != nil {
		return err
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if err := ensureDirectoryTraversalPermissions(filepath.Join(root, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func ensureDirectoryTraversalPermission(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return nil
	}

	mode := info.Mode().Perm()
	fixedMode := mode
	if mode&0400 != 0 {
		fixedMode |= 0100
	}
	if mode&0040 != 0 {
		fixedMode |= 0010
	}
	if mode&0004 != 0 {
		fixedMode |= 0001
	}
	if fixedMode == mode {
		return nil
	}
	return os.Chmod(dir, fixedMode)
}

func HostContainsEbpfDir(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if entry.IsDir() && strings.Contains(entry.Name(), "ebpf") {
			return true
		}
	}

	return false
}

func getArch() string {
	if runtime.GOARCH == "arm64" {
		return "arm64"
	}

	return "x64"
}
